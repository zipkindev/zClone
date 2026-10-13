"""Recovery controller tests using disposable data and an injected object store."""
import importlib.util
import pathlib
import os
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location('recover', pathlib.Path(__file__).parents[1] / 'zclone-recover.py')
recover = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(recover)


class Store:
    def __init__(self):
        self.objects = {}
        self.uploads = []
        self.fail_after_upload = False

    def upload(self, source, remote):
        self.objects[remote] = pathlib.Path(source).read_bytes()
        self.uploads.append(remote)
        if self.fail_after_upload:
            self.fail_after_upload = False
            raise OSError('connection lost after server accepted object')

    def read(self, remote, consume):
        if remote not in self.objects:
            raise OSError('missing object')
        consume(self.objects[remote])


class RecoveryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = pathlib.Path(self.temp.name)
        self.source = self.root / 'source.img'
        self.data = bytes(range(256)) * 37 + b'end'
        self.source.write_bytes(self.data)
        self.args = recover.parser().parse_args([
            'upload', str(self.source), 'remote:session',
            '--journal', str(self.root / 'state' / 'journal.jsonl'),
            '--stage-dir', str(self.root / 'stage'), '--chunk-size', '1024'])
        self.store = Store()
        silence = mock.patch('sys.stderr')
        silence.start()
        self.addCleanup(silence.stop)

    def run_upload(self):
        recover.upload(self.args, self.store)

    def test_restore_round_trip(self):
        self.run_upload()
        out = self.root / 'restored.img'
        args = recover.parser().parse_args(['restore', self.store.uploads[-1], str(out)])
        recover.restore(args, self.store)
        self.assertEqual(out.read_bytes(), self.data)
        self.assertEqual(list((self.root / 'stage').iterdir()), [])
        with self.assertRaises((ValueError, FileExistsError)):
            recover.restore(args, self.store)

    def test_reconcile_upload_before_checkpoint(self):
        self.store.fail_after_upload = True
        with self.assertRaises(OSError):
            self.run_upload()
        first = self.store.uploads[0]
        self.run_upload()
        self.assertEqual(self.store.uploads.count(first), 1)

    def test_reboot_loses_stage_but_preserves_uploaded_bytes(self):
        self.store.fail_after_upload = True
        with self.assertRaises(OSError):
            self.run_upload()
        for p in (self.root / 'stage').iterdir():
            p.unlink()
        first = self.store.uploads[0]
        self.run_upload()
        self.assertEqual(self.store.uploads.count(first), 1)

    def test_repair_corrupt_upload_from_stage(self):
        self.store.fail_after_upload = True
        with self.assertRaises(OSError):
            self.run_upload()
        self.store.objects[self.store.uploads[0]] = b'broken'
        self.run_upload()
        self.assertEqual(self.store.uploads.count(self.store.uploads[0]), 2)

    def test_torn_journal_tail(self):
        self.store.fail_after_upload = True
        with self.assertRaises(OSError):
            self.run_upload()
        with open(self.args.journal, 'ab') as f:
            f.write(b'{"incomplete":')
        self.run_upload()
        self.run_upload()

    def test_corrupt_journal_fails_closed(self):
        self.run_upload()
        p = pathlib.Path(self.args.journal)
        p.write_bytes(p.read_bytes().replace(b'prepared', b'prepXred', 1))
        with self.assertRaises(ValueError):
            self.run_upload()

    def test_changed_source_rejected(self):
        self.run_upload()
        self.source.write_bytes(b'x' * len(self.data))
        with self.assertRaises(ValueError):
            self.run_upload()

    def test_restore_rejects_wrong_raw_hash(self):
        import json
        self.run_upload()
        manifest_path = self.store.uploads[-1]
        manifest = json.loads(self.store.objects[manifest_path])
        manifest['chunks'][0]['raw_sha256'] = '0' * 64
        self.store.objects[manifest_path] = json.dumps(manifest).encode()
        out = self.root / 'bad-raw.img'
        args = recover.parser().parse_args(['restore', manifest_path, str(out)])
        with self.assertRaises(ValueError):
            recover.restore(args, self.store)
        self.assertFalse(out.exists())

    def test_restore_rejects_missing_range(self):
        import json
        self.run_upload()
        manifest_path = self.store.uploads[-1]
        manifest = json.loads(self.store.objects[manifest_path])
        manifest['chunks'].pop(0)
        self.store.objects[manifest_path] = json.dumps(manifest).encode()
        out = self.root / 'gap.img'
        args = recover.parser().parse_args(['restore', manifest_path, str(out)])
        with self.assertRaises(ValueError):
            recover.restore(args, self.store)
        self.assertFalse(out.exists())

    def test_short_source_never_prepared(self):
        with mock.patch.object(recover.os, 'pread', return_value=b''):
            with self.assertRaises(OSError):
                self.run_upload()
        self.assertEqual(self.store.uploads, [])

    def test_restore_corrupt_chunk_does_not_publish_output(self):
        self.run_upload()
        self.store.objects[self.store.uploads[0]] = b'bad gzip'
        out = self.root / 'restored.img'
        args = recover.parser().parse_args(['restore', self.store.uploads[-1], str(out)])
        with self.assertRaises((ValueError, OSError, EOFError)):
            recover.restore(args, self.store)
        self.assertFalse(out.exists())

    def test_concurrent_writer_rejected(self):
        with recover.Journal(self.args.journal):
            with self.assertRaises((OSError, ValueError)):
                self.run_upload()

    def test_full_staging_storage_does_not_upload(self):
        with mock.patch.object(recover.gzip.GzipFile, 'write', side_effect=OSError('no space left')):
            with self.assertRaises(OSError):
                self.run_upload()
        self.assertEqual(self.store.uploads, [])
        self.assertEqual(list((self.root / 'stage').iterdir()), [])

    def test_verification_failure_keeps_staged_bytes(self):
        upload = self.store.upload
        def corrupt(source, remote):
            upload(source, remote)
            self.store.objects[remote] = b'corrupt'
        with mock.patch.object(self.store, 'upload', corrupt):
            with self.assertRaises(ValueError):
                self.run_upload()
        self.assertEqual(len(list((self.root / 'stage').glob('*.gz'))), 1)
        self.run_upload()

    def test_failed_verified_checkpoint_keeps_staging(self):
        append = recover.Journal.append
        def fail(journal, event):
            if event.get('state') == 'verified':
                raise OSError('journal storage full')
            append(journal, event)
        with mock.patch.object(recover.Journal, 'append', fail):
            with self.assertRaises(OSError):
                self.run_upload()
        self.assertEqual(len(list((self.root / 'stage').glob('*.gz'))), 1)
        first = self.store.uploads[0]
        self.run_upload()
        self.assertEqual(self.store.uploads.count(first), 1)

    def test_resume_cleans_verified_stage_without_source_read(self):
        append = recover.Journal.append
        def fail_after_verified(journal, event):
            append(journal, event)
            if event.get('state') == 'verified':
                raise OSError('crash after durable verification')
        with mock.patch.object(recover.Journal, 'append', fail_after_verified):
            with self.assertRaises(OSError):
                self.run_upload()
        self.run_upload()
        self.assertEqual(list((self.root / 'stage').iterdir()), [])


class LocalZcloneTests(unittest.TestCase):
    def setUp(self):
        self.binary = pathlib.Path(os.environ.get('ZCLONE_RECOVERY_TEST_BINARY',
            str(pathlib.Path(__file__).parents[2] / 'build' / 'zclone'))).resolve()
        if not self.binary.is_file():
            self.skipTest('set ZCLONE_RECOVERY_TEST_BINARY to a built zClone executable')
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = pathlib.Path(self.temp.name)
        self.source = self.root / 'source.img'
        self.data = os.urandom(131072) + b'compressible' * 10000
        self.source.write_bytes(self.data)
        self.args = ['upload', str(self.source), str(self.root / 'remote'),
                     '--journal', str(self.root / 'state' / 'journal.jsonl'),
                     '--stage-dir', str(self.root / 'stage'), '--chunk-size', '65536',
                     '--zclone', str(self.binary), '--config', '/notfound']

    def test_sigkill_at_each_checkpoint_and_restore(self):
        script = str(pathlib.Path(__file__).parents[1] / 'zclone-recover.py')
        for state in ('staging', 'prepared', 'uploaded', 'verified'):
            with self.subTest(state=state):
                # Each kill point uses an independent recovery session.
                journal = self.root / ('journal-' + state)
                args = list(self.args)
                args[args.index('--journal') + 1] = str(journal)
                code = """
import importlib.util, os, signal, sys
spec = importlib.util.spec_from_file_location('recover', sys.argv[1])
r = importlib.util.module_from_spec(spec)
spec.loader.exec_module(r)
append = r.Journal.append
def checkpoint(journal, event):
    append(journal, event)
    if event.get('state') == sys.argv[2]:
        os.kill(os.getpid(), signal.SIGKILL)
r.Journal.append = checkpoint
if sys.argv[2] == 'staging':
    write = r.gzip.GzipFile.write
    def interrupted_write(compressor, data):
        write(compressor, data)
        os.kill(os.getpid(), signal.SIGKILL)
    r.gzip.GzipFile.write = interrupted_write
r.upload(r.parser().parse_args(sys.argv[3:]))
"""
                killed = subprocess.run([sys.executable, '-c', code, script, state] + args,
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                self.assertEqual(killed.returncode, -9, killed.stderr.decode())
                objects = list((self.root / 'remote' / 'chunks').glob('*.gz')) if (self.root / 'remote' / 'chunks').exists() else []
                timestamps = {p: p.stat().st_mtime_ns for p in objects}
                # A reboot would also discard tmpfs staging.
                for p in (self.root / 'stage').iterdir():
                    p.unlink()
                resumed = subprocess.run([sys.executable, script] + args,
                                         stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                self.assertEqual(resumed.returncode, 0, resumed.stderr.decode())
                self.assertEqual({p: p.stat().st_mtime_ns for p in objects}, timestamps)
                import json
                header = json.loads(journal.read_text().splitlines()[0])['event']
                manifest = self.root / 'remote' / 'manifests' / (header['session'] + '.json')
                output = self.root / ('restored-' + state)
                restored = subprocess.run([sys.executable, script, 'restore', str(manifest), str(output),
                                           '--zclone', str(self.binary), '--config', '/notfound'],
                                          stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                self.assertEqual(restored.returncode, 0, restored.stderr.decode())
                self.assertEqual(output.read_bytes(), self.data)


if __name__ == '__main__':
    unittest.main()

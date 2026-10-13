#!/usr/bin/env python3
"""Checkpointed, independently compressed image uploads using zClone (POSIX)."""
import argparse
import contextlib
import fcntl
import gzip
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import uuid

BLOCK = 1024 * 1024
VERSION = 1


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode('utf-8')


def make_directory(path):
    missing = []
    current = path
    while not current.exists():
        missing.append(current)
        current = current.parent
    for directory in reversed(missing):
        directory.mkdir(exist_ok=True)
        sync_dir(directory.parent)


def sync_dir(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


class Journal:
    """An exclusively locked, fsynced JSON journal with a checksum chain."""
    def __init__(self, path):
        self.path = Path(path)
        self.events = []
        self.digest = '0' * 64

    def __enter__(self):
        make_directory(self.path.parent)
        fd = os.open(self.path, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
        self.file = os.fdopen(fd, 'r+b')
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            sync_dir(self.path.parent)
            while True:
                start = self.file.tell()
                line = self.file.readline()
                if not line:
                    break
                if not line.endswith(b'\n'):
                    self.file.truncate(start)
                    self.file.seek(start)
                    self.file.flush()
                    os.fsync(fd)
                    break
                try:
                    row = json.loads(line)
                    digest = hashlib.sha256(self.digest.encode() + encode(row['event'])).hexdigest()
                    if row['sha256'] != digest:
                        raise ValueError('checksum mismatch')
                    self.events.append(row['event'])
                    self.digest = digest
                except (ValueError, KeyError, TypeError) as e:
                    raise ValueError('invalid recovery journal; refusing to resume') from e
            return self
        except BaseException:
            self.file.close()
            raise

    def append(self, event):
        digest = hashlib.sha256(self.digest.encode() + encode(event)).hexdigest()
        self.file.write(encode({'event': event, 'sha256': digest}) + b'\n')
        self.file.flush()
        os.fsync(self.file.fileno())
        self.digest = digest
        self.events.append(event)

    def __exit__(self, *_):
        self.file.close()


class Transport:
    """Runs existing zClone commands without shell interpolation."""
    def __init__(self, args):
        self.command = [args.zclone]
        if args.config:
            self.command += ['--config', args.config]
        self.retries = args.retries

    def upload(self, source, remote):
        subprocess.run(self.command + ['copyto', str(source), remote, '--ignore-times',
                                      '--retries', str(self.retries)], check=True)

    def read(self, remote, consume):
        with subprocess.Popen(self.command + ['cat', remote], stdout=subprocess.PIPE) as process:
            try:
                while True:
                    data = process.stdout.read(BLOCK)
                    if not data:
                        break
                    consume(data)
            except BaseException:
                process.kill()
                process.wait()
                raise
            if process.wait() != 0:
                raise OSError('zclone could not read ' + remote)


def remote_path(destination, name):
    return destination.rstrip('/') + ('' if destination.endswith(':') else '/') + name


def chunk_name(header, chunk):
    return 'chunks/{}-{:012d}-{}.gz'.format(header['session'], chunk['index'], chunk['compressed_sha256'])


def stage_path(stage, header, index):
    return stage / '{}-{:012d}.gz'.format(header['session'], index)


def identity(fd, source, args):
    info = os.fstat(fd)
    result = {'path': str(Path(source).resolve()), 'dev': info.st_dev,
              'ino': info.st_ino, 'rdev': info.st_rdev, 'source_id': args.source_id}
    if stat.S_ISREG(info.st_mode):
        size = info.st_size
        if args.size is not None and args.size != size:
            raise ValueError('--size must match the regular file size')
        result.update(size=size, mtime_ns=info.st_mtime_ns, ctime_ns=info.st_ctime_ns)
    elif stat.S_ISBLK(info.st_mode):
        if not args.source_id or args.size is None:
            raise ValueError('block devices require --source-id and an exact --size')
        size = args.size
        if os.lseek(fd, 0, os.SEEK_END) != size:
            raise ValueError('--size does not match block device capacity')
        result.pop('ino')
        result.pop('dev')
        result['size'] = size
    else:
        raise ValueError('source must be a regular file or block device, not a pipe')
    if size <= 0:
        raise ValueError('source size must be positive')
    return result


def verify(store, remote, expected_size, expected_hash):
    digest = hashlib.sha256()
    size = 0
    def consume(data):
        nonlocal size
        size += len(data)
        if size > expected_size:
            raise ValueError('remote object exceeds recorded size: ' + remote)
        digest.update(data)
    store.read(remote, consume)
    if size != expected_size or digest.hexdigest() != expected_hash:
        raise ValueError('remote content verification failed: ' + remote)


def hash_file(path):
    digest = hashlib.sha256()
    size = 0
    with open(path, 'rb') as f:
        while data := f.read(BLOCK):
            size += len(data)
            digest.update(data)
    return size, digest.hexdigest()


def prepare(fd, path, index, offset, length):
    digest = hashlib.sha256()
    remaining = length
    limit = length + length // 100 + BLOCK
    temp_fd, temp_name = tempfile.mkstemp(prefix=path.name + '-', suffix='.partial', dir=path.parent)
    try:
        with os.fdopen(temp_fd, 'wb') as output:
            with gzip.GzipFile(filename='', mode='wb', fileobj=output, mtime=0, compresslevel=6) as compressor:
                while remaining:
                    data = os.pread(fd, min(BLOCK, remaining), offset + length - remaining)
                    if not data:
                        raise OSError('unexpected end of source at offset {}'.format(offset + length - remaining))
                    digest.update(data)
                    compressor.write(data)
                    remaining -= len(data)
                    if output.tell() > limit:
                        raise OSError('compressed chunk exceeded staging limit')
            output.flush()
            os.fsync(output.fileno())
        size, compressed_hash = hash_file(temp_name)
        if size > limit:
            raise OSError('compressed chunk exceeded staging limit')
        os.replace(temp_name, path)
        sync_dir(path.parent)
        return {'index': index, 'offset': offset, 'length': length,
                'raw_sha256': digest.hexdigest(), 'compressed_size': size,
                'compressed_sha256': compressed_hash}
    finally:
        with contextlib.suppress(FileNotFoundError):
            os.unlink(temp_name)


def validate_chunk(chunk, index, offset, length):
    if (chunk['index'], chunk['offset'], chunk['length']) != (index, offset, length):
        raise ValueError('invalid chunk layout')
    if not isinstance(chunk['compressed_size'], int) or not 0 < chunk['compressed_size'] <= length + length // 100 + BLOCK:
        raise ValueError('invalid compressed chunk size')
    for key in ('raw_sha256', 'compressed_sha256'):
        value = chunk[key]
        if not isinstance(value, str) or len(value) != 64 or any(c not in '0123456789abcdef' for c in value):
            raise ValueError('invalid chunk checksum')


def upload(args, store=None):
    if args.chunk_size <= 0 or args.retries <= 0:
        raise ValueError('chunk size and retries must be positive')
    stage = Path(args.stage_dir).resolve()
    make_directory(stage)
    journal_path = Path(args.journal).resolve()
    if journal_path.is_relative_to(stage):
        raise ValueError('journal must be outside the staging directory, on persistent storage')
    store = store or Transport(args)
    with open(args.source, 'rb', buffering=0) as source, Journal(args.journal) as journal:
        source_identity = identity(source.fileno(), args.source, args)
        if not journal.events:
            journal.append({'version': VERSION, 'session': uuid.uuid4().hex,
                            'source': source_identity, 'destination': args.destination,
                            'chunk_size': args.chunk_size, 'compression': 'gzip-level-6-mtime-0'})
        header = journal.events[0]
        if (header.get('version'), header.get('source'), header.get('destination'), header.get('chunk_size')) != (
                VERSION, source_identity, args.destination, args.chunk_size):
            raise ValueError('source, destination or chunk settings differ from journal')
        states = {}
        previous_index = -1
        for event in journal.events[1:]:
            index = event['chunk']['index']
            previous = states.get(index)
            valid = ((event['state'] == 'prepared' and previous is None and index == previous_index + 1 and
                      (previous_index < 0 or states[previous_index]['state'] == 'verified')) or
                     (previous is not None and event['chunk'] == previous['chunk'] and
                      (previous['state'], event['state']) in (('prepared', 'uploaded'), ('uploaded', 'verified'), ('prepared', 'verified'))))
            if not valid:
                raise ValueError('invalid recovery journal state transition')
            states[index] = event
            previous_index = index
        chunks = []
        total = source_identity['size']
        for index, offset in enumerate(range(0, total, args.chunk_size)):
            length = min(args.chunk_size, total - offset)
            path = stage_path(stage, header, index)
            # Only this session's interrupted preparations are disposable.
            for partial in stage.glob(path.name + '-*.partial'):
                partial.unlink()
            event = states.get(index)
            reconcile = event is not None
            if event:
                chunk = event['chunk']
                validate_chunk(chunk, index, offset, length)
            else:
                chunk = prepare(source.fileno(), path, index, offset, length)
                if identity(source.fileno(), args.source, args) != source_identity:
                    raise ValueError('source changed during staging')
                journal.append({'state': 'prepared', 'chunk': chunk})
                event = {'state': 'prepared', 'chunk': chunk}
            if event['state'] != 'verified':
                remote = remote_path(args.destination, chunk_name(header, chunk))
                needs_upload = True
                if reconcile:
                    try:
                        verify(store, remote, chunk['compressed_size'], chunk['compressed_sha256'])
                        needs_upload = False
                    except (OSError, ValueError):
                        pass
                if needs_upload:
                    if not path.exists() or hash_file(path) != (chunk['compressed_size'], chunk['compressed_sha256']):
                        regenerated = prepare(source.fileno(), path, index, offset, length)
                        if regenerated != chunk or identity(source.fileno(), args.source, args) != source_identity:
                            raise ValueError('regenerated chunk differs from journal; refusing to replace recovered bytes')
                    store.upload(path, remote)
                    if event['state'] == 'prepared':
                        journal.append({'state': 'uploaded', 'chunk': chunk})
                    verify(store, remote, chunk['compressed_size'], chunk['compressed_sha256'])
                journal.append({'state': 'verified', 'chunk': chunk})
            with contextlib.suppress(FileNotFoundError):
                path.unlink()
            sync_dir(stage)
            chunks.append(chunk)
            print('Verified chunk {} ({}/{})'.format(index, offset + length, total), file=sys.stderr)
        if len(states) > len(chunks):
            raise ValueError('journal contains chunks beyond the source size')
        if identity(source.fileno(), args.source, args) != source_identity:
            raise ValueError('source changed during upload')
        manifest = {'header': header, 'chunks': chunks}
        if len(encode(manifest)) > 64 * BLOCK:
            raise ValueError('manifest exceeds 64 MiB format limit; use larger chunks')
        with tempfile.NamedTemporaryFile(dir=journal_path.parent) as snapshot:
            snapshot.write(encode(manifest))
            snapshot.flush()
            remote = remote_path(args.destination, 'manifests/' + header['session'] + '.json')
            store.upload(snapshot.name, remote)
            size, digest = hash_file(snapshot.name)
            verify(store, remote, size, digest)
        print('Recovery upload complete; verified manifest: ' + remote, file=sys.stderr)


def restore(args, store=None):
    store = store or Transport(args)
    output = Path(args.output).resolve()
    if output.exists():
        raise FileExistsError('restore output already exists')
    manifest_data = bytearray()
    def collect(data):
        if len(manifest_data) + len(data) > 64 * BLOCK:
            raise ValueError('manifest exceeds 64 MiB limit')
        manifest_data.extend(data)
    store.read(args.manifest, collect)
    manifest = json.loads(manifest_data)
    header, chunks = manifest['header'], manifest['chunks']
    if header['version'] != VERSION or header['chunk_size'] <= 0 or header['source']['size'] <= 0:
        raise ValueError('invalid recovery manifest')
    session = header['session']
    if len(session) != 32 or any(c not in '0123456789abcdef' for c in session):
        raise ValueError('invalid session ID')
    offset = 0
    total = header['source']['size']
    for index, chunk in enumerate(chunks):
        length = min(header['chunk_size'], total - offset)
        if length <= 0:
            raise ValueError('manifest contains extra chunks')
        validate_chunk(chunk, index, offset, length)
        offset += length
    if offset != total:
        raise ValueError('manifest does not cover the complete source')
    fd, partial = tempfile.mkstemp(prefix=output.name + '-', suffix='.partial', dir=output.parent)
    try:
        with os.fdopen(fd, 'wb') as target:
            for chunk in chunks:
                with tempfile.TemporaryFile(dir=output.parent) as compressed:
                    digest = hashlib.sha256()
                    size = 0
                    def consume(data):
                        nonlocal size
                        size += len(data)
                        if size > chunk['compressed_size']:
                            raise ValueError('remote chunk exceeds recorded size')
                        digest.update(data)
                        compressed.write(data)
                    store.read(remote_path(header['destination'], chunk_name(header, chunk)), consume)
                    if size != chunk['compressed_size'] or digest.hexdigest() != chunk['compressed_sha256']:
                        raise ValueError('compressed chunk verification failed')
                    compressed.seek(0)
                    raw_hash = hashlib.sha256()
                    remaining = chunk['length']
                    with gzip.GzipFile(fileobj=compressed, mode='rb') as uncompressed:
                        while remaining:
                            data = uncompressed.read(min(BLOCK, remaining))
                            if not data:
                                raise ValueError('decompressed chunk is truncated')
                            raw_hash.update(data)
                            target.write(data)
                            remaining -= len(data)
                        if uncompressed.read(1) or raw_hash.hexdigest() != chunk['raw_sha256']:
                            raise ValueError('decompressed chunk verification failed')
            target.flush()
            os.fsync(target.fileno())
        # Linking publishes a complete file without replacing an existing output.
        os.link(partial, output)
        sync_dir(output.parent)
    finally:
        os.unlink(partial)


def parser():
    result = argparse.ArgumentParser(description=__doc__)
    sub = result.add_subparsers(dest='action', required=True)
    up = sub.add_parser('upload', help='upload or resume an immutable image in verified gzip chunks')
    up.add_argument('source')
    up.add_argument('destination', help='dedicated destination directory for this session')
    up.add_argument('--journal', required=True, help='persistent local journal outside staging storage')
    up.add_argument('--stage-dir', required=True, help='RAM or healthy disk directory for one staged chunk')
    up.add_argument('--chunk-size', type=int, default=64 * BLOCK, help='uncompressed bytes per chunk (default: 64 MiB)')
    up.add_argument('--size', type=int, help='exact device size in bytes; required for block devices')
    up.add_argument('--source-id', help='stable device identity checked by operator, e.g. disk serial')
    down = sub.add_parser('restore', help='verify and restore a complete session to a new local image file')
    down.add_argument('manifest', help='exact remote manifest path printed by upload')
    down.add_argument('output')
    for command in (up, down):
        command.add_argument('--zclone', default='zclone', help='path to the zClone executable')
        command.add_argument('--config', help='zClone configuration path')
        command.add_argument('--retries', type=int, default=3)
    return result


def main():
    args = parser().parse_args()
    try:
        (upload if args.action == 'upload' else restore)(args)
    except (OSError, ValueError, KeyError, TypeError, EOFError, subprocess.CalledProcessError) as e:
        print('Recovery stopped: {}'.format(e), file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        message = ('rerun upload with the same journal to resume' if args.action == 'upload'
                   else 'rerun restore to restart restoration')
        print('Recovery interrupted; ' + message, file=sys.stderr)
        return 130
    return 0


if __name__ == '__main__':
    sys.exit(main())

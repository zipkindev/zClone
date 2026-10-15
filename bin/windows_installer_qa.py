"""Exercise the experimental updater using disposable, checksummed release fixtures."""
import argparse
import ctypes
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time
import zipfile


def allocated_bytes(binary):
    high = ctypes.c_ulong()
    query = ctypes.WinDLL('kernel32', use_last_error=True).GetCompressedFileSizeW
    query.argtypes = [ctypes.c_wchar_p, ctypes.POINTER(ctypes.c_ulong)]
    query.restype = ctypes.c_ulong
    low = query(str(binary), ctypes.byref(high))
    if low == 0xffffffff and ctypes.get_last_error():
        raise ctypes.WinError(ctypes.get_last_error())
    return (high.value << 32) | low


def install_cases(updater, archive, previous, root):
    root.mkdir()
    binary = root / 'zclone.exe'
    sentinel = root / 'rclone.conf'
    sentinel.write_text('# disposable configuration sentinel\n')
    records = []
    environment = dict(os.environ, LOCALAPPDATA=str(root / 'updater-preferences'))

    def apply(source, expected, explicit_destination=True):
        start = time.perf_counter()
        result = subprocess.run([str(updater), *(['--install-dir', str(root)] if explicit_destination else []), '--archive', str(source),
                                 '--sha256', hashlib.sha256(source.read_bytes()).hexdigest()],
                                capture_output=True, text=True, timeout=180, check=True, env=environment)
        entry = json.loads(result.stdout)
        assert hashlib.sha256(binary.read_bytes()).hexdigest() == expected == entry['sha256']
        allocation = allocated_bytes(binary)
        assert allocation < binary.stat().st_size, 'Installed executable lost native compression'
        assert sentinel.read_text() == '# disposable configuration sentinel\n'
        assert not list(root.glob('.zclone-stage-*'))
        records.append({'seconds': time.perf_counter() - start, 'allocated_bytes': allocation,
                        'sha256': expected, 'algorithm': entry['algorithm']})

    def executable_hash(source):
        with zipfile.ZipFile(source) as package:
            name = next(n for n in package.namelist() if n.endswith('/zclone.exe'))
            return hashlib.sha256(package.read(name)).hexdigest()

    old, new = executable_hash(previous), executable_hash(archive)
    assert old != new, 'Update QA must replace different executable bytes'
    apply(previous, old)
    apply(archive, new, explicit_destination=False)
    apply(archive, new, explicit_destination=False)
    preferences = root / 'updater-preferences/zclone-updater/settings.json'
    saved_preferences = preferences.read_bytes()
    bad = subprocess.run([str(updater), '--install-dir', str(root), '--archive', str(previous),
                          '--sha256', '0' * 64], capture_output=True, timeout=180, env=environment)
    assert bad.returncode != 0
    assert preferences.read_bytes() == saved_preferences
    assert hashlib.sha256(binary.read_bytes()).hexdigest() == new
    # A live Windows image must remain running and unchanged if replacement is rejected.
    process = subprocess.Popen([str(binary), '--config', str(sentinel), 'rcd', '--rc-no-auth',
                                '--rc-addr', '127.0.0.1:0'], stdout=subprocess.DEVNULL,
                               stderr=subprocess.DEVNULL)
    try:
        time.sleep(1)
        assert process.poll() is None
        locked = subprocess.run([str(updater), '--install-dir', str(root), '--archive', str(previous),
                                 '--sha256', hashlib.sha256(previous.read_bytes()).hexdigest()],
                                capture_output=True, timeout=180, env=environment)
        assert locked.returncode != 0, 'Updater replaced a running executable'
        assert preferences.read_bytes() == saved_preferences
        assert process.poll() is None
        assert hashlib.sha256(binary.read_bytes()).hexdigest() == new
    finally:
        process.terminate()
        process.wait(timeout=15)
    apply(previous, old)
    apply(archive, new)
    scanners = sorted((Path(os.environ['ProgramData']) / 'Microsoft/Windows Defender/Platform').glob('*/MpCmdRun.exe'))
    if not scanners:
        raise RuntimeError('Defender scanner is unavailable')
    scan = subprocess.run([str(scanners[-1]), '-Scan', '-ScanType', '3', '-File', str(binary),
                           '-DisableRemediation'], capture_output=True, text=True, timeout=180)
    if scan.returncode:
        raise RuntimeError('Installed executable Defender scan failed: ' + scan.stdout + scan.stderr)
    return binary, {'experimental': True, 'default_algorithm': 'XPRESS4K',
                    'installed_defender_scan': 'passed',
                    'updater_bytes': updater.stat().st_size, 'archive_bytes': archive.stat().st_size,
                    'updater_sha256': hashlib.sha256(updater.read_bytes()).hexdigest(),
                    'operations': records, 'allocated_bytes': allocated_bytes(binary),
                    'metadata_bytes': sum(p.stat().st_size for p in (root / '.zclone-releases').rglob('*') if p.is_file()),
                    'tests': ['first install', 'different executable update', 'reinstall',
                              'bad checksum preserves install', 'running image preserves install',
                              'previous version reinstall', 'compression after every replacement',
                              'configuration unchanged', 'remembered destination', 'staging cleanup']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--archive', type=Path, required=True)
    parser.add_argument('--previous-binary', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    with zipfile.ZipFile(args.archive) as source, zipfile.ZipFile(args.output, 'w', zipfile.ZIP_DEFLATED) as output:
        for name in source.namelist():
            data = source.read(name)
            if name.endswith('/zclone.exe'):
                data = args.previous_binary.read_bytes()
            if name.endswith('/build-info.json'):
                info = json.loads(data)
                info['tags'] = ['cmount']
                info['qa_fixture'] = 'previous JSON implementation; not a published release'
                data = json.dumps(info).encode()
            output.writestr(name, data)


if __name__ == '__main__':
    main()

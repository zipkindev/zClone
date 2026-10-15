#!/usr/bin/env python3
"""Compare executable inventories and exercise only disposable local data."""
import argparse
import ctypes
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import socket
import statistics
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import zipfile


def run(binary, config, *args):
    return subprocess.run([str(binary), '--config', str(config), *map(str, args)],
                          check=True, capture_output=True, timeout=120).stdout


def inventory(binary, config):
    providers = json.loads(run(binary, config, 'config', 'providers'))
    commands = {}
    pending = [()]
    while pending:
        command = pending.pop()
        help_text = run(binary, config, *command, '--help').decode()
        commands[' '.join(command)] = sorted(re.findall(r'^\s+(?:-\w,\s+)?(--[\w-]+)', help_text, re.M))
        section = help_text.split('Available Commands:', 1)
        if len(section) == 2:
            listing = section[1].split('Flags:', 1)[0].split('Global Flags:', 1)[0]
            pending += [(*command, name) for name in re.findall(r'^  ([\w-]+)\s+', listing, re.M)
                        if name != 'help']
    return {'providers': providers, 'commands': commands}


def tree(root):
    return {p.relative_to(root).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in root.rglob('*') if p.is_file()}


def port():
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        return sock.getsockname()[1]


def peak_memory(process):
    if os.name == 'nt':
        class Counters(ctypes.Structure):
            _fields_ = [('cb', ctypes.c_uint32), ('faults', ctypes.c_uint32)] + [
                (name, ctypes.c_size_t) for name in ['peak', 'working', 'peak_paged', 'paged',
                                                   'peak_nonpaged', 'nonpaged', 'pagefile', 'peak_pagefile']]
        counters = Counters()
        counters.cb = ctypes.sizeof(counters)
        query = ctypes.WinDLL('psapi', use_last_error=True).GetProcessMemoryInfo
        query.argtypes = [ctypes.c_void_p, ctypes.POINTER(Counters), ctypes.c_uint32]
        query.restype = ctypes.c_int
        if not query(int(process._handle), ctypes.byref(counters), counters.cb):
            raise ctypes.WinError(ctypes.get_last_error())
        return counters.peak
    status = Path('/proc') / str(process.pid) / 'status'
    if status.exists():
        match = re.search(r'^VmHWM:\s+(\d+)', status.read_text(), re.M)
        return int(match[1]) * 1024 if match else None
    return None


def server_check(binary, config, args, url, expected, data=None):
    with tempfile.TemporaryFile() as log:
        process = subprocess.Popen([str(binary), '--config', str(config), *map(str, args)],
                                   stdout=log, stderr=log)
        try:
            deadline = time.monotonic() + 30
            while time.monotonic() < deadline:
                if process.poll() is not None:
                    log.seek(0)
                    raise AssertionError(log.read().decode(errors='replace'))
                try:
                    with urllib.request.urlopen(url, data=data, timeout=2) as response:
                        body = response.read()
                    assert expected in body, (url, body)
                    return peak_memory(process)
                except (urllib.error.URLError, TimeoutError):
                    time.sleep(0.1)
            raise AssertionError('Server did not become ready: ' + url)
        finally:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=10)


def smoke(binary, root, require_mount=False):
    root.mkdir()
    config = root / 'test.conf'
    config.write_text('')
    source, target = root / 'source', root / 'target'
    (source / 'nested').mkdir(parents=True)
    (source / 'empty').write_bytes(b'')
    (source / 'unicode-é.txt').write_bytes(b'temporary QA data\n')
    (source / 'nested' / 'binary').write_bytes(bytes(range(256)) * 512)
    expected = tree(source)
    run(binary, config, 'copy', source, target)
    assert tree(target) == expected
    run(binary, config, 'check', source, target)
    filtered = root / 'filtered'
    run(binary, config, 'copy', source, filtered, '--include', '*.txt', '--exclude', '**')
    assert tree(filtered) == {name: digest for name, digest in expected.items() if name.endswith('.txt')}
    (target / 'unicode-é.txt').write_bytes(b'corrupted disposable data')
    mismatch = subprocess.run([str(binary), '--config', str(config), 'check', str(source), str(target)],
                              capture_output=True, timeout=120)
    assert mismatch.returncode != 0, 'check accepted corrupted data'
    (source / 'unicode-é.txt').write_bytes(b'updated QA data\n')
    (target / 'extra').write_bytes(b'disposable extra file')
    before = tree(target)
    run(binary, config, 'sync', source, target, '--dry-run')
    assert tree(target) == before, 'dry-run modified destination'
    run(binary, config, 'sync', source, target)
    assert tree(target) == tree(source)
    listing = json.loads(run(binary, config, 'lsjson', target, '--recursive', '--files-only'))
    assert {item['Path'] for item in listing} == set(tree(source))
    for suffix in ['zip', 'tar.gz', 'tar.xz', 'tar.zst']:
        archive = root / ('test.' + suffix)
        extracted = root / ('extracted-' + suffix)
        run(binary, config, 'archive', 'create', source, archive)
        run(binary, config, 'archive', 'extract', archive, extracted)
        assert tree(extracted) == tree(source), suffix
    password = run(binary, config, 'obscure', 'disposable-test-password').decode().strip()
    encrypted = root / 'encrypted'
    config.write_text('[qa-crypt]\ntype = crypt\nremote = ' + encrypted.as_posix()
                      + '\npassword = ' + password + '\n')
    run(binary, config, 'copy', source, 'qa-crypt:')
    restored = root / 'decrypted'
    run(binary, config, 'copy', 'qa-crypt:', restored)
    assert tree(restored) == tree(source)
    assert tree(encrypted) != tree(source)
    template = root / 'listing.html'
    template.write_text('QA-TEMPLATE {{.Title}} {{range .Entries}}{{.Leaf}} {{end}}')
    address = '127.0.0.1:' + str(port())
    server_check(binary, config, ['serve', 'http', source, '--addr', address, '--template', template],
                 'http://' + address + '/', b'QA-TEMPLATE')
    address = '127.0.0.1:' + str(port())
    rc_memory = server_check(binary, config, ['rcd', '--rc-addr', address, '--rc-no-auth'],
                 'http://' + address + '/core/version', b'version', data=b'{}')
    gui, api = '127.0.0.1:' + str(port()), '127.0.0.1:' + str(port())
    gui_memory = server_check(binary, config, ['gui', '--no-open-browser', '--no-auth', '--addr', gui, '--api-addr', api],
                 'http://' + gui + '/', b'<html')
    if require_mount:
        mount = root / 'mount'
        with tempfile.TemporaryFile() as log:
            process = subprocess.Popen([str(binary), '--config', str(config), 'mount', str(source),
                                        str(mount), '--vfs-cache-mode', 'writes', '--cache-dir',
                                        str(root / 'cache')], stdout=log, stderr=log)
            try:
                deadline = time.monotonic() + 30
                while not (mount / 'unicode-é.txt').exists():
                    if process.poll() is not None or time.monotonic() > deadline:
                        log.seek(0)
                        raise AssertionError('Mount failed: ' + log.read().decode(errors='replace'))
                    time.sleep(0.1)
                assert (mount / 'unicode-é.txt').read_bytes() == (source / 'unicode-é.txt').read_bytes()
                (mount / 'written').write_bytes(b'mount write test')
                deadline = time.monotonic() + 30
                while not (source / 'written').exists():
                    assert time.monotonic() < deadline, 'Mount write did not reach source'
                    time.sleep(0.1)
                assert (source / 'written').read_bytes() == b'mount write test'
            finally:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=10)
    return {'tests': ['copy', 'check/corruption detection', 'filters', 'sync/dry-run', 'lsjson', 'archive roundtrips', 'crypt roundtrip',
                      'HTTP custom template', 'RC', 'GUI'] + (['mount read/write'] if require_mount else []),
            'rc_peak_working_set_bytes': rc_memory, 'gui_peak_working_set_bytes': gui_memory}


def compact(binary, algorithm, root):
    candidate = root / ('compact-' + algorithm + '.exe')
    shutil.copyfile(binary, candidate)
    helper = Path(__file__).with_name('compress-windows.ps1').resolve()
    command = ['powershell.exe', '-NoLogo', '-NoProfile', '-NonInteractive', '-File', str(helper),
               '-Path', str(candidate)]
    def apply(*options):
        result = subprocess.run(command + list(options), capture_output=True, text=True, timeout=180)
        if result.returncode:
            raise RuntimeError('Native compression helper failed:\n' + result.stdout + result.stderr)
        return result
    kernel = ctypes.WinDLL('kernel32', use_last_error=True)
    size = kernel.GetCompressedFileSizeW
    size.argtypes = [ctypes.c_wchar_p, ctypes.POINTER(ctypes.c_ulong)]
    size.restype = ctypes.c_ulong
    def allocated():
        ctypes.set_last_error(0)
        high = ctypes.c_ulong()
        low = size(str(candidate), ctypes.byref(high))
        if low == 0xffffffff and ctypes.get_last_error():
            raise ctypes.WinError(ctypes.get_last_error())
        return (high.value << 32) | low
    for selection in ['XPRESS4K', algorithm]:
        apply('-Algorithm', selection)
    assert allocated() < candidate.stat().st_size, 'Native compression did not reduce allocated disk'
    apply('-Undo')
    restored_bytes = allocated()
    assert restored_bytes >= candidate.stat().st_size, 'Undo left executable compression applied'
    result = apply('-Algorithm', algorithm)
    return candidate, {'allocated_bytes': allocated(), 'undo_allocated_bytes': restored_bytes,
                       'compression_undo_reapply_passed': True, 'switch_from_XPRESS4K_passed': True,
                       'compact_output': result.stdout}


def startup_comparison(binaries, config, report):
    samples = {name: [] for name in binaries}
    for binary in binaries.values():
        run(binary, config, 'version')
    for trial in range(9):
        order = list(binaries) if trial % 2 == 0 else list(reversed(binaries))
        for name in order:
            start = time.perf_counter()
            run(binaries[name], config, 'version')
            samples[name].append(time.perf_counter() - start)
    for name, values in samples.items():
        report['binaries'][name]['startup_median_seconds'] = statistics.median(values)
        report['binaries'][name]['startup_samples_seconds'] = values
    report['startup_method'] = 'Nine warm samples per variant after warmup, alternating forward/reverse order; not cold-start measurements'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--baseline', required=True, type=Path)
    parser.add_argument('--candidate', required=True, type=Path)
    parser.add_argument('--packed', type=Path)
    parser.add_argument('--json-baseline', type=Path, help='Same-source build using the previous JSON library')
    parser.add_argument('--mount', action='store_true', help='Require WinFsp mount read/write QA')
    parser.add_argument('--compact', action='store_true', help='Require native Windows compression QA')
    parser.add_argument('--output', type=Path, default=Path('build/executable-comparison.json'))
    args = parser.parse_args()
    binaries = {'baseline': args.baseline.resolve(), 'candidate': args.candidate.resolve()}
    if args.json_baseline:
        binaries['json_baseline'] = args.json_baseline.resolve()
    if args.packed:
        binaries['upx'] = args.packed.resolve()
    if args.compact and os.name != 'nt':
        parser.error('--compact requires Windows')
    report = {'binaries': {}, 'provider_inventory_equal': False, 'existing_commands_and_flags_preserved': False}
    with tempfile.TemporaryDirectory(prefix='zclone-qa-') as directory:
        root = Path(directory)
        config = root / 'inventory.conf'
        config.write_text('')
        inventories = {name: inventory(binary, config) for name, binary in binaries.items()}
        baseline = inventories['baseline']
        for name, value in inventories.items():
            assert value['providers'] == baseline['providers'], 'Provider options changed: ' + name
            for command, flags in baseline['commands'].items():
                assert command in value['commands'], 'Missing command: ' + command
                assert set(flags) <= set(value['commands'][command]), 'Missing flags: ' + command
        if 'upx' in inventories:
            assert inventories['upx'] == inventories['candidate'], 'Packing changed inventory'
        report['provider_inventory_equal'] = True
        report['existing_commands_and_flags_preserved'] = True
        report['added_commands'] = sorted(set(inventories['candidate']['commands']) - set(baseline['commands']))
        if args.compact:
            for algorithm in ['XPRESS4K', 'XPRESS8K', 'LZX']:
                binary, metrics = compact(binaries['candidate'], algorithm, root)
                assert binary.read_bytes() == binaries['candidate'].read_bytes(), 'Native compression changed bytes'
                binaries[algorithm] = binary
                report['binaries'][algorithm] = metrics
        for name, binary in binaries.items():
            result = report['binaries'].setdefault(name, {})
            result['bytes'] = binary.stat().st_size
            result['sha256'] = hashlib.sha256(binary.read_bytes()).hexdigest()
            result.update(smoke(binary, root / name, args.mount and name != 'baseline'))
            with tempfile.TemporaryFile() as compressed:
                with zipfile.ZipFile(compressed, 'w', zipfile.ZIP_DEFLATED) as archive:
                    archive.write(binary, 'zclone.exe')
                result['zip_bytes'] = compressed.tell()
            result['saved_bytes'] = binaries['baseline'].stat().st_size - binary.stat().st_size
        startup_comparison(binaries, config, report)
    failures = []
    if args.compact:
        ordinary = report['binaries']['candidate']
        report['native_compression_performance'] = {
            'preferred_algorithm': 'XPRESS4K', 'startup_limit_percent': 20,
            'working_set_limit_percent': 10, 'change_percent': {}}
        for name in ['XPRESS4K', 'XPRESS8K', 'LZX']:
            values = report['binaries'][name]
            changes = {key: 100 * (values[key] / ordinary[key] - 1)
                       for key in ['startup_median_seconds', 'rc_peak_working_set_bytes',
                                   'gui_peak_working_set_bytes']}
            report['native_compression_performance']['change_percent'][name] = changes
            if name == 'XPRESS4K':
                for key, limit in [('startup_median_seconds', 20), ('rc_peak_working_set_bytes', 10),
                                   ('gui_peak_working_set_bytes', 10)]:
                    if changes[key] > limit:
                        failures.append(name + ' ' + key + ' regression')
        report['native_compression_performance']['regressions'] = failures
    if args.json_baseline:
        old, new = report['binaries']['json_baseline'], report['binaries']['candidate']
        report['arrow_json_change'] = {
            'saved_bytes': old['bytes'] - new['bytes'],
            'change_percent': {key: 100 * (new[key] / old[key] - 1) if old[key] else None
                               for key in ['bytes', 'startup_median_seconds',
                                           'rc_peak_working_set_bytes', 'gui_peak_working_set_bytes']}}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report, indent=2))
    if failures:
        raise SystemExit('; '.join(failures))


if __name__ == '__main__':
    main()

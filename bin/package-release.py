#!/usr/bin/env python3
"""Build a vendored, standalone Zclone release archive for one platform."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import zipfile

from version import read_versions


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('version')
    parser.add_argument('os', choices=['linux', 'darwin', 'windows'])
    parser.add_argument('arch', choices=['amd64', 'arm64'])
    parser.add_argument('--output', default='build/releases')
    parser.add_argument('--compression', choices=['none', 'upx'], default='none')
    args = parser.parse_args()
    if args.compression == 'upx' and (args.os, args.arch) != ('windows', 'amd64'):
        parser.error('UPX packaging is only supported for windows/amd64')
    if read_versions() != args.version:
        parser.error('release version must match VERSION')
    if 'go1.27.0 ' not in subprocess.check_output(['go', 'version'], text=True):
        parser.error('release builds require Go 1.27.0')
    output = Path(args.output).resolve()
    output.mkdir(parents=True, exist_ok=True)
    scratch = Path('build/recovery-verification/tmp').resolve()
    scratch.mkdir(parents=True, exist_ok=True)
    name = 'zclone-{}-{}-{}'.format(args.version, args.os, args.arch)
    if args.compression != 'none':
        name += '-' + args.compression
    tags = ['arrow_json_stdlib']
    if args.os == 'windows':
        tags.append('cmount')
    env = dict(os.environ, GOPROXY='off', GOSUMDB='off', GOFLAGS='-mod=vendor',
               GOTOOLCHAIN='local', CGO_ENABLED='0', GOOS=args.os, GOARCH=args.arch)
    env.setdefault('GOTMPDIR', str(scratch))
    with tempfile.TemporaryDirectory(dir=output) as directory:
        binary = Path(directory) / ('zclone.exe' if args.os == 'windows' else 'zclone')
        command = ['go', 'build', '-buildvcs=false', '-trimpath', '-ldflags',
                   '-s -w -X zclone/fs.Version=' + args.version]
        if tags:
            command += ['-tags', ','.join(tags)]
        subprocess.run(command + ['-o', str(binary), '.'], env=env, check=True)
        original_size = binary.stat().st_size
        upx_version = None
        if args.compression == 'upx':
            upx_version = subprocess.check_output(['upx', '--version'], text=True).splitlines()[0]
            subprocess.run(['upx', '-9', str(binary)], check=True)
            subprocess.run(['upx', '-t', str(binary)], check=True)
        info = {'version': args.version, 'os': args.os, 'arch': args.arch,
                'commit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip(),
                'go': 'go1.27.0', 'cgo': False, 'tags': tags,
                'compression': args.compression, 'uncompressed_bytes': original_size,
                'executable_bytes': binary.stat().st_size}
        if upx_version:
            info['upx'] = upx_version
        native_compression = args.os == 'windows' and args.compression == 'none'
        if native_compression:
            info['windows_native_compression'] = {
                'helper': 'compress-windows.ps1', 'default_algorithm': 'XPRESS4K',
                'alternatives': ['XPRESS8K', 'LZX'], 'applied_to_archive': False}
        with zipfile.ZipFile(output / (name + '.zip'), 'w', zipfile.ZIP_DEFLATED) as archive:
            archive.write(binary, name + '/' + binary.name)
            for source in ['COPYING', 'NOTICE', 'README.md', 'DEPENDENCY_MANIFEST.sha256',
                           'bin/zclone-recover.py', 'docs/content/recovery.md', 'build/zclone.sbom.cdx.json']:
                archive.write(source, name + '/' + source)
            if native_compression:
                archive.write('bin/compress-windows.ps1', name + '/compress-windows.ps1')
                archive.write('RELEASE.md', name + '/RELEASE.md')
            archive.writestr(name + '/build-info.json', json.dumps(info, indent=2) + '\n')
    path = output / (name + '.zip')
    with path.open('rb') as stream:
        hasher = hashlib.sha256()
        while data := stream.read(1024 * 1024):
            hasher.update(data)
        digest = hasher.hexdigest()
    (output / (name + '.zip.sha256')).write_text(digest + '  ' + path.name + '\n')
    print(path)


if __name__ == '__main__':
    main()

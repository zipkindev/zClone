"""Release archive contract tests using a disposable toolchain substitute."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
import zipfile

SCRIPT = Path(__file__).parents[1] / 'package-release.py'


class PackageTests(unittest.TestCase):
    def test_archive_contains_recovery_licenses_and_build_identity(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            sources = {'VERSION': 'v0.1.0\n', 'fs/versiontag.go': 'var VersionTag = "v0.1.0"\n',
                       'docs/layouts/partials/version.html': 'v0.1.0'}
            for name in ['COPYING', 'NOTICE', 'README.md', 'RELEASE.md', 'DEPENDENCY_MANIFEST.sha256',
                         'bin/zclone-recover.py', 'bin/compress-windows.ps1',
                         'docs/content/recovery.md', 'build/zclone.sbom.cdx.json']:
                sources[name] = name
            for name, data in sources.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(data)
            subprocess.run(['git', 'init', '-q'], cwd=root, check=True)
            subprocess.run(['git', 'add', '.'], cwd=root, check=True)
            subprocess.run(['git', '-c', 'user.name=Test', '-c', 'user.email=test@example.invalid',
                            'commit', '-qm', 'fixture'], cwd=root, check=True)
            tools = root / 'tools'
            tools.mkdir()
            go = tools / 'go'
            go.write_text('''#!/usr/bin/env python3
import json, os, pathlib, sys
if sys.argv[1] == 'version':
    print('go version go1.27.0 linux/amd64')
else:
    assert os.environ['GOPROXY'] == 'off'
    assert os.environ['GOSUMDB'] == 'off'
    assert os.environ['GOFLAGS'] == '-mod=vendor'
    assert os.environ['CGO_ENABLED'] == '0'
    output = pathlib.Path(sys.argv[sys.argv.index('-o') + 1])
    output.write_text(json.dumps({'os': os.environ['GOOS'], 'arch': os.environ['GOARCH'],
                                  'tags': sys.argv[sys.argv.index('-tags') + 1] if '-tags' in sys.argv else ''}))
    output.chmod(0o755)
''')
            go.chmod(0o755)
            env = dict(os.environ, PATH=str(tools) + os.pathsep + os.environ['PATH'])
            result = subprocess.run([sys.executable, str(SCRIPT), 'v0.1.0', 'linux', 'amd64'],
                                    cwd=root, env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            archive = root / 'build/releases/zclone-v0.1.0-linux-amd64.zip'
            checksum = archive.with_suffix('.zip.sha256').read_text().split()[0]
            self.assertEqual(checksum, hashlib.sha256(archive.read_bytes()).hexdigest())
            with zipfile.ZipFile(archive) as package:
                prefix = 'zclone-v0.1.0-linux-amd64/'
                for name in ['COPYING', 'NOTICE', 'bin/zclone-recover.py', 'build/zclone.sbom.cdx.json']:
                    self.assertIn(prefix + name, package.namelist())
                info = json.loads(package.read(prefix + 'build-info.json'))
                self.assertFalse(info['cgo'])
                self.assertEqual(info['version'], 'v0.1.0')
                self.assertEqual(info['tags'], ['arrow_json_stdlib'])
                self.assertEqual(len(info['commit']), 40)
                self.assertTrue((package.getinfo(prefix + 'zclone').external_attr >> 16) & 0o111)
                self.assertNotIn(prefix + 'compress-windows.ps1', package.namelist())
                self.assertNotIn('windows_native_compression', info)


            result = subprocess.run([sys.executable, str(SCRIPT), 'v0.1.0', 'windows', 'amd64'],
                                    cwd=root, env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            with zipfile.ZipFile(root / 'build/releases/zclone-v0.1.0-windows-amd64.zip') as package:
                prefix = 'zclone-v0.1.0-windows-amd64/'
                binary = json.loads(package.read(prefix + 'zclone.exe'))
                self.assertEqual(binary['tags'], 'arrow_json_stdlib,cmount')
                info = json.loads(package.read(prefix + 'build-info.json'))
                self.assertEqual(info['tags'], ['arrow_json_stdlib', 'cmount'])
                self.assertEqual(info['compression'], 'none')
                self.assertEqual(package.read(prefix + 'compress-windows.ps1'), b'bin/compress-windows.ps1')
                self.assertIn(prefix + 'RELEASE.md', package.namelist())
                self.assertEqual(info['windows_native_compression'], {
                    'helper': 'compress-windows.ps1', 'default_algorithm': 'XPRESS4K',
                    'alternatives': ['XPRESS8K', 'LZX'], 'applied_to_archive': False})
            upx = tools / 'upx'
            upx.write_text("#!/usr/bin/env python3\nimport sys\nprint('upx 5.1.1')\n")
            upx.chmod(0o755)
            result = subprocess.run([sys.executable, str(SCRIPT), 'v0.1.0', 'windows', 'amd64',
                                     '--compression', 'upx'], cwd=root, env=env,
                                    capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            with zipfile.ZipFile(root / 'build/releases/zclone-v0.1.0-windows-amd64-upx.zip') as package:
                info = json.loads(package.read('zclone-v0.1.0-windows-amd64-upx/build-info.json'))
                self.assertEqual(info['compression'], 'upx')
                self.assertEqual(info['upx'], 'upx 5.1.1')
                self.assertNotIn('windows_native_compression', info)
                self.assertNotIn('zclone-v0.1.0-windows-amd64-upx/compress-windows.ps1', package.namelist())
            result = subprocess.run([sys.executable, str(SCRIPT), 'v0.1.0', 'windows', 'arm64',
                                     '--compression', 'upx'], cwd=root, env=env,
                                    capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('only supported for windows/amd64', result.stderr)


if __name__ == '__main__':
    unittest.main()

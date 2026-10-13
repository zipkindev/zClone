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
            for name in ['COPYING', 'NOTICE', 'README.md', 'DEPENDENCY_MANIFEST.sha256',
                         'bin/zclone-recover.py', 'docs/content/recovery.md', 'build/zclone.sbom.cdx.json']:
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
    output.write_text(json.dumps({'os': os.environ['GOOS'], 'arch': os.environ['GOARCH']}))
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
                self.assertEqual(len(info['commit']), 40)
                self.assertTrue((package.getinfo(prefix + 'zclone').external_attr >> 16) & 0o111)


if __name__ == '__main__':
    unittest.main()

"""Version policy tests against disposable Git indexes."""
import pathlib
import subprocess
import sys
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).parents[1] / 'version.py'


class VersionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = pathlib.Path(self.temp.name)
        (self.root / 'fs').mkdir()
        (self.root / 'docs/layouts/partials').mkdir(parents=True)
        self.write('v0.1.0')
        subprocess.run(['git', 'init', '-q'], cwd=self.root, check=True)
        subprocess.run(['git', 'add', '.'], cwd=self.root, check=True)

    def write(self, version):
        (self.root / 'VERSION').write_text(version + '\n')
        (self.root / 'fs/versiontag.go').write_text('package fs\n\n// VersionTag is the base zClone version.\nvar VersionTag = "' + version + '"\n')
        (self.root / 'docs/layouts/partials/version.html').write_text(version)

    def run_script(self, *args):
        return subprocess.run([sys.executable, str(SCRIPT), *args], cwd=self.root, capture_output=True, text=True)

    def test_matching_versions(self):
        self.assertEqual(self.run_script('check').returncode, 0)

    def test_reject_mismatch(self):
        (self.root / 'docs/layouts/partials/version.html').write_text('v1.76.0')
        self.assertNotEqual(self.run_script('check').returncode, 0)

    def test_hook_checks_index_not_worktree(self):
        (self.root / 'VERSION').write_text('v0.2.0\n')
        self.assertEqual(self.run_script('check', '--staged').returncode, 0)
        subprocess.run(['git', 'add', 'VERSION'], cwd=self.root, check=True)
        self.write('v0.2.0')
        self.assertNotEqual(self.run_script('check', '--staged').returncode, 0)

    def test_patch_minor_major_bumps(self):
        for bump, expected in [('patch', 'v0.1.1'), ('minor', 'v0.2.0'), ('major', 'v1.0.0')]:
            with self.subTest(bump=bump):
                self.assertEqual(self.run_script('bump', bump).returncode, 0)
                self.assertEqual((self.root / 'VERSION').read_text().strip(), expected)
                self.assertEqual(self.run_script('check', '--tag', expected).returncode, 0)

    def test_tag_must_match(self):
        self.assertNotEqual(self.run_script('check', '--tag', 'v1.76.0').returncode, 0)

    def test_invalid_version_rejected(self):
        self.write('v00.1.0')
        self.assertNotEqual(self.run_script('check').returncode, 0)


if __name__ == '__main__':
    unittest.main()

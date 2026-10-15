#!/usr/bin/env python3
"""Validate or explicitly bump zClone's independent release version."""
import argparse
from pathlib import Path
import re
import subprocess
import sys

FILES = ('VERSION', 'fs/versiontag.go', 'docs/layouts/partials/version.html')
SEMVER = re.compile(r'v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\Z')


def read_versions(staged=False):
    def read(name):
        if staged:
            return subprocess.check_output(['git', 'show', ':' + name], text=True)
        return Path(name).read_text()
    values = [read(name) for name in FILES]
    version = values[0].strip()
    match = re.search(r'^var VersionTag = "([^"]+)"$', values[1], re.MULTILINE)
    if not SEMVER.fullmatch(version) or not match or match[1] != version or values[2].strip() != version:
        raise ValueError('VERSION, fs/versiontag.go and website version must agree on vMAJOR.MINOR.PATCH')
    return version


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    check = commands.add_parser('check')
    check.add_argument('--staged', action='store_true', help='check the Git index used by commits')
    check.add_argument('--tag', help='require this exact release tag')
    bump = commands.add_parser('bump')
    bump.add_argument('part', choices=['major', 'minor', 'patch'])
    args = parser.parse_args()
    try:
        version = read_versions(getattr(args, 'staged', False))
        if args.command == 'check':
            if args.tag and args.tag != version:
                raise ValueError('release tag must match VERSION exactly')
        else:
            parts = [int(n) for n in version[1:].split('.')]
            index = ['major', 'minor', 'patch'].index(args.part)
            parts[index] += 1
            parts[index + 1:] = [0] * (2 - index)
            version = 'v' + '.'.join(map(str, parts))
            Path(FILES[0]).write_text(version + '\n')
            source = Path(FILES[1]).read_text()
            Path(FILES[1]).write_text(re.sub(r'^var VersionTag = "[^"]+"$',
                                           'var VersionTag = "' + version + '"', source, flags=re.MULTILINE))
            Path(FILES[2]).write_text(version)
        print(version)
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print('Version check failed: ' + str(error), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())

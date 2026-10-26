#!/usr/bin/env python3
"""Package ELab with its icon using the separately pinned Vellum packager."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
WORKER = """
import pathlib, runpy, sys
tool = pathlib.Path(sys.argv[1])
sys.path.insert(0, str(tool.parent))
sys.argv = sys.argv[1:]
runpy.run_path(str(tool), run_name='__main__')
"""


def verify_packager(root=ROOT):
    pin = json.loads((root / 'third_party/vellum/packager.lock.json').read_text())
    if (pin.get('schema') != 1 or pin.get('repository') != 'https://github.com/dshills/vellum'
            or not re.fullmatch(r'[0-9a-f]{40}', pin.get('upstream_commit', ''))):
        raise ValueError('invalid Vellum packager pin')
    relative = Path(pin.get('root', ''))
    if relative.parent != Path('third_party/vellum') or not relative.name.startswith('packager-'):
        raise ValueError('invalid Vellum packager directory')
    tool_root = root / relative
    files = pin.get('files', {})
    required = {'LICENSE', 'release/version.json', 'scripts/package-app.py', 'scripts/app_icons.py'}
    if not required.issubset(files):
        raise ValueError('incomplete Vellum packager pin')
    actual = set()
    for path in [tool_root, *tool_root.rglob('*')]:
        if path.is_symlink():
            raise ValueError('Vellum packager contains a symlink')
        if path.is_file():
            actual.add(path.relative_to(tool_root).as_posix())
    if actual != set(files):
        raise ValueError('Vellum packager file inventory changed')
    for name, expected in files.items():
        path = Path(name)
        if path.is_absolute() or '..' in path.parts or not re.fullmatch(r'[0-9a-f]{64}', expected):
            raise ValueError('invalid Vellum packager file pin')
        if hashlib.sha256((tool_root / path).read_bytes()).hexdigest() != expected:
            raise ValueError('Vellum packager file changed: ' + name)
    return tool_root


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True, help='new .app path; existing bundles are never replaced')
    parser.add_argument('--version', required=True)
    parser.add_argument('--build-version', required=True)
    parser.add_argument('--icon', type=Path, default=ROOT / 'assets/app-icon.png')
    parser.add_argument('--sign-identity', default='-', help="'-' means ad-hoc signing")
    parser.add_argument('--notice-bundle', type=Path)
    args = parser.parse_args()
    try:
        tools = verify_packager()
    except (ValueError, OSError) as error:
        parser.error(str(error))
    command = [sys.executable, '-I', '-B', '-c', WORKER, str(tools / 'scripts/package-app.py'),
               '--binary', str(args.binary.absolute()), '--output', str(args.output.absolute()),
               '--name', 'ELab', '--identifier', 'com.dshills.elab', '--platform', 'darwin',
               '--version', args.version, '--build-version', args.build_version,
               '--icon', str(args.icon.absolute()), '--sign-identity', args.sign_identity]
    if args.notice_bundle is not None:
        command.extend(['--notice-bundle', str(args.notice_bundle.absolute())])
    return subprocess.run(command, check=False, timeout=900).returncode


if __name__ == '__main__':
    raise SystemExit(main())

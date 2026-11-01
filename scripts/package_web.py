#!/usr/bin/env python3
"""Build a self-contained macOS ELab Web app and optional private-alpha ZIP."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import plistlib
import re
import shutil
import subprocess
import tempfile
import zipfile

from package_app import verify_packager

ROOT = Path(__file__).resolve().parents[1]
LICENSES = {
    'gonum.org/v1/gonum': ('v0.17.0', ROOT / 'third_party/gonum/LICENSE'),
    'golang.org/x/sys': ('v0.41.0', ROOT / 'third_party/agent-mcp/golang.org/x/sys@v0.41.0/LICENSE'),
}


def run(argv, *, env=None, capture=False):
    return subprocess.run([str(x) for x in argv], cwd=ROOT, env=env, check=True,
                          text=True, stdout=subprocess.PIPE if capture else None,
                          timeout=300).stdout


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def bundle_info(version, build, test_data=None):
    match = re.fullmatch(r'(\d+\.\d+\.\d+)(?:-[A-Za-z0-9.-]+)?', version)
    if not match or not re.fullmatch(r'\d+(?:\.\d+){0,2}', build):
        raise ValueError('version must be semver; build must be numeric')
    info = {
        'CFBundleExecutable': 'ELabWebLauncher', 'CFBundleName': 'ELab Web',
        'CFBundleDisplayName': 'ELab Web', 'CFBundleIdentifier': 'com.dshills.elab.web',
        'CFBundlePackageType': 'APPL', 'CFBundleShortVersionString': match[1],
        'CFBundleVersion': build, 'CFBundleIconFile': 'AppIcon.icns',
        'LSMinimumSystemVersion': '13.0', 'NSHighResolutionCapable': True,
        'ELabWebVersion': version,
    }
    if test_data is not None:
        info['ELabWebTestDataDirectory'] = str(Path(test_data).resolve())
        info['CFBundleIdentifier'] += '.test'
        info['CFBundleDisplayName'] = 'ELab Web Test'
    return info


def source_snapshot():
    names = run(['git', 'ls-files', '-co', '--exclude-standard'], capture=True).splitlines()
    selected = [name for name in names if (
        '/' not in name and name.endswith(('.go', '.mod', '.sum')) or
        name.startswith(('exp-web/', 'workbench/', 'internal/')) and
        name.endswith(('.go', '.c', '.h', '.swift', '.js', '.mjs', '.css', '.html')) or
        name in ['scripts/package_web.py', 'scripts/package_app.py', 'scripts/qualify_web_package.py', 'assets/app-icon.png'] or
        name in [str(p.relative_to(ROOT)) for _, p in LICENSES.values()]
    )]
    return {name: digest(ROOT / name) for name in sorted(set(selected))}


def copy_licenses(resources, build_info):
    deps = {}
    for line in build_info.splitlines():
        fields = line.split()
        if fields and fields[0] == 'dep':
            deps[fields[1]] = fields[2]
    if deps != {name: version for name, (version, _) in LICENSES.items()}:
        raise ValueError('linked dependencies changed; update the matching license inventory')
    directory = resources / 'Licenses'
    directory.mkdir()
    for name, (_, path) in LICENSES.items():
        shutil.copyfile(path, directory / (name.replace('/', '-') + '.txt'))
    goroot = Path(run(['go', 'env', 'GOROOT'], capture=True).strip())
    license_path = next((p for p in [goroot / 'LICENSE', goroot.parent / 'LICENSE'] if p.is_file()), None)
    if license_path is None or 'Go Authors' not in license_path.read_text():
        raise ValueError('matching Go runtime license could not be located')
    shutil.copyfile(license_path, directory / 'Go.txt')
    (directory / 'README.txt').write_text(
        'Third-party licenses for this ELab Web build.\n' +
        '\n'.join(name + ' ' + version for name, version in deps.items()) + '\n')


def verify_bundle(app):
    if app.is_symlink() or any(p.is_symlink() or not (p.is_file() or p.is_dir()) for p in app.rglob('*')):
        raise ValueError('app contains a link or special file')
    run(['/usr/bin/codesign', '--verify', '--deep', '--strict', app])
    for relative in ['Contents/MacOS/ELabWebLauncher', 'Contents/Resources/elab-web']:
        binary = app / relative
        if not binary.is_file() or not os.access(binary, os.X_OK):
            raise ValueError('missing executable: ' + relative)
        libraries = run(['/usr/bin/otool', '-L', binary], capture=True)
        for line in libraries.splitlines()[1:]:
            library = line.strip().split(' (', 1)[0]
            if not library.startswith(('/usr/lib/', '/System/Library/', '@rpath/libswift')):
                raise ValueError('non-system runtime dependency: ' + library)


def package(output, version, build, identity='-', archive=None, test_data=None):
    if platform.system() != 'Darwin':
        raise ValueError('macOS packaging requires a Mac with Xcode command-line tools')
    output = Path(output).absolute()
    archive = Path(archive).absolute() if archive else None
    if output.suffix != '.app' or output.exists() or output.is_symlink():
        raise ValueError('output must be a new .app path')
    receipt = output.with_suffix('.package.json')
    if receipt.exists() or receipt.is_symlink():
        raise ValueError('package receipt already exists')
    if archive and (archive.suffix != '.zip' or archive.exists() or archive.is_symlink() or archive.is_relative_to(output)):
        raise ValueError('archive must be a new ZIP outside the app')
    if test_data is not None and archive:
        raise ValueError('test-configured apps cannot be archived for distribution')
    info = bundle_info(version, build, test_data)
    before = source_snapshot()
    head = run(['git', 'rev-parse', 'HEAD'], capture=True).strip()
    clean = not run(['git', 'status', '--porcelain'], capture=True).strip()
    output.parent.mkdir(parents=True, exist_ok=True)
    architecture = {'arm64': 'arm64', 'x86_64': 'x86_64'}.get(platform.machine())
    if architecture is None:
        raise ValueError('unsupported macOS architecture')
    env = dict(os.environ, GOOS='darwin', GOARCH='arm64' if architecture == 'arm64' else 'amd64',
               CGO_ENABLED='1', GOCACHE=os.environ.get('GOCACHE', str(ROOT / 'build/go-cache')),
               MACOSX_DEPLOYMENT_TARGET='13.0', GOWORK='off')
    with tempfile.TemporaryDirectory(prefix='.elab-web-package-', dir=output.parent) as temporary:
        work = Path(temporary)
        app = work / 'ELab Web.app'
        macos = app / 'Contents/MacOS'; resources = app / 'Contents/Resources'
        macos.mkdir(parents=True); resources.mkdir()
        launcher = macos / 'ELabWebLauncher'; backend = resources / 'elab-web'
        run(['go', 'build', '-trimpath', '-o', backend, './exp-web'], env=env)
        cache = ROOT / 'build/swift-module-cache'; cache.mkdir(parents=True, exist_ok=True)
        run(['xcrun', 'swiftc', ROOT / 'exp-web/launcher/main.swift', '-o', launcher,
             '-target', architecture + '-apple-macos13.0', '-module-cache-path', cache, '-framework', 'Cocoa'])
        with (app / 'Contents/Info.plist').open('wb') as stream:
            plistlib.dump(info, stream)
        # Reuse the existing verified icon conversion helper, without modifying
        # the pinned native app packager or its release bundles.
        tools = verify_packager()
        import sys
        sys.path.insert(0, str(tools / 'scripts'))
        try:
            import app_icons
            app_icons.install(ROOT / 'assets/app-icon.png', resources)
        finally:
            sys.path.pop(0)
        build_info = run(['go', 'version', '-m', backend], capture=True)
        copy_licenses(resources, build_info)
        (resources / 'BUILD.txt').write_text(build_info)
        (resources / 'Read me.txt').write_text(
            'ELab Web ' + version + '\n\n'
            'Open ELab Web.app. It starts ELab and opens your default browser.\n'
            'Use the launcher window or Command-Q to save and quit. Closing a browser tab leaves ELab running.\n'
            'Projects remain in ~/Library/Application Support/ELab/web when the app is upgraded.\n'
            'Use Projects > Recovery history after an interrupted session.\n'
            'Go, Node and a network connection are not required to run this app.\n'
            'This private alpha is ad-hoc signed unless an explicit signing identity was supplied.\n'
            'Developer ID/notarized distribution and clean-account/other-OS acceptance are separate gates.\n')
        signing = ['--timestamp=none'] if identity == '-' else ['--timestamp', '--options', 'runtime']
        run(['/usr/bin/codesign', '--force', '--sign', identity, *signing, backend])
        record = {'schema': 1, 'version': version, 'build': build, 'architecture': architecture,
                  'source_commit': head, 'source_clean': clean, 'source_inputs': before,
                  'test_fixture': test_data is not None, 'signing_identity': identity,
                  'minimum_os_build_target': '13.0',
                  'toolchain': {'go': run(['go', 'version'], capture=True).strip(),
                                'swift': run(['xcrun', 'swiftc', '--version'], capture=True).strip()},
                  'executables': {str(backend.relative_to(app)): digest(backend)}}
        (resources / 'PACKAGE.json').write_text(json.dumps(record, indent=2) + '\n')
        run(['/usr/bin/codesign', '--force', '--sign', identity, *signing, app])
        verify_bundle(app)
        if source_snapshot() != before or run(['git', 'rev-parse', 'HEAD'], capture=True).strip() != head:
            raise ValueError('source changed while packaging')
        if output.exists() or output.is_symlink():
            raise ValueError('output appeared while packaging')
        app.rename(output)
    # Signing the bundle also seals/changes its main executable. Its final hash
    # belongs in this external receipt, never in its own sealed resources.
    record['executables'] = {relative: digest(output / relative) for relative in
                             ['Contents/MacOS/ELabWebLauncher', 'Contents/Resources/elab-web']}
    record['files'] = {str(p.relative_to(output)): {'sha256': digest(p), 'mode': p.stat().st_mode & 0o777}
                       for p in sorted(output.rglob('*')) if p.is_file()}
    with receipt.open('x') as stream:
        json.dump(record, stream, indent=2); stream.write('\n')
    if archive:
        archive.parent.mkdir(parents=True, exist_ok=True)
        run(['/usr/bin/ditto', '-c', '-k', '--sequesterRsrc', '--keepParent', output, archive])
        with zipfile.ZipFile(archive) as zipped:
            if zipped.testzip() is not None:
                raise ValueError('ZIP integrity check failed')
            expected = {output.name + '/' + relative: item['sha256'] for relative, item in record['files'].items()}
            actual = {name for name in zipped.namelist() if not name.endswith('/') and not name.startswith('__MACOSX/')}
            if actual != set(expected):
                raise ValueError('ZIP file inventory differs from the sealed app')
            for name, checksum in expected.items():
                if hashlib.sha256(zipped.read(name)).hexdigest() != checksum:
                    raise ValueError('ZIP payload changed: ' + name)
    return output


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--version', default='0.1.0-alpha.1')
    parser.add_argument('--build-version', default='1')
    parser.add_argument('--sign-identity', default='-')
    parser.add_argument('--archive', type=Path)
    parser.add_argument('--test-data-dir', type=Path, help='private GUI fixture only; disables distribution archive')
    args = parser.parse_args()
    try:
        print(package(args.output, args.version, args.build_version, args.sign_identity, args.archive, args.test_data_dir))
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        parser.exit(1, str(error) + '\n')


if __name__ == '__main__':
    main()

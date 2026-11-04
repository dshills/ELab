#!/usr/bin/env python3
"""Verify a pinned notarized ZIP, Gatekeeper, and a fresh isolated runtime."""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import stat
import subprocess
import tempfile
import zipfile

from distribution import verify_distribution
from package_web import verify_bundle
from qualify_web_package import qualify


def safe_archive(archive, byte_limit=512 << 20):
    with zipfile.ZipFile(archive) as zipped:
        entries = zipped.infolist()
        if not entries or len(entries) > 2000 or sum(p.file_size for p in entries) > byte_limit:
            raise ValueError('archive exceeds the installation check bounds')
        names, apps = set(), set()
        for entry in entries:
            name = entry.filename
            path = PurePosixPath(name)
            mode = stat.S_IFMT(entry.external_attr >> 16)
            if (str(path) in names or '\\' in name or '\0' in entry.orig_filename or path.is_absolute()
                    or '..' in path.parts or mode not in (0, stat.S_IFDIR, stat.S_IFREG)):
                raise ValueError('unsafe archive entry')
            names.add(str(path))
            if path.parts and path.parts[0] != '__MACOSX':
                apps.add(path.parts[0])
        if len(apps) != 1 or not next(iter(apps)).endswith('.app'):
            raise ValueError('archive must contain exactly one application')
        return next(iter(apps))


def check(archive, expected_sha256, artifacts):
    archive, artifacts = Path(archive).resolve(), Path(artifacts).absolute()
    checksum = hashlib.sha256(archive.read_bytes()).hexdigest()
    if checksum != expected_sha256:
        raise ValueError('archive SHA-256 differs from the expected release digest')
    name = safe_archive(archive)
    artifacts.mkdir(parents=True, exist_ok=False, mode=0o700)
    with tempfile.TemporaryDirectory(prefix='elab-install-check-') as directory:
        root = Path(directory)
        subprocess.run(['/usr/bin/ditto', '-x', '-k', str(archive), str(root/'extracted')],
                       check=True, timeout=60)
        app = root/'extracted'/name
        verify_bundle(app)
        distribution = verify_distribution(app)
        metadata = json.loads((app/'Contents/Resources/PACKAGE.json').read_text())
        if not metadata.get('source_clean') or metadata.get('test_fixture'):
            raise ValueError('installation check requires clean shipping source')
        runtime = qualify(app, root/'runtime')
        result = dict(passed=True, archive_sha256=checksum, version=metadata['version'],
                      source_commit=metadata['source_commit'], distribution=distribution,
                      runtime=runtime, scope='Current macOS account, OS-extracted ZIP and fresh '
                      'disposable library. Clean-account/Finder acceptance is a separate check.')
        (artifacts/'installation.json').write_text(json.dumps(result, indent=2)+'\n')
        return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--archive', type=Path, required=True)
    parser.add_argument('--sha256', required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    try:
        print(json.dumps(check(args.archive, args.sha256, args.artifacts), indent=2))
    except (ValueError, OSError, subprocess.SubprocessError, zipfile.BadZipFile) as error:
        parser.exit(1, str(error)+'\n')


if __name__ == '__main__':
    main()

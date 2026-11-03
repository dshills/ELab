#!/usr/bin/env python3
"""Check the macOS web package, extracted archive and copied runtime without GUI."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile

from package_web import package, verify_bundle
from qualify_web_package import qualify


def verify_extracted(app, expected):
    entries = list(app.rglob('*'))
    if app.is_symlink() or any(p.is_symlink() or not (p.is_dir() or p.is_file()) for p in entries):
        raise ValueError('extracted app contains a link or special entry')
    actual = {str(p.relative_to(app)): p for p in entries if p.is_file()}
    if set(actual) != set(expected):
        raise ValueError('extracted app inventory differs')
    for name, metadata in expected.items():
        path = actual[name]
        if (hashlib.sha256(path.read_bytes()).hexdigest() != metadata['sha256']
                or path.stat().st_mode & 0o777 != metadata['mode']):
            raise ValueError('extracted app bytes or mode differ: ' + name)


def check(artifacts):
    artifacts = Path(artifacts).resolve()
    artifacts.mkdir(parents=True, exist_ok=False, mode=0o700)
    # Private launch replies/control tokens stay in this temporary tree. Only
    # credential-free receipts are retained in the public quality artifacts.
    with tempfile.TemporaryDirectory(prefix='elab-package-check-') as directory:
        root = Path(directory)
        app, archive = root/'ELab Web.app', root/'ELab-Web.zip'
        package(app, '0.1.0-check', '0', archive=archive)
        receipt = json.loads(app.with_suffix('.package.json').read_text())
        extracted = root/'extracted'
        subprocess.run(['/usr/bin/ditto', '-x', '-k', str(archive), str(extracted)],
                       check=True, timeout=60)
        verify_bundle(extracted/app.name)
        verify_extracted(extracted/app.name, receipt['files'])
        runtime = qualify(extracted/app.name, root/'runtime')
        (artifacts/'package.json').write_text(json.dumps(receipt, indent=2)+'\n')
        report = dict(passed=True, source_commit=receipt['source_commit'],
                      source_clean=receipt['source_clean'],
                      archive_sha256=hashlib.sha256(archive.read_bytes()).hexdigest(),
                      extracted_files=len(receipt['files']), runtime=runtime)
        (artifacts/'report.json').write_text(json.dumps(report, indent=2)+'\n')
        return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    default = os.environ.get('ELAB_QUALITY_ARTIFACTS')
    parser.add_argument('--artifacts', type=Path,
                        default=Path(default)/'package' if default else None)
    args = parser.parse_args()
    if args.artifacts is None:
        parser.error('--artifacts is required outside the quality runner')
    print(json.dumps(check(args.artifacts), indent=2))


if __name__ == '__main__':
    main()

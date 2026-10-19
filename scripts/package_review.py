#!/usr/bin/env python3
"""Archive a sealed private review and verify a macOS extraction.

This is a local packaging rehearsal, not installation or signing acceptance.
The sealed review is read-only; output paths must not already exist.
"""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import platform
import stat
import subprocess
import zipfile

from wiring_replay import ROOT, validate_release


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def members(folder):
    result = {}
    for path in sorted(folder.rglob('*')):
        if path.is_symlink() or not (path.is_file() or path.is_dir()):
            raise ValueError('review contains a link or special entry: '+str(path))
        if path.is_file():
            name = path.relative_to(folder).as_posix()
            result[name] = dict(sha256=sha(path), mode=oct(stat.S_IMODE(path.stat().st_mode)))
    return result


def verify_archive(archive, prefix, expected):
    """Check every entry before an OS extraction; never accept extra paths."""
    with zipfile.ZipFile(archive) as source:
        entries = source.infolist()
        names = [entry.filename for entry in entries]
        wanted = {prefix+'/'+name for name in expected}
        if len(names) != len(set(names)) or set(names) != wanted:
            raise ValueError('archive file set differs from sealed review')
        for entry in entries:
            path = PurePosixPath(entry.filename)
            mode = entry.external_attr >> 16
            if (str(path) != entry.filename or path.is_absolute() or '..' in path.parts or '\\' in entry.filename
                    or not stat.S_ISREG(mode) or entry.create_system != 3):
                raise ValueError('archive contains an unsafe or unsupported entry')
            expected_entry = expected[str(path.relative_to(prefix))]
            if (oct(stat.S_IMODE(mode)) != expected_entry['mode']
                    or hashlib.sha256(source.read(entry)).hexdigest() != expected_entry['sha256']):
                raise ValueError('archive bytes or mode differ: '+entry.filename)


def run(receipt_path, output, artifacts):
    if platform.system() != 'Darwin':
        raise ValueError('the extraction rehearsal requires macOS ditto')
    identity = validate_release(ROOT, receipt_path)
    receipt = json.loads(receipt_path.read_text())
    review = Path(receipt['review_path']).resolve()
    output, artifacts = output.resolve(), artifacts.resolve()
    if (output.is_relative_to(review) or artifacts.is_relative_to(review)
            or output.is_relative_to(artifacts) or artifacts.is_relative_to(output)):
        raise ValueError('archive, extraction and sealed review must be separate')
    if output.exists() or artifacts.exists():
        raise FileExistsError('refusing to reuse archive or artifact destination')
    if not output.parent.is_dir() or not artifacts.parent.is_dir():
        raise ValueError('output parents must already exist')
    expected = members(review)
    artifacts.mkdir(mode=0o700)
    report = dict(passed=False, scope=__doc__.strip(), release=identity,
                  archive=str(output), extraction_parent=str(artifacts),
                  expected_files=expected, deterministic_member_time='1980-01-01T00:00:00',
                  limits=['No clean-account/OS, Finder Archive Utility, quarantine, signing or notarization acceptance',
                          'No executable startup claim; extracted runtime is checked separately'])
    try:
        with output.open('xb') as destination:
            output.chmod(0o600)
            with zipfile.ZipFile(destination, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
                for name, entry in expected.items():
                    member = zipfile.ZipInfo(review.name+'/'+name, date_time=(1980, 1, 1, 0, 0, 0))
                    member.create_system = 3
                    member.external_attr = (stat.S_IFREG | int(entry['mode'], 8)) << 16
                    member.compress_type = zipfile.ZIP_DEFLATED
                    archive.writestr(member, (review/name).read_bytes(), compresslevel=9)
        verify_archive(output, review.name, expected)
        destination = artifacts/'Extracted review µ with spaces'
        extraction = subprocess.run(['/usr/bin/ditto', '-x', '-k', str(output), str(destination)],
                                    capture_output=True, text=True, timeout=120)
        (artifacts/'ditto.txt').write_text(extraction.stdout+extraction.stderr)
        report['ditto_exit_code'] = extraction.returncode
        extraction.check_returncode()
        extracted = destination/review.name
        if members(extracted) != expected:
            raise ValueError('macOS extraction changed file bytes or modes')
        if members(review) != expected:
            raise ValueError('source review changed while archiving')
        report.update(passed=True, archive_sha256=sha(output), archive_bytes=output.stat().st_size,
                      extracted_review=str(extracted), files_verified=len(expected),
                      source_review_unchanged=True, extraction_bytes_and_modes_match=True)
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
        raise
    finally:
        (artifacts/'report.json').write_text(json.dumps(report, indent=2)+'\n')
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--receipt', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    result = run(args.receipt, args.output, args.artifacts)
    print(json.dumps({key: result[key] for key in
                     ['passed', 'archive', 'archive_sha256', 'archive_bytes', 'files_verified']}))


if __name__ == '__main__':
    main()

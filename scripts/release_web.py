#!/usr/bin/env python3
"""Stage and qualify a web alpha; create a verified draft before optional publication."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile

from check_package import verify_extracted
from package_web import bundle_info, package, verify_bundle
from qualify_web_package import qualify
from quality import gate_plan, go_environment, source_inputs

ROOT = Path(__file__).resolve().parents[1]
REPO = 'dshills/ELab'


def run(args, *, capture=False, env=None):
    return subprocess.run([str(a) for a in args], cwd=ROOT, env=env, check=True,
                          text=True, stdout=subprocess.PIPE if capture else None,
                          timeout=7200).stdout


def read(path):
    return json.loads(Path(path).read_text())


def write(path, value):
    with Path(path).open('x') as stream:
        json.dump(value, stream, indent=2)
        stream.write('\n')


def checksum(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def source(candidate=False):
    if run(['git', 'status', '--porcelain'], capture=True).strip():
        raise ValueError('release staging requires clean committed source')
    branch = run(['git', 'branch', '--show-current'], capture=True).strip()
    if branch != 'main' and not candidate:
        raise ValueError('release staging requires main; --candidate permits local non-publishable rehearsal')
    return run(['git', 'rev-parse', 'HEAD'], capture=True).strip(), branch


def check_quality(directory, commit, inputs):
    report = read(directory/'report.json')
    expected = [name for name, _ in gate_plan('macos')]
    if (report.get('passed') is not True or report.get('gates_passed') is not True
            or report.get('inputs_unchanged') is not True or report.get('profile') != 'macos'
            or report.get('source_commit') != commit
            or [s.get('name') for s in report.get('steps', [])] != expected
            or any(s.get('status') != 'passed' or s.get('exit_code') != 0 for s in report['steps'])
            or read(directory/'inputs.json') != inputs):
        raise ValueError('quality receipt does not qualify this exact source')
    return report


def browser_summary(report):
    stats = report.get('stats', {})
    if (stats.get('unexpected', 0) or stats.get('flaky', 0) or stats.get('expected', 0) < 1
            or report.get('errors')):
        raise ValueError('browser qualification failed or did not run')
    projects = {p.get('name') for p in report.get('config', {}).get('projects', [])}
    if projects != {'chromium', 'firefox', 'webkit'}:
        raise ValueError('all three browser engines are required')
    # Require actual successful results per engine, not just configured projects.
    passed = set()
    def visit(suites):
        for suite in suites:
            for spec in suite.get('specs', []):
                for test in spec.get('tests', []):
                    if test.get('status') == 'expected' and any(r.get('status') == 'passed' for r in test.get('results', [])):
                        passed.add(test.get('projectName'))
            visit(suite.get('suites', []))
    visit(report.get('suites', []))
    if passed != projects:
        raise ValueError('a browser engine has no successful journeys')
    return {k: stats.get(k, 0) for k in ['expected', 'skipped', 'unexpected', 'flaky']}


def extract_and_qualify(archive, receipt, destination):
    extracted = destination/'extracted'
    run(['/usr/bin/ditto', '-x', '-k', archive, extracted])
    app = extracted/'ELab Web.app'
    verify_bundle(app)
    verify_extracted(app, receipt['files'])
    report = qualify(app, destination/'runtime')
    if (report.get('passed') is not True or report.get('source_commit') != receipt['source_commit']
            or report.get('source_clean') is not True):
        raise ValueError('downloaded shipping binary identity or runtime qualification failed')
    return report


def stage(args):
    bundle_info(args.version, args.build)
    if not re.fullmatch(r'\d+\.\d+\.\d+-alpha\.[1-9]\d*', args.version):
        raise ValueError('this pipeline publishes alpha versions only')
    commit, branch = source(args.candidate)
    inputs = source_inputs(ROOT)
    output = args.output.absolute()
    output.mkdir(parents=True, exist_ok=False, mode=0o700)
    env = go_environment()
    quality = args.quality.resolve() if args.quality else output/'quality'
    if not args.quality:
        run([sys.executable, '-B', 'scripts/quality.py', '--profile', 'macos', '--artifacts', quality], env=env)
    check_quality(quality, commit, inputs)
    evidence = output/'qualification'
    evidence.mkdir()
    for name in ['report.json', 'inputs.json']:
        shutil.copyfile(quality/name, evidence/name)
    run(['go', 'build', '-o', 'build/elab-web-browser', '.'], env=env)
    browser_report = output/'browsers.json'
    run(['node', 'browser/node_modules/playwright/cli.js', 'test', '--config',
         'browser/playwright.config.mjs', '--reporter=json', '--output', output/'browser-results'],
        env=dict(env, ELAB_BROWSER_TESTS='1', PLAYWRIGHT_JSON_OUTPUT_FILE=str(browser_report)))
    browsers = browser_summary(read(browser_report))
    app = output/'ELab Web.app'
    archive = output/f'ELab-Web-{args.version}-macos-{os.uname().machine}.zip'
    package(app, args.version, args.build, args.sign_identity, archive,
            notary_profile=args.notary_profile)
    receipt = read(app.with_suffix('.package.json'))
    if receipt['source_commit'] != commit or not receipt['source_clean'] or receipt['test_fixture']:
        raise ValueError('package source identity differs from qualified source')
    runtime = extract_and_qualify(archive, receipt, output/'archive-check')
    if runtime.get('passed') is not True or runtime.get('source_commit') != commit:
        raise ValueError('archived shipping binary did not qualify')
    if source(args.candidate) != (commit, branch) or source_inputs(ROOT) != inputs:
        raise ValueError('source changed during release qualification')
    distribution = 'Developer ID / notarized' if receipt.get('notarization_requested') else 'private alpha; notarization not qualified'
    manifest = dict(schema=1, version=args.version, build=args.build, source_commit=commit,
                    publication_ready=branch == 'main' and not args.candidate,
                    architecture=receipt['architecture'], distribution=distribution,
                    archive=archive.name, archive_sha256=checksum(archive),
                    package_receipt_sha256=checksum(app.with_suffix('.package.json')),
                    quality_report_sha256=checksum(evidence/'report.json'),
                    browser_report_sha256=checksum(browser_report), browsers=browsers,
                    runtime_checks=runtime['checks'])
    write(output/'release.json', manifest)
    (output/'SHA256SUMS').write_text(f"{manifest['archive_sha256']}  {archive.name}\n")
    (output/'RELEASE-NOTES.md').write_text(
        f"ELab Web {args.version} (build {args.build})\n\nSource: `{commit}`.\n\n"
        f"macOS {receipt['architecture']}; {distribution}.\n\n"
        'Qualified with the macOS quality plan, three browser engines, extracted archive inventory '
        'and copied shipping-backend restart/backup/restore checks. '
        'Native VoiceOver, clean-account installation and human usability acceptance are not established by these automated gates.\n')
    return manifest


def verify_stage(directory):
    manifest = read(directory/'release.json')
    name = manifest['archive']
    if (Path(name).name != name or not name.endswith('.zip') or name.startswith('-')
            or not re.fullmatch(r'[0-9a-f]{40}', manifest['source_commit'])
            or not re.fullmatch(r'\d+\.\d+\.\d+-alpha\.[1-9]\d*', manifest['version'])):
        raise ValueError('invalid release manifest')
    for path, digest in [(directory/name, manifest['archive_sha256']),
                         (directory/'ELab Web.package.json', manifest['package_receipt_sha256']),
                         (directory/'browsers.json', manifest['browser_report_sha256']),
                         (directory/'qualification/report.json', manifest['quality_report_sha256'])]:
        if path.is_symlink() or checksum(path) != digest:
            raise ValueError('staged release bytes changed: '+path.name)
    receipt = read(directory/'ELab Web.package.json')
    if (receipt.get('source_commit') != manifest['source_commit'] or not receipt.get('source_clean')
            or receipt.get('test_fixture') or receipt.get('version') != manifest['version']
            or receipt.get('build') != manifest['build']):
        raise ValueError('staged package identity differs')
    browser_summary(read(directory/'browsers.json'))
    check_quality(directory/'qualification', manifest['source_commit'], source_inputs(ROOT))
    return manifest, receipt


def publish(args):
    directory = args.directory.resolve()
    manifest, receipt = verify_stage(directory)
    commit, _ = source()
    if not manifest.get('publication_ready') or commit != manifest['source_commit']:
        raise ValueError('only the qualified main revision may be published')
    remote = run(['gh', 'api', f'repos/{REPO}/git/ref/heads/main'], capture=True)
    if json.loads(remote)['object']['sha'] != commit:
        raise ValueError('GitHub main must match the qualified revision')
    tag = 'v'+manifest['version']
    tags = json.loads(run(['gh', 'api', f'repos/{REPO}/git/matching-refs/tags/{tag}'], capture=True))
    if any(ref.get('ref') == 'refs/tags/'+tag for ref in tags):
        raise ValueError('release tag already exists; preserve existing releases')
    assets = [directory/manifest['archive'], directory/'release.json', directory/'SHA256SUMS']
    run(['gh', 'release', 'create', tag, *assets, '--repo', REPO, '--target', commit,
         '--title', f"ELab Web {manifest['version']} (build {manifest['build']})", '--draft',
         '--prerelease', '--notes-file', directory/'RELEASE-NOTES.md'])
    # A failed download/check leaves the draft intact for investigation, never public.
    with tempfile.TemporaryDirectory(prefix='elab-release-download-') as temporary:
        downloaded = Path(temporary)
        run(['gh', 'release', 'download', tag, '--repo', REPO, '--dir', downloaded])
        for asset in assets:
            if checksum(downloaded/asset.name) != checksum(asset):
                raise ValueError('downloaded GitHub asset differs: '+asset.name)
        extract_and_qualify(downloaded/manifest['archive'], receipt, downloaded/'check')
    if source() != (commit, 'main') or json.loads(run(['gh', 'api', f'repos/{REPO}/git/ref/heads/main'], capture=True))['object']['sha'] != commit:
        raise ValueError('main changed during draft verification; draft remains unpublished')
    if args.make_public:
        run(['gh', 'release', 'edit', tag, '--repo', REPO, '--draft=false', '--prerelease'])
    return dict(tag=tag, published=args.make_public, downloaded_assets_verified=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    create = commands.add_parser('stage')
    create.add_argument('--version', required=True)
    create.add_argument('--build', required=True)
    create.add_argument('--output', type=Path, required=True)
    create.add_argument('--quality', type=Path, help='reuse only a matching, passing macOS quality receipt')
    create.add_argument('--candidate', action='store_true', help='non-publishable clean-branch rehearsal')
    create.add_argument('--sign-identity', default='-')
    create.add_argument('--notary-profile')
    release = commands.add_parser('publish')
    release.add_argument('--directory', type=Path, required=True)
    release.add_argument('--make-public', action='store_true', help='publish only after downloaded assets pass')
    args = parser.parse_args()
    try:
        print(json.dumps(stage(args) if args.command == 'stage' else publish(args), indent=2))
    except (ValueError, OSError, KeyError, subprocess.SubprocessError) as error:
        parser.exit(1, str(error)+'\n')


if __name__ == '__main__':
    main()

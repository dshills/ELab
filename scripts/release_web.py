#!/usr/bin/env python3
"""Stage and qualify a web alpha; create a verified draft before optional publication."""
import argparse
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile

from portable_web import TARGETS, archive_name, build_archive, inspect_archive, native_target, sha256 as checksum
from qualify_web_package import exercise_host
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


def source(candidate=False):
    if run(['git', 'status', '--porcelain'], capture=True).strip():
        raise ValueError('release staging requires clean committed source')
    branch = run(['git', 'branch', '--show-current'], capture=True).strip()
    if branch != 'main' and not candidate:
        raise ValueError('release staging requires main; --candidate permits local non-publishable rehearsal')
    return run(['git', 'rev-parse', 'HEAD'], capture=True).strip(), branch


def check_quality(directory, commit, inputs):
    report = read(directory/'report.json')
    expected = [name for name, _ in gate_plan('portable')]
    if (report.get('passed') is not True or report.get('gates_passed') is not True
            or report.get('inputs_unchanged') is not True or report.get('profile') != 'portable'
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
    engines = {'chromium', 'firefox', 'webkit'}
    required = engines | {f'{engine}-large-circuits' for engine in engines}
    if projects != required:
        raise ValueError('workbench and large-circuit projects are required for all three browser engines')
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
        raise ValueError('a browser project has no successful journeys')
    return {k: stats.get(k, 0) for k in ['expected', 'skipped', 'unexpected', 'flaky']}


def qualify_archive(directory, manifest, target):
    if target != native_target() or target not in TARGETS:
        raise ValueError('runtime must be checked on the matching release target')
    entry = manifest['archives'][target]
    with tempfile.TemporaryDirectory(prefix='elab-release-runtime-') as temporary:
        root = Path(temporary)
        binary = inspect_archive(directory/entry['name'], entry, target, root/'extracted')
        metadata = {key: manifest[key] for key in ('version', 'build', 'source_commit')}
        report = exercise_host(binary, dict(metadata, source_clean=True, target=target), root/'runtime')
    return dict(report, target=target, archive_sha256=entry['sha256'])


def stage(args):
    if not re.fullmatch(r'\d+\.\d+\.\d+-alpha\.[1-9]\d*', args.version) or not re.fullmatch(r'[1-9]\d*', args.build):
        raise ValueError('use an alpha version and a positive numeric build')
    native = native_target()
    if native not in TARGETS:
        raise ValueError('stage on a supported release target')
    commit, branch = source(args.candidate)
    inputs = source_inputs(ROOT)
    output = args.output.absolute()
    output.mkdir(parents=True, exist_ok=False, mode=0o700)
    env = go_environment()
    quality = args.quality.resolve() if args.quality else output/'quality'
    if not args.quality:
        run([sys.executable, '-B', 'scripts/quality.py', '--profile', 'portable', '--artifacts', quality], env=env)
    check_quality(quality, commit, inputs)
    evidence = output/'qualification'
    evidence.mkdir()
    for name in ['report.json', 'inputs.json']:
        shutil.copyfile(quality/name, evidence/name)
    browser_report = output/'browsers.json'
    with tempfile.TemporaryDirectory(prefix='elab-release-build-') as temporary:
        work = Path(temporary)
        archives = {target: build_archive(target, args.version, args.build, output, work, env)
                    for target in TARGETS}
        for target, entry in archives.items():
            inspect_archive(output/entry['name'], entry, target)
        shipping = inspect_archive(output/archives[native]['name'], archives[native], native, work/'browser-host')
        run(['node', 'browser/node_modules/playwright/cli.js', 'test', '--config',
             'browser/playwright.config.mjs', '--reporter=json', '--output', output/'browser-results'],
            env=dict(env, ELAB_BROWSER_TESTS='1', ELAB_BROWSER_RELEASE_BINARY=str(shipping),
                     ELAB_BROWSER_RELEASE_VERSION=args.version,
                     PLAYWRIGHT_JSON_OUTPUT_FILE=str(browser_report)))
    browsers = browser_summary(read(browser_report))
    manifest = dict(schema=2, version=args.version, build=args.build, source_commit=commit,
                    publication_ready=branch == 'main' and not args.candidate,
                    archives=archives, quality_report_sha256=checksum(evidence/'report.json'),
                    browser_report_sha256=checksum(browser_report), browsers=browsers)
    runtime = qualify_archive(output, manifest, native)
    write(evidence/f'runtime-{native}.json', runtime)
    if source(args.candidate) != (commit, branch) or source_inputs(ROOT) != inputs:
        raise ValueError('source changed during release qualification')
    (output/'SHA256SUMS').write_text(checksum_text(archives), encoding='utf-8')
    (output/'RELEASE-NOTES.md').write_text(
        f"ELab Web {args.version} (build {args.build})\n\nSource: `{commit}`.\n\n"
        'Choose the ZIP for your computer: Linux x64, macOS Apple Silicon or Intel, or Windows x64. '
        'Extract it, follow START-HERE.txt, then use ELab in your browser at http://localhost:8788. '
        'The download includes the local web host and simulation engine. No Go or Node installation is needed.\n\n'
        'The standalone downloads are unsigned for distribution. The optional macOS launcher has '
        'its own packaging and signing process.\n', encoding='utf-8')
    manifest['release_notes_sha256'] = checksum(output/'RELEASE-NOTES.md')
    write(output/'release.json', manifest)
    return manifest


def checksum_text(archives):
    return ''.join(f"{archives[target]['sha256']}  {archives[target]['name']}\n" for target in TARGETS)


def verify_stage(directory, *, inspect=True):
    manifest = read(directory/'release.json')
    if (manifest.get('schema') != 2 or not re.fullmatch(r'[0-9a-f]{40}', manifest['source_commit'])
            or not re.fullmatch(r'\d+\.\d+\.\d+-alpha\.[1-9]\d*', manifest['version'])
            or not re.fullmatch(r'[1-9]\d*', manifest['build']) or set(manifest['archives']) != set(TARGETS)):
        raise ValueError('invalid release manifest')
    for target, entry in manifest['archives'].items():
        if entry['name'] != archive_name(manifest['version'], target):
            raise ValueError('invalid staged archive filename')
        if inspect:
            inspect_archive(directory/entry['name'], entry, target)
    for path, digest in [(directory/'browsers.json', manifest['browser_report_sha256']),
                         (directory/'RELEASE-NOTES.md', manifest['release_notes_sha256']),
                         (directory/'qualification/report.json', manifest['quality_report_sha256'])]:
        if path.is_symlink() or checksum(path) != digest:
            raise ValueError('staged release bytes changed: '+path.name)
    if (directory/'SHA256SUMS').read_text(encoding='utf-8') != checksum_text(manifest['archives']):
        raise ValueError('staged checksum list differs')
    if browser_summary(read(directory/'browsers.json')) != manifest['browsers']:
        raise ValueError('staged browser summary differs')
    check_quality(directory/'qualification', manifest['source_commit'], source_inputs(ROOT))
    return manifest


def verify_runtime_receipts(directory, manifest):
    for target, entry in manifest['archives'].items():
        report = read(directory/'qualification'/f'runtime-{target}.json')
        if (report.get('passed') is not True or report.get('source_clean') is not True
                or report.get('source_commit') != manifest['source_commit'] or report.get('target') != target
                or report.get('archive_sha256') != entry['sha256'] or not report.get('checks')):
            raise ValueError('archive has no matching passing runtime receipt: '+target)


def verify_runtime(args):
    directory = args.directory.resolve()
    # Each target job only downloads its own archive and the shared evidence.
    manifest = verify_stage(directory, inspect=False)
    report = qualify_archive(directory, manifest, args.target)
    write(args.report, report)
    return report


def publish(args):
    directory = args.directory.resolve()
    manifest = verify_stage(directory)
    verify_runtime_receipts(directory, manifest)
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
    assets = [directory/manifest['archives'][target]['name'] for target in TARGETS]
    assets += [directory/'release.json', directory/'SHA256SUMS']
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
        for target, entry in manifest['archives'].items():
            inspect_archive(downloaded/entry['name'], entry, target)
        qualify_archive(downloaded, manifest, native_target())
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
    create.add_argument('--quality', type=Path, help='reuse only a matching, passing portable quality receipt')
    create.add_argument('--candidate', action='store_true', help='non-publishable clean-branch rehearsal')
    verify = commands.add_parser('verify-runtime')
    verify.add_argument('--directory', type=Path, required=True)
    verify.add_argument('--target', choices=TARGETS, required=True)
    verify.add_argument('--report', type=Path, required=True)
    release = commands.add_parser('publish')
    release.add_argument('--directory', type=Path, required=True)
    release.add_argument('--make-public', action='store_true', help='publish only after downloaded assets pass')
    args = parser.parse_args()
    try:
        print(json.dumps({'stage': stage, 'verify-runtime': verify_runtime, 'publish': publish}[args.command](args), indent=2))
    except (ValueError, OSError, KeyError, subprocess.SubprocessError) as error:
        parser.exit(1, str(error)+'\n')


if __name__ == '__main__':
    main()

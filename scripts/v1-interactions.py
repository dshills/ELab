#!/usr/bin/env python3
"""Run the explicit v1 interaction matrix and verify every named test executed.

This deliberately does not promote rendered/offscreen evidence to native OS
queue, hardware-rate, or first-responder acceptance.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
DEFAULT = ROOT/'docs/validation/v1-acceptance/interaction-matrix.json'


def load_matrix(path):
    matrix = json.loads(path.read_text())
    if matrix.get('format') != 'elab.v1-interaction-matrix' or matrix.get('version') != 1:
        raise ValueError('unsupported interaction matrix')
    families, selected, sources = set(), set(), {}
    for row in matrix['families']:
        if not row['id'] or row['id'] in families or not row['acceptance'] or not row['tests']:
            raise ValueError('empty or duplicate interaction family')
        families.add(row['id'])
        for test in row['tests']:
            name = test['name']
            if not re.fullmatch(r'Test[A-Za-z0-9_]+', name):
                raise ValueError('invalid Go test name')
            source = ROOT/test['source']
            # Matrix sources are repository test files, never arbitrary paths
            # or shell text. Each selector must resolve to a real declaration.
            if source.is_symlink() or source.resolve().parent != ROOT/'desktop' or not source.name.endswith('_test.go'):
                raise ValueError('matrix source must be a desktop Go test')
            data = source.read_bytes()
            if not re.search(rb'^func '+name.encode()+rb'\(t \*testing\.T\)', data, re.M):
                raise ValueError('test declaration missing: '+name)
            selected.add(name)
            sources[test['source']] = hashlib.sha256(data).hexdigest()
    if not selected or matrix.get('native_open_checks') != [3, 6, 9, 11]:
        raise ValueError('matrix omitted acceptance scope or tests')
    return matrix, sorted(selected), sources


def summarize_events(path, selected):
    passed, skipped, failed, subtests = set(), set(), set(), 0
    package_passed = False
    for line in path.read_text().splitlines():
        event = json.loads(line)
        name, action = event.get('Test'), event.get('Action')
        if name and '/' in name:
            if action == 'pass':
                subtests += 1
            if action == 'fail':
                failed.add(name)
        elif name:
            if action == 'pass':
                passed.add(name)
            elif action == 'skip':
                skipped.add(name)
            elif action == 'fail':
                failed.add(name)
        elif action == 'pass':
            package_passed = True
    missing = set(selected)-passed
    if failed or skipped or missing or passed-set(selected) or not package_passed:
        raise ValueError('interaction matrix incomplete: '+json.dumps(dict(
            failed=sorted(failed), skipped=sorted(skipped), missing=sorted(missing),
            unexpected=sorted(passed-set(selected)), package_passed=package_passed)))
    return dict(top_level_passed=len(passed), subtests_passed=subtests,
                tests=sorted(passed), package_passed=package_passed)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--matrix', type=Path, default=DEFAULT)
    parser.add_argument('--artifacts', type=Path)
    parser.add_argument('--list', action='store_true')
    args = parser.parse_args()
    matrix, selected, sources = load_matrix(args.matrix)
    if args.list:
        print(json.dumps(dict(families=len(matrix['families']), tests=selected), indent=2))
        return
    if not args.artifacts:
        parser.error('--artifacts is required to run')
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    command = ['go', 'test', '-json', '-race', '-tags', 'offscreen', '-count=1',
               '-timeout', '15m', '-run', '^('+'|'.join(selected)+')$', '.']
    env = dict(os.environ, GOMAXPROCS='2', GOWORK='off', GOPROXY='off',
               GOTOOLCHAIN='local', GOFLAGS='-p=1 -mod=readonly', GOCACHE=str(ROOT/'build/go-cache'))
    started = time.monotonic()
    report = dict(passed=False, scope=matrix['scope'], native_open_checks=matrix['native_open_checks'],
                  matrix_sha256=hashlib.sha256(args.matrix.read_bytes()).hexdigest(),
                  source_sha256=sources, command=command, family_count=len(matrix['families']))
    try:
        with (assets/'events.jsonl').open('w') as output, (assets/'stderr.txt').open('w') as errors:
            result = subprocess.run(command, cwd=ROOT/'desktop', env=env,
                                    stdout=output, stderr=errors, timeout=930)
        report['exit_code'] = result.returncode
        if result.returncode:
            raise RuntimeError('interaction tests failed; see retained events/stderr')
        if any(hashlib.sha256((ROOT/path).read_bytes()).hexdigest() != sha
               for path, sha in sources.items()):
            raise RuntimeError('matrix sources changed during the run')
        report.update(summarize_events(assets/'events.jsonl', selected))
        report['passed'] = True
    finally:
        report['elapsed_s'] = time.monotonic()-started
        (assets/'report.json').write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps(dict(passed=True, families=report['family_count'],
                         tests=report['top_level_passed'], subtests=report['subtests_passed'])))


if __name__ == '__main__':
    main()

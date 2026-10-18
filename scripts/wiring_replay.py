#!/usr/bin/env python3
"""Run fresh wiring observations against an exactly matched private review.

No live endpoint is accepted. Admission precedes test/agent execution. The
worker runs under the pinned SDK's bounded process-group runner.
"""
import argparse
import hashlib
import importlib.util
import json
import math
import os
from pathlib import Path
import re
import stat
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
LEDGER = 'research/13-native-command-inventory.json'
RUNTIME_DIRS = ('cmd', 'desktop', 'internal', 'transport', 'workbench')
CASES = {'mounted-complete', 'mounted-undo', 'native-adapter-complete'}
TEST_ARGS = ['go', 'test', '-race', '-tags=offscreen', '-count=1', '-timeout=60s',
             '-run', '^(TestMountedNamedWiring|TestNativeNamedWiring)', '-v', './...']


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def git_blob(root, source, name):
    return subprocess.check_output(['git', 'show', source+':'+name], cwd=root, timeout=15)


def git_files(root, source):
    tree = subprocess.check_output(['git', 'ls-tree', '-r', '-z', source], cwd=root, timeout=15)
    result = {}
    for entry in tree.decode().split('\0')[:-1]:
        header, name = entry.split('\t', 1)
        _, kind, identity = header.split()
        if kind != 'blob':
            raise ValueError('unsupported source tree entry')
        result[name] = identity
    return result


def git_identity(data):
    return hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()


def runtime_input(name):
    path = Path(name)
    return (name in ('go.mod', 'go.sum') or
            (path.parts[0] in RUNTIME_DIRS and not path.name.endswith('_test.go')
             and not path.name.startswith('.') and path.suffix != '.md'))


def regular(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError('expected a regular file: '+str(path))
    return path


def validate_release(root, receipt_path):
    receipt = json.loads(regular(receipt_path).read_text())
    source = receipt.get('source_commit', '')
    if not re.fullmatch('[0-9a-f]{40}', source) or receipt.get('clean_build_vcs') is not True:
        raise ValueError('review requires a clean, exact source commit')
    ledger_bytes = regular(root/LEDGER).read_bytes()
    if ledger_bytes != git_blob(root, source, LEDGER):
        raise ValueError('current source ledger differs from review '+source[:7]+'; select a matching review')
    ledger = json.loads(ledger_bytes)
    tree = git_files(root, source)
    expected_inputs = {name for name in tree if runtime_input(name)}
    actual_inputs = {str(p.relative_to(root)) for folder in RUNTIME_DIRS
                     for p in (root/folder).rglob('*') if p.is_file() and runtime_input(str(p.relative_to(root)))}
    actual_inputs.update(name for name in ('go.mod', 'go.sum') if (root/name).is_file())
    if actual_inputs != expected_inputs:
        raise ValueError('runtime source input set differs from review')
    for name in sorted(expected_inputs):
        if git_identity(regular(root/name).read_bytes()) != tree[name]:
            raise ValueError('runtime source differs from review: '+name)
    for name, expected in ledger['source_sha256'].items():
        if digest(regular(root/name)) != expected:
            raise ValueError('source ledger hash differs: '+name)
    if json.loads(regular(root/'third_party/vellum/sdk.lock.json').read_text()) != receipt['sdk']:
        raise ValueError('SDK pin differs from review')
    review = Path(receipt['review_path'])
    if review.is_symlink() or not review.resolve().is_relative_to((root/'build').resolve()):
        raise ValueError('review must be an owned build folder')
    manifest_path = regular(review/'RELEASE_MANIFEST.json')
    manifest = json.loads(manifest_path.read_text())
    files = manifest['files']
    paths = list(review.rglob('*'))
    if any(p.is_symlink() or not (p.is_file() or p.is_dir()) for p in paths):
        raise ValueError('review contains a link or special entry')
    if {str(p.relative_to(review)) for p in paths if p.is_file()} != set(files) | {'RELEASE_MANIFEST.json'}:
        raise ValueError('review file set differs from manifest')
    if len(files) != receipt['manifest_file_entries'] or manifest['source_record_commit'] != source or manifest['source_build_clean'] is not True:
        raise ValueError('review manifest identity differs from receipt')
    for name, expected in files.items():
        path = regular(review/name)
        if digest(path) != expected['sha256'] or oct(stat.S_IMODE(path.stat().st_mode)) != expected['mode']:
            raise ValueError('review manifest mismatch: '+name)
    if digest(regular(review/'ELab.app/Contents/MacOS/ELab')) != receipt['native_sha256']:
        raise ValueError('native app differs from receipt')
    agent = regular(review/'elab-agent')
    if digest(agent) != receipt['agent_sha256']:
        raise ValueError('agent differs from receipt')
    info = regular(review/'Evidence/Agent-build-info.txt').read_text()
    if 'vcs.revision='+source not in info or 'vcs.modified=false' not in info:
        raise ValueError('agent build identity differs from receipt')
    return dict(source_commit=source, release=receipt['version']+' /'+receipt['build'],
                agent=str(agent), agent_sha256=digest(agent), receipt_sha256=digest(receipt_path),
                ledger_sha256=hashlib.sha256(ledger_bytes).hexdigest(),
                production_source_files_verified=len(ledger['source_sha256']),
                runtime_inputs_verified=len(expected_inputs), manifest_file_entries_verified=len(files),
                manifest_sha256=digest(manifest_path))


def producer_sources(root):
    paths = [*sorted((root/'desktop').rglob('*_test.go')),
             root/'scripts/wiring_replay.py', root/'scripts/persistence-demo.py']
    return {str(p.relative_to(root)): digest(regular(p)) for p in paths}


def observations(log):
    lines = log.read_text().splitlines()
    if 'PASS' not in lines or any(line.startswith('FAIL') or '--- FAIL:' in line or '(cached)' in line for line in lines):
        raise ValueError('focused run must pass without cached observations')
    result = {}
    for line in lines:
        if 'named-wiring-observation ' in line:
            item = json.loads(line.split('named-wiring-observation ', 1)[1])
            if item['case'] in result:
                raise ValueError('duplicate wiring observation')
            result[item['case']] = item
    if set(result) != CASES:
        raise ValueError('missing wiring observation cases')
    return result


def observe(state):
    result = state.get('result')
    stale = not result or result['stale']
    return dict(components=state['components'], wires=state['wires'], nets=state['nets'],
                drawing=state['drawing'], selected=state['selected'], undo_name=state['undo_name'],
                result_stale=stale, current_a=None if stale else result['current_a'])


def compare(expected, actual):
    expected = {key: value for key, value in expected.items() if key != 'case'}
    if expected.keys() != actual.keys():
        raise ValueError('observation fields differ')
    for key, value in expected.items():
        equal = (math.isclose(value, actual[key], rel_tol=1e-10, abs_tol=5e-12)
                 if key == 'current_a' and value is not None else value == actual[key])
        if not equal:
            raise ValueError('observation differs: '+key)


def replay(root, proof, captured):
    spec = importlib.util.spec_from_file_location('qualified_wiring_client', root/'scripts/persistence-demo.py')
    helper = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(helper)
    client = helper.Client(command_preconditions=True, agent_binary=proof['agent'])
    try:
        initial = client.ask('project.inspect')
        if initial['components'] or initial['wires'] or initial['document']['generation'] != 1:
            raise ValueError('owned agent did not start fresh')
        parts = [dict(id='V1', kind='dc-source', value='5V', position=dict(x=-200, y=0)),
                 dict(id='R1', kind='resistor', value='1k', position=dict(x=0, y=-120)),
                 dict(id='D1', kind='led', position=dict(x=200, y=0)),
                 dict(id='GND', kind='ground', position=dict(x=-200, y=120))]
        for part in parts:
            client.ask('component.place', part, True)
            client.ask('simulation.wait')
        client.ask('drawing.tool.set', dict(tool='wire'), True)
        for source, target in [('V1.positive', 'R1.1'), ('R1.2', 'D1.anode'),
                               ('D1.cathode', 'GND.reference'), ('GND.reference', 'V1.negative')]:
            pair = {'from': source, 'to': target}
            client.ask('wire.terminals.set', pair, True)
            client.ask('wire.connect', pair, True)
            client.ask('simulation.wait')
        complete = observe(client.ask('simulation.wait'))
        for name in ('mounted-complete', 'native-adapter-complete'):
            compare(captured[name], complete)
        if not math.isclose(complete['current_a'], .002916796577441, rel_tol=0, abs_tol=5e-12):
            raise ValueError('LED reference current differs')
        client.ask('edit.undo', {}, True)
        undone = observe(client.ask('simulation.wait'))
        compare(captured['mounted-undo'], undone)
        if len(undone['wires']) != 3 or not undone['result_stale']:
            raise ValueError('Undo did not disconnect/stale the circuit')
        return dict(complete=complete, undone=undone)
    finally:
        client.close()


def worker(root, receipt, artifacts):
    proof = validate_release(root, receipt)
    producer = json.loads(regular(artifacts/'producer.json').read_text())
    if producer['release'] != proof or producer['sources'] != producer_sources(root) or producer['test_command'] != TEST_ARGS or producer['test_exit_code'] != 0:
        raise ValueError('observation producer identity differs')
    log = regular(artifacts/'focused.txt')
    if digest(log) != producer['observations_sha256']:
        raise ValueError('observation transcript changed')
    result = replay(root, proof, observations(log))
    if proof != validate_release(root, receipt) or producer['sources'] != producer_sources(root):
        raise ValueError('release or producer changed during replay')
    report = dict(passed=True, release_identity=proof, producer=producer, compared_cases=sorted(CASES),
                  transport='fresh owned headless stdio', **result,
                  limits=['No physical OS menu/IME/VoiceOver or visible p95 evidence',
                          'No product rebuild, promotion or owner acceptance'])
    with (artifacts/'report.json').open('x') as output:
        output.write(json.dumps(report, indent=2)+'\n')


def run(root, receipt, artifacts):
    proof = validate_release(root, receipt)  # Before any test/agent execution.
    if artifacts.exists() or artifacts.is_symlink():
        raise FileExistsError('replay artifacts already exist')
    from runtime_isolation import pinned_child_runner
    _, run_child = pinned_child_runner()
    artifacts.mkdir(parents=True, exist_ok=False)
    sources = producer_sources(root)
    environment = dict(os.environ, GOWORK='off', GOFLAGS='', GOTOOLCHAIN='local', GOPROXY='off',
                       GOCACHE=str(root/'build/go-cache'), CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native')
    with (artifacts/'focused.txt').open('x') as log:
        completed = run_child(TEST_ARGS, cwd=root/'desktop', env=environment,
                              stdout=log, stderr=subprocess.STDOUT, timeout=90, check=True)
    observations(artifacts/'focused.txt')
    if validate_release(root, receipt) != proof or sources != producer_sources(root):
        raise ValueError('release or producer changed during observation')
    producer = dict(release=proof, sources=sources, test_command=TEST_ARGS,
                    test_exit_code=completed.returncode, observations_sha256=digest(artifacts/'focused.txt'))
    with (artifacts/'producer.json').open('x') as output:
        output.write(json.dumps(producer, indent=2)+'\n')
    with (artifacts/'replay.txt').open('x') as log:
        run_child([sys.executable, '-B', str(root/'scripts/wiring_replay.py'), '--receipt', str(receipt),
                   '--artifacts', str(artifacts), '--worker'], cwd=root, stdout=log,
                  stderr=subprocess.STDOUT, timeout=60, check=True)
    report = json.loads((artifacts/'report.json').read_text())
    if report.get('passed') is not True or report['release_identity'] != proof:
        raise ValueError('replay did not qualify the admitted review')
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--receipt', required=True, type=Path)
    parser.add_argument('--artifacts', required=True, type=Path)
    parser.add_argument('--worker', action='store_true', help=argparse.SUPPRESS)
    args = parser.parse_args()
    try:
        if args.worker:
            worker(ROOT, args.receipt.resolve(), args.artifacts.resolve())
        else:
            report = run(ROOT, args.receipt.resolve(), args.artifacts.resolve())
            print(json.dumps(dict(passed=True, release=report['release_identity']['release'],
                                  source=report['release_identity']['source_commit'], report=str(args.artifacts/'report.json'))))
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        parser.exit(1, str(error)+'\n')


if __name__ == '__main__':
    main()

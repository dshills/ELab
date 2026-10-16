#!/usr/bin/env python3
"""Run every connected demo's refusal against an owned nonempty host."""
import argparse
import ast
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('viewport', ROOT/'scripts/viewport-fit-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def demo_arguments(path):
    options = {}
    for node in ast.walk(ast.parse(path.read_text())):
        if (isinstance(node, ast.Call) and getattr(node.func, 'attr', None) == 'add_argument'
                and node.args and isinstance(node.args[0], ast.Constant)
                and isinstance(node.args[0].value, str)):
            options[node.args[0].value] = {keyword.arg: keyword.value for keyword in node.keywords}
    return options


def run_audit(binary, native, base):
    base.mkdir(parents=True, exist_ok=False)
    endpoint = base/'endpoint.json'
    argv = [str(binary), '--agent-endpoint' if native else '--listen', str(endpoint),
            '--recovery-directory', str(base/'recovery'), '--appearance-file', str(base/'appearance.json')]
    sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
    client = None
    result = None
    with (base/'host.log').open('x') as log:
        host = subprocess.Popen(argv, stdout=log, stderr=log)
        try:
            deadline = time.monotonic()+20
            while not endpoint.exists():
                assert host.poll() is None and time.monotonic() < deadline, 'owned host did not start'
                time.sleep(.02)
            client = helper.Client(str(endpoint))
            mutate = lambda command, params=None: client.ask(command, params, True)
            mutate('recovery.configure', {'directory': str(base/'recovery'), 'enabled': False, 'interval_s': 1})
            mutate('example.load', {'id': 'rlc'})
            mutate('simulation.pause')
            client.ask('simulation.wait')
            mutate('simulation.step', {'duration_s': '200 us'})
            operation = mutate('simulation.verify', {'duration_s': '2 ms', 'probe_ids': ['v:C1', 'i:L1']})['last_operation']
            deadline = time.monotonic()+20
            while client.ask('operation.inspect', {'id': operation})['operation']['status'] != 'completed':
                assert time.monotonic() < deadline, 'fixture capture did not complete'
                time.sleep(.01)
            capture = base/'capture-before.json'
            mutate('verification.export', {'id': operation, 'path': str(capture), 'format': 'json'})
            project = base/'saved-fixture.elab'
            mutate('project.save', {'path': str(project)})
            mutate('component.set', {'id': 'R1', 'value': '60 ohm'})
            # Let automatic native viewport publication settle. The circuit is
            # paused, recovery is disabled, and every subsequent public field
            # must stay identical across each refused demo, including view/UI.
            deadline, stable_since, previous = time.monotonic()+10, None, None
            while True:
                before = client.ask('project.inspect')
                if before == previous and (not native or before['viewport']['registered']):
                    if stable_since is None:
                        stable_since = time.monotonic()
                    if time.monotonic()-stable_since >= .3:
                        break
                else:
                    stable_since = None
                previous = before
                assert time.monotonic() < deadline, 'owned fixture did not settle'
                time.sleep(.02)
            with (base/'fixture-snapshot.json').open('x') as output:
                output.write(json.dumps(before, sort_keys=True, indent=2)+'\n')
            assert before['document']['dirty'] and before['document']['recording_count'] == 1
            assert before['paused'] and before['mixed']['time_s'] == .0002 and before['undo_entry']
            project_hash, capture_bytes = sha(project), capture.read_bytes()
            cases = []
            for path in sorted((ROOT/'scripts').glob('*.py')):
                options = demo_arguments(path)
                if '--connect' not in options:
                    continue
                case = base/path.stem
                case.mkdir()
                command = ['python3', str(path), '--connect', str(endpoint)]
                for option, keywords in options.items():
                    if option == '--connect':
                        continue
                    if option in ('--output', '--discovery-output'):
                        command += [option, str(case/(option[2:]+'.json'))]
                    elif option == '--artifacts':
                        command += [option, str(case/'assets')]
                    elif option == '--legacy':
                        command += [option, str(project)]
                    elif isinstance(keywords.get('required'), ast.Constant) and keywords['required'].value:
                        assert 'choices' in keywords, ('unhandled required demo argument', path.name, option)
                        command += [option, str(ast.literal_eval(keywords['choices'])[0])]
                with (case/'run.log').open('x') as case_log:
                    run = subprocess.run(command, stdout=case_log, stderr=subprocess.STDOUT, timeout=35)
                assert run.returncode != 0, ('demo accepted nonempty target', path.name)
                assert 'demo requires a fresh blank owned session' in (case/'run.log').read_text(), path.name
                assert host.poll() is None, ('demo closed target', path.name)
                observed = client.ask('project.inspect')
                if observed != before:
                    with (case/'failure-snapshot.json').open('x') as output:
                        output.write(json.dumps(observed, sort_keys=True, indent=2)+'\n')
                assert observed == before, ('demo changed target', path.name,
                    [key for key in before.keys() | observed.keys() if before.get(key) != observed.get(key)])
                assert sha(project) == project_hash, ('demo changed saved fixture', path.name)
                assert not (case/'output.json').exists(), ('refusal wrote a success report', path.name)
                cases.append(dict(script=str(path.relative_to(ROOT)), sha256=sha(path), refused=True,
                    target_live=True, state_preserved=True, saved_bytes_preserved=True))
            capture_after = base/'capture-after.json'
            mutate('verification.export', {'id': operation, 'path': str(capture_after), 'format': 'json'})
            assert capture_after.read_bytes() == capture_bytes
            result = dict(passed=True, transport='native-socket' if native else 'headless-socket',
                binary_sha256=sha(binary), demos=len(cases), cases=cases,
                fixture_snapshot=before,
                fixture=dict(components=len(before['components']), public_snapshot_fields=len(before), recording_count=1,
                    accepted_time_s=.0002, paused_stale=True, dirty=True,
                    saved_project_sha256=project_hash, captured_json_sha256=sha(capture)),
                capture_bytes_preserved=True,
                limitations=['Admission check is not an ownership credential or exclusive editing lease',
                    'All supplied hosts and fixture paths are created and owned by this audit',
                    'Refusal coverage does not establish successful behavior for every historical demo'])
            mutate('app.close', {'discard_changes': True})
            client.close()
            client = None
            assert host.wait(timeout=10) == 0
            result['owned_host_exit'] = host.returncode
            with (base/'report.json').open('x') as output:
                output.write(json.dumps(result, indent=2)+'\n')
            print(json.dumps(dict(passed=True, transport=result['transport'], demos=len(cases))), flush=True)
        finally:
            if client is not None:
                client.close()
            if host.poll() is None:
                host.terminate()
                try:
                    host.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    host.kill()
                    host.wait(timeout=5)
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--artifacts', type=Path, required=True, help='New audit directory')
    parser.add_argument('--output', type=Path, required=True, help='New summary JSON')
    parser.add_argument('--native-binary', type=Path, help='Also create and close an owned native host')
    args = parser.parse_args()
    assert not args.output.exists()
    base = args.artifacts.resolve()
    base.mkdir(parents=True, exist_ok=False)
    reports = [run_audit(ROOT/'build/elab-agent', False, base/'headless')]
    if args.native_binary:
        reports.append(run_audit(args.native_binary.resolve(), True, base/'native'))
    report = dict(passed=True, adapters=reports)
    with args.output.open('x') as output:
        output.write(json.dumps(report, indent=2)+'\n')


if __name__ == '__main__':
    main()

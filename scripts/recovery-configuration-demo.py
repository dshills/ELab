#!/usr/bin/env python3
"""Qualify observable recovery setup through public headless or native requests."""
import argparse
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    root = args.artifacts.resolve()
    root.mkdir(mode=0o700, parents=True, exist_ok=False)
    old, new = root / 'original-recovery', root / 'new-recovery'
    old.mkdir(mode=0o700)
    new.mkdir(mode=0o700)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def completed(operation_id):
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            state = client.ask('project.inspect')
            work = state['project_work']
            assert work['id'] == operation_id and work['status'] not in ('failed', 'canceled', 'superseded'), work
            if work['status'] == 'completed':
                return state
            time.sleep(.01)
        raise AssertionError('Recovery setup did not complete')

    def live(state):
        return {k: copy.deepcopy(state.get(k)) for k in ('components', 'wires', 'run_id', 'electrical_revision',
                'result', 'transient', 'mixed', 'npn', 'document', 'undo_entry', 'view')}

    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        source = root / 'source.elab'
        client.ask('project.save', {'path': str(source)}, True)
        source_digest = hashlib.sha256(source.read_bytes()).hexdigest()
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': '1 ms'}, True)
        configured = client.ask('recovery.configure', {'directory': str(old), 'enabled': False, 'interval_s': 2}, True)
        assert configured['project_work']['recovery_configuration_applied']
        client.ask('recovery.checkpoint', {}, True)
        candidate = client.ask('project.inspect')['recovery']['candidates'][0]
        capsule = old / (candidate['id'] + '.elab-recovery')
        capsule_digest = hashlib.sha256(capsule.read_bytes()).hexdigest()
        before = client.ask('project.inspect')
        full = client.ask('capabilities.list')
        details = [client.ask('capabilities.list', {'command': command}) for command in
                   ('recovery.configure', 'recovery.configure.start')]
        for detail in details:
            interval = detail['capabilities'][0]['input_schema']['properties']['interval_s']
            assert interval['$ref'] == 'urn:elab:time:1' and interval['minimum'] == 1 and interval['maximum'] == 60
        assert before == client.ask('project.inspect')
        admitted = client.ask('recovery.configure.start', {'directory': str(new), 'enabled': True, 'interval_s': '60 s'}, True)
        operation_id = admitted['project_work']['id']
        assert admitted['project_work']['status'] == 'queued' and admitted['project_work']['cancelable']
        assert not admitted['project_work']['recovery_configuration_applied']
        assert admitted['recovery'] == before['recovery']
        state = completed(operation_id)
        assert live(before) == live(state)
        assert state['recovery']['directory'] == str(new) and state['recovery']['enabled']
        assert state['project_work']['recovery_configuration_applied']
        assert client.ask('project.work.inspect', {'id': operation_id})['recovery_configuration_applied']
        assert not list(old.glob('session-*.json'))
        assert hashlib.sha256(capsule.read_bytes()).hexdigest() == capsule_digest
        leases = sorted(p.name for p in new.glob('session-*.json'))
        same = client.ask('recovery.configure', {'directory': str(new), 'enabled': False, 'interval_s': '1500 ms'}, True)
        assert same['recovery']['interval_s'] == 1.5 and not same['recovery']['enabled']
        assert sorted(p.name for p in new.glob('session-*.json')) == leases
        client.ask('recovery.select', {'id': ''}, True)
        same = client.ask('recovery.configure', {'directory': str(new), 'enabled': False, 'interval_s': 2}, True)
        assert same['recovery']['selected'] == ''
        for directory, code in [(root / 'insecure', 'file-error'), (root / 'overfull', 'invalid-params')]:
            directory.mkdir(mode=0o700)
            if code == 'file-error':
                directory.chmod(0o755)
            else:
                for i in range(128):
                    (directory / f'entry-{i:03}').write_text('owned fixture to preserve')
            previous = client.ask('project.inspect')
            client.ask('recovery.configure', {'directory': str(directory), 'enabled': True, 'interval_s': 2}, True, code)
            failed = client.ask('project.inspect')
            assert failed['recovery'] == previous['recovery'] and live(failed) == live(previous)
            assert not failed['project_work']['recovery_configuration_applied']
            assert not list(directory.glob('session-*.json'))
        previous = client.ask('project.inspect')
        client.ask('recovery.configure.start', {'directory': str(new), 'enabled': True, 'interval_s': '0 s'}, True, 'invalid-params')
        assert previous == client.ask('project.inspect')
        restored = client.ask('recovery.configure', {'directory': str(old), 'enabled': False, 'interval_s': 2}, True)
        assert restored['recovery']['candidates'][0]['id'] == candidate['id']
        assert restored['recovery']['candidates'][0]['sha256'] == capsule_digest
        assert live(restored) == live(before)
        assert hashlib.sha256(source.read_bytes()).hexdigest() == source_digest
        semantic = {k: live(restored)[k] for k in ('components', 'wires', 'run_id', 'electrical_revision',
                                               'result', 'transient', 'mixed', 'npn', 'view')}
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'commands': len(full['capabilities']), 'live_semantic': semantic,
                  'checks': ['waiting and immediate setup through shared project work', 'admission differs from installed configuration',
                             'unit-time detailed schemas and query purity', 'unchanged paused RC graph/physics/history/dirty state',
                             'old lease released; same-store lease reused', 'older capsule and explicit saved file preserved',
                             'failed private-directory/overflow setup preserves previous configuration and cleans staged lease',
                             'invalid interval refuses before admission', 'explicit cleared selection preserved',
                             'previous recovery directory/capsule can be selected again'],
                  'limits': ['Blocked filesystem responsiveness, cancellation/cleanup and retry interleavings are covered by controlled core tests',
                             'Dismiss/inspection and serialized shutdown remain separate work; no power-loss or full accessibility claim']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as target:
            target.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'passed': True, 'transport': report['transport'], 'commands': report['commands']}))
    finally:
        client.close()


if __name__ == '__main__':
    main()

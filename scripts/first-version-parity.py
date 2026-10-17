#!/usr/bin/env python3
"""Reconstruct the native walkthrough circuits through public agent commands.

An optional --connect must designate a fresh blank process owned by the caller.
The native host is left paused; the forwarding client alone is closed.
"""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path

from demo_session import connect_demo_client

ROOT = Path(__file__).resolve().parents[1]
FIXTURES = ROOT / 'research/ui-audit/first-version'
spec = importlib.util.spec_from_file_location('persistence', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def numerical(state):
    result = {key: state[key] for key in ('result', 'mixed') if key in state}
    if 'transient' in state:
        result['sample'] = state['transient']['sample']
    return result


def compare(left, right, path='result'):
    """Compare every reported field except independent run identities."""
    if isinstance(left, dict):
        assert isinstance(right, dict), path
        left = {k: v for k, v in left.items() if k not in ('run_id', 'electrical_revision')}
        right = {k: v for k, v in right.items() if k not in ('run_id', 'electrical_revision')}
        assert left.keys() == right.keys(), (path, left.keys(), right.keys())
        for key in left:
            compare(left[key], right[key], path+'.'+key)
    elif isinstance(left, list):
        assert isinstance(right, list) and len(left) == len(right), path
        for i, (a, b) in enumerate(zip(left, right)):
            compare(a, b, f'{path}[{i}]')
    elif isinstance(left, (float, int)) and not isinstance(left, bool):
        assert math.isclose(left, right, rel_tol=1e-9, abs_tol=1e-12), (path, left, right)
    else:
        assert left == right, (path, left, right)


def construct(client, document):
    client.ask('simulation.pause', {}, True)
    for part in document['graph']['components']:
        params = {key: part[key] for key in ('id', 'kind', 'position')}
        if part['value']:
            params['value'] = format(part['value'], '.17g')
        client.ask('component.place', params, True)
        for _ in range(part['rotation']//90):
            client.ask('component.rotate', {'id': part['id']}, True)
        if 'initial_voltage_v' in part:
            client.ask('capacitor.initial.set', {'id': part['id'], 'value': format(part['initial_voltage_v'], '.17g')}, True)
    for wire in document['graph']['wires']:
        client.ask('wire.connect', {key: wire[key] for key in ('id', 'from', 'to')}, True)
    client.ask('simulation.transient.set', document['transient_settings'], True)
    client.ask('scope.probes.set', document['scope_probes'], True)
    client.ask('simulation.restart', {}, True)
    return client.ask('simulation.wait')


def observe(client, name):
    if name == 'rc':
        client.ask('simulation.step', {'duration_s': .01}, True)
    elif name == 'flasher':
        for _ in range(16):
            client.ask('simulation.step', {'duration_s': 1}, True)
        client.ask('simulation.step', {'duration_s': .575}, True)
    return client.ask('simulation.wait')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--artifacts', required=True, type=Path)
    args = parser.parse_args()
    output = args.artifacts.resolve()
    output.mkdir(parents=True, exist_ok=False)
    native = connect_demo_client(helper.Client, args.connect, command_preconditions=True) if args.connect else None
    cases = [('led', 'keyboard-led-ui.elab'), ('rc', 'rc-built-ui.elab'), ('flasher', 'flasher-timing.elab')]
    reports = []
    try:
        for name, filename in cases:
            document = json.loads((FIXTURES/filename).read_text())
            client = helper.Client(command_preconditions=True)
            try:
                catalog = {c['id'] for c in client.ask('capabilities.list')['capabilities']}
                assert {'component.place', 'wire.connect', 'simulation.step', 'project.save'} <= catalog
                built = construct(client, document)
                assert built['components'] == document['graph']['components'], name+' components'
                assert built['wires'] == document['graph']['wires'], name+' wires'
                project = output/(name+'.elab')
                client.ask('project.save', {'path': str(project)}, True)
                observed = observe(client, name)
                if name == 'led':
                    reference = json.loads((FIXTURES/'workflow-checkpoint.json').read_text())['native_edited_result']
                    compare(reference, observed['result'])
                elif name == 'rc':
                    reference = json.loads((FIXTURES/'rc-observed.json').read_text())['transient']['sample']
                    compare(reference, observed['transient']['sample'])
                else:
                    reference = json.loads((FIXTURES/'flasher-observed-reference.json').read_text())
                    compare(reference['mixed'], observed['mixed'])
                    compare(reference['transient']['sample'], observed['transient']['sample'])
                if native:
                    native.ask('project.load', {'path': str(project), 'discard_changes': True}, True)
                    native.ask('simulation.wait')
                    mounted = observe(native, name)
                    assert mounted['components'] == observed['components'] and mounted['wires'] == observed['wires']
                    compare(numerical(observed), numerical(mounted))
                (output/(name+'-observed.json')).write_text(json.dumps(observed, indent=2)+'\n')
                reports.append({'task': name, 'constructed_graph_matches_native_fixture': True,
                                'native_ui_numerical_reference_matches': True,
                                'native_endpoint_matches': bool(native),
                                'fixture_sha256': hashlib.sha256((FIXTURES/filename).read_bytes()).hexdigest(),
                                'project_sha256': hashlib.sha256(project.read_bytes()).hexdigest()})
            finally:
                client.close()
        (output/'report.json').write_text(json.dumps({'passed': True, 'cases': reports,
            'comparison': 'Every component/wire field; every reported result field except run/electrical identities; numerical rel 1e-9 / abs 1e-12',
            'limitations': ['Native endpoint loading/observation does not replace the recorded physical UI construction walkthrough',
                            'File failure, crash recovery, cancellation and live interaction have separate acceptance checks']}, indent=2)+'\n')
        print(json.dumps({'passed': True, 'cases': reports}, indent=2))
    finally:
        if native:
            native.close()


if __name__ == '__main__':
    main()

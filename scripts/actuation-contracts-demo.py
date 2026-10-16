#!/usr/bin/env python3
"""Discover and exercise switch/wiper semantics through public commands."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path

from validation_artifacts import discovery_bundle

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
COMMANDS = ['switch.set', 'potentiometer.wiper.set']


def physics(state):
    return {key: state.get(key) for key in ('mixed', 'transient', 'run_id', 'electrical_revision')}


def invariant(state):
    return dict(physics(state), **{key: state.get(key) for key in ('components', 'wires', 'document', 'undo_entry', 'redo_entry')})


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native session; closed unless --keep-open succeeds')
    parser.add_argument('--keep-open', action='store_true')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assert not args.keep_open or args.connect
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect)
    mutate = lambda name, params=None: client.ask(name, params, True)
    fresh = None
    keep = False

    def refuse(command, params):
        before = invariant(client.ask('project.inspect'))
        client.ask(command, params, True, 'invalid-params')
        assert invariant(client.ask('project.inspect')) == before

    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh owned blank session'
        before = invariant(client.ask('project.inspect'))
        responses = {name: client.ask('capabilities.list', {'command': name}) for name in COMMANDS}
        assert invariant(client.ask('project.inspect')) == before
        schemas = {name: result['capabilities'][0]['input_schema'] for name, result in responses.items()}
        for name, schema in schemas.items():
            assert schema['x-command-contract']['scope'] == 'interactive-part-actuation'
            assert schema['x-interaction-summary'] and schema['x-target-inventory']
        assert schemas['potentiometer.wiper.set']['properties']['position']['minimum'] == 0
        assert schemas['potentiometer.wiper.set']['properties']['position']['maximum'] == 1
        discovery = discovery_bundle(responses, assets/'discovery.json')
        mutate('simulation.pause')
        for identity, kind, x, y, value in [('V1', 'dc-source', -280, 0, '5 V'),
                                           ('S1', 'switch', -160, -100, ''),
                                           ('P1', 'potentiometer', 0, -100, '10k'),
                                           ('C1', 'capacitor', 220, 0, '1uF'),
                                           ('GND1', 'ground', 80, 180, '')]:
            mutate('component.place', {'id': identity, 'kind': kind, 'position': {'x': x, 'y': y}, 'value': value})
        # Incomplete drawings accept saved startup configuration without a live job.
        mutate('switch.set', {'id': 'S1', 'closed': True})
        for start, end in [('V1.positive', 'S1.1'), ('S1.2', 'P1.1'), ('P1.2', 'GND1.reference'),
                           ('P1.wiper', 'C1.1'), ('C1.2', 'GND1.reference'), ('V1.negative', 'GND1.reference')]:
            mutate('wire.connect', {'from': start, 'to': end})
        mutate('simulation.restart')
        client.ask('simulation.wait')
        mutate('scope.probes.set', {'ids': ['v:C1', 'i:P1.wiper', 'i:S1']})
        mutate('project.save', {'path': str(assets/'startup.elab')})
        mutate('simulation.step', {'duration_s': .005})
        before = client.ask('project.inspect')
        mutate('switch.set', {'id': 'S1', 'closed': False})
        queued = client.ask('simulation.wait')
        assert physics(queued) == physics(before)
        assert queued['switch_states'][0] == dict(id='S1', desired_closed=False, applied_closed=True, pending=True)
        unchanged = invariant(queued)
        mutate('switch.set', {'id': 'S1', 'closed': False})
        assert invariant(client.ask('project.inspect')) == unchanged
        mutate('switch.set', {'id': 'S1', 'closed': True})
        coalesced = client.ask('project.inspect')
        assert not coalesced['switch_states'][0]['pending'] and physics(coalesced) == physics(before) and not coalesced['document']['dirty']
        mutate('switch.set', {'id': 'S1', 'closed': False})
        admission = mutate('potentiometer.wiper.set', {'id': 'P1', 'position': .75})
        assert admission['input_work']['command'] == 'potentiometer.wiper.set'
        assert admission['input_work']['requested_wiper'] == dict(id='P1', position=.75)
        moved = client.ask('simulation.wait')
        assert moved['input_work']['status'] == 'completed' and moved['switch_states'][0]['pending']
        assert moved['mixed']['potentiometers'][0]['wiper_position'] == .75
        assert moved['transient']['sample']['time_s'] == before['transient']['sample']['time_s']
        assert moved['mixed']['capacitors'][0]['voltage_v'] == before['mixed']['capacitors'][0]['voltage_v']
        assert moved['run_id'] == before['run_id'] and moved['electrical_revision'] == before['electrical_revision']
        no_op = invariant(moved)
        mutate('potentiometer.wiper.set', {'id': 'P1', 'position': .75})
        assert invariant(client.ask('project.inspect')) == no_op
        mutate('simulation.step', {'duration_s': .001})
        applied = client.ask('project.inspect')
        target = 5*7499.5/10001
        tau = (2501.5*7499.5/10001)*1e-6
        expected = target + (before['mixed']['capacitors'][0]['voltage_v']-target)*math.exp(-.001/tau)
        assert abs(applied['mixed']['capacitors'][0]['voltage_v']-expected) < .002
        assert not applied['switch_states'][0]['pending'] and not applied['switch_states'][0]['applied_closed']
        assert abs(applied['transient']['sample']['time_s']-.006) < 1e-12
        for command, params in [('switch.set', {'id': 'P1', 'closed': True}),
                                ('potentiometer.wiper.set', {'id': 'V1', 'position': .5}),
                                ('potentiometer.wiper.set', {'id': 'P1', 'position': 50}),
                                ('potentiometer.wiper.set', {'id': 'P1', 'position': math.nextafter(0, -math.inf)}),
                                ('potentiometer.wiper.set', {'id': 'P1', 'position': math.nextafter(1, math.inf)})]:
            refuse(command, params)
        mutate('edit.batch', {'name': 'Actuator startup', 'edits': [
            {'command': 'switch.set', 'params': {'id': 'S1', 'closed': True}},
            {'command': 'potentiometer.wiper.set', 'params': {'id': 'P1', 'position': .25}}]})
        batch = client.ask('project.inspect')
        assert batch['mixed']['stale'] and batch['undo_name'] == 'Actuator startup'
        mutate('edit.undo')
        assert next(p for p in client.ask('project.inspect')['components'] if p['id'] == 'P1')['wiper_position'] == .75
        mutate('edit.redo')
        mutate('simulation.restart')
        client.ask('simulation.wait')
        mutate('simulation.step', {'duration_s': .001})
        path = assets/'actuators.elab'
        mutate('project.save', {'path': str(path)})
        saved = json.loads(path.read_bytes())
        fresh = helper.Client(None)
        fresh.ask('project.load', {'path': str(path)}, True)
        loaded = fresh.ask('simulation.wait')
        assert loaded['components'] == saved['graph']['components'] and loaded['wires'] == saved['graph']['wires']
        assert loaded['paused'] and loaded['transient']['sample']['time_s'] == 0
        assert loaded['mixed']['potentiometers'][0]['wiper_position'] == .25 and loaded['switch_states'][0]['applied_closed']
        fresh.close()
        fresh = None
        exports = {}
        before_export = invariant(client.ask('project.inspect'))
        for fmt in ['svg', 'png', 'pdf']:
            output = assets/f'actuators.{fmt}'
            mutate('export.schematic.'+fmt, {'path': str(output)})
            if fmt == 'pdf':
                assert output.read_bytes().startswith(b'%PDF-')
            else:
                exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
        assert invariant(client.ask('project.inspect')) == before_export
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      discovery=discovery, hints={name: schema['x-interaction-summary'] for name, schema in schemas.items()},
                      capacitor_before_actuation_v=before['mixed']['capacitors'][0]['voltage_v'],
                      capacitor_after_pending_switch_v=applied['mixed']['capacitors'][0]['voltage_v'],
                      independent_capacitor_reference_v=expected, exports=exports,
                      checks=['discovery is pure, target-kind and completion paths explicit',
                              'incomplete startup edit; fresh paused switch queues, Wait preserves pending, coalescing returns clean source',
                              'wiper completes at same charge/time/run while preserving pending applied switch',
                              'next sample advances old circuit then applies pending switch; independent Thevenin RC check',
                              'same-value commands preserve physics/history; invalid range/wrong kind refuses without mutation',
                              'atomic batch uses paused-stale startup semantics with one undo/redo; restart and fresh-process save/open preserve values',
                              'SVG/PNG/PDF capture preserves graph, physics, history and document'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: report[k] for k in ('passed', 'transport', 'capacitor_after_pending_switch_v', 'independent_capacitor_reference_v')}, indent=2))
        if args.keep_open:
            mutate('component.select', {'id': 'P1'})
            mutate('scope.visibility.set', {'visible': False})
            mutate('view.fit', {'follow_viewport': True})
            keep = True
    finally:
        if fresh:
            fresh.close()
        if args.connect and not keep:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()

#!/usr/bin/env python3
"""Discover and exercise schematic contracts in an owned blank session."""
import argparse
import importlib.util
import json
from pathlib import Path

from validation_artifacts import discovery_summary

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)

COMMANDS = ('parts.search.set', 'component.duplicate', 'component.move', 'component.rotate',
            'component.select', 'object.delete', 'drawing.tool.set', 'drawing.position.set',
            'drawing.cancel', 'wire.connect', 'wire.begin', 'wire.terminals.set',
            'wire.junction.insert', 'editor.value.set')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)
    ask = lambda name, params=None: client.ask(name, params, True)
    inspect = lambda: client.ask('project.inspect')

    def refuse(name, params, code='invalid-params'):
        before = inspect()
        client.ask(name, params, True, code)
        assert inspect() == before

    try:
        original = inspect()
        assert not original['components'], 'use an owned blank session'
        contracts, registry, discovery = {}, None, None
        for name in COMMANDS:
            response = client.ask('capabilities.list', {'command': name})
            cap = response['capabilities'][0]
            assert cap['input_schema']['x-command-contract']['scope'] == 'schematic-editing'
            contracts[name] = cap
            if registry is None:
                registry = response['schemas']
                discovery = discovery_summary(response, assets/'schematic-discovery.json')
            else:
                assert response['schemas'] == registry
        assert inspect() == original
        assert contracts['component.move']['input_schema']['properties']['position']['properties']['x']['maximum'] == 10000
        assert contracts['wire.junction.insert']['input_schema']['properties']['wire_ids']['maxItems'] == 8
        ask('parts.search.set', {'query': 'battery'})
        assert inspect()['matching_parts'][0]['kind'] == 'dc-source'
        for identity, kind, x, y in [('V1', 'dc-source', -200, 0), ('R1', 'resistor', 0, 0),
                                     ('D1', 'led', 200, 0), ('GND', 'ground', 0, 110)]:
            ask('component.place', dict(id=identity, kind=kind, position=dict(x=x, y=y)))
        for previous in ({}, {'x': 0}, {'y': 0}, {'x': None, 'y': 0}):
            refuse('component.move', dict(id='R1', position=dict(x=15, y=-15), previous_position=previous))
        ask('component.move', dict(id='R1', position=dict(x=15, y=-15), previous_position=None))
        moved = next(p for p in inspect()['components'] if p['id'] == 'R1')['position']
        assert moved == dict(x=20, y=-20)
        refuse('component.move', dict(id='R1', position=dict(x=40, y=40), previous_position=dict(x=20, y=-19)), 'conflict')
        ask('drawing.tool.set', {'tool': 'wire', 'kind': None})
        ask('wire.terminals.set', {'from': 'V1.positive', 'to': 'R1.1'})
        ask('wire.begin', {'terminal': 'V1.positive'})
        ask('drawing.position.set', {'position': {'x': 200, 'y': 0}})
        ask('editor.value.set', {'value': 'not parsed yet'})
        ask('drawing.cancel')
        draft = inspect()
        assert not draft['wires'] and draft['drawing']['tool'] == 'select'
        assert draft['drawing']['from'] == 'V1.positive' and draft['drawing']['to'] == 'R1.1'
        assert draft['drawing']['value'] == 'not parsed yet'
        for a, b in [('V1.positive', 'R1.1'), ('R1.2', 'D1.anode'),
                     ('D1.cathode', 'GND.reference'), ('GND.reference', 'V1.negative')]:
            ask('wire.connect', {'id': None, 'from': a, 'to': b})
        live = client.ask('simulation.wait')
        assert abs(live['result']['current_a']-.00291679) < 2e-8
        refuse('wire.connect', {'from': 'R1.1', 'to': 'V1.positive'})
        ask('component.move', dict(id='R1', position=dict(x=40, y=40), previous_position=moved))
        ask('component.rotate', {'id': 'R1'})
        geometry = inspect()
        for key in ('run_id', 'electrical_revision', 'result'):
            assert geometry[key] == live[key], key
        ask('component.duplicate', {'id': 'R1', 'position': None})
        duplicate = inspect()['selected']
        assert len(inspect()['wires']) == 4 and inspect()['result']['stale']
        ask('object.delete', {'id': duplicate})
        client.ask('simulation.wait')
        refuse('wire.junction.insert', {'wire_ids': ['W1', 'W2']})
        ask('wire.junction.insert', {'id': None, 'wire_ids': ['W1']})
        joined = client.ask('simulation.wait')
        junction = joined['selected']
        assert len(joined['wires']) == 5 and abs(joined['result']['current_a']-live['result']['current_a']) < 1e-12
        # This pair shares a net via the new junction, but is no longer a
        # directly duplicated wire pair. It explicitly adds a copper cycle.
        ask('wire.connect', {'from': 'V1.positive', 'to': 'R1.1'})
        cyclic = client.ask('simulation.wait')
        assert len(cyclic['wires']) == 6 and abs(cyclic['result']['current_a']-live['result']['current_a']) < 1e-12
        ask('component.select', {'id': ''})
        assert inspect()['selected'] == ''
        ask('project.save', {'path': str(assets/'schematic-contracts.elab')})
        summary = dict(discovery)
        summary.pop('capabilities')
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      contracts=contracts, discovery=summary,
                      observations=dict(contracts=len(contracts), snapped_position=moved,
                                        current_a=cyclic['result']['current_a'], junction_id=junction,
                                        explicit_wire_count=len(cyclic['wires']), planning_preserved_named_choices=True,
                                        malformed_previous_positions_refused=4),
                      checks=['All 14 detailed contracts are pure and share the same full schema registry',
                              'Incomplete move preimages refuse without state changes; null/complete guards and signed snapping match discovery',
                              'Drawing/wire/draft planning and cancel preserve graph and documented choices',
                              'Blank LED construction, geometry-only preservation, duplication/delete recovery and atomic junction',
                              'Direct duplicate wire pair refuses; explicit same-net copper cycle retains electrical readings'],
                      limits=['Detailed schematic family coverage only; other contract/native input/release gates remain open'])
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(report['observations']))
    finally:
        if args.connect:
            inspect()
            ask('app.close', {'discard_changes': True})
        client.close()


if __name__ == '__main__':
    main()

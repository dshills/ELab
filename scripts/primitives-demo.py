#!/usr/bin/env python3
"""Exercise junctions and named connectivity through stdio or an owned desktop."""
import argparse
import importlib.util
import json
from pathlib import Path
import tempfile

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    client = helper.Client(args.connect)
    fresh = None
    try:
        catalog = client.ask('capabilities.list')
        kinds = {part['kind'] for part in client.ask('parts.list')}
        assert {'junction', 'net-label'} <= kinds
        client.ask('example.reset', {'discard_changes': True}, True)
        initial = client.ask('simulation.wait')
        if initial['paused']:
            client.ask('simulation.resume', {}, True)
            initial = client.ask('simulation.wait')
        client.ask('wire.junction.insert', {'wire_ids': ['W1'], 'id': 'Branch'}, True)
        inserted = client.ask('simulation.wait')
        assert abs(inserted['result']['current_a'] - initial['result']['current_a']) < 1e-12
        assert len(inserted['components']) == 5 and len(inserted['wires']) == 5
        client.ask('edit.undo', {}, True)
        assert client.ask('simulation.wait')['wires'] == initial['wires']
        client.ask('edit.batch', {'name': 'Named supply', 'edits': [
            {'command': 'object.delete', 'params': {'id': 'W1'}},
            {'command': 'component.place', 'params': {'id': 'SupplyA', 'kind': 'net-label', 'position': {'x': -300, 'y': -100}, 'value': 'VCC'}},
            {'command': 'component.place', 'params': {'id': 'SupplyB', 'kind': 'net-label', 'position': {'x': 300, 'y': -100}, 'value': 'VCC'}},
            {'command': 'wire.connect', 'params': {'from': 'V1.positive', 'to': 'SupplyA.node'}},
            {'command': 'wire.connect', 'params': {'from': 'SupplyB.node', 'to': 'R1.1'}},
        ]}, True)
        named = client.ask('simulation.wait')
        assert abs(named['result']['current_a'] - initial['result']['current_a']) < 1e-12
        assert any(net.get('labels') == ['VCC'] for net in named['nets'])
        client.ask('component.set', {'id': 'SupplyB', 'value': 'vcc'}, True)
        broken = client.ask('project.inspect')
        assert broken['status'] == 'incomplete' and broken['result']['stale']
        client.ask('edit.undo', {}, True)
        client.ask('simulation.wait')
        with tempfile.TemporaryDirectory(prefix='elab-primitives-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            path, svg = directory / 'named-led.elab', directory / 'named-led.svg'
            client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            client.ask('export.schematic.svg', {'path': str(svg), 'overwrite': True}, True)
            assert 'VCC' in svg.read_text() and 'Net label' in svg.read_text()
            fresh = helper.Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            reopened = fresh.ask('simulation.wait')
            assert abs(reopened['result']['current_a'] - initial['result']['current_a']) < 1e-12
            assert not reopened['document']['dirty']
            report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                          capabilities=len(catalog['capabilities']), library_kinds=len(kinds), current_a=named['result']['current_a'],
                          checks=['explicit single-wire junction', 'atomic insertion undo', 'named remote supply',
                                  'exact case-sensitive label rename', 'incomplete/stale feedback', 'rename undo',
                                  'SVG symbols/names', 'fresh-process named project reopen'])
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

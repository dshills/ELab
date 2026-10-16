#!/usr/bin/env python3
"""Qualify filtered command discovery using only public requests."""
import argparse
import copy
import importlib.util
import json
from pathlib import Path


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='owned native validation endpoint; replaces its project')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        before = client.ask('simulation.wait')
        full = client.ask('capabilities.list')
        details = []
        for command in ['source.sine.set', 'source.pulse.set', 'simulation.transient.set',
                        'potentiometer.wiper.set', 'example.load', 'scope.cursor.set']:
            result = client.ask('capabilities.list', {'command': command})
            assert result['version'] == full['version'] and result['schema_version'] == full['schema_version']
            cap = result['capabilities'][0]
            assert len(result['capabilities']) == 1 and cap['id'] == command
            assert cap['input_schema']['type'] == 'object' and '$ref' not in cap['input_schema']
            details.append(cap)
        assert client.ask('project.inspect') == before
        client.ask('capabilities.list', {'command': 'missing'}, expect='not-found')
        client.ask('capabilities.list', {'command': None}, expect='invalid-params')
        client.ask('capabilities.list', {'command': 'x'*129}, expect='invalid-params')
        assert client.ask('project.inspect') == before
        pulse = next(c['input_schema'] for c in details if c['id'] == 'source.pulse.set')
        assert pulse['properties']['waveform']['properties']['rise_s']['x-normalized-si-range']['minimum'] == 1e-6
        assert len(pulse['x-normalized-si-relations']) == 4
        client.ask('project.clear', {'discard_changes': True}, True)
        for command, kind in [('source.sine.set', 'sine-source'), ('source.pulse.set', 'finite-pulse-source')]:
            client.ask('project.clear', {'discard_changes': True}, True)
            source_id = 'DemoSine' if command == 'source.sine.set' else 'DemoPulse'
            client.ask('component.place', {'id': source_id, 'kind': kind, 'position': {'x': 0, 'y': 0}}, True)
            schema = next(c['input_schema'] for c in details if c['id'] == command)
            params = copy.deepcopy(schema['examples'][0])
            params['id'] = source_id
            client.ask(command, params, True)
            if command == 'source.pulse.set':
                params['waveform'].update(high_s='50 ms', low_s='50 ms', rise_s='1 ms', fall_s='1 ms')
                client.ask(command, params, True)
                current = client.ask('project.inspect')
                params['waveform']['rise_s'] = '60 ms'
                client.ask(command, params, True, 'invalid-params')
                assert client.ask('project.inspect') == current
        client.ask('project.clear', {'discard_changes': True}, True)
        raw = json.dumps(full['capabilities'], ensure_ascii=False, separators=(',', ':'))
        raw = raw.replace('&', r'\u0026').replace('<', r'\u003c').replace('>', r'\u003e')
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'commands': len(full['capabilities']), 'catalog_bytes': len(raw.encode()),
                  'checks': ['ordinary catalog compatibility', 'filtered identity/expanded root input',
                             'paused graph/history/physics query purity', 'unknown/null/oversized refusal purity',
                             'finite-pulse SI range and coupled constraints', 'runnable sine/pulse examples',
                             'unit-text pulse entry and invalid coupled value preserve state'],
                  'detailed_capabilities': details,
                  'limitations': ['five commands have additional bounds/examples; other filtered inputs expose their existing type schemas',
                                  'stateful preconditions/error discovery and exhaustive native capability coverage remain incomplete']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps({k: v for k, v in report.items() if k != 'detailed_capabilities'}))
    finally:
        client.close()


if __name__ == '__main__':
    main()

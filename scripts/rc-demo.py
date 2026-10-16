#!/usr/bin/env python3
"""Build an RC circuit through shared commands and verify exact transient samples."""
import argparse
import json
import math
from pathlib import Path
import tempfile
import importlib.util


helper_spec = importlib.util.spec_from_file_location("persistence_demo", Path(__file__).with_name("persistence-demo.py"))
helper = importlib.util.module_from_spec(helper_spec)
helper_spec.loader.exec_module(helper)
Client = helper.Client


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--fixture', type=Path)
    args = parser.parse_args()
    client = connect_demo_client(Client, args.connect)
    fresh = None
    try:
        catalog = client.ask('capabilities.list')['capabilities']
        client.ask('project.clear', {'discard_changes': True}, True)
        ids = {}
        for kind, x, y, value in [('pulse-source', -200, 0, '5 V'), ('resistor', 0, -110, '10k'), ('capacitor', 200, 0, '1uF'), ('ground', 0, 110, '')]:
            p = {'kind': kind, 'position': {'x': x, 'y': y}}
            if value:
                p['value'] = value
            state = client.ask('component.place', p, True)
            ids[kind] = state['selected']
        v, r, c, g = [ids[k] for k in ('pulse-source', 'resistor', 'capacitor', 'ground')]
        for source, dest in [(v+'.positive', r+'.1'), (r+'.2', c+'.1'), (c+'.2', g+'.reference'), (g+'.reference', v+'.negative')]:
            client.ask('wire.connect', {'from': source, 'to': dest}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        initial = client.ask('project.inspect')
        assert initial['analysis'] == 'RC transient' and initial['transient']['sample']['time_s'] == 0
        at10 = client.ask('simulation.step', {'duration_s': .01}, True)
        assert abs(at10['transient']['sample']['voltage_v'] - 3.1606027941427883) < 1e-12
        at50 = client.ask('simulation.step', {'duration_s': .04}, True)
        assert at50['transient']['sample']['input_v'] == 0
        at100 = client.ask('simulation.step', {'duration_s': .05}, True)
        assert abs(at100['transient']['sample']['voltage_v'] - 5*(1-math.exp(-5))*math.exp(-5)) < 1e-12
        client.ask('simulation.input.set', {'mode': 'charge'}, True)
        charged = client.ask('simulation.step', {'duration_s': .03}, True)
        before = charged['transient']['sample']['voltage_v']
        run = charged['run_id']
        client.ask('simulation.input.set', {'mode': 'discharge'}, True)
        discharged = client.ask('simulation.step', {'duration_s': .01}, True)
        assert abs(discharged['transient']['sample']['voltage_v'] - before*math.exp(-1)) < 1e-12
        assert discharged['run_id'] == run
        client.ask('component.move', {'id': c, 'position': {'x': 240, 'y': 0}}, True)
        client.ask('component.rotate', {'id': c}, True)
        pose = client.ask('project.inspect')
        assert pose['transient']['sample'] == discharged['transient']['sample']
        assert pose['run_id'] == run
        client.ask('component.set', {'id': c, 'value': '0'}, True, 'invalid-params')
        with tempfile.TemporaryDirectory(prefix='elab-rc-') as directory:
            path = args.fixture.resolve() if args.fixture else Path(directory)/'RC.elab'
            client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            doc = json.loads(path.read_text())
            assert doc['version'] == 4 and doc['transient_settings']['step_s'] == .001
            fresh = Client()
            restored = fresh.ask('project.load', {'path': str(path)}, True)
            assert restored['transient']['sample']['voltage_v'] == 0
            assert restored['transient']['sample']['time_s'] == 0
            assert restored['components'] == pose['components']
            assert restored['transient']['input_mode'] == 'auto'
        report = {'passed': True, 'transport': 'desktop-socket' if args.connect else 'headless-stdio',
                  'engine': at10['transient']['engine'], 'capability_count': len(catalog),
                  'voltage_10ms_v': at10['transient']['sample']['voltage_v'],
                  'voltage_50ms_v': at50['transient']['sample']['voltage_v'],
                  'voltage_100ms_v': at100['transient']['sample']['voltage_v'],
                  'checks': ['construction from blank', '10 ms analytical charge reference',
                             'automatic pulse falling/rising edges', 'manual charge/discharge continuity',
                             'paused stepping', 'move/rotate preserves time and energy',
                             'invalid capacitance refusal', 'save/reopen RC configuration'],
                  'samples': at100['transient']['samples']}
        if args.output:
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: v for k, v in report.items() if k != 'samples'}, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()

if __name__ == '__main__':
    main()

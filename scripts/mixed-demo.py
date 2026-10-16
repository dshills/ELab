#!/usr/bin/env python3
"""Build an autonomous RC/NPN LED flasher through shared user/agent commands."""
import argparse
import importlib.util
import json
from pathlib import Path
import tempfile
import time


spec = importlib.util.spec_from_file_location('persistence_demo', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--fixture', type=Path)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect)
    fresh = None
    try:
        catalog = client.ask('capabilities.list')['capabilities']
        assert any(c['id'] == 'example.flasher' for c in catalog)
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        ids = {}
        for name, kind, x, y, value, rotation in [
            ('Supply', 'dc-source', -300, 0, '5 V', 0),
            ('Load1', 'resistor', -140, -200, '1k', 90), ('Load2', 'resistor', 140, -200, '1k', 90),
            ('Base1', 'resistor', -220, -100, '47k', 90), ('Base2', 'resistor', 60, -100, '47k', 90),
            ('LED1', 'led', -140, -70, '', 0), ('LED2', 'led', 140, -70, '', 0),
            ('Q1', 'npn', -140, 70, '', 0), ('Q2', 'npn', 140, 70, '', 0),
            ('C1', 'capacitor', -20, 10, '10uF', 270), ('C2', 'capacitor', 20, 160, '10uF', 90),
            ('Ground', 'ground', 0, 240, '', 0)]:
            p = {'kind': kind, 'position': {'x': x, 'y': y}}
            if value:
                p['value'] = value
            ids[name] = client.ask('component.place', p, True)['selected']
            for _ in range(rotation//90):
                client.ask('component.rotate', {'id': ids[name]}, True)
        client.ask('capacitor.initial.set', {'id': ids['C1'], 'value': '.1 V'}, True)
        ref = lambda pin: ids[pin.split('.')[0]] + '.' + pin.split('.')[1]
        for source, target in [
            ('Supply.positive', 'Load1.1'), ('Supply.positive', 'Load2.1'),
            ('Supply.positive', 'Base1.1'), ('Supply.positive', 'Base2.1'),
            ('Load1.2', 'LED1.anode'), ('Load2.2', 'LED2.anode'),
            ('LED1.cathode', 'Q1.collector'), ('LED2.cathode', 'Q2.collector'),
            ('Base1.2', 'Q1.base'), ('Base2.2', 'Q2.base'),
            ('Q1.collector', 'C1.1'), ('C1.2', 'Q2.base'),
            ('Q2.collector', 'C2.1'), ('C2.2', 'Q1.base'),
            ('Q1.emitter', 'Ground.reference'), ('Q2.emitter', 'Ground.reference'),
            ('Supply.negative', 'Ground.reference')]:
            client.ask('wire.connect', {'from': ref(source), 'to': ref(target)}, True)
        cfg = {'step_s': .005, 'high_s': .05, 'low_s': .05, 'initial_voltage_v': 0,
               'seconds_per_wall_second': .5}
        client.ask('simulation.transient.set', cfg, True)
        client.ask('simulation.restart', {}, True)
        initial = client.ask('project.inspect')
        assert initial['analysis'] == 'RC/NPN transient' and initial['mixed']['time_s'] == 0

        def measure(cap_value, factor):
            for cid in [ids['C1'], ids['C2']]:
                client.ask('component.set', {'id': cid, 'value': cap_value}, True)
            config = dict(cfg, step_s=.005*factor)
            client.ask('simulation.transient.set', config, True)
            client.ask('simulation.restart', {}, True)
            high = client.ask('project.inspect')['mixed']['leds'][0]['current_a'] > .001
            rises, readings = [], []
            # Keep output sample phase relative to RC constant for the scaling check.
            for _ in range(300):
                s = client.ask('simulation.step', {'duration_s': .01*factor}, True)
                r = s['mixed']
                assert r['kcl_residual_a'] < 1e-9 and abs(r['power_residual_w']) < 1e-8
                new_high = r['leds'][0]['current_a'] > .001
                if new_high and not high and r['time_s'] > .4*factor:
                    rises.append(r['time_s'])
                high = new_high
                readings.append({'time_s': r['time_s'], 'led1_a': r['leds'][0]['current_a'],
                                 'led2_a': r['leds'][1]['current_a'],
                                 'cap1_v': r['capacitors'][0]['voltage_v'],
                                 'cap2_v': r['capacitors'][1]['voltage_v']})
            assert len(rises) >= 4, rises
            assert min(p['led1_a'] for p in readings) < 1e-5
            assert max(p['led1_a'] for p in readings) > .002
            return rises[2]-rises[1], readings

        period1, trace1 = measure('10uF', 1)
        period2, trace2 = measure('20uF', 2)
        assert abs(period2/period1-2) < .03
        current = client.ask('project.inspect')
        run = current['run_id']
        client.ask('component.move', {'id': ids['C1'], 'position': {'x': -20, 'y': 30}}, True)
        assert client.ask('project.inspect')['mixed']['time_s'] == current['mixed']['time_s']
        client.ask('simulation.input.set', {'mode': 'low'}, True)
        powered_off = client.ask('simulation.step', {'duration_s': .01}, True)
        assert powered_off['transient']['input_mode'] == 'low' and powered_off['run_id'] == run
        client.ask('simulation.input.set', {'mode': 'high'}, True)
        client.ask('simulation.resume', {}, True)
        deadline = time.monotonic()+2
        while client.ask('project.inspect')['mixed']['time_s'] <= powered_off['mixed']['time_s']:
            assert time.monotonic() < deadline
            time.sleep(.02)
        client.ask('simulation.pause', {}, True)
        before = client.ask('project.inspect')
        stale = client.ask('capacitor.initial.set', {'id': ids['C1'], 'value': '-.2 V'}, True)
        assert stale['mixed']['stale'] and stale['transient']['stale']
        client.ask('edit.undo', {}, True)
        client.ask('simulation.restart', {}, True)
        with tempfile.TemporaryDirectory(prefix='elab-mixed-') as directory:
            path = args.fixture.resolve() if args.fixture else Path(directory)/'Flasher.elab'
            client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            document = json.loads(path.read_text())
            assert next(p for p in document['graph']['components'] if p['id'] == ids['C1'])['initial_voltage_v'] == .1
            fresh = helper.Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            loaded = fresh.ask('project.inspect')
            assert loaded['mixed']['time_s'] < .05 and loaded['transient']['input_mode'] == 'auto'
            assert loaded['components'] == before['components'] and not loaded['document']['dirty']
        report = {'passed': True, 'transport': 'desktop-socket' if args.connect else 'headless-stdio',
                  'capability_count': len(catalog), 'engine': initial['mixed']['engine'],
                  'period_10uf_s': period1, 'period_20uf_s': period2,
                  'checks': ['construction from blank', 'DC-powered autonomous alternation',
                             'capacitance doubles physical period', 'KCL and power balance with BE damping',
                             'geometry preserves memory', 'power override preserves run', 'live resume',
                             'signed initial voltage / undo', 'save and fresh Auto initialization'],
                  'trace_10uf': trace1, 'trace_20uf': trace2}
        if args.output:
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: v for k, v in report.items() if not k.startswith('trace_')}, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

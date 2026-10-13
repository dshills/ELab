#!/usr/bin/env python3
"""Build a pulse-driven NPN LED and verify shared transient commands."""
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
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--fixture', type=Path)
    args = parser.parse_args()
    client = helper.Client(args.connect)
    fresh = None
    try:
        catalog = client.ask('capabilities.list')['capabilities']
        assert any(c['id'] == 'example.npn-pulse' for c in catalog)
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        ids = {}
        for name, kind, x, y, value in [
            ('Supply', 'dc-source', 260, -100, '5 V'),
            ('Drive', 'pulse-source', -260, 60, '5 V'),
            ('BaseR', 'resistor', -100, 60, '10k'),
            ('LoadR', 'resistor', 100, -180, '1k'),
            ('Light', 'led', 100, -80, ''),
            ('Switch', 'npn', 100, 60, ''),
            ('Earth', 'ground', 100, 180, '')]:
            p = {'kind': kind, 'position': {'x': x, 'y': y}}
            if value:
                p['value'] = value
            placed = client.ask('component.place', p, True)
            ids[name] = placed['selected']
            if name == 'LoadR':
                client.ask('component.rotate', {'id': ids[name]}, True)
        ref = lambda pin: ids[pin.split('.')[0]] + '.' + pin.split('.')[1]
        for source, target in [
            ('Supply.positive', 'LoadR.1'), ('LoadR.2', 'Light.anode'),
            ('Light.cathode', 'Switch.collector'), ('Drive.positive', 'BaseR.1'),
            ('BaseR.2', 'Switch.base'), ('Switch.emitter', 'Earth.reference'),
            ('Supply.negative', 'Earth.reference'), ('Drive.negative', 'Earth.reference')]:
            client.ask('wire.connect', {'from': ref(source), 'to': ref(target)}, True)
        config = {'step_s': .007, 'high_s': .031, 'low_s': .047,
                  'initial_voltage_v': 0, 'seconds_per_wall_second': .1}
        client.ask('simulation.transient.set', config, True)
        client.ask('simulation.restart', {}, True)
        start = client.ask('project.inspect')
        assert start['analysis'] == 'NPN transient' and start['transient']['sample']['time_s'] == 0
        assert start['npn']['state'] == 'saturation'
        on = client.ask('simulation.step', {'duration_s': .02}, True)
        off = client.ask('simulation.step', {'duration_s': .011}, True)
        assert off['npn']['state'] == 'cutoff' and off['transient']['sample']['input_v'] == 0
        period = client.ask('simulation.step', {'duration_s': .047}, True)
        trace = period['transient']
        assert period['npn']['state'] == 'saturation' and abs(trace['sample']['time_s']-.078) < 1e-12
        assert [e['mode'] for e in trace['events']] == ['restart', 'pulse-low', 'pulse-high']
        for edge in [.031, .078]:
            assert {p['input_v'] for p in trace['samples'] if abs(p['time_s']-edge) < 1e-12} == {0, 5}
        assert abs(on['npn']['collector_current_a']-.002860330492) < 1e-10
        run = period['run_id']
        queued = client.ask('simulation.input.set', {'mode': 'low'}, True)
        assert queued['transient']['pending_input_mode'] == 'low' and queued['npn']['state'] == 'saturation'
        low = client.ask('simulation.step', {'duration_s': .007}, True)
        assert low['npn']['state'] == 'cutoff' and low['run_id'] == run
        client.ask('simulation.input.set', {'mode': 'high'}, True)
        client.ask('simulation.resume', {}, True)
        deadline = time.monotonic()+2
        while True:
            live = client.ask('project.inspect')
            if live['transient']['sample']['time_s'] > low['transient']['sample']['time_s']:
                break
            assert time.monotonic() < deadline, 'live clock did not advance'
            time.sleep(.02)
        assert live['npn']['state'] == 'saturation' and live['run_id'] == run
        frozen = client.ask('simulation.pause', {}, True)
        time.sleep(.08)
        assert client.ask('project.inspect')['transient']['sample'] == frozen['transient']['sample']
        client.ask('component.move', {'id': ids['Switch'], 'position': {'x': 120, 'y': 60}}, True)
        client.ask('component.rotate', {'id': ids['Switch']}, True)
        assert client.ask('project.inspect')['transient']['sample'] == frozen['transient']['sample']
        stale = client.ask('component.set', {'id': ids['Drive'], 'value': '.8 V'}, True)
        assert stale['npn']['stale'] and stale['transient']['stale']
        client.ask('simulation.restart', {}, True)
        assert client.ask('project.inspect')['npn']['state'] == 'active'
        client.ask('component.set', {'id': ids['Drive'], 'value': '5 V'}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.input.set', {'mode': 'low'}, True)
        client.ask('simulation.step', {'duration_s': .007}, True)
        client.ask('view.fit', {}, True)
        with tempfile.TemporaryDirectory(prefix='elab-npn-pulse-') as directory:
            path = args.fixture.resolve() if args.fixture else Path(directory)/'Blink.elab'
            saved = client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            fresh = helper.Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            loaded = fresh.ask('simulation.wait')
            assert loaded['transient']['input_mode'] == 'auto' and loaded['transient_settings'] == config
            assert loaded['components'] == saved['components'] and not loaded['document']['dirty']
            assert loaded['npn']['state'] == 'saturation'
        report = {'passed': True, 'transport': 'desktop-socket' if args.connect else 'headless-stdio',
                  'capability_count': len(catalog), 'engine': trace['engine'],
                  'model_limit': 'Memoryless junction response; no storage or switching delay.',
                  'on_current_a': on['npn']['collector_current_a'], 'off_current_a': off['npn']['collector_current_a'],
                  'checks': ['construction from blank', 'DC phase reference', 'edges between samples',
                             'paused override and step', 'resume continues run', 'pause freezes time',
                             'geometry preserves time', 'amplitude edit stales and restarts',
                             'saved pulse settings and fresh Auto run'],
                  'first_period': trace}
        if args.output:
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: v for k, v in report.items() if k != 'first_period'}, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

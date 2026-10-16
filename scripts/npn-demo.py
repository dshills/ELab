#!/usr/bin/env python3
"""Build and switch an NPN LED circuit through the shared desktop/headless API."""
import argparse
import importlib.util
import json
from pathlib import Path
import tempfile


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
        capabilities = client.ask('capabilities.list')['capabilities']
        assert any(c['id'] == 'example.npn' for c in capabilities)
        parts = client.ask('parts.list')
        assert next(p for p in parts if p['kind'] == 'npn')['terminals'] == ['base', 'collector', 'emitter']
        client.ask('project.clear', {'discard_changes': True}, True)
        ids = {}
        for cid, kind, x, y, value in [
            ('Supply', 'dc-source', 260, -100, '5 V'),
            ('Drive', 'dc-source', -260, 60, '5 V'),
            ('BaseR', 'resistor', -100, 60, '10k'),
            ('LoadR', 'resistor', 100, -180, '1k'),
            ('Light', 'led', 100, -80, ''),
            ('Switch', 'npn', 100, 60, ''),
            ('Earth', 'ground', 100, 180, '')]:
            params = {'kind': kind, 'position': {'x': x, 'y': y}}
            if value:
                params['value'] = value
            placed = client.ask('component.place', params, True)
            ids[cid] = placed['selected']
            if cid == 'LoadR':
                client.ask('component.rotate', {'id': ids[cid]}, True)
        for source, target in [
            ('Supply.positive', 'LoadR.1'), ('LoadR.2', 'Light.anode'),
            ('Light.cathode', 'Switch.collector'), ('Drive.positive', 'BaseR.1'),
            ('BaseR.2', 'Switch.base'), ('Switch.emitter', 'Earth.reference'),
            ('Supply.negative', 'Earth.reference'), ('Drive.negative', 'Earth.reference')]:
            ref = lambda name: ids[name.split('.')[0]] + '.' + name.split('.')[1]
            client.ask('wire.connect', {'from': ref(source), 'to': ref(target)}, True)
        readings = []
        for drive, expected in [('0 V', 'cutoff'), ('0.8 V', 'active'), ('5 V', 'saturation')]:
            client.ask('component.set', {'id': ids['Drive'], 'value': drive}, True)
            snapshot = client.ask('simulation.wait')
            result = snapshot['npn']
            assert result['state'] == expected and not result['stale']
            assert result['electrical_revision'] == snapshot['electrical_revision']
            assert result['kcl_residual_a'] < 1e-10 and abs(result['kvl_residual_v']) < 1e-10
            supply_power = 5*result['collector_current_a'] + float(drive.split()[0])*result['base_current_a']
            dissipated = sum(result[k] for k in ['led_power_w', 'transistor_power_w', 'base_resistor_power_w', 'load_resistor_power_w'])
            assert abs(supply_power-dissipated) < 1e-10
            readings.append({'drive': drive, **result})
        client.ask('simulation.pause', {}, True)
        client.ask('component.set', {'id': ids['Drive'], 'value': '0 V'}, True)
        assert client.ask('project.inspect')['npn']['stale']
        client.ask('simulation.resume', {}, True)
        assert client.ask('simulation.wait')['npn']['state'] == 'cutoff'
        client.ask('edit.undo', {}, True)
        on = client.ask('simulation.wait')
        assert on['npn']['state'] == 'saturation'
        run = on['run_id']
        client.ask('component.rotate', {'id': ids['Switch']}, True)
        assert client.ask('project.inspect')['run_id'] == run
        client.ask('edit.undo', {}, True)
        with tempfile.TemporaryDirectory(prefix='elab-npn-') as directory:
            path = args.fixture.resolve() if args.fixture else Path(directory)/'Transistor LED.elab'
            saved = client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            document = json.loads(path.read_text())
            assert document['npn_model']['forward_beta'] == 100
            fresh = helper.Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            loaded = fresh.ask('simulation.wait')
            assert loaded['components'] == saved['components']
            assert loaded['npn']['collector_current_a'] == on['npn']['collector_current_a']
            assert not loaded['document']['dirty']
        report = {'passed': True, 'transport': 'desktop-socket' if args.connect else 'headless-stdio',
                  'capability_count': len(capabilities), 'engine': on['npn']['engine'],
                  'checks': ['build from blank with generated IDs', 'cutoff / active / saturation',
                             'KCL / KVL / power balance', 'pause / stale / resume', 'undo drive',
                             'rotation preserves electrical run', 'save / fresh-process reopen'],
                  'readings': readings}
        if args.output:
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: v for k, v in report.items() if k != 'readings'}, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

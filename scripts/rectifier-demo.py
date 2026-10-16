#!/usr/bin/env python3
"""Verify rectifier learning, reconstruction, recordings and persistence.
--connect must identify an owned development session; its schematic is replaced.
"""
import argparse
import importlib.util
import json
import math
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
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    fresh = None
    try:
        info = next(i for i in client.ask('examples.list') if i['id'] == 'rectifier')
        client.ask('example.rectifier', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        initial = client.ask('simulation.wait')
        assert initial['selected'] == 'C1' and not initial['example']['modified']
        assert set(initial['scope']['selected']) == {'i:D1', 'v:C1', 'v:V1.positive'}
        client.ask('simulation.step', {'duration_s': .2}, True)
        source = client.ask('project.inspect')
        output_200ms = source['mixed']['capacitors'][0]['voltage_v']
        accepted = client.ask('simulation.verify', {'duration_s': .2, 'probe_ids': ['v:C1', 'v:V1.positive', 'i:D1']}, True)
        op = accepted['last_operation']
        deadline = time.monotonic() + 10
        while True:
            view = client.ask('operation.inspect', {'id': op})
            if view['operation']['status'] == 'completed':
                break
            assert view['operation']['status'] in ['queued', 'running'] and time.monotonic() < deadline, view
            time.sleep(.01)
        assert abs(view['metadata']['mixed']['capacitors'][0]['voltage_v'] - output_200ms) < 1e-9
        assert view['metadata']['diode_model']['id'] == 'elab.generic-silicon-diode.v1'
        rows = []
        for offset in [0, 200]:
            rows.extend(client.ask('operation.samples', {'id': op, 'offset': offset, 'limit': 200})['rows'])
        negative = next(r for r in rows if abs(r['time_s'] - .075) < 1e-9)
        assert abs(negative['values'][1] + 5) < 1e-9 and negative['values'][0] > 1
        assert abs(negative['values'][2] + 1e-9) < 1e-12
        after_verify = client.ask('project.inspect')
        for field in ['transient', 'components', 'wires', 'run_id', 'electrical_revision', 'undo_entry']:
            assert source.get(field) == after_verify.get(field), field
        for field in ['path', 'generation', 'format_version', 'schematic_dirty', 'preferences_dirty']:
            assert source['document'][field] == after_verify['document'][field], field
        # Completed Verify intentionally adds a kept recording asset.
        assert after_verify['document']['recording_count'] == source['document']['recording_count'] + 1
        assert after_verify['document']['recordings_dirty']
        client.ask('verification.export', {'id': op, 'format': 'json', 'path': str(directory / 'rectifier-record.json')}, True)

        ripples = []
        for cap in ['10u', '47u']:
            client.ask('component.set', {'id': 'C1', 'value': cap}, True)
            # Paused electrical edits retain stale readings until explicit Restart.
            client.ask('simulation.restart', {}, True)
            client.ask('simulation.wait')
            for _ in range(20):
                result = client.ask('simulation.step', {'duration_s': .1}, True)
                assert result['step_work']['status'] == 'completed'
            state = client.ask('project.inspect')
            trace = next(t for t in state['scope']['traces'] if t['probe']['id'] == 'v:C1')
            values = [p['value'] for p in trace['points'] if p['time_s'] >= 1.9 - 1e-9]
            ripples.append({'capacitance': cap, 'minimum_v': min(values), 'maximum_v': max(values), 'ripple_v': max(values) - min(values)})
        assert 0 < ripples[1]['ripple_v'] < .5 * ripples[0]['ripple_v']
        client.ask('project.save', {'path': str(directory / 'rectifier.elab')}, True)
        client.ask('export.schematic.png', {'path': str(directory / 'rectifier.png')}, True)
        saved = client.ask('project.inspect')
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': str(directory / 'rectifier.elab')}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['example']['info'] == info and reopened['example']['modified']
        assert reopened['components'] == saved['components'] and reopened['wires'] == saved['wires']
        assert reopened['paused'] and reopened['transient']['sample']['time_s'] == 0

        # Construct the same electrical network through ordinary drawing commands,
        # using new object IDs because deleted IDs cannot be reused in a session.
        client.ask('project.clear', {'discard_changes': True}, True)
        parts = [('Supply', 'sine-source', -300, 20, '5'), ('Limit', 'resistor', -160, -100, '100'),
                 ('Junction', 'diode', 0, -100, ''), ('Storage', 'capacitor', 170, 20, '10u'),
                 ('Load', 'resistor', 340, 20, '10k'), ('Earth', 'ground', 170, 180, '')]
        wires = [('Supply.positive', 'Limit.1'), ('Limit.2', 'Junction.anode'), ('Junction.cathode', 'Storage.1'),
                 ('Storage.1', 'Load.1'), ('Load.2', 'Storage.2'), ('Storage.2', 'Earth.reference'), ('Supply.negative', 'Earth.reference')]
        edits = [{'command': 'component.place', 'params': {'id': id, 'kind': kind, 'position': {'x': x, 'y': y}, 'value': value}} for id, kind, x, y, value in parts]
        edits += [{'command': 'component.rotate', 'params': {'id': 'Junction'}} for _ in range(3)]
        edits += [{'command': 'component.rotate', 'params': {'id': 'Load'}}]
        edits += [{'command': 'wire.connect', 'params': {'from': a, 'to': b}} for a, b in wires]
        client.ask('edit.batch', {'name': 'Build rectifier', 'edits': edits}, True)
        client.ask('source.sine.set', {'id': 'Supply', 'waveform': {'amplitude_v': 5, 'offset_v': 0, 'frequency_hz': 10, 'phase_degrees': 0}}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        built = client.ask('simulation.step', {'duration_s': .2}, True)
        constructed_200ms = built['mixed']['capacitors'][0]['voltage_v']
        assert math.isclose(constructed_200ms, output_200ms, abs_tol=1e-9)
        client.ask('project.clear', {'discard_changes': True}, True)
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'example_id': info['id'], 'output_200ms_v': output_200ms, 'constructed_200ms_v': constructed_200ms,
                  'negative_peak': negative, 'ripple_after_2s': ripples,
                  'checks': ['default input/output/diode probes', 'live/Verify equality and source preservation',
                             'negative-half reverse isolation', 'larger-capacitor ripple reduction',
                             'source/template/model persistence', 'blank-sheet component/wire reconstruction'],
                  'limits': ['Pinned educational equations; no physical supply/diode rating or mains qualification',
                             'Independent RK4 error bounds are in Go tests; this demo checks public workflow parity']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        if fresh:
            fresh.close()
        client.close()


if __name__ == '__main__':
    main()

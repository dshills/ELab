#!/usr/bin/env python3
"""Capture shared source annotations as SVG/PDF/PNG in an owned session."""
import argparse
import importlib.util
import json
from pathlib import Path

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)
    exported = []

    def capture(name):
        before = client.ask('project.inspect')
        annotations = before['source_annotations']
        assert annotations and all(a['lines'] for a in annotations)
        for format in ['svg', 'pdf', 'png']:
            client.ask('export.schematic.' + format, {'path': str(directory / (name + '.' + format))}, True)
            after = client.ask('project.inspect')
            for field in ['source_annotations', 'components', 'wires', 'transient', 'run_id', 'electrical_revision', 'document', 'undo_entry']:
                assert before.get(field) == after.get(field), field
        svg = (directory / (name + '.svg')).read_text()
        for a in annotations:
            for line in a['lines']:
                assert line in svg
        client.ask('view.fit', {'width_px': 600, 'height_px': 300}, True)
        fitted = client.ask('project.inspect')
        assert before.get('transient') == fitted.get('transient')
        exported.append({'case': name, 'annotations': annotations, 'view_after_fit': fitted['view']})

    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        settings = {'step_s': .001, 'high_s': .123456, 'low_s': .234567, 'initial_voltage_v': 0, 'seconds_per_wall_second': .02}
        client.ask('simulation.transient.set', settings, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.input.set', {'mode': 'auto'}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        capture('legacy-pulse')

        client.ask('example.rectifier', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('source.sine.set', {'id': 'V1', 'waveform': {'amplitude_v': 2.5, 'offset_v': -1, 'frequency_hz': 123.456, 'phase_degrees': -90}}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.input.set', {'mode': 'auto'}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        capture('sine')

        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('edit.batch', {'name': 'Finite pulse source', 'edits': [
            {'command': 'component.place', 'params': {'id': 'RampWave', 'kind': 'finite-pulse-source', 'position': {'x': -200, 'y': 0}}},
            {'command': 'component.place', 'params': {'id': 'WaveLoad', 'kind': 'resistor', 'value': '1k', 'position': {'x': 0, 'y': -100}}},
            {'command': 'component.place', 'params': {'id': 'WaveReturn', 'kind': 'ground', 'position': {'x': 200, 'y': 100}}},
            {'command': 'wire.connect', 'params': {'from': 'RampWave.positive', 'to': 'WaveLoad.1'}},
            {'command': 'wire.connect', 'params': {'from': 'WaveLoad.2', 'to': 'WaveReturn.reference'}},
            {'command': 'wire.connect', 'params': {'from': 'WaveReturn.reference', 'to': 'RampWave.negative'}},
        ]}, True)
        client.ask('source.pulse.set', {'id': 'RampWave', 'waveform': {'low_v': -1, 'high_v': 4, 'high_s': .02, 'low_s': .03, 'rise_s': .0002, 'fall_s': .0005}}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.input.set', {'mode': 'auto'}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        capture('finite-pulse')

        client.ask('example.rectifier', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.input.set', {'mode': 'high'}, True)
        queued = client.ask('project.inspect')['source_annotations'][0]
        assert queued.get('pending_input_mode') == 'high' and queued.get('input_mode') == 'auto'
        client.ask('simulation.step', {'duration_s': .001}, True)
        client.ask('simulation.input.set', {'mode': 'low'}, True)
        applied = client.ask('project.inspect')['source_annotations'][0]
        assert applied['input_mode'] == 'high' and applied['pending_input_mode'] == 'low'
        capture('held-and-pending')

        client.ask('source.sine.set', {'id': 'V1', 'waveform': {'amplitude_v': 3, 'offset_v': 1, 'frequency_hz': 50, 'phase_degrees': 30}}, True)
        stale = client.ask('project.inspect')['source_annotations'][0]
        assert stale['state'] == 'stale' and not stale.get('input_mode') and not stale.get('pending_input_mode')
        capture('stale-sine')
        client.ask('project.clear', {'discard_changes': True}, True)
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio', 'exports': exported,
                  'checks': ['same annotation lines in snapshot and SVG/PDF/PNG layout', 'global/per-source timings',
                             'configured versus applied/pending/stale provenance', 'source/history/physics preservation', 'shared Fit'],
                  'limits': ['Six significant display digits; SVG metadata and normal project snapshots retain exact settings',
                             'Visual/PDF bounds inspection recorded separately; no printer submission']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        client.close()


if __name__ == '__main__':
    main()

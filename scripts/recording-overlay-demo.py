#!/usr/bin/env python3
"""Inspect bounded reference traces through shared recording commands."""
import argparse
import copy
import hashlib
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

    def capture(duration, step, probes, mode='charge'):
        state = client.ask('simulation.verify', dict(duration_s=duration, sample_step_s=step,
                           input_mode=mode, probe_ids=probes), True)
        identity = state['last_operation']
        deadline = time.monotonic() + 15
        while True:
            view = client.ask('operation.inspect', {'id': identity})
            assert view['operation']['status'] not in ('failed', 'canceled'), view
            if view['operation']['status'] == 'completed':
                return identity
            assert time.monotonic() < deadline
            time.sleep(.01)

    def semantic(overlay):
        result = copy.deepcopy(overlay)
        result.pop('id')
        result.pop('origin_id')
        return result

    def configure(selected, reference):
        client.ask('operation.select', {'id': selected}, True)
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        return client.ask('recording.compare.set', dict(id=selected, reference_id=reference,
                          time_s=0, show_overlay=True), True)

    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        contract = client.ask('capabilities.list', {'command': 'recording.compare.set'})['capabilities'][0]['input_schema']
        assert contract['properties']['show_overlay']['description']
        assert contract['x-max-reference-points-per-trace'] == 258
        first = capture('20 ms', '1 ms', ['v:C1', 'i:R1'])
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        second = capture('30 ms', '2 ms', ['i:R1', 'v:C1'])
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        client.ask('recording.probes.set', {'ids': ['v:C1', 'i:R1']}, True)
        before = client.ask('project.inspect')
        params = dict(id=second, reference_id=first, time_s='10 ms', show_overlay=True)
        state = client.ask('recording.compare.set', params, True)
        reference = state['recording']['reference_overlay']
        assert reference['time_basis'] == 'captured-seconds-from-startup'
        assert reference['scope']['window']['end_s'] == .03 and reference['duration_s'] == .02
        traces = reference['scope']['traces']
        assert [t['probe']['id'] for t in traces] == ['v:C1', 'i:R1']
        assert traces[0]['points'][-1]['time_s'] == .02 and reference['window_sample_count'] == 21
        for point in traces[0]['points']:
            assert abs(point['value'] - 5 * (1 - math.exp(-point['time_s'] / .01))) < 1e-12
        for key in ('components', 'wires', 'document', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert state.get(key) == before.get(key), key
        queried_before = client.ask('project.inspect')
        assert client.ask('recording.compare', params) == state['recording']['comparison']
        assert queried_before == client.ask('project.inspect')
        client.ask('recording.window.set', {'start_s': '1.5 ms', 'end_s': '2 ms'}, True)
        client.ask('recording.cursor.set', {'cursor': 'a', 'time_s': '2 ms'}, True)
        narrow = client.ask('project.inspect')['recording']
        assert narrow['scope']['a']['time_s'] == .002
        assert narrow['reference_overlay']['scope']['traces'][0]['points'] == [traces[0]['points'][2]]
        client.ask('recording.window.set', {'start_s': '21 ms', 'end_s': '30 ms'}, True)
        empty = client.ask('project.inspect')['recording']['reference_overlay']
        assert not empty['scope']['window_valid'] and empty['window_sample_count'] == 0
        assert all(t['points'] == [] for t in empty['scope']['traces'])
        client.ask('recording.window.set', {'start_s': 0, 'end_s': '30 ms'}, True)
        expected = semantic(client.ask('project.inspect')['recording']['reference_overlay'])
        project = directory / 'overlay.elab'
        client.ask('project.save', {'path': str(project)}, True)
        digest = hashlib.sha256(project.read_bytes()).hexdigest()
        client.ask('project.load', {'path': str(project)}, True)
        reopened = client.ask('simulation.wait')
        assert 'reference_overlay' not in reopened['recording'] and 'comparison' not in reopened['recording']
        by_resistance = {}
        for op in reopened['operations']:
            metadata = client.ask('operation.inspect', {'id': op['id']})['metadata']
            resistance = next(c['value'] for c in metadata['graph']['components'] if c['id'] == 'R1')
            by_resistance[resistance] = op['id']
        configure(by_resistance[20000], by_resistance[10000])
        client.ask('recording.probes.set', {'ids': ['v:C1', 'i:R1']}, True)
        assert semantic(client.ask('project.inspect')['recording']['reference_overlay']) == expected
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest
        client.ask('recording.view.set', {'mode': 'live'}, True)
        assert 'reference_overlay' not in client.ask('project.inspect')['recording']
        client.ask('example.npn-pulse', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        pulse_a = capture('100 ms', '1 ms', ['v:VIN.positive'], 'auto')
        pulse_b = capture('100 ms', '2 ms', ['v:VIN.positive'], 'auto')
        pulse = configure(pulse_b, pulse_a)['recording']['reference_overlay']
        edges = [p for p in pulse['scope']['traces'][0]['points'] if abs(p['time_s'] - .05) < 1e-12]
        assert any(p['value'] == 5 for p in edges) and any(p['value'] == 0 for p in edges)
        client.ask('project.clear', {'discard_changes': True}, True)
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      contract=contract, rc_overlay=semantic(reference), pulse_overlay=semantic(pulse),
                      narrow_reference_points=narrow['reference_overlay']['scope']['traces'][0]['points'],
                      checks=['exact analytic RC reference, channel order and independent sample times',
                              'shorter reference ends naturally; empty overlap publishes no fabricated values',
                              'sub-step interval and exact selected cursor', 'query purity and live/history/dirty preservation',
                              'reopened captures rebuild identical overlay without changing saved bytes',
                              'reference absent from live view', 'both stored NPN ideal-edge sides retained'],
                      limits=['plot reduction preserves bucket extrema; exact full rows remain in operation.samples',
                              'no time alignment/extrapolation, cross-run cursor inference or physical device validation'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'])))
    finally:
        client.close()


if __name__ == '__main__':
    main()

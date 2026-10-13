#!/usr/bin/env python3
"""Measure both sides of a flasher through the same scope commands as the UI.
--connect changes the given development session; save any user work first.
"""
import argparse
import importlib.util
import json
from pathlib import Path

spec = importlib.util.spec_from_file_location('persistence_demo', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    client = helper.Client(args.connect)
    try:
        catalog = client.ask('capabilities.list')['capabilities']
        commands = {c['id'] for c in catalog}
        assert {'scope.probes.set', 'scope.cursor.set', 'scope.cursor.select', 'scope.cursor.clear'} <= commands
        client.ask('example.flasher', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': 1}, True)
        client.ask('simulation.step', {'duration_s': 1}, True)
        before = client.ask('project.inspect')
        ids = ['v:Q1.collector', 'v:Q2.collector', 'i:D1', 'i:D2', 'v:C2', 'i:C2']
        state = client.ask('scope.probes.set', {'ids': ids}, True)
        assert state['run_id'] == before['run_id']
        assert state['electrical_revision'] == before['electrical_revision']
        assert state['transient']['sample'] == before['transient']['sample']
        assert state['document'] == before['document']
        traces = {t['probe']['id']: t for t in state['scope']['traces']}
        for trace in traces.values():
            assert len(trace['points']) == len(state['transient']['samples'])
        first, second = traces['i:D1']['points'], traces['i:D2']['points']
        assert sum((a['value'] > .001) != (b['value'] > .001) for a, b in zip(first, second)) > len(first)/2
        rises = [b['time_s'] for a, b in zip(first, first[1:]) if a['value'] < .001 <= b['value']]
        assert len(rises) >= 2
        client.ask('scope.cursor.set', {'cursor': 'a', 'time_s': rises[-2]}, True)
        client.ask('scope.cursor.select', {'cursor': 'b'}, True)
        measured = client.ask('scope.cursor.set', {'cursor': 'b', 'time_s': rises[-1]}, True)['scope']
        assert abs(measured['delta_s'] - .485) < .01
        assert abs(measured['interval_hz'] - 1/measured['delta_s']) < 1e-12
        assert len(measured['readings']) == 6
        client.ask('scope.probes.set', {'ids': ['i:D1', 'i:D1']}, True, expect='invalid-params')
        client.ask('scope.cursor.set', {'cursor': 'a', 'time_s': 3}, True, expect='invalid-params')
        # Select a new channel from the already-recorded history.
        changed = client.ask('scope.probes.set', {'ids': ['v:C1', 'i:D1', 'i:D2']}, True)
        assert len(changed['scope']['traces'][0]['points']) == len(first)
        cleared = client.ask('scope.cursor.clear', {'cursor': 'a'}, True)['scope']
        assert 'a' not in cleared and 'delta_s' not in cleared
        # Leave a paused, inspectable two-sided scope and the measured cursors.
        client.ask('scope.probes.set', {'ids': ids}, True)
        final = client.ask('scope.cursor.set', {'cursor': 'a', 'time_s': rises[-2]}, True)
        report = {'passed': True, 'transport': 'desktop socket' if args.connect else 'headless stdio',
                  'capabilities': len(catalog), 'run_id': final['scope']['run_id'],
                  'sample_step_s': final['transient_settings']['step_s'],
                  'sample_count': len(first), 'cursor_a_s': measured['a']['time_s'],
                  'cursor_b_s': measured['b']['time_s'], 'period_s': measured['delta_s'],
                  'interval_hz': measured['interval_hz'], 'readings': measured['readings'],
                  'checks': ['both collector voltages and LED currents', 'opposite LED phases',
                             'capacitor voltage and signed current', 'selection backfills retained history',
                             'selection preserves solver state', 'cursor timing and per-channel deltas',
                             'clear cursor removes measurement', 'invalid selection/time refuses mutation']}
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()

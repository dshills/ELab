#!/usr/bin/env python3
"""Exercise shared scope contracts and sub-sample visible-window fidelity."""
import argparse
import csv
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
COMMANDS = ['scope.visibility.set', 'scope.probes.set', 'scope.probe.remove', 'scope.trace.select',
            'scope.window.set', 'scope.cursor.set', 'scope.cursor.select', 'scope.cursor.clear', 'export.scope.csv']


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)

    def refuse(command, params, code):
        before = client.ask('project.inspect')
        client.ask(command, params, True, code)
        assert client.ask('project.inspect') == before

    def capture():
        state = client.ask('simulation.verify', dict(duration_s='1 ps', input_mode='charge', probe_ids=['v:C1']), True)
        identity = state['last_operation']
        deadline = time.monotonic() + 10
        while True:
            operation = client.ask('operation.inspect', {'id': identity})['operation']
            assert operation['status'] not in ('failed', 'canceled'), operation
            if operation['status'] == 'completed':
                return identity
            assert time.monotonic() < deadline
            time.sleep(.01)

    try:
        schemas = {command: client.ask('capabilities.list', {'command': command})['capabilities'][0]['input_schema'] for command in COMMANDS}
        assert all(schema['x-command-contract']['scope'] == 'live-scope-controls-and-csv' for schema in schemas.values())
        assert schemas['scope.probes.set']['properties']['ids']['maxItems'] == 6
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': '1 ps'}, True)
        client.ask('scope.probes.set', {'ids': ['v:C1', 'i:C1']}, True)
        before = client.ask('project.inspect')
        empty = client.ask('scope.window.set', {'mode': 'fixed', 'start_s': '.4 ps', 'end_s': '.6 ps'}, True)
        scope = empty['scope']
        assert not scope['window_valid'] and scope['window_reason']
        assert scope['statistics'] == [] and all(trace['points'] == [] for trace in scope['traces'])
        for cursor in ('a', 'b'):
            empty = client.ask('scope.cursor.set', {'cursor': cursor, 'time_s': '.5 ps'}, True)
        assert not empty['scope']['a']['valid'] and not empty['scope']['b']['valid']
        assert 'delta_s' not in empty['scope'] and empty['scope']['readings'] == []
        sentinel = assets / 'untouched.csv'
        sentinel.write_bytes(b'untouched')
        refuse('export.scope.csv', {'path': str(sentinel), 'overwrite': True}, 'unavailable')
        assert sentinel.read_bytes() == b'untouched'
        one = client.ask('scope.window.set', {'mode': 'fixed', 'start_s': '.75 ps', 'end_s': '1 ps'}, True)
        assert one['scope']['window_valid'] and all(len(trace['points']) == 1 for trace in one['scope']['traces'])
        assert all(stat['count'] == 1 for stat in one['scope']['statistics'])
        csv_path = assets / 'one-row.csv'
        exported = client.ask('export.scope.csv', {'path': str(csv_path)}, True)
        assert exported['project_work']['status'] == 'completed'
        rows = list(csv.reader(csv_path.read_text().splitlines()))
        assert len(rows) == 2 and rows[0][-2:] == ['v:C1 [V]', 'i:C1 [A]']
        assert float(rows[1][6]) == 1e-12
        voltage = float(rows[1][-2])
        assert math.isclose(voltage, -5 * math.expm1(-1e-10), rel_tol=1e-12, abs_tol=1e-24)
        client.ask('scope.cursor.set', {'cursor': 'b', 'time_s': 0}, True)
        whole = client.ask('scope.window.set', {'mode': 'live'}, True)
        assert whole['scope']['delta_s'] == -1e-12 and whole['scope']['interval_hz'] == 1e12
        zero = client.ask('scope.cursor.set', {'cursor': 'b', 'time_s': '1 ps'}, True)
        assert zero['scope']['delta_s'] == 0 and 'interval_hz' not in zero['scope']
        client.ask('scope.cursor.select', {'cursor': 'b'}, True)
        cleared = client.ask('scope.cursor.clear', {'cursor': 'a'}, True)
        assert 'a' not in cleared['scope'] and cleared['scope']['active_cursor'] == 'b' and cleared['scope']['readings'] == []
        hidden = client.ask('scope.visibility.set', {'visible': False}, True)
        assert not hidden['scope_visible'] and hidden['transient'] == before['transient']
        client.ask('scope.visibility.set', {'visible': True}, True)
        after = client.ask('project.inspect')
        for key in ('document', 'components', 'wires', 'transient', 'run_id', 'electrical_revision', 'undo_entry'):
            assert before.get(key) == after.get(key), key

        reference, selected = capture(), capture()
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        captured_before = client.ask('project.inspect')
        client.ask('recording.window.set', {'start_s': '.4 ps', 'end_s': '.6 ps'}, True)
        recorded = client.ask('recording.compare.set', dict(id=selected, reference_id=reference, time_s='.5 ps', show_overlay=True), True)
        view = recorded['recording']
        assert not view['scope']['window_valid'] and view['window_sample_count'] == 0
        assert view['scope']['statistics'] == [] and view['readings'] == []
        assert view['reference_overlay']['window_sample_count'] == 0 and not view['reference_overlay']['scope']['window_valid']
        assert all(trace['points'] == [] for trace in view['reference_overlay']['scope']['traces'])
        restored = client.ask('recording.window.set', {'start_s': 0, 'end_s': view['last_sample_time_s']}, True)['recording']
        assert restored['scope']['window_valid'] and restored['window_sample_count'] == restored['sample_count'] == 2
        raw = client.ask('operation.samples', {'id': selected, 'offset': 0, 'limit': 200})
        asset = assets / 'tiny-rc.json'
        client.ask('verification.export', {'id': selected, 'path': str(asset), 'format': 'json'}, True)
        digest = hashlib.sha256(asset.read_bytes()).hexdigest()
        client.ask('verification.import', {'path': str(asset)}, True)
        imported = client.ask('project.inspect')['recording']['id']
        imported_rows = client.ask('operation.samples', {'id': imported, 'offset': 0, 'limit': 200})
        assert imported_rows['rows'] == raw['rows'] and hashlib.sha256(asset.read_bytes()).hexdigest() == digest
        repeated = client.ask('recording.window.set', {'start_s': '.4 ps', 'end_s': '.6 ps'}, True)['recording']
        assert repeated['window_sample_count'] == 0 and repeated['scope']['statistics'] == [] and repeated['readings'] == []
        after = client.ask('project.inspect')
        for key in ('components', 'wires', 'transient', 'run_id', 'electrical_revision', 'undo_entry'):
            assert captured_before.get(key) == after.get(key), key

        client.ask('recording.view.set', {'mode': 'live'}, True)
        client.ask('scope.probes.set', {'ids': ['i:C1']}, True)
        client.ask('scope.trace.select', {'id': 'i:C1'}, True)
        removed_source = client.ask('object.delete', {'id': 'C1'}, True)
        assert removed_source['scope']['unavailable'][0]['id'] == 'i:C1'
        cleaned = client.ask('scope.probe.remove', {'id': 'i:C1'}, True)
        assert cleaned['scope'].get('unavailable', []) == [] and 'trace_focus' not in cleaned
        assert cleaned['components'] == removed_source['components'] and cleaned['document'] == removed_source['document']
        refuse('scope.probe.remove', {'id': 'i:C1'}, 'not-found')
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schemas=schemas,
                      observations=dict(empty_interval_s=[4e-13, 6e-13], outside_saved_times_s=[0, 1e-12], empty_rows=0,
                                        one_row_interval_s=[7.5e-13, 1e-12], csv_times_s=[float(rows[1][6])], voltage_v=voltage,
                                        signed_cursor_delta_s=whole['scope']['delta_s'], reciprocal_interval_hz=whole['scope']['interval_hz'],
                                        captured_rows=restored['sample_count'], captured_last_sample_time_s=restored['last_sample_time_s']),
                      checks=['Nine scope/CSV commands have discoverable shared contracts', 'Narrow windows cannot borrow outside rows or report fake statistics',
                              'Empty CSV refusal preserves destination bytes', 'CSV and scope contain the same one saved row with signed SI columns',
                              'Cursor/window validity, signed delta, zero interval and clear/select behavior', 'Visibility/probe/window/cursor/export preserve live source/history',
                              'Recorded and reference windows remain empty between samples', 'Whole captured range restores both saved rows',
                              'Export/import preserves raw captured samples', 'Requested unavailable source can be discovered and removed'],
                      limits=['Window membership uses bounded roundoff, no interpolation; file validation/requested-time tolerance retains prior compatibility',
                              'Tiny-time RC check is a software/model regression, not physical device qualification',
                              'No complete physical focus/typing/IME/screen-reader/visible-latency acceptance established'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], csv_rows=len(rows) - 1)))
    finally:
        try:
            if args.connect:
                client.ask('project.inspect')
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()

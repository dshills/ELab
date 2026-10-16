#!/usr/bin/env python3
"""Exercise independent recorded import/replay through stdio or an owned desktop.
--connect replaces that session's live schematic with the RC example.
"""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import tempfile
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect)
    try:
        catalog = client.ask('capabilities.list')
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': .012}, True)
        accepted = client.ask('simulation.verify', {'duration_s': .02, 'probe_ids': ['v:C1', 'i:C1']}, True)
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            record = client.ask('operation.inspect', {'id': accepted['last_operation']})
            if record['operation']['status'] == 'completed':
                break
            assert record['operation']['status'] not in ['failed', 'canceled'], record
            time.sleep(.01)
        assert record['operation']['status'] == 'completed'
        with tempfile.TemporaryDirectory(prefix='elab-recorded-view-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            original_path = directory / 'rc-record.json'
            copy_path = directory / 'reimported-record.json'
            client.ask('verification.export', {'id': accepted['last_operation'], 'path': str(original_path), 'format': 'json', 'overwrite': True}, True)
            original = original_path.read_bytes()
            client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
            before = client.ask('project.inspect')
            imported = client.ask('verification.import', {'path': str(original_path)}, True)
            identifier = imported['recording']['id']
            assert imported['recording']['viewing'] and imported['recording']['imported']
            view = client.ask('operation.inspect', {'id': identifier})
            assert not view['operation']['source_unchanged']
            assert view['metadata'] == record['metadata']
            assert view['operation']['source_sha256'] == hashlib.sha256(original).hexdigest()
            client.ask('recording.seek', {'index': 10}, True)
            at_10ms = client.ask('project.inspect')['recording']['readings'][0]['value']
            assert abs(at_10ms - 5*(1-math.exp(-1))) < 1e-12
            client.ask('recording.cursor.set', {'cursor': 'a', 'time_s': .005}, True)
            client.ask('recording.cursor.set', {'cursor': 'b', 'time_s': .015}, True)
            client.ask('recording.window.set', {'start_s': .005, 'end_s': .015}, True)
            client.ask('recording.probes.set', {'ids': ['v:C1']}, True)
            measured = client.ask('project.inspect')['recording']
            assert abs(measured['scope']['delta_s'] - .01) < 1e-12
            assert abs(measured['scope']['interval_hz'] - 100) < 1e-9
            assert measured['scope']['statistics'][0]['count'] == 11
            client.ask('recording.seek', {'time_s': 0}, True)
            client.ask('recording.speed.set', {'seconds_per_wall_second': 1}, True)
            client.ask('recording.play', {'seconds_per_wall_second': 1}, True)
            deadline = time.monotonic() + 2
            state = client.ask('project.inspect')
            while state['recording']['playing'] and time.monotonic() < deadline:
                time.sleep(.02)
                state = client.ask('project.inspect')
            assert not state['recording']['playing'] and state['recording']['sample_index'] == 20
            for key in ['document', 'transient', 'scope', 'components', 'wires', 'run_id', 'electrical_revision']:
                assert state[key] == before[key], key
            page = client.ask('operation.samples', {'id': identifier, 'limit': 100})
            assert page['rows'] == json.loads(original)['rows']
            client.ask('verification.export', {'id': identifier, 'path': str(copy_path), 'format': 'json', 'overwrite': True}, True)
            assert copy_path.read_bytes() == original
            bad_path = directory / 'incomplete-record.json'
            damaged = json.loads(original)
            del damaged['rows'][0]['time_s']
            bad_path.write_text(json.dumps(damaged))
            before_bad = client.ask('project.inspect')
            client.ask('verification.import', {'path': str(bad_path)}, True, expect='invalid-recording')
            assert client.ask('project.inspect')['recording'] == before_bad['recording']
            client.ask('recording.seek', {'index': 10}, True)
            report = {
                'passed': True, 'transport': 'packaged-native-private-socket' if args.connect else 'headless-stdio',
                'capabilities': len(catalog['capabilities']), 'schema_version': catalog['schema_version'],
                'record_sha256': hashlib.sha256(original).hexdigest(), 'rc_voltage_10ms_v': at_10ms,
                'delta_s': measured['scope']['delta_s'], 'interval_hz': measured['scope']['interval_hz'],
                'interval_exact_samples': 11, 'record_exact_samples': page['total'],
                'checks': ['validated complete import', 'immutable captured input and provenance',
                           'RC analytical recorded measurement', 'exact cursor interval/readings',
                           'captured probe selection', 'playback completion', 'live state/undo/file point preserved',
                           'full paged rows', 'byte-identical portable reexport', 'invalid import preserves viewer'],
            }
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()

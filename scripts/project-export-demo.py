#!/usr/bin/env python3
"""Captured background SVG/live CSV/recorded JSON+CSV and legacy artifact parity.
--connect replaces only the designated owned development session.
"""
import argparse
import csv
import hashlib
import importlib.util
import json
from pathlib import Path
import time
import xml.etree.ElementTree as ET


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def protected(state):
    return {key: state.get(key) for key in ['components', 'wires', 'document', 'view', 'transient',
        'mixed', 'npn', 'scope', 'undo_entry', 'redo_entry', 'run_id', 'electrical_revision']}


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': .01}, True)
        client.ask('scope.probes.set', {'ids': ['v:C1', 'i:C1']}, True)
        client.ask('scope.window.set', {'mode': 'fixed', 'start_s': .002, 'end_s': .01}, True)
        admissions = []

        def start(kind, filename, record_id=None):
            before = client.ask('project.inspect')
            params = dict(kind=kind, path=str(directory/filename), overwrite=True)
            if record_id:
                params['id'] = record_id
            began = time.perf_counter()
            admitted = client.ask('export.start', params, True)
            admissions.append(dict(kind=kind, wall_ms=(time.perf_counter()-began)*1000))
            assert admitted['project_work']['status'] == 'queued' and admitted['project_work']['cancelable']
            assert protected(admitted) == protected(before)
            return admitted['project_work']['id'], params['path']

        def complete(work_id, path, edited):
            state = client.ask('simulation.wait')
            work = client.ask('project.work.inspect', {'id': work_id})
            assert work['status'] == 'completed' and not work['cancelable']
            assert protected(state) == protected(edited)
            data = Path(path).read_bytes()
            assert work['artifact_sha256'] == hashlib.sha256(data).hexdigest()
            assert state['last_export']['sha256'] == work['artifact_sha256']
            assert state['last_export']['bytes'] == len(data)
            assert state['last_export']['document_generation'] == work['document_generation']
            return data

        work_id, svg_path = start('schematic-svg', 'captured.svg')
        client.ask('component.move', {'id': 'R1', 'position': {'x': 300, 'y': -200}}, True)
        edited = client.ask('project.inspect')
        svg = complete(work_id, svg_path, edited)
        ET.fromstring(svg)
        work_id, csv_path = start('scope-csv', 'captured-scope.csv')
        client.ask('scope.probes.set', {'ids': ['v:R1']}, True)
        edited = client.ask('project.inspect')
        complete(work_id, csv_path, edited)
        with open(csv_path) as source:
            rows = list(csv.DictReader(source))
        assert len(rows) == 9 and 'v:C1 [V]' in rows[0] and 'i:C1 [A]' in rows[0]
        assert float(rows[0]['time_s']) == .002 and float(rows[-1]['time_s']) == .01
        accepted = client.ask('simulation.verify', {'duration_s': .01, 'probe_ids': ['v:C1', 'i:C1']}, True)
        record_id = accepted['last_operation']
        deadline = time.monotonic()+10
        while True:
            recorded = client.ask('operation.inspect', {'id': record_id})
            if recorded['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        assert recorded['operation']['status'] == 'completed'
        work_id, json_path = start('verification-json', 'captured-record.json', record_id)
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        edited = client.ask('project.inspect')
        encoded = complete(work_id, json_path, edited)
        document = json.loads(encoded)
        assert document['metadata'] == recorded['metadata'] and len(document['rows']) == 11
        work_id, record_csv = start('verification-csv', 'captured-record.csv', record_id)
        client.ask('recording.keep.set', {'id': record_id, 'keep': False}, True)
        edited = client.ask('project.inspect')
        complete(work_id, record_csv, edited)
        with open(record_csv) as source:
            rows = list(csv.DictReader(source))
        assert len(rows) == 11 and rows[0]['operation_id'] == recorded['metadata']['operation_id']
        legacy_path = str(directory/'legacy-record.json')
        client.ask('verification.export', {'id': record_id, 'path': legacy_path, 'format': 'json', 'overwrite': True}, True)
        assert Path(legacy_path).read_bytes() == Path(json_path).read_bytes()
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('scope.window.set', {'mode': 'live'}, True)
        saved = client.ask('project.save', {'path': str(directory/'export-source.elab'), 'overwrite': True}, True)
        assert saved['paused'] and not saved['document']['dirty']
        assert not list(directory.glob('.elab-export-*'))
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            admissions=admissions,
            checks=['all four immediate-start export kinds and stable inspection',
                'admission captures source before later geometry/probe/value/keep edits',
                'current project/physics/history/scope preserved after publication',
                'captured scope window/units/samples', 'record JSON retains original metadata/rows',
                'record CSV retains operation identity', 'exact hashes/byte counts/captured generation',
                'legacy recorded export bytes match immediate-start output', 'temporary cleanup and durable clean source'],
            limitations=['blocked-worker tests separately prove lock responsiveness, actual record eviction and publication fencing',
                'OS calls may outlast cancellation/deadline; publishing cannot be canceled',
                'portable final check/rename has no universal cross-process compare-and-swap guarantee',
                'backend timing excludes input-to-pixel latency', 'manual recovery/import/shutdown still have synchronous paths'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()

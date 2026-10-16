#!/usr/bin/env python3
"""Validate the next ten hobbyist workflow improvements through public commands.
--connect replaces the given development session's schematic; save work first.
"""
import argparse
import csv
import importlib.util
import json
from pathlib import Path
import tempfile
import xml.etree.ElementTree as ET


spec = importlib.util.spec_from_file_location('persistence_demo', Path(__file__).with_name('persistence-demo.py'))
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
    fresh = None
    try:
        catalog = client.ask('capabilities.list')['capabilities']
        searched = client.ask('parts.search.set', {'query': 'capacitor'}, True)
        assert [p['kind'] for p in searched['matching_parts']] == ['capacitor']
        client.ask('parts.search.set', {'query': ''}, True)
        client.ask('example.flasher', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        initial = client.ask('simulation.restart', {}, True)
        duplicated = client.ask('component.duplicate', {'id': 'C1'}, True)
        copy_id = duplicated['selected']
        original_cap = next(c for c in initial['components'] if c['id'] == 'C1')
        copied_cap = next(c for c in duplicated['components'] if c['id'] == copy_id)
        assert copied_cap['rotation'] == original_cap['rotation']
        assert copied_cap['initial_voltage_v'] == original_cap['initial_voltage_v']
        assert duplicated['wires'] == initial['wires'] and duplicated['undo_name'] == 'Duplicate component'
        undone = client.ask('edit.undo', {}, True)
        assert undone['components'] == initial['components'] and undone['redo_name'] == 'Duplicate component'
        client.ask('edit.redo', {}, True)
        client.ask('edit.undo', {}, True)
        edits = [{'command': 'component.set', 'params': {'id': cid, 'value': '20uF'}} for cid in ['C1', 'C2']]
        before = client.ask('project.inspect')
        paired = client.ask('edit.batch', {'name': 'Set capacitor pair', 'edits': edits}, True)
        assert paired['electrical_revision'] == before['electrical_revision'] + 1
        assert paired['undo_name'] == 'Set capacitor pair'
        reverted = client.ask('edit.undo', {}, True)
        assert reverted['components'] == initial['components']
        client.ask('edit.batch', {'name': 'Must rollback', 'edits': [edits[0], {'command': 'component.set', 'params': {'id': 'missing', 'value': '1k'}}]}, True, expect='invalid-params')
        assert client.ask('project.inspect')['components'] == reverted['components']
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': 1}, True)
        client.ask('simulation.step', {'duration_s': 1}, True)
        ids = ['v:Q1.collector', 'v:Q2.collector', 'i:D1', 'i:D2', 'v:C1', 'v:C2']
        client.ask('scope.probes.set', {'ids': ids}, True)
        window = client.ask('scope.window.set', {'mode': 'fixed', 'start_s': .5, 'end_s': 1.5}, True)
        assert window['scope']['window_valid'] and len(window['scope']['statistics']) == 6
        assert all(s['count'] == 201 for s in window['scope']['statistics'])
        client.ask('scope.cursor.set', {'cursor': 'a', 'time_s': 1}, True)
        measured = client.ask('scope.cursor.set', {'cursor': 'b', 'time_s': 1.4}, True)
        assert abs(measured['scope']['delta_s'] - .4) < .0051
        with tempfile.TemporaryDirectory(prefix='elab-next-ten-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            doc_path, csv_path, svg_path = [directory / name for name in ['flasher-probed.elab', 'flasher-interval.csv', 'flasher-schematic.svg']]
            saved = client.ask('project.save', {'path': str(doc_path), 'overwrite': True}, True)
            assert not saved['document']['dirty']
            doc = json.loads(doc_path.read_text())
            assert doc['version'] == 4 and doc['scope_probes']['ids'] == ids
            fresh = helper.Client()
            reopened = fresh.ask('project.load', {'path': str(doc_path)}, True)
            assert reopened['scope']['selected'] == ids and reopened['transient']['sample']['time_s'] == 0
            assert reopened['scope']['window']['mode'] == 'live' and 'a' not in reopened['scope']
            csv_state = client.ask('export.scope.csv', {'path': str(csv_path), 'overwrite': True}, True)
            client.ask('export.schematic.svg', {'path': str(svg_path), 'overwrite': True}, True)
            with csv_path.open() as f:
                rows = list(csv.reader(f))
            assert len(rows) == 202 and rows[0][-1] == 'v:C2 [V]'
            assert rows[1][2] == 'elab.mixed-mna-be.v1'
            ET.parse(svg_path)
            assert 'C1.1' in svg_path.read_text()
            after = client.ask('project.inspect')
            assert after['components'] == saved['components'] and after['wires'] == saved['wires']
            assert after['transient']['sample'] == saved['transient']['sample'] and not after['document']['dirty']
            client.ask('export.scope.csv', {'path': str(csv_path)}, True, expect='file-exists')
            client.ask('export.schematic.svg', {'path': str(doc_path), 'overwrite': True}, True, expect='invalid-params')
            report = {'passed': True, 'transport': 'desktop socket' if args.connect else 'headless stdio',
                      'capabilities': len(catalog), 'document_version': 4,
                      'scope_window': window['scope']['window'], 'statistics': window['scope']['statistics'],
                      'interval_rows': len(rows) - 1, 'cursor_interval_s': measured['scope']['delta_s'],
                      'svg_bytes': svg_path.stat().st_size,
                      'checks': ['component search', 'duplicate preserves value/rotation/startup without wires',
                                 'named undo/redo', 'atomic edit commit/rollback and one undo',
                                 'direct plot command path (native parity suite)', 'fixed scope interval',
                                 'interval min/max/sample mean/RMS', 'probe selections reopen in v3 file',
                                 'CSV interval units and run/engine identity', 'headless SVG using shared routes',
                                 'exports preserve schematic and refuse unsafe destinations']}
            if args.artifacts:
                report['artifacts'] = [str(p) for p in [doc_path, csv_path, svg_path]]
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(report, indent=2))
        # Leave a paused, inspectable experiment in the connected app.
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

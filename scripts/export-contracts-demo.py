#!/usr/bin/env python3
"""Owned export contracts, native/headless output and associated-file alias protection."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import struct
import time
import xml.etree.ElementTree as ET

from demo_session import connect_demo_client
from recovery_timing import settle_native_view

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
COMMANDS = ['export.page.set', 'export.start', 'export.schematic.svg', 'export.schematic.pdf',
            'export.schematic.png', 'verification.export']


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def protected(state):
    return {key: state.get(key) for key in ['components', 'wires', 'document', 'view', 'transient',
            'mixed', 'npn', 'scope', 'undo_entry', 'redo_entry', 'run_id', 'electrical_revision']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve();assets.mkdir(parents=True, mode=0o700, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        initial = client.ask('project.inspect')
        if initial['recovery'].get('directory'):
            client.ask('recovery.configure', {'directory': initial['recovery']['directory'], 'enabled': False, 'interval_s': 2}, True)
        before = client.ask('project.inspect')
        schemas = {command: client.ask('capabilities.list', {'command': command})['capabilities'][0]['input_schema'] for command in COMMANDS}
        assert all(s['x-command-contract']['scope'] == 'captured-file-export' for s in schemas.values())
        assert before == client.ask('project.inspect')
        client.ask('example.rc', {}, True);client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True);client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': .003}, True)
        client.ask('scope.probes.set', {'ids': ['v:C1', 'i:C1']}, True)
        accepted = client.ask('simulation.verify', {'duration_s': .003, 'probe_ids': ['v:C1', 'i:C1']}, True)
        record_id = accepted['last_operation'];deadline = time.monotonic()+10
        while True:
            record = client.ask('operation.inspect', {'id': record_id})
            if record['operation']['status'] == 'completed': break
            assert record['operation']['status'] not in ('failed', 'canceled') and time.monotonic()<deadline
            time.sleep(.005)
        sweep = client.ask('sweep.start', {'settings': {'component_id': 'V1', 'parameter': 'voltage', 'start': '0 V', 'stop': '1 V', 'points': 2, 'probe_ids': ['v:C1']}}, True)
        sweep_id = sweep['dc_sweep']['selected_operation'];deadline = time.monotonic()+10
        while True:
            sweep = client.ask('sweep.inspect', {'id': sweep_id})
            if sweep['operation']['status'] == 'completed': break
            assert sweep['operation']['status'] not in ('failed', 'canceled') and time.monotonic()<deadline
            time.sleep(.005)
        before_sweep_hide = client.ask('project.inspect')
        client.ask('sweep.visibility.set', {'visible': False}, True)
        if args.connect:
            settle_native_view(client);client.ask('view.pan', {'dx': 0, 'dy': 0}, True)
        source = assets/'source.elab'
        client.ask('project.save', {'path': str(source)}, True)
        source_hash = digest(source);baseline = client.ask('project.inspect')
        alias = assets/'directory-alias';alias.symlink_to(assets, target_is_directory=True)
        outcomes = [];files = [];refusals = []

        def terminal(identity):
            deadline = time.monotonic()+10
            while True:
                work = client.ask('project.work.inspect', {'id': identity})
                if work['status'] not in ('queued', 'running', 'canceling'):return work
                assert time.monotonic()<deadline
                time.sleep(.005)

        def preserve(state):
            assert protected(state) == protected(baseline)
            assert digest(source) == source_hash

        def refuse(command, params):
            before = client.ask('project.inspect')
            client.ask(command, params, True, 'invalid-params')
            assert before == client.ask('project.inspect')
            refusals.append(dict(command=command, params=params, code='invalid-params'))

        for bad in [{'tile': True}, {'scale': 0}, {'overlap_mm': 1}, {'paper': 'legal'}]:refuse('export.page.set', bad)
        client.ask('export.page.set', {'paper': 'a4', 'orientation': 'portrait', 'color': True}, True)
        client.ask('export.page.set', {'margin_mm': 0, 'paper': None}, True)
        page = client.ask('project.inspect')['schematic_page']
        assert page['paper']=='letter' and page['orientation']=='landscape' and page['margin_mm']==0 and not page.get('color') and not page.get('scale')
        preserve(client.ask('project.inspect'))

        kinds = ['schematic-svg', 'schematic-pdf', 'schematic-png', 'scope-csv', 'verification-json', 'verification-csv', 'sweep-json', 'sweep-csv']
        for kind in kinds:
            params = dict(kind=kind, path=str(alias/'source.elab'), overwrite=True)
            if kind.startswith('verification-'):params['id']=record_id
            if kind.startswith('sweep-'):params['id']=sweep_id
            admitted = client.ask('export.start', params, True)
            work = terminal(admitted['project_work']['id'])
            assert work['status']=='failed' and work['error']['code']=='invalid-params', work
            preserve(client.ask('project.inspect'))
            outcomes.append(dict(kind=kind, status=work['status'], code=work['error']['code']))
        for command, params in [('export.schematic.svg', {}), ('export.schematic.pdf', {}), ('export.schematic.png', {}),
                                ('verification.export', {'id':record_id, 'format':'json'})]:
            client.ask(command, dict(params, path=str(alias/'source.elab'), overwrite=True), True, 'invalid-params')
            preserve(client.ask('project.inspect'))
            outcomes.append(dict(command=command, code='invalid-params'))

        for command, name, options in [('export.schematic.svg', 'schematic.svg', {}),
                    ('export.schematic.pdf', 'schematic.pdf', {'page': {'paper': 'a4', 'orientation': 'portrait'}}),
                    ('export.schematic.png', 'schematic.png', {'dpi':72}),
                    ('verification.export', 'record.json', {'id':record_id, 'format':'json'})]:
            path = alias/name
            state = client.ask(command, dict(options, path=str(path)), True)
            preserve(state);artifact=state['last_export'];data=path.read_bytes()
            assert artifact['sha256']==digest(path) and artifact['bytes']==len(data)
            if name.endswith('.svg'):ET.fromstring(data)
            if name.endswith('.pdf'):assert data.startswith(b'%PDF-')
            if name.endswith('.png'):assert data[:8]==b'\x89PNG\r\n\x1a\n' and struct.unpack('>II',data[16:24])==(792,612)
            if name=='record.json':assert json.loads(data)['metadata']==record['metadata']
            files.append(dict(name=name, sha256=digest(path), bytes=len(data), format=artifact['format']))
        no_overwrite = client.ask('project.inspect')
        client.ask('export.schematic.svg', {'path':str(alias/'schematic.svg')}, True, 'file-exists')
        preserve(client.ask('project.inspect'))
        assert client.ask('project.inspect')['last_export']==no_overwrite['last_export']
        assert not list(assets.glob('.elab-export-*'))
        report=dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', contracts=schemas,
                    atomic_refusals=refusals, alias_outcomes=outcomes, files=files, source_sha256=source_hash,
                    live=protected(baseline), capture_rows=client.ask('operation.samples', {'id':record_id, 'limit':100})['rows'],
                    allocation_before_sweep_hide=dict(viewport=before_sweep_hide['viewport'], sweep_visible=before_sweep_hide['dc_sweep']['visible']),
                    page_reset=page, limits=['Named owned-session callback/socket and file tests; no full physical typing/AX/printer or arbitrary filesystem race guarantee',
                                          'Backend image magic/dimensions and valid PDF/SVG are not fresh pixel/readability qualification'])
        args.output.parent.mkdir(parents=True,exist_ok=True)
        with args.output.open('x') as output:json.dump(report,output,indent=2);output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], contracts=len(schemas), aliases=len(outcomes), files=len(files))))
    finally:
        try:
            if args.connect:client.ask('app.close', {'discard_changes':True}, True)
        finally:client.close()


if __name__ == '__main__':
    main()

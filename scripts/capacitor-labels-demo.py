#!/usr/bin/env python3
"""Render capacitor captions using shared commands in a fresh owned session."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import xml.etree.ElementTree as ET

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def invariant(state):
    return {key: state.get(key) for key in ('components', 'wires', 'document', 'undo_name',
                                           'redo_name', 'run_id', 'electrical_revision',
                                           'mixed', 'transient', 'result')}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closed unless --keep-open succeeds')
    parser.add_argument('--keep-open', action='store_true', help='Leave owned rotation grid for visual inspection')
    parser.add_argument('--legacy', type=Path, required=True, help='Existing saved flasher with one recording; read only')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assert not args.keep_open or args.connect
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect)
    mutate = lambda name, params=None: client.ask(name, params, True)
    keep = False
    exports = {}

    def export(name):
        before = invariant(client.ask('project.inspect'))
        files = {}
        for fmt in ('svg', 'png', 'pdf'):
            path = assets/f'{name}.{fmt}'
            mutate('export.schematic.'+fmt, {'path': str(path)})
            data = path.read_bytes()
            assert len(data) > 500
            if fmt == 'pdf':
                assert data.startswith(b'%PDF-')
            else:
                files[fmt+'_sha256'] = hashlib.sha256(data).hexdigest()
        assert invariant(client.ask('project.inspect')) == before, name
        exports[name] = files

    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh owned blank session'
        mutate('simulation.pause')
        expected = {}
        for rotation, x, y in [(0, -200, -150), (90, 200, -150), (180, -200, 150), (270, 200, 150)]:
            identity = f'C_{rotation}'
            mutate('component.place', {'kind': 'capacitor', 'id': identity, 'value': '10uF',
                                       'position': {'x': x, 'y': y}})
            for _ in range(rotation//90):
                mutate('component.rotate', {'id': identity})
            expected[identity] = {'x': x-90 if rotation in (90, 270) else x+34,
                                  'y': y-36 if rotation in (90, 270) else y+3}
        mutate('scope.visibility.set', {'visible': False})
        mutate('view.fit', {'width_px': 700, 'height_px': 450})
        mutate('project.save', {'path': str(assets/'rotations.elab')})
        export('rotations')
        xml = ET.fromstring((assets/'rotations.svg').read_bytes())
        labels = {}
        for node in xml.iter('{http://www.w3.org/2000/svg}text'):
            # Main captions inherit 14px; the separate component legend declares 12px.
            if 'font-size' in node.attrib:
                continue
            for identity in expected:
                if (node.text or '').startswith(identity+' · '):
                    assert identity not in labels
                    labels[identity] = {'x': float(node.attrib['x']), 'y': float(node.attrib['y'])}
        assert labels == expected, labels
        original = args.legacy.read_bytes()
        legacy = json.loads(original)
        mutate('project.load', {'path': str(args.legacy.resolve())})
        reopened = client.ask('simulation.wait')
        assert reopened['components'] == legacy['graph']['components'] and reopened['wires'] == legacy['graph']['wires']
        assert reopened['paused'] == legacy['paused'] and reopened['document']['recording_count'] == 1
        mutate('project.save', {'path': str(assets/'legacy-reopened.elab')})
        saved = json.loads((assets/'legacy-reopened.elab').read_bytes())
        assert saved['graph'] == legacy['graph'] and saved['recordings'] == legacy['recordings']
        mutate('simulation.pause')
        export('legacy-flasher')
        assert args.legacy.read_bytes() == original
        examples = client.ask('examples.list')
        for example in examples:
            identity = example['id']
            mutate('example.load', {'id': identity, 'discard_changes': True})
            mutate('simulation.pause')
            mutate('simulation.restart')
            state = client.ask('simulation.wait')
            assert state['status'] != 'error', (identity, state['diagnostics'])
            export('example-'+identity)
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      rotation_labels=labels, prior_project_sha256=hashlib.sha256(original).hexdigest(),
                      example_ids=[e['id'] for e in examples], exports=exports,
                      checks=['four upright capacitor rotations render in incomplete schematic',
                              'SVG/PNG/PDF capture preserves circuit, physics, history and document state',
                              'previous-release saved flasher geometry and kept recording reopen/resave unchanged',
                              'all guided examples render through shared export commands'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: report[k] for k in ('passed', 'transport', 'rotation_labels')}, indent=2))
        if args.keep_open:
            mutate('project.load', {'path': str(assets/'rotations.elab'), 'discard_changes': True})
            mutate('scope.visibility.set', {'visible': False})
            mutate('view.fit', {'follow_viewport': True})
            keep = True
    finally:
        if args.connect and not keep:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()

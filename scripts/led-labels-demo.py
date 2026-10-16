#!/usr/bin/env python3
"""Render upright LED labels through shared commands; compare native/headless output."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import xml.etree.ElementTree as ET

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def invariants(state):
    return {key: state.get(key) for key in ('components', 'wires', 'document', 'undo_name',
                                           'redo_name', 'run_id', 'electrical_revision',
                                           'mixed', 'transient', 'result')}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closes unless --keep-open succeeds')
    parser.add_argument('--keep-open', action='store_true', help='Leave owned native rotation grid for visual inspection')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--legacy', type=Path, required=True, help='Existing previous-release 30 V LED fixture; read only')
    args = parser.parse_args()
    assert not args.keep_open or args.connect
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect)
    mutate = lambda command, params=None: client.ask(command, params, True)
    keep = False
    reports = {}

    def export(name, pdf=False):
        before = invariants(client.ask('project.inspect'))
        files = {}
        for fmt in ('svg', 'png') + (('pdf',) if pdf else ()):
            path = assets/f'{name}.{fmt}'
            mutate('export.schematic.'+fmt, {'path': str(path)})
            data = path.read_bytes()
            assert len(data) > 500
            if fmt == 'pdf':
                assert data.startswith(b'%PDF-')
            else:
                files[fmt+'_sha256'] = hashlib.sha256(data).hexdigest()
        assert invariants(client.ask('project.inspect')) == before, name
        reports[name] = files

    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh owned blank session'
        mutate('simulation.pause')
        for rotation, x, y in [(0, -160, -120), (90, 160, -120), (180, -160, 120), (270, 160, 120)]:
            identity = f'D_{rotation}'
            mutate('component.place', {'id': identity, 'kind': 'led', 'position': {'x': x, 'y': y}})
            for _ in range(rotation//90):
                mutate('component.rotate', {'id': identity})
        mutate('scope.visibility.set', {'visible': False})
        mutate('view.fit', {'width_px': 700, 'height_px': 450})
        mutate('project.save', {'path': str(assets/'rotations.elab')})
        export('rotations', pdf=True)
        xml = ET.fromstring((assets/'rotations.svg').read_bytes())
        labels = {n.text: {'x': float(n.attrib['x']), 'y': float(n.attrib['y'])}
                  for n in xml.iter('{http://www.w3.org/2000/svg}text') if n.text in {'D_0', 'D_90', 'D_180', 'D_270'}}
        assert set(labels) == {'D_0', 'D_90', 'D_180', 'D_270'}, labels
        original = args.legacy.resolve().read_bytes()
        mutate('project.load', {'path': str(args.legacy.resolve())})
        state = client.ask('simulation.wait')
        assert state['status'] != 'error' and not state['mixed']['stale'], state['diagnostics']
        current = state['mixed']['leds'][0]['current_a']
        assert abs(current-.027775356354112588) < 1e-10
        export('led30', pdf=True)
        mutate('object.delete', {'id': 'V3'})
        mutate('wire.connect', {'from': 'V1.positive', 'to': 'R1.1'})
        mutate('simulation.restart')
        state = client.ask('simulation.wait')
        current24 = state['mixed']['leds'][0]['current_a']
        assert abs(current24-.02179388334119657) < 1e-10
        export('led24', pdf=True)
        assert args.legacy.resolve().read_bytes() == original
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
                      rotation_labels=labels, prior_project_current_a=current, recreated24_current_a=current24,
                      prior_project_sha256=hashlib.sha256(original).hexdigest(),
                      example_ids=[e['id'] for e in examples], exports=reports,
                      checks=['four upright LED rotations render in incomplete schematic',
                              'SVG/PNG/PDF capture preserves graph, physics, history and document state',
                              'actual previous-release 30 V LED project reopens without changing original bytes',
                              'all guided examples render through shared export commands'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: report[k] for k in ('passed', 'transport', 'rotation_labels', 'prior_project_current_a')}, indent=2))
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

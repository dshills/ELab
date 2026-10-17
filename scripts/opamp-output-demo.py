#!/usr/bin/env python3
"""Exercise amplifier exports/preview through a fresh owned public session."""
import argparse
import base64
import hashlib
import importlib.util
import json
from pathlib import Path
import time

from demo_session import connect_demo_client


spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def invariant(state):
    return {key: state.get(key) for key in ('components', 'wires', 'mixed', 'transient',
        'run_id', 'electrical_revision', 'undo_entry', 'redo_entry', 'document')}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closed afterward')
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect)
    mutate = lambda command, params=None: client.ask(command, params, True)
    observations = []

    def rotate(identity, rotation):
        before = client.ask('project.inspect')
        for _ in range(4):
            part = next(p for p in client.ask('project.inspect')['components'] if p['id'] == identity)
            if part['rotation'] == rotation:
                break
            mutate('component.rotate', {'id': identity})
        else:
            raise AssertionError('rotation did not settle')
        after = client.ask('project.inspect')
        for key in ('mixed', 'transient', 'run_id', 'electrical_revision'):
            assert before[key] == after[key], ('geometry changed physics', identity, key)

    def wait_preview():
        deadline = time.monotonic()+10
        while True:
            preview = client.ask('project.inspect')['print_preview']
            assert preview['status'] not in ('failed', 'canceled'), preview
            if preview['status'] == 'completed':
                return preview
            assert time.monotonic() < deadline, preview
            time.sleep(.01)

    def capture(name, tiled=False):
        before = invariant(client.ask('project.inspect'))
        paths = [assets/f'{name}.{suffix}' for suffix in ('svg', 'pdf')]
        for path in paths:
            mutate('export.schematic.'+path.suffix[1:], {'path': str(path)})
        assert paths[1].read_bytes().startswith(b'%PDF-')
        mutate('print.preview.start')
        preview = wait_preview()
        count = preview['page_count']
        assert count == preview['rows']*preview['columns']
        assert 1 < count <= 16 if tiled else count == 1
        pages = []
        for index in range(count):
            if index:
                mutate('print.preview.page.set', {'id': preview['id'], 'index': index})
                preview = wait_preview()
            assert preview['page_index'] == index and preview['page_count'] == count and preview['source_current']
            data = bytearray()
            while len(data) < preview['bytes']:
                chunk = client.ask('print.preview.image', {'id': preview['id'], 'offset': len(data)})
                assert chunk['offset'] == len(data) and chunk['sha256'] == preview['sha256']
                data.extend(base64.b64decode(chunk['png_base64'], validate=True))
                assert chunk['next_offset'] == len(data) and chunk['total'] == preview['bytes']
            assert hashlib.sha256(data).hexdigest() == preview['sha256']
            shown = assets/f'{name}-preview-{index+1}.png'
            shown.write_bytes(data)
            png = assets/f'{name}-page-{index+1}.png'
            params = {'path': str(png)}
            if tiled:
                params['page_index'] = index
            mutate('export.schematic.png', params)
            assert png.read_bytes() == data, 'preview differs from current exported page'
            paths.extend((png, shown))
            pages.append({'index': index, 'sha256': preview['sha256'],
                          'width_px': preview['width_px'], 'height_px': preview['height_px']})
        mutate('print.preview.close', {'id': preview['id']})
        assert invariant(client.ask('project.inspect')) == before, 'output changed drawing, physics, history or document'
        observations.append({'name': name, 'pages': pages, 'source_state_unchanged': True,
            'preview_matches_export': True, 'sha256': {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}})

    try:
        for identity in ('opamp-follower', 'opamp-inverting'):
            mutate('example.load', {'id': identity, 'discard_changes': True})
            mutate('simulation.pause')
            mutate('simulation.restart')
            client.ask('simulation.wait')
            mutate('simulation.step', {'duration_s': .025})
            mutate('export.page.set', {'paper': 'letter', 'orientation': 'landscape', 'margin_mm': 10, 'color': True})
            capture(identity)
        for rotation in (0, 180, 270):
            rotate('VIN', rotation)
            capture(f'source-{rotation}')
        rotate('VIN', 90)
        for rotation in (0, 180, 270):
            rotate('GNDPLUS', rotation)
            capture(f'ground-{rotation}')
        rotate('GNDPLUS', 90)
        mutate('export.page.set', {'paper': 'a4', 'orientation': 'portrait', 'margin_mm': 10, 'color': False})
        capture('inverting-grayscale')
        mutate('export.page.set', {'paper': 'letter', 'orientation': 'landscape', 'margin_mm': 10,
                                  'color': True, 'scale': 1, 'tile': True, 'overlap_mm': 5})
        capture('inverting-tiled', tiled=True)
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
            'observations': observations,
            'checks': ['guided amplifiers and all sine-source/ground rotations',
                       'Letter color, A4 portrait grayscale and fixed-scale tiled output',
                       'every captured preview page matches exported PNG bytes and checksums',
                       'rotation preserves physics; output preserves graph, physics, document and history'],
            'limitations': ['PDF layout acceptance requires independent rendering and visual inspection',
                            'no physical print job is submitted',
                            'native rendering, pointer, accessibility and visible latency require separate evidence']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({'passed': True, 'captures': len(observations), 'transport': report['transport']}))
    finally:
        try:
            if args.connect:
                mutate('app.close', {'discard_changes': True})
        finally:
            client.close()


if __name__ == '__main__':
    main()

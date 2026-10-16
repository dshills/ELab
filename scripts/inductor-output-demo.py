#!/usr/bin/env python3
"""Qualify coil PDF/PNG/preview output in a fresh owned public session."""
import argparse
import base64
import hashlib
import importlib.util
import json
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def physics(state):
    return {key: state.get(key) for key in ('components', 'wires', 'mixed', 'transient',
                                           'run_id', 'electrical_revision', 'undo_entry')}


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native session; closed afterward')
    parser.add_argument('--artifacts', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect)
    mutate = lambda command, params=None: client.ask(command, params, True)
    observations = []

    def load(identity):
        mutate('example.load', {'id': identity, 'discard_changes': True})
        mutate('simulation.pause')
        mutate('simulation.restart')
        state = client.ask('simulation.wait')
        assert state['mixed']['engine'] == 'elab.rlc-mna-radau5.v1' and not state['mixed']['stale']
        return state

    def capture(name):
        before = physics(client.ask('project.inspect'))
        pdf, png = assets/f'{name}.pdf', assets/f'{name}.png'
        mutate('export.schematic.pdf', {'path': str(pdf)})
        assert pdf.read_bytes().startswith(b'%PDF-')
        mutate('print.preview.start')
        deadline = time.monotonic()+10
        while True:
            preview = client.ask('project.inspect')['print_preview']
            assert preview['status'] not in ('failed', 'canceled'), preview
            if preview['status'] == 'completed':
                break
            assert time.monotonic() < deadline, preview
            time.sleep(.01)
        assert preview['page_count'] == 1 and preview['source_current']
        data = bytearray()
        while len(data) < preview['bytes']:
            chunk = client.ask('print.preview.image', {'id': preview['id'], 'offset': len(data)})
            assert chunk['offset'] == len(data) and chunk['sha256'] == preview['sha256']
            data.extend(base64.b64decode(chunk['png_base64'], validate=True))
            assert chunk['next_offset'] == len(data) and chunk['total'] == preview['bytes']
        assert hashlib.sha256(data).hexdigest() == preview['sha256']
        mutate('export.schematic.png', {'path': str(png)})
        assert png.read_bytes() == data
        (assets/f'{name}-preview.png').write_bytes(data)
        mutate('print.preview.close', {'id': preview['id']})
        assert physics(client.ask('project.inspect')) == before
        observations.append({'name': name, 'pdf_sha256': hashlib.sha256(pdf.read_bytes()).hexdigest(),
                             'png_sha256': preview['sha256'], 'width_px': preview['width_px'],
                             'height_px': preview['height_px'], 'preview_matches_export': True,
                             'physical_state_unchanged': True})

    try:
        assert not client.ask('project.inspect')['components'], 'requires a fresh owned blank session'
        mutate('export.page.set', {'paper': 'letter', 'orientation': 'landscape', 'margin_mm': 10, 'color': True})
        for identity in ('rl', 'rlc', 'flyback'):
            load(identity)
            capture(identity)
        load('rl')
        for rotation in (0, 90, 180, 270):
            while True:
                part = next(p for p in client.ask('project.inspect')['components'] if p['id'] == 'L1')
                if part['rotation'] == rotation:
                    break
                mutate('component.rotate', {'id': 'L1'})
            capture(f'coil-{rotation}')
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'observations': observations,
                  'checks': ['all three coil examples and four symbol rotations',
                             'single-page Letter landscape PDF and captured preview',
                             'preview chunk length/checksum/ownership',
                             'captured preview equals PNG export bytes',
                             'output and preview preserve graph, physics and history'],
                  'limitations': ['PDF visual acceptance requires independent rendering and inspection',
                                  'no printer job is submitted; physical printing remains open']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({'passed': True, 'outputs': len(observations), 'transport': report['transport']}))
    finally:
        if args.connect:
            mutate('app.close', {'discard_changes': True})
        client.close()


if __name__ == '__main__':
    main()

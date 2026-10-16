#!/usr/bin/env python3
"""Exercise captured print preview through the public agent protocol."""
import argparse
import base64
import hashlib
import importlib.util
import json
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def completed():
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            state = client.ask('project.inspect')
            preview = state['print_preview']
            assert preview['status'] not in ('failed', 'canceled'), preview
            if preview['status'] == 'completed':
                return state
            time.sleep(.01)
        raise AssertionError('preview did not complete')

    def pixels(preview):
        data = bytearray()
        while len(data) < preview['bytes']:
            chunk = client.ask('print.preview.image', {'id': preview['id'], 'offset': len(data)})
            assert chunk['offset'] == len(data) and chunk['sha256'] == preview['sha256']
            data.extend(base64.b64decode(chunk['png_base64'], validate=True))
            assert chunk['next_offset'] == len(data) and chunk['total'] == preview['bytes']
        assert hashlib.sha256(data).hexdigest() == preview['sha256']
        return bytes(data)

    try:
        client.ask('example.rectifier', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        client.ask('export.page.set', {'paper': 'a4', 'orientation': 'portrait', 'margin_mm': 12, 'color': True}, True)
        before = client.ask('project.inspect')
        client.ask('print.preview.start', {}, True)
        after = completed()
        first = after['print_preview']
        assert first['source_current'] and first['visible']
        for key in ('document', 'components', 'wires', 'transient', 'run_id', 'electrical_revision', 'undo_entry'):
            assert before.get(key) == after.get(key), key
        captured = pixels(first)
        (directory / 'captured-preview.png').write_bytes(captured)
        exported = directory / 'exported-page.png'
        client.ask('export.schematic.png', {'path': str(exported)}, True)
        assert exported.read_bytes() == captured
        client.ask('component.set', {'id': 'RL', 'value': '20k'}, True)
        stale = client.ask('project.inspect')['print_preview']
        assert not stale['source_current'] and pixels(stale) == captured
        client.ask('print.preview.start', {}, True)
        refreshed = completed()['print_preview']
        assert refreshed['source_current'] and refreshed['id'] != first['id']
        (directory / 'refreshed-preview.png').write_bytes(pixels(refreshed))
        client.ask('print.preview.close', {'id': first['id']}, True, expect='not-found')
        client.ask('print.preview.close', {'id': refreshed['id']}, True)
        closed = client.ask('project.inspect')['print_preview']
        assert not closed['visible'] and closed['bytes'] == 0
        client.ask('print.preview.image', {'id': refreshed['id']}, expect='not-ready')
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'captured': first, 'refreshed': refreshed, 'closed': closed,
                  'checks': ['capture preserves source/physics/history', 'chunk checksum and ownership',
                             'preview equals PNG export bytes', 'later edits retain captured pixels and mark stale',
                             'refresh captures latest source', 'old preview ID cannot close new capture', 'close releases image'],
                  'limits': ['macOS CoreGraphics page backend', 'preview only; no printer job or physical print proof']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as target:
            target.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'passed': True, 'transport': report['transport'], 'checks': len(report['checks'])}))
        client.ask('project.clear', {'discard_changes': True}, True)
    finally:
        client.close()


if __name__ == '__main__':
    main()

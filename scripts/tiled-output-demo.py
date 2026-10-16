#!/usr/bin/env python3
"""Public tiled PDF/PNG/preview workflow; never submits to a physical printer."""
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
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def inspect():
        return client.ask('project.inspect')

    def wait_preview():
        deadline = time.monotonic()+10
        while True:
            p = inspect()['print_preview']
            assert p['status'] not in ('failed', 'canceled'), p
            if p['status'] == 'completed':
                return p
            assert time.monotonic() < deadline
            time.sleep(.01)

    def image(p):
        data = bytearray()
        while len(data) < p['bytes']:
            chunk = client.ask('print.preview.image', dict(id=p['id'], offset=len(data), limit=65536))
            assert chunk['offset'] == len(data) and chunk['sha256'] == p['sha256']
            data.extend(base64.b64decode(chunk['png_base64']))
            assert chunk['next_offset'] == len(data) and chunk['total'] == p['bytes']
        assert hashlib.sha256(data).hexdigest() == p['sha256']
        return bytes(data)

    def source(state):
        return {key: state.get(key) for key in ('components', 'wires', 'transient', 'document', 'undo_entry', 'run_id', 'electrical_revision')}

    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('component.move', dict(id='C1', position=dict(x=800, y=600)), True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        original = source(inspect())
        contract = client.ask('capabilities.list', {'command': 'print.preview.page.set'})['capabilities'][0]['input_schema']
        assert contract['x-command-contract']['scope'] == 'captured-preview-page'
        observations, manifests = [], []
        for paper, orientation in [('letter', 'landscape'), ('a4', 'portrait')]:
            page = dict(paper=paper, orientation=orientation, margin_mm=10, scale=1, tile=True, overlap_mm=5)
            client.ask('export.page.set', page, True)
            client.ask('print.preview.start', {}, True)
            p = wait_preview()
            count, rows, columns = p['page_count'], p['rows'], p['columns']
            assert count == rows*columns and 4 <= count <= 16
            pdf = assets / f'tiled-{paper}.pdf'
            client.ask('export.schematic.pdf', {'path': str(pdf)}, True)
            assert pdf.read_bytes().startswith(b'%PDF-')
            first_image = None
            for index in range(count):
                if index:
                    old = p['id']
                    client.ask('print.preview.page.set', dict(id=old, index=index), True)
                    p = wait_preview()
                    assert p['id'] != old
                    client.ask('print.preview.image', {'id': old}, expect='not-found')
                assert p['page_index'] == index and p['page_count'] == count and p['source_current']
                before = inspect()
                client.ask('print.preview.page.set', dict(id=p['id'], index=count), True, 'invalid-params')
                assert inspect() == before
                data = image(p)
                if index == 0:
                    first_image = data
                shown = assets / f'{paper}-shown-{index+1}.png'
                client.ask('export.schematic.png', dict(path=str(shown), preview_id=p['id']), True)
                assert shown.read_bytes() == data
                direct = assets / f'{paper}-direct-{index+1}.png'
                client.ask('export.schematic.png', dict(path=str(direct), page_index=index), True)
                assert direct.read_bytes() == data
                manifests.append(dict(paper=paper, index=index, shown=shown.name, direct=direct.name,
                                      sha256=hashlib.sha256(data).hexdigest(), width=p['width_px'], height=p['height_px']))
            bad = assets / f'{paper}-implicit.png'
            client.ask('export.schematic.png', dict(path=str(bad)), True, 'file-error')
            assert not bad.exists() and source(inspect()) == original
            observations.append(dict(paper=paper, orientation=orientation, rows=rows, columns=columns,
                                     page_count=count, scale=1, overlap_mm=5, width=p['width_px'], height=p['height_px']))
            if paper == 'a4':
                # Navigation must retain old captured pixels after electrical edits.
                client.ask('component.set', dict(id='R1', value='20k'), True)
                client.ask('print.preview.page.set', dict(id=p['id'], index=0), True)
                p = wait_preview()
                assert not p['source_current'] and image(p) == first_image
                stale = assets / 'captured-before-edit.png'
                client.ask('export.schematic.png', dict(path=str(stale), preview_id=p['id']), True)
                assert stale.read_bytes() == first_image
                client.ask('print.preview.close', {'id': p['id']}, True)
                client.ask('export.schematic.png', dict(path=str(assets/'closed.png'), preview_id=p['id']), True, 'not-ready')
        (assets/'page-manifest.json').write_text(json.dumps(manifests, indent=2)+'\n')
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', page_command_schema=contract,
                      observations=observations,
                      checks=['Large connected RC schematic retains graph/physics/history while output settings and previews change',
                              'Letter landscape and A4 portrait PDFs cover complete fixed-scale row-major grids with 5 mm overlap',
                              'Every page navigates under a new preview ID; stale chunk ID and out-of-range page refuse',
                              'Exact shown-page PNG export and explicit current-source PNG page export match captured bytes for every tile',
                              'Multi-page PNG without page_index refuses without partial output',
                              'Later electrical edit does not recapture earlier page; stale captured PNG remains byte-identical',
                              'Closed preview cannot be exported; no physical printer submission'],
                      limits=['At most 16 pages and 8 MiB per output; one preview page retained/rendered at a time',
                              'PDF and print submission include all pages; PNG selects one page',
                              'Rendered PDF page-count/text/pixel review is recorded separately',
                              'Physical printer output, keyboard/IME/accessibility and visible latency remain separate gates'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], pages=[x['page_count'] for x in observations])))
    finally:
        try:
            if args.connect:
                inspect()
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()

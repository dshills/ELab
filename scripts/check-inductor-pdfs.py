#!/usr/bin/env python3
"""Parse and render actual coil PDFs; visual acceptance remains a human/agent review."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

import pdfplumber
from PIL import Image, ImageChops
from pypdf import PdfReader


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--headless', type=Path, required=True)
    parser.add_argument('--native', type=Path, required=True)
    parser.add_argument('--renders', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.renders.mkdir(parents=True, exist_ok=False)
    names = ('rl', 'rlc', 'flyback', 'coil-0', 'coil-90', 'coil-180', 'coil-270')
    reports = []
    for name in names:
        pair = []
        for transport, directory in (('headless', args.headless), ('native', args.native)):
            source = directory/f'{name}.pdf'
            reader = PdfReader(source)
            assert len(reader.pages) == 1
            with pdfplumber.open(source) as pdf:
                page = pdf.pages[0]
                assert abs(page.width-792) < .001 and abs(page.height-612) < .001
                text = ' '.join((page.extract_text() or '').split())
                assert 'L1' in text and 'ELab' in text
                assert all(0 <= c['x0'] <= c['x1'] <= 792 and 0 <= c['top'] <= c['bottom'] <= 612
                           for c in page.chars)
                assert len(page.curves) > 0
                curve_count = len(page.curves)
            prefix = args.renders/f'{transport}-{name}'
            subprocess.run(['pdftoppm', '-r', '120', '-singlefile', '-png', str(source), str(prefix)],
                           check=True, capture_output=True, timeout=30)
            image_path = prefix.with_suffix('.png')
            with Image.open(image_path) as image:
                pair.append(image.convert('RGB'))
            reports.append({'name': name, 'transport': transport, 'pages': 1,
                            'width_pt': 792, 'height_pt': 612, 'text_inside_page': True,
                            'curve_count': curve_count,
                            'pdf_sha256': hashlib.sha256(source.read_bytes()).hexdigest(),
                            'render': str(image_path),
                            'render_sha256': hashlib.sha256(image_path.read_bytes()).hexdigest()})
        assert pair[0].size == pair[1].size and ImageChops.difference(pair[0], pair[1]).getbbox() is None, name
    report = {'passed': True, 'pdfs': reports, 'native_headless_pixels_identical': True,
              'scope': '14 actual command PDFs parsed and independently rendered with Poppler at 120 DPI',
              'visual_review': 'Pending direct image inspection; parser/pixel checks do not establish layout quality',
              'limitations': ['Letter landscape only in this run', 'No physical print submitted']}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open('x') as output:
        output.write(json.dumps(report, indent=2)+'\n')
    print(json.dumps({'passed': True, 'PDFs': len(reports), 'identical_render_pairs': len(names)}))


if __name__ == '__main__':
    main()

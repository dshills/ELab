#!/usr/bin/env python3
"""Inspect public tiled outputs with pypdf/pdfplumber and Poppler-rendered pixels."""
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
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--renders', type=Path, required=True)
    args = parser.parse_args()
    args.renders.mkdir(parents=True, exist_ok=False)
    manifest = json.loads((args.artifacts/'page-manifest.json').read_text())
    reports = []
    for paper in ('letter', 'a4'):
        path = args.artifacts/f'tiled-{paper}.pdf'
        reader = PdfReader(path)
        expected = [x for x in manifest if x['paper'] == paper]
        assert len(reader.pages) == len(expected) == 6
        metadata = json.loads(str(reader.metadata['/Subject']))
        assert metadata['page']['tile'] and metadata['page']['scale'] == 1 and metadata['page']['overlap_mm'] == 5
        prefix = args.renders/paper
        subprocess.run(['pdftoppm', '-r', '100', '-png', str(path), str(prefix)], check=True, capture_output=True)
        page_reports = []
        with pdfplumber.open(path) as document:
            for i, page in enumerate(document.pages):
                width, height = (792, 612) if paper == 'letter' else (210*72/25.4, 297*72/25.4)
                assert abs(page.width-width) < .001 and abs(page.height-height) < .001
                text = ' '.join((page.extract_text() or '').split())
                footer = f'ELab | {i+1}/6 | row {i//3+1}/2 col {i%3+1}/3 | scale 100% | overlap 5 mm'
                assert footer in text, (paper, i, text[-180:])
                image_path = args.renders/f'{paper}-{i+1}.png'
                im = Image.open(image_path).convert('RGB')
                margin = int(10/25.4*100)-2
                for crop in ((0, 0, im.width, margin), (0, 0, margin, im.height),
                             (im.width-margin, 0, im.width, im.height), (0, im.height-margin, im.width, im.height)):
                    assert ImageChops.difference(im.crop(crop), Image.new('RGB', (crop[2]-crop[0], crop[3]-crop[1]), 'white')).getbbox() is None, (paper, i, crop)
                font_sizes = sorted({round(c['size'], 3) for c in page.chars})
                assert min(font_sizes) >= 8.49
                # Text outside the clip is still present in PDF content streams;
                # page geometry tests and visual review, not text extraction,
                # establish that the assembled schematic remains complete.
                page_reports.append(dict(index=i, width_pt=page.width, height_pt=page.height,
                                         footer=footer, font_sizes_pt=font_sizes, margins_white=True,
                                         render=str(image_path), render_sha256=hashlib.sha256(image_path.read_bytes()).hexdigest()))
        reports.append(dict(paper=paper, pages=page_reports, pdf_sha256=hashlib.sha256(path.read_bytes()).hexdigest()))
    result = dict(passed=True, source='public command PDFs; independent PDF parsers and Poppler rasterization',
                  checks=['Actual PDF page counts match captured grid', 'Paper dimensions, footer order/scale/overlap and font sizes',
                          'All four outer paper margins contain no ink at 100 DPI with a two-pixel rounding allowance'],
                  visual_review='Pending direct inspection of the rendered PNGs; this script does not claim visual acceptance',
                  pdfs=reports)
    with args.output.open('x') as output:
        json.dump(result, output, indent=2)
        output.write('\n')
    print(json.dumps(dict(passed=True, pages_checked=sum(len(x['pages']) for x in reports))))


if __name__ == '__main__':
    main()

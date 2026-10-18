#!/usr/bin/env python3
"""Check that the engineering audit covers every current P0 functional PRD ID."""
import argparse
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser()
    destination = parser.add_mutually_exclusive_group()
    destination.add_argument('--output', type=Path)
    destination.add_argument('--verify', type=Path, help='check a saved report without changing it')
    args = parser.parse_args()
    requirements = ROOT/'research/07-product-requirements.md'
    audit = ROOT/'research/11-prd-implementation-audit.md'
    pattern = re.compile(r'^\| ([A-Z]+-\d+) \| (.*)$', re.MULTILINE)
    required = [m[0] for m in pattern.findall(requirements.read_text())]
    rows = []
    for identifier, remainder in pattern.findall(audit.read_text()):
        columns = [s.strip() for s in remainder.strip().strip('|').split('|')]
        evidence, assessment = columns
        status = assessment.split(':', 1)[0].split()[0].lower()
        assert status in ['implemented', 'partial', 'pending'], (identifier, assessment)
        rows.append(dict(id=identifier, evidence=evidence, status=status, assessment=assessment))
    assert len(required) == len(set(required)), 'duplicate PRD ID'
    assert len(rows) == len({r['id'] for r in rows}), 'duplicate audit ID'
    assert set(required) == {r['id'] for r in rows}, 'audit coverage differs from the PRD'
    report = dict(passed=True, scope='Requirement coverage validation only; does not establish product acceptance',
                  requirements=len(required), counts={status: sum(r['status'] == status for r in rows)
                                                     for status in ['implemented', 'partial', 'pending']},
                  rows=rows)
    if args.verify:
        try:
            saved = json.loads(args.verify.read_text())
        except (OSError, ValueError) as error:
            parser.exit(1, f'Cannot read PRD audit report {args.verify}: {error}\n')
        if saved != report:
            parser.exit(1, f'PRD audit report differs from current Markdown: {args.verify}; review the changes and regenerate with --output\n')
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps(dict(passed=True, requirements=report['requirements'], counts=report['counts'], scope=report['scope']), indent=2))


if __name__ == '__main__':
    main()

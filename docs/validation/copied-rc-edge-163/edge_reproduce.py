#!/usr/bin/env python3
"""Check a fresh owned agent's RC transition against independent electrical laws."""
import argparse
import csv
import gzip
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import sys

ROOT = Path('/Users/dshills/Development/projects/ELab')
sys.path.insert(0, str(ROOT / 'scripts'))
spec = importlib.util.spec_from_file_location('persistence', ROOT / 'scripts/persistence-demo.py')
persistence = importlib.util.module_from_spec(spec)
spec.loader.exec_module(persistence)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True, exist_ok=False)
    agent = args.agent.resolve()
    report = dict(passed=False, agent_sha256=hashlib.sha256(agent.read_bytes()).hexdigest(),
                  scope='Exact copied /163 native socket or headless stdio; manual RC edge, scope, cursor and CSV',
                  transport='native-socket' if args.connect else 'headless-stdio',
                  errors=[])
    client = persistence.Client(args.connect, command_preconditions=True, agent_binary=Path('/private/tmp/elab-first-version-163-runtime/agent'))
    try:
        assert client.ask('project.inspect')['components'] == []
        client.ask('example.rc', {}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        ids = ['v:V1', 'v:R1', 'v:C1', 'i:V1', 'i:R1', 'i:C1']
        client.ask('scope.probes.set', dict(ids=ids), True)
        client.ask('simulation.input.set', dict(mode='charge'), True)
        client.ask('simulation.step', dict(duration_s=.01), True)
        client.ask('simulation.input.set', dict(mode='discharge'), True)
        client.ask('simulation.step', dict(duration_s=.01), True)
        state = client.ask('simulation.wait')
        client.ask('scope.cursor.set', dict(cursor='a', time_s=.01), True)
        client.ask('scope.cursor.set', dict(cursor='b', time_s=.01), True)
        measured = client.ask('simulation.wait')
        client.ask('export.scope.csv', dict(path=str(out / 'RC edge.csv')), True)
        client.ask('simulation.wait')
        rows = list(csv.DictReader((out / 'RC edge.csv').open(newline='')))
        samples = state['transient']['samples']
        assert len(rows) == len(samples) == 23
        edge_indices = [i for i, p in enumerate(samples) if math.isclose(p['time_s'], .01, abs_tol=1e-12)]
        assert len(edge_indices) == 2
        edge = edge_indices[1]
        v10 = 5 * -math.expm1(-1)
        want_current = -v10 / 10000

        def check(label, actual, expected):
            if not math.isclose(actual, expected, rel_tol=1e-10, abs_tol=1e-12):
                report['errors'].append(dict(check=label, actual=actual, expected=expected))

        expected_columns = [[] for _ in ids]
        for i, sample in enumerate(samples):
            at = sample['time_s']
            voltage = 5 * -math.expm1(-at / .01) if i < edge else v10 * math.exp(-(at - .01) / .01)
            source = 5 if i < edge else 0
            current = (source - voltage) / 10000
            check(f'solver voltage row {i}', sample['voltage_v'], voltage)
            check(f'solver input row {i}', sample['input_v'], source)
            check(f'solver current row {i}', sample['current_a'], current)
            values = [source, source - voltage, voltage, current, current, current]
            for j, (probe, value) in enumerate(zip(ids, values)):
                expected_columns[j].append(value)
                trace = state['scope']['traces'][j]
                assert trace['probe']['id'] == probe and len(trace['points']) == len(samples)
                check(f'scope {probe} time row {i}', trace['points'][i]['time_s'], at)
                check(f'scope {probe} row {i}', trace['points'][i]['value'], value)
                unit = 'A' if probe.startswith('i:') else 'V'
                check(f'CSV {probe} row {i}', float(rows[i][f'{probe} [{unit}]']), value)
        assert len(measured['scope']['readings']) == len(ids)
        for i, reading in enumerate(measured['scope']['readings']):
            check(f'cursor {ids[i]} A', reading['a'], expected_columns[i][edge])
            check(f'cursor {ids[i]} B', reading['b'], expected_columns[i][edge])
            check(f'cursor {ids[i]} delta', reading['delta'], 0)
        for i, stats in enumerate(state['scope']['statistics']):
            check(f'{ids[i]} minimum', stats['min'], min(expected_columns[i]))
            check(f'{ids[i]} maximum', stats['max'], max(expected_columns[i]))
        report.update(expected_edge_current_a=want_current,
                      actual_scope_edge_current_a=state['scope']['traces'][4]['points'][edge]['value'],
                      actual_csv_edge_current_a=float(rows[edge]['i:R1 [A]']),
                      solver_edge_current_a=samples[edge]['current_a'], samples=len(samples))
        with gzip.GzipFile(filename=str(out / 'state.json.gz'), mode='wb', mtime=0) as stream:
            stream.write((json.dumps(dict(state=state, measured=measured), indent=2) + '\n').encode())
        report['passed'] = not report['errors']
    except BaseException as error:
        report['exception'] = type(error).__name__ + ': ' + str(error)
        raise
    finally:
        client.close()
        report['agent_exit_code'] = client.process.returncode
        (out / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps({k: v for k, v in report.items() if k != 'errors'}))
    return 0 if report['passed'] else 1


if __name__ == '__main__':
    raise SystemExit(main())

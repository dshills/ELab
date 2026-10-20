"""Read-only replay of FLOATING-INPUT-01 source qualification."""
from pathlib import Path
import copy
import gzip
import hashlib
import json
import math

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def raw(path):
    data = path.read_bytes()
    return gzip.decompress(data) if path.suffix == '.gz' else data


def read(path):
    return json.loads(raw(path))


def cases(kind):
    summary = read(HERE/'public'/kind/'report.json')
    assert summary['passed'] and len(summary['cases']) == 4
    stage_map = {}
    for case in summary['cases']:
        returned, switched = case['return_resistor'], case['switched']
        name = f'return-{int(returned)}-switch-{int(switched)}'
        folder = HERE/'public'/kind/name
        rows = [json.loads(line) for line in raw(folder/'transcript.jsonl.gz').splitlines()]
        assert len(rows) == case['requests'] == (13 if switched else 10)
        ids = set()
        for row in rows:
            req, res = row['request'], row['response']
            assert req['version'] == res['version'] == 1
            assert req['id'] == res['id'] and req['id'] not in ids and not res.get('error')
            ids.add(req['id'])
            assert req['command'] != 'events.poll'
        assert rows[0]['request']['command'] == 'project.inspect'
        assert not rows[0]['response']['result']['components']
        batch = next(row['request']['params'] for row in rows if row['request']['command'] == 'edit.batch')
        assert len(batch['edits']) == (10 if switched else 8)
        final = rows[-1]['response']['result']
        assert rows[-1]['request']['command'] == 'app.close' and final['closed']
        assert final['shutdown']['status'] == 'completed' and final['shutdown']['exit_ready']
        assert final['shutdown']['workers_joined'] and final['shutdown']['recovery_finalized']
        assert case['agent_exit'] == 0 and raw(folder/'stderr.txt.gz') == b''
        stages = [row['response']['result'] for row in rows if row['request']['command'] == 'simulation.step']
        assert len(stages) == (3 if switched else 2)
        stage_map[name] = stages
        charged = stages[0]
        tau = (10001 if switched else 10000)*1e-6
        assert abs(charged['mixed']['capacitors'][0]['voltage_v']-5*-math.expm1(-.02/tau)) < .003
        mismatch = 0
        for stage in stages[:-1]:
            samples = stage['transient']['samples']
            points = stage['scope']['traces'][0]['points']
            assert len(samples) == len(points)
            for sample, point in zip(samples, points):
                assert point['time_s'] == sample['time_s'] and abs(point['value']-5) < 1e-12
                mismatch += abs(sample['input_v']-5) > 1e-12
            assert abs(stage['transient']['sample']['input_v']-samples[-1]['input_v']) < 1e-12
        previous, off = stages[-2], stages[-1]
        count = len(previous['transient']['samples'])
        assert previous['transient']['samples'] == off['transient']['samples'][:count]
        for sample in off['transient']['samples'][count:]:
            mismatch += abs(sample['input_v']) > 1e-12
        assert mismatch == case['mismatch_count']
        assert bool(mismatch) == case['expected_bug'] == (kind == 'original')
    assert sum(case['requests'] for case in summary['cases']) == 46
    return stage_map


def main():
    manifest_bytes = (HERE/'manifest.json').read_bytes()
    assert sha(manifest_bytes) == read(HERE/'checkpoint.json')['manifest_sha256']
    manifest = json.loads(manifest_bytes)
    found = {str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file()
             and p.name not in ('manifest.json', 'checkpoint.json')}
    assert set(manifest) == found
    for relative, entry in manifest.items():
        p = HERE/relative
        assert p.is_file() and not p.is_symlink() and p.resolve().is_relative_to(HERE)
        data = p.read_bytes()
        assert sha(data) == entry['sha256'] and len(data) == entry['bytes'], relative
    report = read(HERE/'report.json')
    quality = read(HERE/'quality.json')
    assert report['passed'] and quality['passed'] and quality['exit_code'] == 0 and quality['source_unchanged']
    assert len(quality['source_sha256']) == 651
    for relative, digest in quality['source_sha256'].items():
        assert sha((ROOT/relative).read_bytes()) == digest, relative
    for relative, digest in report['artifacts'].items():
        assert sha((ROOT/relative).read_bytes()) == digest, relative
    base = read(ROOT/report['base_receipt']['path'])
    assert sha((ROOT/report['base_receipt']['path']).read_bytes()) == report['base_receipt']['sha256']
    changed = {relative for relative, digest in base['quality']['source_sha256'].items()
               if sha((ROOT/relative).read_bytes()) != digest}
    assert changed == {'workbench/mixed_transient.go', 'workbench/switch_actuation.go'}
    for relative in changed:
        assert sha((HERE/'sources'/(Path(relative).name+'.original.txt')).read_bytes()) == base['quality']['source_sha256'][relative]
    for relative in changed | {'workbench/floating_input_test.go'}:
        assert (HERE/'sources'/(Path(relative).name+'.txt')).read_bytes() == (ROOT/relative).read_bytes()
    negative = raw(HERE/'checks/final-test-original-source.txt.gz').decode()
    assert 'FAIL' in negative and negative.count('--- FAIL: TestFloatingRCInputReportsSourceDifferenceAcrossStepsAndEvents/') == 4
    assert 'ok  ' in raw(HERE/'checks/focused.txt.gz').decode()
    original, final = cases('original'), cases('final')
    assert set(original) == set(final)
    paired = 0
    for name in original:
        assert len(original[name]) == len(final[name])
        for before, after in zip(original[name], final[name]):
            for field in ('components', 'wires', 'mixed', 'scope', 'document', 'run_id', 'electrical_revision', 'undo_entry'):
                assert before.get(field) == after.get(field), (name, field)
            x, y = copy.deepcopy(before['transient']), copy.deepcopy(after['transient'])
            for trace in (x, y):
                trace['sample'].pop('input_v')
                for sample in trace['samples']:
                    sample.pop('input_v')
            assert x == y, name
            paired += 1
    assert paired == report['paired_step_snapshots'] == 10
    assert not report['new_native_interaction'] and not report['replacement_packaged']
    print('Verified original failures, four repaired cases, 92 public requests, ten physics-preserving pairs and full source quality.')


if __name__ == '__main__':
    main()

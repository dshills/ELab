#!/usr/bin/env python3
"""Verify preserved paging evidence offline; never launch a process or write files."""
import copy
import gzip
import hashlib
import json
from pathlib import Path
import sys

HERE = Path(__file__).resolve().parent


def read(name):
    data = (HERE / name).read_bytes()
    return gzip.decompress(data) if name.endswith('.gz') else data


def load(name):
    return json.loads(read(name))


def capability(catalog):
    return next(c for c in catalog['capabilities'] if c['id'] == 'operation.samples')


def check_session(prefix, count, total):
    report = load(prefix + '/report.json.gz')
    transcript = [json.loads(line) for line in read(prefix + '/transcript.jsonl.gz').splitlines()]
    assert len(transcript) == report['requests'] == total
    assert len(report['cases']) == count and report['process_exit_code'] == 0
    fixture = load(prefix + '/recorded-rc.elab.gz')
    recorded = fixture['recordings']['entries'][0]['data']
    assert len(recorded['rows']) == recorded['metadata']['sample_count'] == 209
    plan = load(prefix + ('/plan.json' if prefix == 'original' else '/plan-source.json'))
    assert hashlib.sha256(read(prefix + '/recorded-rc.elab.gz')).hexdigest() == plan['fixture_sha256']
    baseline = load(prefix + '/baseline.json.gz')
    assert transcript[2]['response']['result'] == baseline
    assert baseline['paused'] and not baseline['document']['dirty']
    assert next(c['value'] for c in baseline['components'] if c['id'] == 'R1') == 30000
    identifier = baseline['recording']['id']
    for entry in transcript:
        request, response = json.loads(entry['request_json']), entry['response']
        assert response['version'] == 1
        if response.get('error', {}).get('code') == 'invalid-json':
            assert '"limit":1,"limit":2' in entry['request_json']
            assert response['id'] == '' and response['revision'] == 0
        else:
            assert response['id'] == request['id']
    for index, case in enumerate(report['cases']):
        before, query, after = transcript[3+3*index:6+3*index]
        assert json.loads(before['request_json'])['command'] == 'project.inspect'
        assert json.loads(after['request_json'])['command'] == 'project.inspect'
        assert before['response']['result'] == after['response']['result'] == baseline
        request, response = json.loads(query['request_json']), query['response']
        assert request['command'] == case['command']
        assert response.get('error', {}).get('code') == case['error']
        if case['error']:
            assert 'result' not in response
            continue
        result = response['result']
        if request['command'] == 'operation.samples':
            offset = request['params'].get('offset') or 0
            limit = request['params'].get('limit') or 100
            rows = recorded['rows'][offset:offset+limit]
            assert result == dict(id=identifier, offset=offset, next_offset=offset+len(rows),
                                  total=209, probes=recorded['metadata']['probes'], rows=rows)
        elif request['command'] == 'operation.inspect':
            assert result['metadata'] == recorded['metadata']
            assert result['operation']['status'] == 'completed'
            assert result['operation']['source_unchanged'] is False
    closed = transcript[-1]['response']['result']
    assert closed == load(prefix + '/close-response.json.gz')
    assert closed['closed'] and closed['shutdown']['status'] == 'completed'
    assert all(closed['shutdown'][k] for k in ['workers_joined', 'exit_ready', 'recovery_finalized'])
    if prefix == 'original':
        assert not report['passed'] and not report['behavior_passed']
        assert transcript[-2]['response']['error']['code'] == 'invalid-json'
        assert load('original/analysis.json')['collector_complete'] is False
    else:
        assert report['passed'] and report['behavior_passed'] and not report['findings']
        assert report['source_file_unchanged'] and load('source/final.json.gz') == baseline
    return report


def verify():
    assert sys.flags.optimize == 0, 'Run without Python optimization'
    for name, entry in load('manifest.json')['files'].items():
        relative = Path(name)
        assert not relative.is_absolute() and '..' not in relative.parts
        path = HERE / relative
        assert path.is_file() and not path.is_symlink(), name
        data = path.read_bytes()
        assert len(data) == entry['bytes'] and hashlib.sha256(data).hexdigest() == entry['sha256'], name
    check_session('original', 29, 93)
    check_session('source', 30, 95)
    old = load('original/catalog.json.gz')
    new = load('source/catalog.json.gz')
    assert len(old['capabilities']) == len(new['capabilities']) == 164
    expected = copy.deepcopy(old)
    for field in ['offset', 'limit']:
        capability(expected)['input_schema']['properties'][field]['type'] = ['integer', 'null']
        for form in ['catalog', 'samples-detail']:
            original = capability(load('original/' + form + '.json.gz'))
            repaired = capability(load('source/' + form + '.json.gz'))
            assert original['input_schema']['properties'][field]['type'] == 'integer'
            assert repaired['input_schema']['properties'][field]['type'] == ['integer', 'null']
    assert expected == new, 'Unexpected ordinary catalog/output schema change'
    detailed = capability(load('source/samples-detail.json.gz'))
    assert detailed['input_schema']['properties']['offset']['default'] == 0
    assert detailed['input_schema']['properties']['limit']['default'] == 100
    output = detailed['output_schema']
    if '$ref' in output:
        assert output['$ref'] == 'urn:elab:verification-page:1'
        output = new['schemas'][output['$ref']]
    assert all(output['properties'][k]['type'] == 'integer' for k in ['offset', 'next_offset', 'total'])
    assert 'FAIL' in read('source/full-quality-initial.txt').decode()
    assert 'integration-time-limit' in read('source/full-quality-initial.txt').decode()
    assert 'PASS' in read('source/coil-ordinary-agent-race.txt').decode()
    initial_pins = load('source/initial-full-quality-source-pins.json')
    final_pins = load('source/final-quality-source-pins.json')['source_sha256']
    assert len(initial_pins) == len(final_pins) == 650
    assert [n for n in initial_pins if initial_pins[n] != final_pins[n]] == ['internal/conformance/performance_test.go']
    for name in ['workbench/commands.go', 'workbench/verification_capabilities.go',
                 'workbench/verification_capabilities_test.go', 'internal/conformance/performance_test.go']:
        assert hashlib.sha256(read('source/' + Path(name).name + '.txt')).hexdigest() == final_pins[name]
    quality = load('source/final-quality-status.json')
    assert quality['exit_code'] == 0 and quality['elapsed_seconds'] > 0
    assert 'FAIL' not in read('source/final-quality.txt').decode()
    assert 'does not advertise accepted null defaults' in read('source/original-metadata-overlay-failure.txt').decode()
    checkpoint = load('checkpoint.json')
    assert checkpoint['full_mvp_complete'] is False
    assert hashlib.sha256(read('manifest.json')).hexdigest() == checkpoint['manifest_sha256']
    return dict(passed=True, original_pure_cases=29, development_pure_cases=30,
                requests=188, schema_mismatches_repaired=4, full_quality_exit=0,
                clean_replacement_packaging='separate gate')


if __name__ == '__main__':
    print(json.dumps(verify(), indent=2))

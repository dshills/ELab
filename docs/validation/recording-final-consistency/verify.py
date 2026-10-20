"""Read-only replay of saved LED/NPN final-result consistency evidence."""
from pathlib import Path
import gzip
import hashlib
import json
import sys

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def raw(name):
    return gzip.decompress((HERE/'raw'/(name+'.gz')).read_bytes())


def read(name):
    return json.loads(raw(name))


def same_project(a, b):
    excluded = {'project_work', 'revision', 'command_revision', 'last_actor'}
    assert {k:v for k,v in a.items() if k not in excluded} == {k:v for k,v in b.items() if k not in excluded}


def main():
    cp = json.loads((HERE/'checkpoint.json').read_text())['files']
    assert set(cp) == {str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
    for name, digest in cp.items():
        p = HERE/name
        assert not p.is_symlink() and sha(p.read_bytes()) == digest, name
    for name, entry in json.loads((HERE/'manifest.json').read_text()).items():
        stored = (HERE/name).read_bytes(); data = gzip.decompress(stored)
        assert sha(stored) == entry['stored_sha256'] and sha(data) == entry['original_sha256'] and len(data) == entry['original_bytes']
    report = json.loads((HERE/'report.json').read_text())
    receipt_path = ROOT/'research/ui-audit/first-version/terminal-validation-review-checkpoint.json'
    assert sha(receipt_path.read_bytes()) == report['preserved_receipt_sha256']
    receipt = json.loads(receipt_path.read_text())
    for name, digest in report['dependencies'].items():
        assert sha((ROOT/name).read_bytes()) == digest, name
    quality = read('stage/full-quality.json')
    assert quality['exit_code'] == 0 and quality['sources_unchanged'] and quality['source_files'] == 660
    assert quality['source_sha256'] == read('stage/source-before-quality.json')
    for name, digest in quality['source_sha256'].items():
        assert sha((ROOT/name).read_bytes()) == digest, name
    tools = {'tools/baseline-admission/'+n for n in ('admission.go','admission_test.go','main.go')}
    current = {p:h for p,h in quality['source_sha256'].items() if p not in tools}
    previous = receipt['quality']['source_sha256']
    assert len(current) == 657 and len(previous) == 656
    assert set(current)-set(previous) == {'workbench/recording_final_consistency_test.go'} and not set(previous)-set(current)
    assert {p for p in previous if previous[p] != current[p]} == {'workbench/recording_import.go'}
    assert sha(raw('stage/recording_import-original.go.txt')) == previous['workbench/recording_import.go']
    assert sha(raw('stage/recording_import-final.go.txt')) == current['workbench/recording_import.go']
    assert sha(raw('stage/recording_final-final.go.txt')) == current['workbench/recording_final_consistency_test.go']
    old_log = raw('stage/final-regression-original.log').decode()
    assert old_log.count('changed final result was accepted with untouched probe rows/statistics') == 13
    fixed_log = raw('stage/final-focused.log').decode()
    assert 'FAIL' not in fixed_log
    assert fixed_log.count('--- PASS: TestRecordedLEDAndNPNFinalResultsMatchCapturedChannelsAndIDs/example.') == 16
    assert fixed_log.count('--- PASS: TestRecordedNPNFinalResultRetainsPulseEdgeSides/') == 6
    assert 'Ran 65 tests' in raw('stage/full-quality.log').decode()
    assert 'panic: runtime error: invalid memory address' in raw('stage/original-focused.log').decode()
    assert 'panic: runtime error: invalid memory address' in raw('stage/fixed-focused-initial.log').decode()
    assert 'undefined: FormatEditableValue' in raw('stage/edge-test-compile-error.log').decode()
    requests = exits = 0
    for mode in ('original','fixed'):
        result = read(mode+'/report.json'); plan = read(mode+'/plan.json')
        assert result['passed'] and result['mode'] == mode and result['mutations'] == 13 and result['embedded_projects'] == 3
        assert len(result['process_exits']) == 4
        assert plan['agent_sha256'] == (receipt['agent_sha256'] if mode == 'original' else report['development_agent_sha256'])
        for outcome in result['process_exits']:
            assert outcome['exit_code'] == 0
            rows = [json.loads(line) for line in raw(mode+'/'+outcome['name']+'-protocol.jsonl').splitlines()]
            req = [r['request'] for r in rows if 'request' in r]; res = [r['response'] for r in rows if 'response' in r]
            assert len(req) == len(res) == outcome['requests']
            for i, response in enumerate(res):
                assert req[i]['id'] == response['id'] == f'junction-{i+1}'
            shutdown = res[-1]['result']['shutdown']
            assert req[-1]['command'] == 'app.close' and res[-1]['result']['closed']
            assert shutdown == outcome['shutdown'] and shutdown['status'] == 'completed'
            assert all(shutdown[k] for k in ('workers_joined','recovery_finalized','exit_ready'))
            assert not raw(mode+'/'+outcome['name']+'-stderr.txt')
            codes = [r.get('error',{}).get('code') for r in res if 'error' in r]
            expected = [] if mode == 'original' else (['invalid-recording']*13 if outcome['name'] == 'fixed' else ['invalid-document'])
            assert codes == expected, (mode, outcome['name'], codes)
            requests += len(req); exits += 1
        for family in result['families']:
            name = family['name']; prefix = mode+'/'+name+'/'
            good = read('original/'+name+'/good.json')
            assert sha(raw('original/'+name+'/good.json')) == family['good_sha256']
            assert raw(prefix+'good-reexport.json') == raw('original/'+name+'/good.json')
            assert family['intact_reexport_exact'] and family['cases'] == (1 if name == 'led' else 6)
            fields = ['current_a'] if name == 'led' else ['base_current_a','collector_current_a','emitter_current_a','base_resistor_id','load_resistor_id','led_id']
            for field in fields:
                bad_bytes = raw('original/'+name+'/'+field+'.json'); bad = json.loads(bad_bytes)
                outcome = read(prefix+field+'-outcome.json')
                assert sha(bad_bytes) == outcome['malformed_sha256']
                assert good['rows'] == bad['rows'] and good['metadata']['statistics'] == bad['metadata']['statistics']
                assert good['metadata']['graph'] == bad['metadata']['graph']
                kind = 'led' if name == 'led' else 'npn'
                a, b = good['metadata'][kind], bad['metadata'][kind]
                assert {k for k in a if a[k] != b[k]} == {field}
                assert b[field] == ('Other' if field.endswith('_id') else a[field]+.001)
                if mode == 'original':
                    assert outcome['accepted']['metadata'] == bad['metadata']
                else:
                    assert outcome['accepted'] is None
                    same_project(outcome['before'], outcome['after'])
                    assert outcome['after']['project_work']['error']['code'] == 'invalid-recording'
            embedded = read(prefix+'embedded.json'); project_bytes = raw('original/'+name+'/bad-embedded.elab')
            assert sha(project_bytes) == embedded['project_sha256']
            project = json.loads(project_bytes)
            assert any(e['data'] == read('original/'+name+'/'+fields[0]+'.json') for e in project['recordings']['entries'])
            if mode == 'fixed':
                same_project(embedded['before'], embedded['after'])
                assert embedded['after']['project_work']['error']['code'] == 'invalid-document'
            else:
                assert not embedded['after']['document']['dirty']
    assert requests == report['retained_declared_requests'] == 197 and exits == report['normal_declared_exits'] == 8
    runtime = read('stage/development-runtime.json')
    assert not runtime['release_candidate'] and runtime['agent_sha256'] == report['development_agent_sha256']
    for key in ('development_agent','copied_agent'):
        assert sha(Path(runtime[key]).read_bytes()) == runtime['agent_sha256']
    assert sha(Path('/private/tmp/elab-first-version-181-signed-runtime/elab-agent').read_bytes()) == receipt['agent_sha256']
    sys.path.insert(0,str(ROOT/'scripts'))
    from package_review import members
    folder = Path(receipt['review_path']); actual = members(folder); actual.pop('RELEASE_MANIFEST.json')
    assert actual == json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual) == 506
    assert sha((ROOT/receipt['sdk']['archive']).read_bytes()) == receipt['sdk']['sha256']
    assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()) == receipt['canonical']['native_sha256']
    assert report['source_qualified'] and report['replacement_packaging_pending']
    assert not any(report[k] for k in ('solver_changed','new_release','native_input','owner_acceptance','full_mvp_complete'))
    print(json.dumps(dict(original_acceptances=13,fixed_record_refusals=13,embedded_refusals=3,normal_exits=8,requests=197,application_sources=657,full_quality_exit=0)))


if __name__ == '__main__':
    main()

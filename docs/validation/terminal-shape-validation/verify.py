"""Read-only malformed-terminal crash/refusal evidence replay; never starts an app."""
from pathlib import Path
import gzip
import hashlib
import json
import math
import sys

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def raw(relative):
    return gzip.decompress((HERE/'raw'/(relative+'.gz')).read_bytes())


def read(relative):
    return json.loads(raw(relative))


def same_project(before, after):
    excluded = {'project_work', 'revision', 'command_revision', 'last_actor'}
    assert {k:v for k,v in before.items() if k not in excluded} == {k:v for k,v in after.items() if k not in excluded}


def protocol(relative, crash=False):
    rows = [json.loads(line) for line in raw(relative).splitlines()]
    requests = [r['request'] for r in rows if 'request' in r]
    responses = [r['response'] for r in rows if 'response' in r]
    assert len(requests) == len(responses)+(1 if crash else 0)
    for i, response in enumerate(responses):
        assert requests[i]['id'] == response['id'] == f'junction-{i+1}'
    if crash:
        assert requests[-1]['command'] == 'project.load' and requests[-1]['params']['discard_changes']
    else:
        state = responses[-1]['result']; work = state['shutdown']
        assert requests[-1]['command'] == 'app.close' and state['closed'] and work['status'] == 'completed'
        assert all(work[k] for k in ('workers_joined','recovery_finalized','exit_ready'))
    return requests, responses


def main():
    checkpoint = json.loads((HERE/'checkpoint.json').read_text())
    files = {str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
    assert files == set(checkpoint['files'])
    for relative,digest in checkpoint['files'].items():
        p = HERE/relative
        assert not p.is_symlink() and p.resolve().is_relative_to(HERE) and sha(p.read_bytes()) == digest, relative
    for relative,entry in json.loads((HERE/'manifest.json').read_text()).items():
        stored = (HERE/relative).read_bytes(); original = gzip.decompress(stored)
        assert sha(stored) == entry['stored_sha256'] and sha(original) == entry['original_sha256'] and len(original) == entry['original_bytes']
    report = json.loads((HERE/'report.json').read_text())
    for relative,digest in report['dependencies'].items():
        assert sha((ROOT/relative).read_bytes()) == digest, relative
    receipt_path = ROOT/'research/ui-audit/first-version/integration-discovery-review-checkpoint.json'
    assert sha(receipt_path.read_bytes()) == report['preserved_receipt_sha256']
    receipt = json.loads(receipt_path.read_text())
    quality = read('stage/full-quality.json')
    assert quality['command'] == './scripts/check.sh' and quality['exit_code'] == 0 and quality['sources_unchanged']
    assert quality['source_sha256'] == read('stage/source-before-quality.json') and quality['source_files'] == 659
    for relative,digest in quality['source_sha256'].items():
        assert sha((ROOT/relative).read_bytes()) == digest, relative
    old = receipt['quality']['source_sha256']
    tools = {'tools/baseline-admission/'+name for name in ('admission.go','admission_test.go','main.go')}
    current = {p:h for p,h in quality['source_sha256'].items() if p not in tools}
    assert len(old) == 655 and len(current) == 656
    assert set(current)-set(old) == {'workbench/document_terminals_test.go'} and not set(old)-set(current)
    assert {p for p in old if old[p] != current[p]} == {'workbench/persistence.go'}
    assert sha(raw('stage/persistence-original.go.txt')) == old['workbench/persistence.go']
    assert sha(raw('stage/persistence-final.go.txt')) == current['workbench/persistence.go']
    assert sha(raw('stage/document_terminals-final.go.txt')) == current['workbench/document_terminals_test.go']
    log = raw('stage/final-regression-original.log').decode()
    assert log.count('--- FAIL: TestDocumentRejectsMalformedTerminalsBeforeProbeInventory/') == 24
    assert log.count('malformed file must return an error rather than panic') == 24
    fixed_log = raw('stage/fixed-focused.log').decode()
    assert 'FAIL' not in fixed_log and fixed_log.count('--- PASS: TestDocumentRejectsMalformedTerminalsBeforeProbeInventory/') == 24
    assert '--- PASS: TestMalformedTerminalOpenPreservesDirtyProjectAndSavedFile' in fixed_log
    assert '--- PASS: TestRecoveryMalformedTerminalsRemainProblemsWithoutDeletingCapsule' in fixed_log
    assert 'Ran 65 tests' in raw('stage/full-quality.log').decode()
    originals = read('original-final/report.json'); fixed = read('fixed/report.json'); controls = read('import-controls/report.json')
    assert originals['passed'] and originals['fixture_unchanged'] and originals['agent_unchanged']
    assert fixed['passed'] and len(fixed['process_exits']) == 4 and controls['passed']
    assert len(originals['results']) == len(fixed['results']) == 2
    requests = 0
    for label in ('scope-empty','pins-short'):
        prefix = 'original-final/'+label+'/'
        result = read(prefix+'result.json')
        assert result['exit_code'] == 2 and result['reproduced'] and result['expected_eof'] and not result['normal_shutdown']
        req,res = protocol(prefix+label+'-protocol.jsonl',True); requests += len(req)
        assert len(req) == result['requests'] == 6
        stderr = raw(prefix+label+'-stderr.txt').decode()
        assert 'panic: runtime error: index out of range' in stderr and 'workbench.probeInventory' in stderr and 'workbench.decodeDocument' in stderr
        assert ('workbench.validatePinPreferences' in stderr) == (label == 'pins-short')
        original, malformed = read(prefix+'original.elab'),read(prefix+'malformed.elab')
        assert sha(raw(prefix+'original.elab')) == result['original_sha256']
        assert sha(raw(prefix+'malformed.elab')) == result['malformed_sha256']
        source = next(p for p in malformed['graph']['components'] if p['kind'] == 'resistor')
        assert source['terminals'] == ([] if label == 'scope-empty' else ['1'])
        assert ('pinned_readings' in malformed) == (label == 'pins-short')
        before_crash = read(prefix+'before-open.json')
        assert before_crash['document']['dirty'] and next(p['value'] for p in before_crash['components'] if p['kind']=='resistor') == 20000
        fp = 'fixed/'+label+'/'
        repaired = read(fp+'result.json')
        assert repaired['passed'] and repaired['original_sha256'] == result['original_sha256'] and repaired['malformed_sha256'] == result['malformed_sha256']
        before,refused = read(fp+'before.json'),read(fp+'refused.json'); same_project(before,refused)
        assert refused['project_work']['status'] == 'failed' and refused['project_work']['error']['code'] == 'invalid-document'
        assert before['components'] == before_crash['components'] and before['wires'] == before_crash['wires']
        scan_before,scanned = read(fp+'scan-before.json'),read(fp+'scanned.json')
        assert not scanned['recovery']['candidates'] and len(scanned['recovery']['problems']) == 1
        assert 'terminal metadata' in scanned['recovery']['problems'][0]['message']
        for key in ('components','wires','transient','mixed','run_id','electrical_revision','document','undo_entry','redo_entry'):
            assert scan_before.get(key) == scanned.get(key), key
        capsule_name = fp+'recovery/rec-'+('a'*24)+'-0000000001.elab-recovery'
        assert sha(raw(capsule_name)) == repaired['capsule_sha256']
        capsule = read(capsule_name)
        compact = json.dumps(capsule['document'],separators=(',',':')).encode()
        assert sha(compact) == capsule['document_sha256'] and capsule['document'] == malformed
        saved,reopened = read(fp+'surviving-edit.elab'),read(fp+'reopened-surviving.json')
        assert sha(raw(fp+'surviving-edit.elab')) == repaired['saved_sha256']
        assert saved['graph']['components'] == reopened['components'] == before['components']
        assert saved['graph']['wires'] == reopened['wires'] == before['wires'] and not reopened['document']['dirty']
        # Independent startup RC law after the surviving 20 kΩ edit is reopened.
        sample = reopened['transient']['sample']
        assert sample['time_s'] == 0 and sample.get('capacitor_voltage_v',0) == 0
        assert math.isclose(sample['current_a'],5/20000,rel_tol=0,abs_tol=1e-12)
        valid = read(fp+'valid-open.json')
        assert not valid['document']['dirty'] and valid['components'] == original['graph']['components'] and valid['wires'] == original['graph']['wires']
        for outcome in [e for e in fixed['process_exits'] if e['name'].startswith(label)]:
            assert outcome['exit_code'] == 0
            req,res = protocol(fp+outcome['name']+'-protocol.jsonl'); requests += len(req)
            assert len(req) == outcome['requests'] and res[-1]['result']['shutdown'] == outcome['shutdown']
            assert not raw(fp+outcome['name']+'-stderr.txt')
    assert sha(raw('import-controls/malformed-record.json')) == controls['malformed_record_sha256']
    for outcome in controls['outcomes']:
        label = outcome['label']; prefix = 'import-controls/'+label+'/'
        assert outcome['invalid_recording_refused'] and outcome['snapshot_preserved'] and outcome['process']['exit_code'] == 0
        req,res = protocol(prefix+label+'-protocol.jsonl'); requests += len(req)
        assert len(req) == 5 and len([r for r in res if r.get('error',{}).get('code')=='invalid-recording']) == 1
        same_project(read(prefix+'before.json'),read(prefix+'after.json'))
    assert requests == report['retained_declared_requests'] == 64
    # Preserve the initial fixture mistakes without relabeling them app failures.
    assert 'unknown component ID' in raw('stage/original-fixture-id-error.log').decode()
    protocol('original-initial/scope-empty/scope-empty-protocol.jsonl')
    assert 'recovery directory must be private' in raw('stage/fixed-fixture-permission-error.log').decode()
    runtime = read('stage/development-runtime.json')
    assert not runtime['release_candidate'] and sha(Path(runtime['copied_agent']).read_bytes()) == runtime['agent_sha256'] == report['development_agent_sha256']
    assert sha(Path(runtime['development_agent']).read_bytes()) == runtime['agent_sha256']
    assert sha(Path('/private/tmp/elab-first-version-180-signed-runtime/elab-agent').read_bytes()) == receipt['agent_sha256']
    sys.path.insert(0,str(ROOT/'scripts'))
    from package_review import members
    folder = Path(receipt['review_path']); actual = members(folder); actual.pop('RELEASE_MANIFEST.json')
    assert actual == json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual) == 473
    assert sha((ROOT/receipt['sdk']['archive']).read_bytes()) == receipt['sdk']['sha256']
    assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()) == receipt['canonical']['native_sha256']
    assert report['source_qualified'] and report['replacement_packaging_pending']
    assert not any(report[k] for k in ('solver_changed','new_release','native_input','owner_acceptance','full_mvp_complete'))
    print(json.dumps(dict(original_panic_regressions=24,owned_original_crashes=2,
                          repaired_file_cases=2,recovery_problem_cases=2,
                          noncrashing_record_import_controls=2,normal_declared_exits=6,
                          retained_requests=64,application_sources=656,full_quality_exit=0)))


if __name__ == '__main__':
    main()

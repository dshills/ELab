"""Read-only verification for the exact signed /175 private review."""
from pathlib import Path
import gzip
import hashlib
import json
import plistlib
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'scripts'))
from package_review import members, verify_archive
from wiring_replay import validate_release


def read(path):
    data = path.read_bytes()
    return json.loads(gzip.decompress(data) if path.suffix == '.gz' else data)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()



def verify_paging(folder, plan):
    report = read(folder/'report.json.gz')
    assert report['passed'] and report['behavior_passed'] and report['source_file_unchanged']
    assert report['process_exit_code'] == 0 and len(report['cases']) == 30 and report['requests'] == 95
    fixture_bytes = gzip.decompress((folder/'recorded-rc.elab.gz').read_bytes())
    assert hashlib.sha256(fixture_bytes).hexdigest() == plan['fixture_sha256']
    recorded = json.loads(fixture_bytes)['recordings']['entries'][0]['data']
    assert len(recorded['rows']) == recorded['metadata']['sample_count'] == 209
    transcript = [json.loads(line) for line in gzip.decompress((folder/'transcript.jsonl.gz').read_bytes()).splitlines()]
    assert len(transcript) == 95
    baseline = read(folder/'baseline.json.gz')
    assert transcript[2]['response']['result'] == baseline == read(folder/'final.json.gz')
    for i, case in enumerate(report['cases']):
        before, query, after = transcript[3+3*i:6+3*i]
        assert json.loads(before['request_json'])['command'] == json.loads(after['request_json'])['command'] == 'project.inspect'
        assert before['response']['result'] == after['response']['result'] == baseline
        request, response = json.loads(query['request_json']), query['response']
        assert request['command'] == case['command']
        assert response.get('error', {}).get('code') == case['error']
        if case['error'] == 'invalid-json':
            assert response['id'] == '' and response['revision'] == 0 and '"limit":1,"limit":2' in query['request_json']
        else:
            assert response['id'] == request['id']
        if case['error'] is None and request['command'] == 'operation.samples':
            offset, limit = request['params'].get('offset') or 0, request['params'].get('limit') or 100
            rows = recorded['rows'][offset:offset+limit]
            assert response['result'] == dict(id=baseline['recording']['id'], offset=offset, next_offset=offset+len(rows), total=209, probes=recorded['metadata']['probes'], rows=rows)
    for form in ['catalog', 'samples-detail']:
        cap = next(c for c in read(folder/(form+'.json.gz'))['capabilities'] if c['id']=='operation.samples')
        assert all(cap['input_schema']['properties'][f]['type'] == ['integer','null'] for f in ['offset','limit'])
    closed = transcript[-1]['response']['result']
    assert closed == read(folder/'close-response.json.gz') and closed['closed']
    assert closed['shutdown']['status']=='completed' and closed['shutdown']['workers_joined'] and closed['shutdown']['exit_ready']


def main():
    folder = Path(__file__).resolve().parent
    checkpoint = read(folder/'checkpoint.json')
    actual = {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
    assert actual == checkpoint['evidence_sha256']
    receipt_path = ROOT/'research/ui-audit/first-version/operation-page-review-checkpoint.json'
    receipt = read(receipt_path)
    assert sha(receipt_path) == checkpoint['receipt_sha256']
    identity = validate_release(ROOT, receipt_path)
    assert identity['release'] == '0.158.0 /175' and identity['source_commit'] == receipt['source_commit'] == 'cc3cb7b142350767a6fc383f85b73c3183e4e85b'
    assert identity['runtime_inputs_verified'] == receipt['runtime_source_input_count'] == 358
    assert identity['production_source_files_verified'] == 72
    assert identity['manifest_file_entries_verified'] == receipt['manifest_file_entries']
    assert identity['manifest_sha256'] == checkpoint['manifest_sha256']
    review = Path(receipt['review_path'])
    assert (review/'Evidence/Acceptance.json').read_bytes() == receipt_path.read_bytes()
    info = plistlib.loads((review/'ELab.app/Contents/Info.plist').read_bytes())
    assert info['CFBundleIdentifier'] == 'com.dshills.elab' and info['CFBundleVersion'] == '175' and info['CFBundleShortVersionString'] == '0.158.0'
    manifest = read(review/'ELab.app/Contents/Resources/APP_MANIFEST.json')
    assert manifest['source_binary_sha256'] == receipt['source_native_sha256'] == receipt['signature']['source_input_sha256']
    assert manifest['signed_identity'] == '-' and sha(review/'ELab.app/Contents/MacOS/ELab') == receipt['signature']['packaged_sha256']
    subprocess.run(['/usr/bin/codesign', '--verify', '--deep', '--strict', str(review/'ELab.app')], capture_output=True, check=True)
    assert read(folder/'Signature-admission.json') == receipt['signature']['exact_artifact_admission']
    assert len(read(folder/'Signature-admission.json')['refusals']) == 5
    proof = ROOT/'docs/validation/operation-page-contract-174'
    status = read(proof/'source/final-quality-status.json')
    quality = dict(passed=status['exit_code']==0, environment={'GOPROXY':'off','GOWORK':'off'}, source_sha256=read(proof/'source/final-quality-source-pins.json')['source_sha256'])
    assert quality['passed'] and quality['environment'] == {'GOPROXY': 'off', 'GOWORK': 'off'}
    assert all(sha(ROOT/name) == digest for name, digest in quality['source_sha256'].items())
    proof = ROOT/'docs/validation/operation-page-contract-174'
    source_checkpoint = read(proof/'checkpoint.json')
    assert sha(proof/'checkpoint.json') == receipt['source_proof_checkpoint_sha256'] == checkpoint['source_proof_checkpoint_sha256']
    assert sha(proof/'manifest.json') == source_checkpoint['manifest_sha256']
    assert all(sha(proof/name) == entry['sha256'] for name, entry in read(proof/'manifest.json')['files'].items())
    verify_paging(folder/'paging', read(folder/'Paging-plan.json'))
    rr = read(folder/'Runtime.json')
    ar = read(folder/'acceptance/report.json')
    assert rr['passed'] and ar['passed'] and rr['shared_geometry_unchanged'] and ar['shared_geometry_unchanged']
    assert ar['discovery_commands']['commands'] == ar['discovery_commands']['semantic_annotations'] == 164
    assert ar['discovery_commands']['appearance_job_checks']['passed']
    assert ar['sweep_capture_commands']['passed'] and ar['sweep_capture_commands']['points_per_capture'] == [51,51,7,9]
    assert ar['sweep_capture_commands']['invalid_settings_refusals'] == 8
    assert ar['sweep_capture_commands']['exact_retained_capture_document_transition']
    assert isinstance(ar['elapsed_s'], (int, float)) and ar['elapsed_s'] > 0
    assert ar['native_host_closed_normally']
    public = read(folder/'Public-report.json.gz')
    assert public['passed'] and len(public['adapters']) == 4
    exits = read(folder/'Process-exits.json.gz')
    shutdowns = read(folder/'Shutdown-responses.json.gz')
    assert len(exits) == 102 and len(shutdowns) == 112
    assert all(row['exit_code'] == 0 and not row['graceful_wait_timed_out'] and not row.get('close_error') and not row.get('wait_error') for row in exits.values())
    for row in shutdowns.values():
        state = row['response']['result']
        assert state['closed'] and state['shutdown']['status'] == 'completed' and state['shutdown']['workers_joined'] and state['shutdown']['exit_ready']
    archive = read(folder/'archive/package-report.json.gz')
    zip_path = ROOT/'build/ELab-0.158.0-175-review.zip'
    assert archive['passed'] and sha(zip_path) == archive['archive_sha256'] == checkpoint['archive_sha256']
    assert archive['files_verified'] == receipt['manifest_file_entries']+1
    verify_archive(zip_path, review.name, members(review))
    checksum = (ROOT/'build/ELab-0.158.0-175-review.zip.sha256').read_text()
    assert checksum == archive['archive_sha256']+'  '+zip_path.name+'\n'
    assert read(folder/'archive/example-report.json.gz')['passed'] and read(folder/'archive/signature-report.json.gz')['passed']
    preserved = read(folder/'preservation.json')
    for name, expected in preserved['preserved_bundles'].items():
        older = ROOT/'build'/('ELab-'+name+'-review')
        files = members(older)
        assert files.pop('RELEASE_MANIFEST.json')['sha256'] == expected['manifest_sha256']
        assert files == read(older/'RELEASE_MANIFEST.json')['files'] and len(files) == expected['files_verified']
    assert len(preserved['preserved_bundles']) == 33
    for path, key in [('ELab-0.130.0-147-review', 'original_147_candidate'),
                      ('ELab-0.147.0-164-review', 'failed_164_candidate'),
                      ('ELab-0.154.0-171-unsigned-candidate', 'unsigned_171_candidate'),
                      ('ELab-0.155.0-172-failed-candidate', 'failed_172_candidate')]:
        candidate = ROOT/'build'/path
        files = members(candidate)
        assert files.pop('CANDIDATE_MANIFEST.json')['sha256'] == preserved['preserved_'+key]['manifest_sha256']
        assert files == read(candidate/'CANDIDATE_MANIFEST.json')['files'] and len(files) == preserved['preserved_'+key]['files_verified']
    assert sha(ROOT/'build/ELab.app/Contents/MacOS/ELab') == receipt['canonical']['native_sha256']
    assert not receipt['canonical_promoted'] and receipt['owner_acceptance'].startswith('Pending:')
    assert receipt['p0_counts'] == dict(implemented=17, partial=10)
    print(dict(passed=True, release=identity['release'], manifest_entries=receipt['manifest_file_entries'],
               archive_files=archive['files_verified'], strict_signature_valid=True, preserved_reviews=33,
               owner_acceptance=False, canonical_promoted=False))


if __name__ == '__main__':
    main()

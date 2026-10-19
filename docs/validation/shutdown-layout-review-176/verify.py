"""Read-only verification for the exact signed /176 private review."""
from pathlib import Path
import gzip
import hashlib
import json
import plistlib
import subprocess
import sys
import struct

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



def jpeg_size(path):
    data=path.read_bytes()
    assert data[:2]==b'\xff\xd8'
    at=2
    while at<len(data):
        assert data[at]==255
        while data[at]==255:at+=1
        marker=data[at];at+=1
        if marker in [216,217] or 208<=marker<=215:continue
        size=struct.unpack('>H',data[at:at+2])[0]
        assert size>=2 and at+size<=len(data)
        if marker in [192,193,194]:
            height,width=struct.unpack('>HH',data[at+3:at+7]);return width,height
        at+=size
    raise AssertionError('JPEG frame dimensions missing')


def verify_native(folder, receipt):
    report=read(folder/'report.json')
    assert report==receipt['native_shutdown']
    assert report['status']=='completed' and report['shared_user_acknowledgement_observed'] and not report['fallback_acknowledgement']
    assert report['native_sha256']==receipt['native_sha256'] and report['agent_sha256']==receipt['agent_sha256']
    assert report['source_commit']==receipt['source_commit']
    plan=read(folder/'plan.json.gz')
    assert hashlib.sha256(gzip.decompress((folder/'launcher.swift.gz').read_bytes())).hexdigest()==plan['launcher_source_sha256']
    assert sha(folder.parent/'native_shutdown.py.txt')==plan['controller_sha256']
    assert hashlib.sha256(gzip.decompress((folder.parent/'runtime.sb.gz').read_bytes())).hexdigest()==plan['policy_sha256']
    assert report['open_exit_code']==report['connected_agent_exit_code']==0 and report['native_exit_status'] is None
    assert all(report[k] for k in ['owned_process_absent','endpoint_removed','protected_files_unchanged','shared_geometry_unchanged','review_unchanged','copied_bundle_unchanged','canonical_unchanged'])
    assert report['shared_geometry_before']==report['shared_geometry_after']
    assert all(v['denied'] and v['errno'] in [1,13] for v in report['controls'].values())
    baseline,failed,before,after=[read(folder/(n+'.json.gz')) for n in ['baseline','failed-shutdown','before-ack','after-ack']]
    assert before==failed
    assert before['closed'] and not before['shutdown']['exit_ready'] and after['closed'] and after['shutdown']['exit_ready']
    assert after['shutdown']['status']=='failed' and after['shutdown']['workers_joined']
    assert after['shutdown']['error']==before['shutdown']['error']
    assert all(before['shutdown'][k]==after['shutdown'][k] for k in ['recovery_error','appearance_error'])
    assert len(before['shutdown']['recovery_error'])>=700 and len(before['shutdown']['appearance_error'])>=3500
    assert after['last_actor']=='user' and after['command_revision']==before['command_revision']+1
    for key in ['components','wires','nets','result','transient','mixed','paused','run_id','electrical_revision','document','undo_entry','redo_entry']:
        assert after.get(key)==before.get(key),key
    assert all(failed[k]==baseline[k] for k in ['components','wires','result','document'])
    for name,key in [('Manual-LED.elab','manual_sha256'),('appearance.json','appearance_sha256'),('Protected-fault.txt','protected_sha256')]:
        assert hashlib.sha256(gzip.decompress((folder/(name+'.gz')).read_bytes())).hexdigest()==report[key]
    ui=read(folder/'cua-actions.json.gz')
    assert ui['control_plane_acknowledgement_count']==0 and ui['cua_pointer_acknowledgement_count']==1
    assert ui['actions'][-1]['api']=='click' and ui['actions'][-1]['coordinates']==[120,705]
    images=read(folder/'screenshots.json.gz')
    for name,size in [('native-initial',(1053,802)),('native-minimum',(1053,762)),('native-scrolled',(1053,762))]:
        im=folder/images[name]['file']
        assert sha(im)==images[name]['sha256'] and jpeg_size(im)==size
    assert images['native-minimum']['sha256']!=images['native-scrolled']['sha256']
    for name in ['native-initial','native-minimum','native-scrolled']:
        text=gzip.decompress((folder/(name+'.ax.txt.gz')).read_bytes()).decode()
        assert 'button Close window anyway' in text and 'scroll area node:c2h1dGRvd24tZXJyb3Itc2Nyb2xs' in text
    ref=read(folder/'raw-evidence.json')['transcript']
    path=Path(receipt['review_path'])/'Evidence/Native-shutdown/transcript.jsonl.gz'
    assert ref['path']==path.relative_to(ROOT).as_posix() and not path.is_symlink()
    assert path.stat().st_size==ref['bytes'] and sha(path)==ref['sha256']
    requests=polls=unchanged=changed=0
    with gzip.open(path,'rt') as stream:
        for line in stream:
            row=json.loads(line);request=json.loads(row['request_json']);response=row['response']
            assert response['version']==1 and response['id']==request['id'] and 'error' not in response
            assert request['command']!='app.close.acknowledge'
            requests+=1
            if request['command']=='events.poll':
                polls+=1;event=response['result']
                assert event['closed'] and event['coalesced'] and not event['timed_out']
                if event['changed']:
                    changed+=1;assert event['snapshot']==after
                else:
                    unchanged+=1;assert event['snapshot']==before and event['revision']==before['revision']
    assert (requests,polls,unchanged,changed)==(5857,5844,5843,1)
    assert not report.get('preparatory_guard_retries')


def main():
    folder = Path(__file__).resolve().parent
    checkpoint = read(folder/'checkpoint.json')
    actual = {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
    assert actual == checkpoint['evidence_sha256']
    receipt_path = ROOT/'research/ui-audit/first-version/shutdown-layout-review-checkpoint.json'
    receipt = read(receipt_path)
    assert sha(receipt_path) == checkpoint['receipt_sha256']
    identity = validate_release(ROOT, receipt_path)
    assert identity['release'] == '0.159.0 /176' and identity['source_commit'] == receipt['source_commit'] == 'f08e77ecde66a08d2799428e440a8f4fef15b0f7'
    assert identity['runtime_inputs_verified'] == receipt['runtime_source_input_count'] == 358
    assert identity['production_source_files_verified'] == 72
    assert identity['manifest_file_entries_verified'] == receipt['manifest_file_entries']
    assert identity['manifest_sha256'] == checkpoint['manifest_sha256']
    review = Path(receipt['review_path'])
    assert (review/'Evidence/Acceptance.json').read_bytes() == receipt_path.read_bytes()
    info = plistlib.loads((review/'ELab.app/Contents/Info.plist').read_bytes())
    assert info['CFBundleIdentifier'] == 'com.dshills.elab' and info['CFBundleVersion'] == '176' and info['CFBundleShortVersionString'] == '0.159.0'
    manifest = read(review/'ELab.app/Contents/Resources/APP_MANIFEST.json')
    assert manifest['source_binary_sha256'] == receipt['source_native_sha256'] == receipt['signature']['source_input_sha256']
    assert manifest['signed_identity'] == '-' and sha(review/'ELab.app/Contents/MacOS/ELab') == receipt['signature']['packaged_sha256']
    subprocess.run(['/usr/bin/codesign', '--verify', '--deep', '--strict', str(review/'ELab.app')], capture_output=True, check=True)
    assert read(folder/'Signature-admission.json') == receipt['signature']['exact_artifact_admission']
    assert len(read(folder/'Signature-admission.json')['refusals']) == 5
    proof = ROOT/'docs/validation/shutdown-error-layout'
    status = read(proof/'quality/status.json')
    quality = dict(passed=status['exit_code']==0, environment={'GOPROXY':'off','GOWORK':'off'}, source_sha256=read(proof/'quality/source-pins.json')['source_sha256'])
    assert quality['passed'] and quality['environment'] == {'GOPROXY': 'off', 'GOWORK': 'off'}
    assert all(sha(ROOT/name) == digest for name, digest in quality['source_sha256'].items())
    proof = ROOT/'docs/validation/shutdown-error-layout'
    source_checkpoint = read(proof/'checkpoint.json')
    assert sha(proof/'checkpoint.json') == receipt['source_proof_checkpoint_sha256'] == checkpoint['source_proof_checkpoint_sha256']
    assert sha(proof/'manifest.json') == source_checkpoint['manifest_sha256']
    assert all(sha(proof/name) == entry['sha256'] for name, entry in read(proof/'manifest.json')['files'].items())
    verify_paging(folder/'paging', read(folder/'Paging-plan.json'))
    verify_native(folder/'native', receipt)
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
    zip_path = ROOT/'build/ELab-0.159.0-176-review.zip'
    assert archive['passed'] and sha(zip_path) == archive['archive_sha256'] == checkpoint['archive_sha256']
    assert archive['files_verified'] == receipt['manifest_file_entries']+1
    verify_archive(zip_path, review.name, members(review))
    checksum = (ROOT/'build/ELab-0.159.0-176-review.zip.sha256').read_text()
    assert checksum == archive['archive_sha256']+'  '+zip_path.name+'\n'
    assert read(folder/'archive/example-report.json.gz')['passed'] and read(folder/'archive/signature-report.json.gz')['passed']
    preserved = read(folder/'preservation.json')
    for name, expected in preserved['preserved_bundles'].items():
        older = ROOT/'build'/('ELab-'+name+'-review')
        files = members(older)
        assert files.pop('RELEASE_MANIFEST.json')['sha256'] == expected['manifest_sha256']
        assert files == read(older/'RELEASE_MANIFEST.json')['files'] and len(files) == expected['files_verified']
    assert len(preserved['preserved_bundles']) == 34
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
               archive_files=archive['files_verified'], strict_signature_valid=True, preserved_reviews=34,
               owner_acceptance=False, canonical_promoted=False))


if __name__ == '__main__':
    main()

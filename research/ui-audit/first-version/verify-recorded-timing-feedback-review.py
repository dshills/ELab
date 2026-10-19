"""Read-only verification of the /165 review and its retained numerical evidence."""
import gzip
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import stat
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'scripts'))
from wiring_replay import validate_release


def read(path):
    return json.loads(gzip.decompress(path.read_bytes()) if path.suffix == '.gz' else path.read_text())


def verify_readme(receipt, text):
    advertised = re.findall(r'\*\*Private review ready: ([0-9.]+) /([0-9]+)\.\*\*', text)
    assert advertised == [(receipt['version'], receipt['build'])], 'README advertises a different private review'
    review = Path(receipt['review_path']).relative_to(ROOT).as_posix()
    assert ']('+review+'/ELab.app)' in text, 'README app link differs from receipt'
    assert '](research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json)' in text
    blocks = [block for block in re.findall(r'```sh\n(.*?)```', text, re.S) if 'scripts/package-app.py' in block]
    assert len(blocks) == 1
    assert 'python3 -B scripts/prepare-vellum.py' in blocks[0], 'Packaging must resolve and verify the current SDK'
    assert '"$ELAB_SDK_CACHE/scripts/package-app.py"' in blocks[0]
    assert 'build/.vellum-sdk-' not in blocks[0], 'Packaging must not select an obsolete cache'


def main():
    receipt_path = ROOT/'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json'
    identity = validate_release(ROOT,receipt_path)
    receipt = read(receipt_path)
    verify_readme(receipt, (ROOT/'README.md').read_text())
    review = Path(receipt['review_path'])
    assert receipt_path.read_bytes() == (review/'Evidence/Acceptance.json').read_bytes()
    assert receipt['source_commit'] == '0e75875abe07d0c172902754aa8acec6062df7dc'
    assert receipt['quality']['passed'] and receipt['quality']['exit_code'] == 0
    for relative,expected in receipt['quality']['source_sha256'].items():
        assert hashlib.sha256((ROOT/relative).read_bytes()).hexdigest() == expected
    runtime = read(ROOT/'docs/validation/first-version-165-runtime.json')
    assert runtime == read(review/'Evidence/Runtime.json')
    assert runtime['passed'] and runtime['copied_bundle_unchanged'] and runtime['agent_unchanged'] and runtime['controller_unchanged']
    assert runtime['controls']['allowed_read']
    assert all(item['denied'] and item['errno'] in [1,13] for category in ['files','network'] for item in runtime['controls'][category].values())
    assert {stage['name'] for stage in runtime['stages']} == {'native-smoke','public-suites'}
    assert all(stage['status']=='completed' and stage['exit_code']==0 for stage in runtime['stages'])
    public = read(ROOT/'docs/validation/first-version-165-public.json.gz')
    assert public == read(review/'Evidence/Public-report.json.gz') and public['passed']
    assert [adapter['name'] for adapter in public['adapters']] == ['in-process','headless-stdio','headless-socket','native-socket']
    assert receipt['acceptance'] == read(ROOT/'docs/validation/first-version-165-acceptance.json')
    assert receipt['acceptance']['passed'] and receipt['acceptance']['native_host_closed_normally']
    proof = ROOT/'docs/validation/copied-mosfet-precision-165'
    for folder,packaged in [(proof,review/'Evidence/Copied-MOSFET-precision-165'),(ROOT/'docs/validation/copied-rc-edge-165',review/'Evidence/Copied-RC-edge-165'),(ROOT/'docs/validation/copied-recorded-timing-165',review/'Evidence/Copied-recorded-timing-165')]:
        names = {str(path.relative_to(folder)) for path in folder.rglob('*') if path.is_file()}
        assert names == {str(path.relative_to(packaged)) for path in packaged.rglob('*') if path.is_file()}
        for name in names:
            assert (folder/name).read_bytes() == (packaged/name).read_bytes(),name
    spec = importlib.util.spec_from_file_location('precision',proof/'precision_reproduce.py')
    precision = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(precision)
    spec = importlib.util.spec_from_file_location('parity',ROOT/'scripts/first-version-parity.py')
    parity = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parity)
    timingproof = ROOT/'docs/validation/copied-recorded-timing-165'
    timing = read(timingproof/'report.json')
    assert timing['passed'] and timing['stage_matches']==8 and timing['native_exit_code']==timing['final_native_exit_code']==0
    assert timing['fresh_agent_reopens']==2 and timing['capture_metadata_rows_preserved']
    timing_results = {surface:read(timingproof/surface/'report.json.gz') for surface in ['native','headless']}
    protected = ['components','wires','nets','document','run_id','electrical_revision','transient','mixed','npn','result','scope','undo_entry','redo_entry']
    expected_names = {'baseline','accepted-field','stale-page-refused','next-page','invalid-field-refused','corrected-field','corrected-next','previous-page'}
    for surface,case in timing_results.items():
        assert case['passed'] and case['original_capture_unchanged'] and set(case['states'])==expected_names
        states = case['states']
        for name,state in states.items():
            for key in protected:
                assert state.get(key)==states['baseline'].get(key),(surface,name,key)
        assert states['stale-page-refused']==states['accepted-field'] and states['invalid-field-refused']==states['next-page']
        assert states['next-page']['command_revision']==states['baseline']['command_revision']+2
        for name in ['accepted-field','corrected-field','previous-page']:
            assert states[name]['recording']['timing']['settings'].get('offset',0)==0
        for name in ['next-page','corrected-next']:
            assert states[name]['recording']['timing']['settings']['offset']==8
        assert states['accepted-field']['recording']['timing']['settings']['level']=='3 V'
        assert states['previous-page']['recording']['timing']['settings']['level']=='2 V'
        rows = [row for page in case['pages'] for row in page['rows']]
        path = timingproof/surface/'Timing capture.elab.gz'
        assert hashlib.sha256(gzip.decompress(path.read_bytes())).hexdigest()==case['saved_sha256']
        document = read(path)
        assert document['recordings']['entries'][0]['data']==dict(metadata=case['capture']['metadata'],rows=rows)
        assert case['reopened']['paused'] and not case['reopened']['document']['dirty'] and case['reopened']['document']['recording_count']==1
    for name in expected_names:
        a,b = [timing_results[surface]['states'][name] for surface in ['native','headless']]
        for key in ['components','wires','nets','paused','transient_settings']:
            assert a[key]==b[key],(name,key)
        for key in ['result','transient','mixed','npn','scope']:
            parity.compare(a.get(key),b.get(key),name+'.'+key)
        parity.compare(a['recording']['timing'],b['recording']['timing'],name+'.timing')
    assert [row for page in timing_results['native']['pages'] for row in page['rows']]==[row for page in timing_results['headless']['pages'] for row in page['rows']]
    results = {surface:read(proof/surface/'report.json.gz') for surface in ['native','headless']}
    for surface,report in results.items():
        assert report['passed'] and report['agent_sha256'] == receipt['agent_sha256']
        for kind,case in report['cases'].items():
            traces = {trace['probe']['id']:trace['points'] for trace in case['live']['scope']['traces']}
            count = len(traces[precision.PROBES[0]])
            assert count == 505 and all(len(traces[p])==count for p in precision.PROBES)
            rows = []
            for index in range(count):
                at = traces[precision.PROBES[0]][index]['time_s']
                assert all(traces[p][index]['time_s']==at for p in precision.PROBES)
                rows.append(dict(time_s=at,values=[traces[p][index]['value'] for p in precision.PROBES]))
            assert precision.check_rows(rows,kind) == case['live_check'] and case['live_check']['passed']
            captured = [row for page in case['pages'] for row in page['rows']]
            assert precision.check_rows(captured,kind) == case['captured_check'] and case['captured_check']['passed']
            model = case['capture']['metadata'][kind+'_model']
            assert model['threshold_v'] == (1.5 if kind=='nmos' else -1.5)
            assert model['beta_a_per_v2']==.1 and model['lambda_per_v']==.02
            assert {key:model['body_diode'][key] for key in ['is_a','n','rs_ohm','temperature_c']} == dict(is_a=1e-9,n=1.7,rs_ohm=1,temperature_c=27)
            path = proof/surface/(kind+'.elab.gz')
            assert hashlib.sha256(gzip.decompress(path.read_bytes())).hexdigest()==case['saved_sha256']
            document = read(path)
            assert document['graph']['components']==case['live']['components'] and document['graph']['wires']==case['live']['wires']
            assert document['recordings']['entries'][0]['data'] == dict(metadata=case['capture']['metadata'],rows=captured)
            assert case['reopened']['paused'] and not case['reopened']['document']['dirty']
    comparison = read(proof/'report.json')
    exclusions = {'operation_id','created_at','input_revision','electrical_revision','document_generation'}
    assert set(comparison['capture_metadata_provenance_exclusions']) == exclusions
    assert comparison['passed'] and comparison['native_exit_code']==comparison['final_native_exit_code']==0 and comparison['fresh_agent_reopens']==4
    for kind in ['nmos','pmos']:
        a,b = [results[surface]['cases'][kind] for surface in ['native','headless']]
        for key in ['components','wires','nets','paused','transient_settings']:
            assert a['live'][key]==b['live'][key]
        for key in ['mixed','transient','scope','npn','result']:
            parity.compare(a['live'].get(key),b['live'].get(key),kind+'.'+key)
        assert [row for page in a['pages'] for row in page['rows']]==[row for page in b['pages'] for row in page['rows']]
        parity.compare({k:v for k,v in a['capture']['metadata'].items() if k not in exclusions},{k:v for k,v in b['capture']['metadata'].items() if k not in exclusions})
    edge = read(ROOT/'docs/validation/copied-rc-edge-165/report.json')
    assert edge['passed'] and edge['native_exit_code']==edge['final_native_exit_code']==0
    for surface in ['native','headless']:
        data = read(ROOT/'docs/validation/copied-rc-edge-165'/surface/'report.json.gz')
        assert data['passed'] and not data['errors'] and data['samples']==23
        assert data['actual_scope_edge_current_a']==data['actual_csv_edge_current_a']==data['solver_edge_current_a']==data['expected_edge_current_a']
    nativeproof = ROOT/'docs/validation/native-recorded-timing-165'
    packaged_native = review/'Evidence/Native-recorded-timing-165'
    ncp = read(nativeproof/'checkpoint.json')
    assert ncp == receipt['recorded_timing']['native_input'] and ncp['passed']
    assert ncp['native_sha256']==receipt['native_sha256'] and ncp['agent_sha256']==receipt['agent_sha256']
    native_names={str(q.relative_to(nativeproof)) for q in nativeproof.rglob('*') if q.is_file()}
    assert native_names==set(ncp['evidence_sha256'])|{'checkpoint.json'}
    assert native_names=={str(q.relative_to(packaged_native)) for q in packaged_native.rglob('*') if q.is_file()}
    for name in native_names:
        assert (nativeproof/name).read_bytes()==(packaged_native/name).read_bytes()
        if name!='checkpoint.json':assert hashlib.sha256((nativeproof/name).read_bytes()).hexdigest()==ncp['evidence_sha256'][name]
    nr=read(nativeproof/'report.json.gz');cleanup=read(nativeproof/'cleanup.json');reopens=read(nativeproof/'fresh-reopens.json.gz')
    assert nr['passed'] and nr['normal_closure'] and nr['final_host_exit_code']==0 and nr['capture_metadata_pages_unchanged']
    assert cleanup['passed'] and cleanup['original_process_absent'] and cleanup['no_test_bundle_processes'] and cleanup['restored_from_verified_backups']
    assert reopens['passed'] and reopens['fresh_agent_reopens']==2
    original_rows=[row for page in nr['original_pages'] for row in page['rows']];assert len(original_rows)==521
    native_protected=protected+['paused']
    stages={s['name']:s for s in nr['stages']};assert len(stages)==17
    expected={
        'visible-baseline':('2.5 V','both',0), 'private-3V':('2.5 V','both',0),
        'AX-3V-rising':('3 V','rising',0), 'pointer-private-4V':('3 V','rising',0),
        'pointer-4V-falling':('4 V','falling',0), 'private-invalid':('4 V','falling',0),
        'invalid-direction-refused':('4 V','falling',0), 'private-correction':('4 V','falling',0),
        'corrected-3.5V-rising':('3.5 V','rising',0), 'paging-both-directions':('3.5 V','both',0),
        'paging-private-3V':('3.5 V','both',0), 'typed-threshold-next':('3 V','both',8),
        'paging-private-invalid':('3 V','both',8), 'invalid-previous-refused':('3 V','both',8),
        'paging-private-correction':('3 V','both',8), 'corrected-previous':('3.5 V','both',0), 'finish':('3.5 V','both',0)}
    for name,stage in stages.items():
        assert stage['passed'];state=stage['state'];settings=state['recording']['timing']['settings']
        assert (settings['level'],settings['direction'],settings.get('offset',0))==expected[name]
        for key in native_protected:assert state.get(key)==nr['baseline'].get(key),(name,key)
        parity.compare(state['recording']['timing'],stage['reference']['recording']['timing'])
    for label in ['Native','Agent']:
        document=read(nativeproof/(label+' timing.elab.gz'));data=document['recordings']['entries'][0]['data']
        assert reopens['files'][label]['sha256']==hashlib.sha256(gzip.decompress((nativeproof/(label+' timing.elab.gz')).read_bytes())).hexdigest()
        assert reopens['files'][label]['capture']['metadata']==data['metadata']
        assert [row for page in reopens['files'][label]['pages'] for row in page['rows']]==data['rows']==original_rows
        assert reopens['files'][label]['state']['paused'] and not reopens['files'][label]['state']['document']['dirty']
    assert not receipt['canonical_promoted'] and len(receipt['preserved_bundles'])==24
    assert receipt['preserved_failed_164_candidate']==dict(files_verified=4,manifest_sha256='0092d861e829d66af5c68b249fab6dc28688e5397487af3b5c59eed1ef8331ed')
    def verify_preserved(folder,manifest_name,expected):
        manifest_path=folder/manifest_name;manifest=read(manifest_path)
        assert hashlib.sha256(manifest_path.read_bytes()).hexdigest()==expected['manifest_sha256']
        assert len(manifest['files'])==expected['files_verified']
        paths=list(folder.rglob('*'));assert not any(q.is_symlink() or not(q.is_file() or q.is_dir()) for q in paths)
        assert {str(q.relative_to(folder)) for q in paths if q.is_file()}==set(manifest['files'])|{manifest_name}
        for name,entry in manifest['files'].items():
            q=folder/name
            assert hashlib.sha256(q.read_bytes()).hexdigest()==entry['sha256'] and oct(stat.S_IMODE(q.stat().st_mode))==entry['mode'],name
    for name,expected in receipt['preserved_bundles'].items():
        verify_preserved(ROOT/'build'/('ELab-'+name+'-review'),'RELEASE_MANIFEST.json',expected)
    verify_preserved(ROOT/'build/ELab-0.130.0-147-review','CANDIDATE_MANIFEST.json',receipt['preserved_original_147_candidate'])
    verify_preserved(ROOT/'build/ELab-0.147.0-164-review','CANDIDATE_MANIFEST.json',receipt['preserved_failed_164_candidate'])
    assert hashlib.sha256((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest()==receipt['canonical']['native_sha256']
    for folder in ['recorded-timing-feedback','recorded-timing-direction']:
        source_proof=ROOT/'docs/validation'/folder;checkpoint=read(source_proof/'checkpoint.json')
        for name,expected in checkpoint['evidence_sha256'].items():assert hashlib.sha256((source_proof/name).read_bytes()).hexdigest()==expected,(folder,name)
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)',(review/'README.md').read_text()):
        assert (review/target.strip('<>')).exists(),target
    assert not receipt['canonical_promoted'] and receipt['owner_acceptance'].startswith('Pending:')
    print(json.dumps(dict(passed=True,release=identity['release'],source_commit=identity['source_commit'],manifest_entries=identity['manifest_file_entries_verified'],runtime_inputs=identity['runtime_inputs_verified'],native_source_files=identity['production_source_files_verified'],timing_stages=8,timing_fresh_reopens=2,native_input_stages=17,native_capture_rows=521,native_fresh_reopens=2,precision_curves=8,rows_per_curve=505,precision_fresh_reopens=4,owner_acceptance=False,canonical_promoted=False)))


if __name__ == '__main__':
    main()

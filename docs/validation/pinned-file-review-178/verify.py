"""Read-only exact /178 artifact and scoped electrical regression replay."""
from pathlib import Path
import gzip
import hashlib
import importlib.util
import json
import math
import sys
from importlib.machinery import SourceFileLoader

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')

HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]
sys.path.insert(0,str(ROOT/'scripts'))
from package_review import members, verify_archive
from wiring_replay import validate_release


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    data=path.read_bytes()
    return json.loads(gzip.decompress(data) if path.suffix=='.gz' else data)


def module(name,path):
    spec=importlib.util.spec_from_loader(name,SourceFileLoader(name,str(path)))
    result=importlib.util.module_from_spec(spec)
    previous=sys.dont_write_bytecode
    try:
        sys.dont_write_bytecode=True
        spec.loader.exec_module(result)
    finally:
        sys.dont_write_bytecode=previous
    return result


def equal(a,b):
    if isinstance(a,dict):
        a={k:v for k,v in a.items() if k not in ('run_id','electrical_revision')}
        b={k:v for k,v in b.items() if k not in ('run_id','electrical_revision')}
        assert set(a)==set(b)
        for k in a:
            equal(a[k],b[k])
    elif isinstance(a,list):
        assert isinstance(b,list) and len(a)==len(b)
        for x,y in zip(a,b):
            equal(x,y)
    elif isinstance(a,(int,float)) and not isinstance(a,bool):
        assert math.isclose(a,b,rel_tol=1e-9,abs_tol=1e-12),(a,b)
    else:
        assert a==b,(a,b)


def floating():
    labels={}
    for label in ('native','headless'):
        cases=[]
        for returned,switched in ((False,False),(False,True),(True,False),(True,True)):
            name=f'return-{int(returned)}-switch-{int(switched)}'
            result=read(HERE/'floating'/f'{label}-floating-{name}.json.gz')
            assert result['name']==name and result['return_resistor']==returned and result['switched']==switched
            base,stages,reopened=result['baseline'],result['stages'],result['reopened']
            assert len(stages)==(3 if switched else 2) and base['paused'] and base['transient']['sample']['time_s']==0
            charged=stages[0]
            error=abs(charged['mixed']['capacitors'][0]['voltage_v']-5*-math.expm1(-.02/((10001 if switched else 10000)*1e-6)))
            assert error==result['charging_error_v'] and error<.003
            for key in ('document','undo_entry','run_id','electrical_revision'):
                assert charged.get(key)==base.get(key)
            for stage in stages[:-1]:
                samples=stage['transient']['samples'];points=stage['scope']['traces'][0]['points']
                assert len(samples)==len(points)
                for sample,point in zip(samples,points):
                    assert sample['time_s']==point['time_s'] and abs(sample['input_v']-5)<1e-12 and abs(point['value']-5)<1e-12
            if switched:
                switch=stages[1]['switch_states'][0]
                assert not switch['pending'] and not switch['applied_closed']
                last=stages[1]['transient']['samples'][-2:]
                assert last[0]['time_s']==last[1]['time_s']
                assert abs(last[0]['capacitor_voltage_v']-last[1]['capacitor_voltage_v'])<1e-10
            previous,off=stages[-2],stages[-1]
            count=len(previous['transient']['samples'])
            assert off['transient']['samples'][:count]==previous['transient']['samples']
            assert off['transient']['input_mode']=='low'
            for sample,point in zip(off['transient']['samples'][count:],off['scope']['traces'][0]['points'][count:]):
                assert abs(sample['input_v'])<1e-12 and abs(point['value'])<1e-12
            saved=read(HERE/'floating'/f'{label}-floating-{name}.elab.gz')
            assert saved['graph']['components']==off['components']==reopened['components']
            assert saved['graph']['wires']==off['wires']==reopened['wires']
            assert not reopened['document']['dirty'] and reopened['paused'] and reopened['transient']['input_mode']=='auto'
            assert reopened['transient']['sample']['time_s']==0 and abs(reopened['transient']['sample']['input_v']-5)<1e-12
            cases.append(result)
        labels[label]=cases
    paired=0
    for a,b in zip(labels['native'],labels['headless']):
        for x,y in zip(a['stages'],b['stages']):
            for key in ('components','wires','paused','transient_settings'):
                assert x[key]==y[key]
            for key in ('mixed','transient','scope'):
                equal(x.get(key),y.get(key))
            paired+=1
        for key in ('mixed','transient','scope'):
            equal(a['reopened'].get(key),b['reopened'].get(key))
    assert paired==10


def captured(path):
    base=Path('/private/tmp/elab-first-version-178-signed-acceptance-final')
    relative=Path(path).relative_to(base)
    return HERE/'pins'/(str(relative)+'.gz')


def pins():
    plan=read(HERE/'pins/legacy-pin-plan.json.gz')
    assert plan['source_proof_checkpoint_sha256']==read(HERE/'Acceptance.json')['source_proof_checkpoint_sha256']
    assert len(plan['cases'])==6 and sum(len(c['ids']) for c in plan['cases'])==44
    adapters={}
    for label in ('native','headless'):
        values=[]
        for case in plan['cases']:
            for kind in ('present','missing'):
                source=captured(case[kind]);data=gzip.decompress(source.read_bytes())
                assert hashlib.sha256(data).hexdigest()==case[kind+'_sha256']
                doc=json.loads(data)
                result=read(HERE/'pins'/f'{label}-pin-{case["name"]}-{kind}.json.gz')
                state=result['loaded']
                assert (result['name'],result['kind'])==(case['name'],kind)
                assert result['source_sha256']==case[kind+'_sha256'] and not state['document']['dirty']
                assert state['components']==doc['graph']['components'] and state['wires']==doc['graph']['wires']
                assert [p['definition'] for p in state['pinned_readings']]==doc['pinned_readings']['items']
                assert [p['definition']['id'] for p in state['pinned_readings']]==case['ids']
                if kind=='present':
                    for p in state['pinned_readings']:
                        assert p['state']=='live' and math.isfinite(p['value']) and p['time_s']==0
                    moved,undone,reopened=result['moved'],result['undone'],result['reopened']
                    assert moved['document']['preferences_dirty'] and not moved['document']['schematic_dirty']
                    for key in ('components','wires','mixed','transient','run_id','electrical_revision'):
                        assert moved.get(key)==state.get(key)
                    assert not undone['document']['dirty'] and undone['pinned_readings']==state['pinned_readings']
                    saved=captured(result['saved_path'])
                    assert hashlib.sha256(gzip.decompress(saved.read_bytes())).hexdigest()==result['saved_sha256']
                    written=read(saved)
                    assert written['graph']==doc['graph'] and written['pinned_readings']==doc['pinned_readings']
                    assert reopened['components']==state['components'] and reopened['wires']==state['wires'] and not reopened['document']['dirty']
                    equal(reopened['pinned_readings'],state['pinned_readings'])
                else:
                    assert all(p['state']=='unavailable' and 'value' not in p for p in state['pinned_readings'])
                values.append(result)
        adapters[label]=values
    for a,b in zip(adapters['native'],adapters['headless']):
        assert (a['name'],a['kind'])==(b['name'],b['kind'])
        equal(a['loaded']['pinned_readings'],b['loaded']['pinned_readings'])
        for key in ('components','wires'):
            assert a['loaded'][key]==b['loaded'][key]
        if a['kind']=='present':
            for key in ('mixed','transient','scope'):
                equal(a['loaded'].get(key),b['loaded'].get(key))
            equal(a['reopened']['pinned_readings'],b['reopened']['pinned_readings'])
    recoveries={}
    for label in ('native','headless'):
        values=[]
        for example in ('nmos','opamp-follower'):
            result=read(HERE/'pins'/f'{label}-pin-recovery-{example}.json.gz')
            before,restored,reopened=result['before'],result['restored'],result['reopened']
            assert result['manual_and_capsule_unchanged']
            for field in ('manual','capsule','saved'):
                data=gzip.decompress(captured(result[field+'_path']).read_bytes())
                assert hashlib.sha256(data).hexdigest()==result[field+'_sha256']
            candidate,inactive=result['candidate'],result['inactive']
            assert candidate['active_session'] and candidate['current_session']
            assert candidate['id']==inactive['id'] and candidate['sha256']==inactive['sha256']==result['capsule_sha256']
            assert not inactive['active_session'] and not inactive['current_session'] and inactive['owner_status']=='inactive'
            capsule=read(captured(result['capsule_path']))
            assert capsule['document']['graph']['components']==before['components'] and capsule['document']['graph']['wires']==before['wires']
            assert capsule['document']['pinned_readings']['items']==[p['definition'] for p in before['pinned_readings']]
            assert restored['document']['dirty'] and restored['document']['path']=='' and restored['document']['recovered_from']==result['manual_path']
            assert restored['components']==before['components'] and restored['wires']==before['wires']
            equal(restored['pinned_readings'],before['pinned_readings'])
            assert not reopened['document']['dirty']
            equal(reopened['pinned_readings'],before['pinned_readings'])
            values.append(result)
        recoveries[label]=values
    for a,b in zip(recoveries['native'],recoveries['headless']):
        assert a['example']==b['example']
        equal(a['restored']['pinned_readings'],b['restored']['pinned_readings'])
    summary=read(HERE/'Reference-acceptance.json.gz')['pinned_file_commands']
    assert summary['passed'] and summary['native_headless_legacy_opens']==24 and summary['new_saves_and_fresh_reopens']==12
    assert summary['inactive_store_recovery_cases']==4 and summary['recovered_saved_reopens']==4
    assert summary==read(HERE/'Acceptance.json')['packaged_pins']


def main():
    cp=read(HERE/'checkpoint.json')
    files={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
    assert files==set(cp['evidence_sha256'])
    for relative,digest in cp['evidence_sha256'].items():
        path=HERE/relative
        assert path.is_file() and not path.is_symlink() and path.resolve().is_relative_to(HERE)
        assert sha(path)==digest,relative
    receipt_path=ROOT/'research/ui-audit/first-version/pin-reader-review-checkpoint.json'
    assert sha(receipt_path)==cp['receipt_sha256'] and receipt_path.read_bytes()==(HERE/'Acceptance.json').read_bytes()
    receipt=read(receipt_path);review=Path(receipt['review_path'])
    identity=validate_release(ROOT,receipt_path)
    assert identity['release']=='0.161.0 /178' and identity['runtime_inputs_verified']==358
    assert identity['manifest_file_entries_verified']==529
    assert sha(review/'RELEASE_MANIFEST.json')==cp['manifest_sha256']
    assert (HERE/'Release-manifest.json').read_bytes()==(review/'RELEASE_MANIFEST.json').read_bytes()
    for target,entry in read(HERE/'selected-source-files.json').items():
        assert sha(HERE/target)==entry['sha256']==sha(review/entry['source'])
    assert len(receipt['quality']['source_sha256'])==652 and receipt['quality']['passed']
    assert all(sha(ROOT/name)==digest for name,digest in receipt['quality']['source_sha256'].items())
    source_proof=ROOT/'docs/validation/pinned-probe-roundtrip'
    assert sha(source_proof/'checkpoint.json')==cp['source_proof_checkpoint_sha256']==receipt['source_proof_checkpoint_sha256']
    module('pin_source_replay',source_proof/'verify.py').main()
    built=read(HERE/'Build-result.json')
    assert built['clean_build_vcs'] and built['source_commit']==receipt['source_commit']
    admission=module('signed_release_admission',HERE/'sources/signed_bundle.py.txt')
    assert built['source_bundle']==admission.bundle_tree(review/'ELab.app')
    raw=ROOT/'build/first-version-178-tools/build/elab'
    admission.admit(review/'ELab.app',built,raw)
    runtime=read(HERE/'Runtime.json')
    assert runtime['passed'] and runtime['shared_geometry_unchanged']
    assert runtime['source_bundle']==built['source_bundle'] and runtime['source_agent_sha256']==built['agent_sha256']
    assert all(s['status']=='completed' and s['exit_code']==0 for s in runtime['stages'])
    public=read(HERE/'Public-report.json.gz')
    assert public['passed'] and [a['name'] for a in public['adapters']]==['in-process','headless-stdio','headless-socket','native-socket']
    exits,shutdowns=read(HERE/'Process-exits.json.gz'),read(HERE/'Shutdown-responses.json.gz')
    assert len(exits)==102 and len(shutdowns)==112
    for value in exits.values():
        assert value['exit_code']==0 and not value['graceful_wait_timed_out']
        assert not any(value.get(k) for k in ('wait_error','close_error','context_error_before_cancel'))
    for value in shutdowns.values():
        assert not value.get('error') and not value['response'].get('error')
        state=value['response']['result'];job=state['shutdown']
        assert state['closed'] and job['status']=='completed' and job['workers_joined'] and job['exit_ready']
    accepted=read(HERE/'Reference-acceptance.json.gz')
    assert accepted['passed'] and accepted['native_host_closed_normally'] and accepted['shared_geometry_unchanged']
    assert accepted['floating_source_commands']==receipt['packaged_floating']
    assert accepted['floating_source_commands']['fresh_saved_reopens']==8
    floating()
    pins()
    previous=module('previous_paging_replay',ROOT/'docs/validation/shutdown-layout-review-176/verify.py')
    previous.verify_paging(HERE/'paging',read(HERE/'Paging-plan.json'))
    for label,entry in receipt['preserved_bundles'].items():
        folder=ROOT/'build'/('ELab-'+label+'-review')
        manifest=read(folder/'RELEASE_MANIFEST.json');actual=members(folder);actual.pop('RELEASE_MANIFEST.json')
        assert actual==manifest['files'] and len(actual)==entry['files_verified']
        assert sha(folder/'RELEASE_MANIFEST.json')==entry['manifest_sha256']
    assert len(receipt['preserved_bundles'])==36
    for key,name in [('preserved_original_147_candidate','ELab-0.130.0-147-review'),
                     ('preserved_failed_164_candidate','ELab-0.147.0-164-review'),
                     ('preserved_unsigned_171_candidate','ELab-0.154.0-171-unsigned-candidate'),
                     ('preserved_failed_172_candidate','ELab-0.155.0-172-failed-candidate')]:
        folder=ROOT/'build'/name;manifest=read(folder/'CANDIDATE_MANIFEST.json')
        actual=members(folder);actual.pop('CANDIDATE_MANIFEST.json')
        assert actual==manifest['files'] and len(actual)==receipt[key]['files_verified']
        assert sha(folder/'CANDIDATE_MANIFEST.json')==receipt[key]['manifest_sha256']
    assert sha(ROOT/'build/ELab.app/Contents/MacOS/ELab')==receipt['canonical']['native_sha256']
    assert sha(ROOT/receipt['sdk']['archive'])==receipt['sdk']['sha256']
    archive=ROOT/'build/ELab-0.161.0-178-review.zip';ar=read(HERE/'archive/report.json')
    assert ar['passed'] and ar['files_verified']==530 and sha(archive)==cp['archive_sha256']==ar['archive_sha256']
    expected=members(review)
    assert expected==ar['expected_files']
    verify_archive(archive,review.name,expected)
    assert members(Path(ar['extracted_review']))==expected
    examples=read(HERE/'archive/example-report.json')
    assert examples['passed'] and len(examples['cases'])==3 and examples['agent_sha256']==receipt['agent_sha256']
    assert all(c['fixture_unchanged'] and c['saved_reopen_fresh'] for c in examples['cases'])
    assert not any(read(HERE/'report.json')[k] for k in ('canonical_promoted','new_physical_input','new_visible_latency','owner_acceptance','full_mvp_complete'))
    print('Verified /178: exact sources/signature/ZIP, four-adapter lifecycle, advanced pin files/recovery, floating-source, paging and preserved reviews.')


if __name__=='__main__':
    main()

"""Read-only replay of native /179 geometry input and retained limitations."""
from pathlib import Path
import gzip,hashlib,importlib.util,json,math,sys

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]

def sha(data):return hashlib.sha256(data).hexdigest()
def bytes_for(name):
    p=HERE/'native'/name
    if p.is_file():return p.read_bytes()
    return gzip.decompress(p.with_name(p.name+'.gz').read_bytes())
def read(name):return json.loads(bytes_for(name))
def main():
    cp=json.loads((HERE/'checkpoint.json').read_text())
    files={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
    assert files==set(cp['files'])
    for name,digest in cp['files'].items():
        path=HERE/name;assert not path.is_symlink() and path.resolve().is_relative_to(HERE) and sha(path.read_bytes())==digest,name
    manifest=json.loads((HERE/'manifest.json').read_text())
    for name,entry in manifest.items():
        stored=(HERE/name).read_bytes();raw=gzip.decompress(stored) if entry['encoding']=='gzip' else stored
        assert sha(stored)==entry['stored_sha256'] and sha(raw)==entry['original_sha256'] and len(raw)==entry['original_bytes']
    report=json.loads((HERE/'report.json').read_text());native=read('report.json');reference=read('reference/report.json')
    receipt_path=ROOT/'research/ui-audit/first-version/geometry-action-review-checkpoint.json'
    assert sha(receipt_path.read_bytes())==report['receipt_sha256']==native['receipt_sha256']
    receipt=json.loads(receipt_path.read_text())
    assert native['native_sha256']==report['native_sha256']==receipt['native_sha256']
    assert native['agent_sha256']==reference['agent_sha256']==report['agent_sha256']==receipt['agent_sha256']
    assert len(receipt['quality']['source_sha256'])==report['source_files_unchanged']==654
    for name,digest in receipt['quality']['source_sha256'].items():assert sha((ROOT/name).read_bytes())==digest,name
    assert all(native[k] for k in ('owned_process_absent','endpoint_removed','same_pid_after_exec','shared_geometry_unchanged','review_unchanged','copied_bundle_unchanged','canonical_unchanged','baseline_file_unchanged'))
    assert native['open_exit_code']==0 and native['native_exit_status'] is report['native_exit_code'] is None
    assert not native['forced_cleanup'] and not native.get('cleanup_error')
    assert native['shared_geometry_before']==native['shared_geometry_after']
    assert all(control['denied'] and control['errno'] in (1,13) for control in native['controls'].values())
    assert read('close-response.json')['shutdown']['exit_ready']
    for key in ('workers_joined','recovery_finalized','exit_ready'):assert read('close-response.json')['shutdown'][key]
    assert read('close-response.json')['shutdown']['status']=='completed'
    for name,digest in native['baseline_files'].items():assert sha(bytes_for(name))==digest
    assert bytes_for('Final-Geometry-179.elab')==bytes_for('Reference-Final-Geometry-179.elab')
    sys.path.insert(0,str(ROOT/'scripts'))
    spec=importlib.util.spec_from_file_location('native_geometry_compare',ROOT/'scripts/first-version-parity.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    assert reference['passed'] and reference['matched_state_count']==len(reference['matched_states'])==14
    for name in reference['matched_states']:
        actual,expected=read(name+'.json'),read('reference/'+name+'.json')
        for key in ('components','wires','nets','selected','drawing','transient_settings','paused','analysis'):assert actual.get(key)==expected.get(key),(name,key)
        for key in ('result','mixed','transient','scope','pinned_readings'):module.compare(actual.get(key),expected.get(key),name+'.'+key)
        for key in ('path','dirty','schematic_dirty','preferences_dirty','recordings_dirty','recording_count','format_version','recovered_from'):
            assert actual['document'].get(key)==expected['document'].get(key),(name,key)
    for close in reference['closed_processes']:
        assert close['exit_code']==0 and close['shutdown']['status']=='completed'
        assert all(close['shutdown'][k] for k in ('workers_joined','recovery_finalized','exit_ready'))
    assert len(reference['closed_processes'])==2 and reference['native_final_byte_identical_resave'] and reference['fresh_native_final_reopen']
    cases=('led-rotate','led-duplicate-named','rl-duplicate','pulse-duplicate')
    for name in ('led-rotate','led-duplicate','led-duplicate-named','rl-duplicate','pulse-duplicate'):
        before,private=read('before-'+name+'.json'),read('private-'+name+'.json')
        changed={k for k in before if before.get(k)!=private.get(k)}
        if name=='rl-duplicate':
            assert changed=={'viewport','command_revision','revision','last_actor'}
            assert private['command_revision']==before['command_revision']+1
            assert before['viewport']['height_px']!=private['viewport']['height_px']
        else:assert not changed
    for name in cases:
        before=read('before-'+name+'.json')
        private=read('private-'+name+'.json')
        after=read('after-'+name+'.json')
        undone=read('after-'+name+'-undo.json')
        tip=after['undo_entry'];assert tip['actor']=='user' and tip['command_revision']==private['command_revision']+2
        assert after['command_revision'] in (tip['command_revision'],tip['command_revision']+1)
        assert undone['redo_entry']==tip
        assert undone['components']==before['components'] and undone['wires']==before['wires'] and not undone['document']['dirty']
        assert after['wires']==before['wires']
        if name=='led-rotate':
            assert tip['name']=='Rotate component'
            for key in ('result','scope','run_id','electrical_revision'):assert before[key]==after[key]
            assert next(p for p in after['components'] if p['id']=='R1')['rotation']==90
        else:
            assert tip['name']=='Duplicate component' and len(after['components'])==len(before['components'])+1
            owner=before['selected'];original=next(p for p in before['components'] if p['id']==owner)
            clone=next(p for p in after['components'] if p['id']==after['selected'])
            assert clone['id']!=owner and clone['position']!=original['position']
            assert {k:v for k,v in clone.items() if k not in ('id','position')}=={k:v for k,v in original.items() if k not in ('id','position')}
            if before.get('transient'):
                assert after['transient']['sample']==before['transient']['sample'] and after['transient']['samples']==before['transient']['samples']
                assert after['transient']['stale'] and after['mixed']['stale']
    missed=read('after-led-duplicate-attempt.json');miss_before=read('before-led-duplicate.json')
    assert missed['components']==miss_before['components'] and missed['wires']==miss_before['wires'] and missed['undo_entry']==miss_before['undo_entry']
    assert missed['command_revision']==miss_before['command_revision']+1 and missed['drawing']['value']=='unfinished'
    actions=json.loads((HERE/'producers/action-record.json').read_text())
    assert len(actions['cases'])==5 and actions['cases'][1]['outcome'].startswith('inconclusive') and not actions['cases'][1]['replacement']
    assert report['declared_cases']==4 and report['successful_declared_cases']==3 and report['inconclusive_declared_cases']==1 and report['additional_declared_named_case_passed']
    maxima=dict(current_a=0.,voltage_v=0.,energy_j=0.)
    for row in read('before-rl-duplicate.json')['transient']['samples']:
        current=.05-.07*math.exp(-row['time_s']/.01)
        for key,expected in [('current_a',current),('voltage_v',5-100*current),('energy_j',.5*current*current)]:
            maxima[key]=max(maxima[key],abs(row[key]-expected))
    assert maxima['current_a']<1e-12 and maxima['voltage_v']<1e-10 and maxima['energy_j']<1e-14
    sys.path.insert(0,str(ROOT/'scripts'))
    from package_review import members
    folder=Path(receipt['review_path']);actual=members(folder);actual.pop('RELEASE_MANIFEST.json')
    assert actual==json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual)==418
    assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==receipt['canonical']['native_sha256']
    assert not any(report[k] for k in ('application_changed','new_release','new_latency_observations','owner_acceptance','full_mvp_complete'))
    print(json.dumps(dict(native_qualified_cases=4,original_duplicate_attempt='inconclusive and preserved',matched_agent_states=14,rl_equation_max_errors=maxima,normal_owned_cleanup=True,owner_acceptance=False)))

if __name__=='__main__':main()

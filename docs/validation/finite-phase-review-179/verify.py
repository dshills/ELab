"""Read-only independent finite-pulse/RC and captured-file evidence replay."""
from pathlib import Path
import gzip,hashlib,importlib.util,json,math,sys
from importlib.machinery import SourceFileLoader

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]

def digest(data):return hashlib.sha256(data).hexdigest()
def raw(name):return gzip.decompress((HERE/'raw'/(name+'.gz')).read_bytes())
def read(name):return json.loads(raw(name))
def module(name,path):
    spec=importlib.util.spec_from_loader(name,SourceFileLoader(name,str(path)))
    result=importlib.util.module_from_spec(spec)
    previous=sys.dont_write_bytecode
    try:
        sys.dont_write_bytecode=True;spec.loader.exec_module(result)
    finally:sys.dont_write_bytecode=previous
    return result

def main():
    checkpoint=json.loads((HERE/'checkpoint.json').read_text())
    files={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
    assert files==set(checkpoint['files'])
    for relative,sha in checkpoint['files'].items():
        path=HERE/relative
        assert not path.is_symlink() and path.resolve().is_relative_to(HERE) and digest(path.read_bytes())==sha,relative
    for relative,entry in json.loads((HERE/'manifest.json').read_text()).items():
        stored=(HERE/relative).read_bytes();original=gzip.decompress(stored)
        assert digest(stored)==entry['stored_sha256'] and digest(original)==entry['original_sha256'] and len(original)==entry['original_bytes']
    report=json.loads((HERE/'report.json').read_text());plan=read('plan.json');collected=read('report.json')
    receipt_path=ROOT/'research/ui-audit/first-version/geometry-action-review-checkpoint.json'
    assert digest(receipt_path.read_bytes())==report['receipt_sha256']==plan['receipt_sha256']
    receipt=json.loads(receipt_path.read_text())
    assert report['agent_sha256']==plan['agent_sha256']==receipt['agent_sha256']
    assert len(receipt['quality']['source_sha256'])==report['source_files_unchanged']==654
    for relative,sha in receipt['quality']['source_sha256'].items():assert digest((ROOT/relative).read_bytes())==sha,relative
    assert collected['passed'] and collected['failure'] is None and collected['agent_unchanged']
    assert len(plan['cases'])==len(collected['results'])==report['cases']==8
    assert len(collected['process_exits'])==report['normal_process_exits']==16
    for result in collected['process_exits']:
        work=result['shutdown'];assert result['exit_code']==0 and work['status']=='completed'
        assert all(work[k] for k in ('workers_joined','recovery_finalized','exit_ready'))
    reference=module('independent_finite_phase_reference',HERE/'producers/reference.py.txt')
    sys.path.insert(0,str(ROOT/'scripts'))
    parity=module('finite_phase_parity',ROOT/'scripts/first-version-parity.py')
    totals=dict(rows=0,channels=0,corners=0)
    aggregate=dict(source_v=0.,capacitor_v=0.,current_a=0.)
    probes=['v:V1','v:C1','i:R1','i:C1']
    warning_counts={'full-rise':1,'full-fall':1,'triangle':0,'negative-triangle':0,'minimum-high':1,'minimum-low':1,'constant-level':0,'decimal-cycle':0}
    for case,summary in zip(plan['cases'],collected['results']):
        name=case['name'];assert summary['passed'] and name==summary['name']
        prefix=name+'/'
        before,live,reopened=(read(prefix+p+'.json') for p in ('baseline','live','reopened'))
        record=read(prefix+'record.json');view=read(prefix+'operation.json');project=read(prefix+'project.elab')
        assert digest(raw(prefix+'project.elab'))==summary['project_sha256']
        assert digest(raw(prefix+'record.json'))==summary['record_sha256']
        assert raw(prefix+'record.json')==raw(prefix+'reopened-record.json')
        assert next(p for p in before['components'] if p['id']=='V1')['pulse']==case['pulse']
        assert next(p for p in before['components'] if p['id']=='R1')['value']==10000
        assert next(p for p in before['components'] if p['id']=='C1')['value']==1e-6
        assert math.ceil(case['duration']/case['step'])<=plan['limits']['maximum_verify_output_steps']==1000
        for key in ('components','wires','transient_settings','document','run_id','electrical_revision','undo_entry','redo_entry'):
            assert before.get(key)==live.get(key),(name,key)
        assert before['paused'] and not live['transient']['stale']
        assert live['transient']['sample']['time_s']==case['duration']
        metadata,rows=record['metadata'],record['rows']
        assert [p['id'] for p in metadata['probes']]==probes
        assert metadata['duration_s']==case['duration'] and metadata['settings']['step_s']==case['step']
        assert len(rows)==metadata['sample_count']==summary['recorded_rows'] and view['operation']['status']=='completed'
        parity.compare(metadata['mixed'],live['mixed'],name+'.live-Verify-final')
        maxima=dict(source_v=0.,capacitor_v=0.,current_a=0.)
        for row in rows:
            source,capacitor=reference.at(case['pulse'],row['time_s'])
            wanted=(source,capacitor,(source-capacitor)/10000,(source-capacitor)/10000)
            for index,(actual,expected) in enumerate(zip(row['values'],wanted)):
                key='source_v' if index==0 else 'capacitor_v' if index==1 else 'current_a'
                maxima[key]=max(maxima[key],abs(actual-expected))
            assert abs(row['values'][2]-row['values'][3])<1e-12
        assert maxima==summary['max_errors']
        assert maxima['source_v']<plan['limits']['source_voltage_error_v']==1e-9
        assert maxima['capacitor_v']<plan['limits']['capacitor_voltage_error_v']==.003
        assert maxima['current_a']<plan['limits']['current_error_a']==3e-7
        matched=0
        for index,trace in enumerate(live['scope']['traces']):
            assert trace['probe']['id']==probes[index]
            for point in trace['points']:
                assert any(row['time_s']==point['time_s'] and math.isclose(row['values'][index],point['value'],rel_tol=1e-9,abs_tol=1e-12) for row in rows),(name,point)
                matched+=1
        assert matched==summary['live_record_values_matched']
        expected=reference.corners(case['pulse'],case['duration'])
        events=[e['time_s'] for e in live['transient']['events'] if e['mode']=='pulse:V1:corner']
        assert events==summary['observed_corners'] and expected==summary['expected_corners'] and len(events)==len(expected)
        assert all(abs(a-b)<plan['limits']['corner_time_error_s']==1e-12 for a,b in zip(events,expected))
        assert live['components']==reopened['components']==project['graph']['components']
        assert live['wires']==reopened['wires']==project['graph']['wires']
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count']==1
        assert reopened['transient']['sample']['time_s']==0
        assert project['recordings']['entries'][0]['data']==record
        assert len(before.get('source_warnings',[]))==warning_counts[name]
        assert before.get('source_sampling')==reopened.get('source_sampling')
        if name.startswith('minimum-'):
            advice=before['source_sampling'][0]
            assert advice['minimum_limited'] and advice['unresolved'] and advice['target_step_s']==.25e-6
            assert advice['recommended_step_s']==1e-6 and before['transient_settings']['step_s']==5e-6
            assert len(live['transient']['samples'])==512<len(rows)
        for key in aggregate:aggregate[key]=max(aggregate[key],maxima[key])
        totals['rows']+=len(rows);totals['channels']+=matched;totals['corners']+=len(events)
    assert aggregate==report['max_errors']
    assert totals==dict(rows=report['recorded_rows'],channels=report['live_channel_values_matched'],corners=report['unique_corner_events'])==dict(rows=1685,channels=5968,corners=49)
    assert not any(report[k] for k in ('application_changed','new_release','new_native_input','new_latency_observations','owner_acceptance','full_mvp_complete'))
    sys.path.insert(0,str(ROOT/'scripts'))
    from package_review import members
    folder=Path(receipt['review_path']);actual=members(folder);actual.pop('RELEASE_MANIFEST.json')
    assert actual==json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual)==418
    assert digest((ROOT/receipt['sdk']['archive']).read_bytes())==receipt['sdk']['sha256']
    assert digest((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==receipt['canonical']['native_sha256']
    print(json.dumps(dict(finite_phase_cases=8,recorded_rows=1685,live_channel_values_matched=5968,corner_events=49,exact_fresh_record_reopens=8,normal_process_exits=16,max_errors=aggregate,application_changed=False)))

if __name__=='__main__':main()

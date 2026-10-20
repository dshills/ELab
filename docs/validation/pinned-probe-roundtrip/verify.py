"""Read-only source, raw protocol and exact-file replay for PIN-REOPEN-01."""
from pathlib import Path
import gzip
import hashlib
import json
import math
import sys

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')

HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]


def raw(path):
    data=path.read_bytes()
    return gzip.decompress(data) if path.suffix=='.gz' else data


def read(path):
    return json.loads(raw(path))


def sha(data):
    return hashlib.sha256(data).hexdigest()


def protocol(folder, error=None):
    rows=[json.loads(line) for line in raw(folder/'transcript.jsonl.gz').splitlines()]
    status=read(folder/'process-exit.json.gz')
    assert status['exit_code']==0 and status['requests']==len(rows)
    assert raw(folder/'stderr.txt.gz')==b''
    failures=[]
    ids=set()
    for row in rows:
        req,res=row['request'],row['response']
        assert req['version']==res['version']==1 and req['id']==res['id'] and req['id'] not in ids
        ids.add(req['id'])
        if res.get('error'):
            failures.append((req['command'],res['error']['code']))
    assert failures==([('project.load',error)] if error else [])
    assert rows[0]['request']['command']=='project.inspect' and rows[0]['response']['result']['components']==[]
    assert rows[-1]['request']['command']=='app.close'
    state=rows[-1]['response']['result'];job=state['shutdown']
    assert state['closed'] and job['status']=='completed' and job['workers_joined'] and job['exit_ready']
    return rows


def public():
    original=read(HERE/'original/report.json.gz')
    final=read(HERE/'final/report.json.gz')
    assert original['passed'] and original['expected_original_failures']
    assert final['passed'] and not final['expected_original_failures'] and original['cases']==final['cases']
    cases=original['cases'];assert len(cases)==6 and sum(len(c['ids']) for c in cases)==44
    assert len(original['reopens'])==len(final['reopens'])==12
    counts=dict(original=dict(processes=0,requests=0),final=dict(processes=0,requests=0))
    for case in cases:
        name=case['name'];folder=HERE/'original'/name
        creator=protocol(folder/'creator')
        counts['original']['processes']+=1;counts['original']['requests']+=len(creator)
        pin_requests=[r['request']['params']['id'] for r in creator if r['request']['command']=='measurement.pin.add']
        assert pin_requests==case['ids']
        saves=[r for r in creator if r['request']['command']=='project.save']
        assert len(saves)==2
        before=read(folder/'source-state.json.gz')
        assert before['paused'] and [p['definition']['id'] for p in before['pinned_readings']]==case['ids']
        for kind,index in [('present',0),('missing',1)]:
            saved=raw(folder/(kind+'.elab.gz'))
            assert sha(saved)==case[kind+'_sha256']
            document=json.loads(saved)
            assert document['format']=='elab.schematic' and document['version']==4
            definitions=document['pinned_readings']['items']
            assert [p['id'] for p in definitions]==case['ids']
            assert all(set(p)=={'id','position'} for p in definitions)
            work=saves[index]['response']['result']['project_work']
            assert work['status']=='completed' and work['artifact_sha256']==sha(saved)
            assert not saves[index]['response']['result']['document']['dirty']
            assert saves[index]['request']['params']['path']==case[kind]
            old_folder=HERE/'original/readers'/(name+'-'+kind)
            old=protocol(old_folder,'invalid-document')
            counts['original']['processes']+=1;counts['original']['requests']+=len(old)
            assert all(r['request']['command'] in ('project.inspect','project.load','app.close') for r in old)
            failed=next(r for r in old if r['request']['command']=='project.load')
            assert failed['request']['params']['path']==case[kind]
            assert 'pinned reading' in failed['response']['error']['message']
            incoming=old[1]['response']['result']
            refused=read(old_folder/'state.json.gz')
            for key in ('components','wires','nets','mixed','transient','document','run_id','electrical_revision','undo_entry','redo_entry'):
                assert incoming.get(key)==refused.get(key)
            fresh_folder=HERE/'final/readers'/(name+'-'+kind)
            fresh=protocol(fresh_folder)
            counts['final']['processes']+=1;counts['final']['requests']+=len(fresh)
            assert all(r['request']['command'] in ('project.inspect','project.load','app.close') for r in fresh)
            load=next(r for r in fresh if r['request']['command']=='project.load')
            assert load['request']['params']['path']==case[kind]
            state=read(fresh_folder/'state.json.gz')
            assert not state['document']['dirty'] and state['components']==document['graph']['components'] and state['wires']==document['graph']['wires']
            pins=state['pinned_readings']
            assert [p['definition'] for p in pins]==definitions
            for i,pin in enumerate(pins):
                if kind=='present':
                    expected=before['pinned_readings'][i]
                    assert pin['state']==expected['state']=='live' and pin['probe']==expected['probe'] and pin['time_s']==0
                    assert math.isfinite(pin['value']) and abs(pin['value']-expected['value'])<=1e-12
                else:
                    assert pin['state']=='unavailable' and 'value' not in pin
    assert counts==dict(original=dict(processes=18,requests=176),final=dict(processes=12,requests=60))
    for result in original['reopens']+final['reopens']:
        assert result['file_unchanged']


def main():
    cp=read(HERE/'checkpoint.json')
    files={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
    assert files==set(cp['evidence_sha256'])
    for relative,digest in cp['evidence_sha256'].items():
        p=HERE/relative
        assert not p.is_symlink() and p.is_file() and p.resolve().is_relative_to(HERE)
        assert sha(p.read_bytes())==digest,relative
    report=read(HERE/'report.json');quality=read(HERE/'quality.json')
    assert report['passed'] and quality['passed'] and quality['exit_code']==0 and quality['source_unchanged']
    assert len(quality['source_sha256'])==652
    assert all(sha((ROOT/name).read_bytes())==digest for name,digest in quality['source_sha256'].items())
    base_pin=report['base_receipt'];assert sha((ROOT/base_pin['path']).read_bytes())==base_pin['sha256']
    base=read(ROOT/base_pin['path'])
    changed={name for name,digest in base['quality']['source_sha256'].items() if sha((ROOT/name).read_bytes())!=digest}
    assert changed=={'workbench/pinned_readings.go'}
    assert set(quality['source_sha256'])-set(base['quality']['source_sha256'])=={'workbench/pinned_probe_roundtrip_test.go'}
    assert sha((HERE/'sources/pinned_readings.go.original.txt').read_bytes())==base['quality']['source_sha256']['workbench/pinned_readings.go']
    for name in ('pinned_readings.go','pinned_probe_roundtrip_test.go'):
        assert (ROOT/'workbench'/name).read_bytes()==(HERE/'sources'/(name+'.txt')).read_bytes()
    assert read(HERE/'negative-control.json')['expected_test_exit_code']==1
    negative=raw(HERE/'checks/final-tests-original-source.txt.gz').decode()
    assert negative.count('successful save must be readable')==6
    assert negative.count('want one candidate, got []')==2
    assert 'ok  ' in raw(HERE/'checks/focused-final.txt.gz').decode()
    for name,digest in report['artifacts'].items():
        assert sha((ROOT/name).read_bytes())==digest,name
    public()
    assert not any(report[k] for k in ('replacement_packaged','new_native_input','new_visible_latency','owner_acceptance','full_mvp_complete'))
    print('Verified PIN-REOPEN-01: original refusals, twelve exact-file reopens, 236 requests/30 normal exits and source/artifact pins.')


if __name__=='__main__':
    main()

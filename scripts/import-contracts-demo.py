#!/usr/bin/env python3
"""Owned complete-record import contracts, failure stages, retries and persistence."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import time

from demo_session import connect_demo_client
from recovery_timing import settle_native_view

spec=importlib.util.spec_from_file_location('helper',Path(__file__).with_name('persistence-demo.py'))
helper=importlib.util.module_from_spec(spec);spec.loader.exec_module(helper)
COMMANDS=['verification.import','verification.import.start']


def digest(path):return hashlib.sha256(path.read_bytes()).hexdigest()


def live(state):
    return {key:state.get(key) for key in ['components','wires','view','transient','mixed','npn','scope','paused','undo_entry','redo_entry','run_id','electrical_revision']}


def attachment(state):
    return {key:state.get(key) for key in ['document','operations','recording','selected_operation','last_operation']}


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect');parser.add_argument('--output',type=Path,required=True);parser.add_argument('--artifacts',type=Path,required=True)
    args=parser.parse_args();assets=args.artifacts.resolve();assets.mkdir(parents=True,mode=0o700,exist_ok=False)
    target=connect_demo_client(helper.Client,args.connect,command_preconditions=True);source=None;fresh=None
    failures=[];admission_refusals=[]
    try:
        initial=target.ask('project.inspect')
        if initial['recovery'].get('directory'):
            target.ask('recovery.configure',{'directory':initial['recovery']['directory'],'enabled':False,'interval_s':2},True)
        contracts={c:target.ask('capabilities.list',{'command':c})['capabilities'][0]['input_schema'] for c in COMMANDS}
        assert all(s['x-command-contract']['scope']=='captured-recording-import' for s in contracts.values())
        source=helper.Client(command_preconditions=True)
        source.ask('example.rc',{},True);source.ask('simulation.pause',{},True)
        accepted=source.ask('simulation.verify',{'duration_s':.006,'probe_ids':['v:C1','i:C1']},True)
        deadline=time.monotonic()+10
        while True:
            original=source.ask('operation.inspect',{'id':accepted['last_operation']})
            if original['operation']['status']=='completed':break
            assert original['operation']['status'] not in ('failed','canceled') and time.monotonic()<deadline
            time.sleep(.005)
        original_path=assets/'source-record.json'
        source.ask('verification.export',{'id':accepted['last_operation'],'format':'json','path':str(original_path)},True)
        original_bytes=original_path.read_bytes();original_hash=digest(original_path)
        target.ask('example.rc',{},True);target.ask('simulation.pause',{},True)
        target.ask('simulation.restart',{},True);target.ask('simulation.wait');target.ask('simulation.step',{'duration_s':.003},True)
        existing=target.ask('simulation.verify',{'duration_s':.003,'probe_ids':['v:C1']},True)
        deadline=time.monotonic()+10
        while True:
            retained=target.ask('operation.inspect',{'id':existing['last_operation']})
            if retained['operation']['status']=='completed':break
            assert retained['operation']['status'] not in ('failed','canceled') and time.monotonic()<deadline
            time.sleep(.005)
        target.ask('simulation.wait')
        if args.connect:
            settle_native_view(target);target.ask('view.pan',{'dx':0,'dy':0},True)
        manual=assets/'manual.elab';target.ask('project.save',{'path':str(manual)},True);manual_hash=digest(manual)
        baseline=target.ask('project.inspect');baseline_live=live(baseline)

        def inspect():return target.ask('project.inspect')

        def preserve(state):
            assert live(state)==baseline_live and digest(manual)==manual_hash and digest(original_path)==original_hash

        def terminal(identity):
            deadline=time.monotonic()+10
            while True:
                work=target.ask('project.work.inspect',{'id':identity})
                if work['status'] not in ('queued','running','canceling'):return work
                assert time.monotonic()<deadline;time.sleep(.005)

        for params,code in [({'path':None},'invalid-params'),({'path':'   '},'invalid-params'),
                            ({'path':str(original_path),'extra':True},'invalid-params'),
                            ({'path':str(original_path),'expected_document_generation':baseline['document']['generation']+1},'conflict')]:
            before=inspect();target.ask('verification.import.start',params,True,code);after=inspect()
            assert attachment(before)==attachment(after) and before['project_work']==after['project_work'];preserve(after)
            admission_refusals.append(dict(code=code,case='guard' if code=='conflict' else 'params'))
        admitted=target.ask('verification.import.start',{'path':str(original_path),'expected_document_generation':None},True)
        assert admitted['project_work']['status']=='queued' and admitted['document']['recording_count']==1;preserve(admitted)
        work=terminal(admitted['project_work']['id']);assert work['status']=='completed' and work['viewer_selected'] and work['source_sha256']==original_hash
        first_id=work['result_operation_id'];first=target.ask('operation.inspect',{'id':first_id})
        assert first['metadata']==original['metadata'] and not first['operation']['source_unchanged']
        attached=inspect();assert attached['recording']['id']==first_id and attached['recording']['viewing'] and attached['document']['recording_count']==2 and attached['document']['recordings_dirty'];preserve(attached)

        invalid=assets/'invalid.json';invalid.write_text('not recorded JSON\n')
        duplicate=assets/'duplicate.json';duplicate.write_text('{"metadata":{},"metadata":{},"rows":[]}')
        version=assets/'version.json';damaged=json.loads(original_bytes);damaged['metadata']['version']=99;version.write_text(json.dumps(damaged))
        exact=assets/'exact-limit.json'
        with exact.open('wb') as f:f.truncate(8*1024*1024)
        oversized=assets/'oversized.json'
        with oversized.open('wb') as f:f.truncate(8*1024*1024+1)
        link=assets/'linked.json';link.symlink_to(original_path)
        directory=assets/'directory';directory.mkdir(mode=0o700)
        for label,path,code in [('missing',assets/'missing.json','file-error'),('final-symlink',link,'file-error'),
                ('directory',directory,'file-error'),('oversized',oversized,'file-error'),('exact-size-invalid',exact,'invalid-recording'),
                ('malformed',invalid,'invalid-recording'),('duplicate-fields',duplicate,'invalid-recording'),('version',version,'invalid-recording'),('schematic-format',manual,'invalid-recording')]:
            before=inspect();started=target.ask('verification.import.start',{'path':str(path)},True);failed=terminal(started['project_work']['id'])
            assert failed['status']=='failed' and failed['error']['code']==code,failed
            after=inspect();assert attachment(before)==attachment(after);preserve(after)
            failures.append(dict(case=label,code=code))

        repair=assets/'repair.json';repair.write_text('broken input')
        before=inspect();request=dict(version=1,id='repair-first',command='verification.import',params={'path':str(repair)},expected_command_revision=before['command_revision'],key='import-repair-owned')
        def raw(params):
            target.process.stdin.write(json.dumps(params)+'\n');target.process.stdin.flush()
            line=target.process.stdout.readline();assert line
            return json.loads(line)
        failed=raw(request);assert failed['error']['code']=='invalid-recording'
        failed_id=failed['result']['project_work']['id'];repair.write_bytes(original_bytes)
        retry=raw(dict(request,id='repair-retry'))
        assert retry['duplicate'] and retry['error']['code']=='invalid-recording' and retry['result']['project_work']['id']==failed_id
        after=inspect();assert attachment(after)==attachment(before);preserve(after)
        request.update(id='repair-new',key='import-repair-corrected',expected_command_revision=after['command_revision'])
        repaired=raw(request);assert 'error' not in repaired and not repaired.get('duplicate')
        attached=inspect();second_id=attached['project_work']['result_operation_id'];assert second_id!=first_id and attached['document']['recording_count']==3;preserve(attached)
        for identity,name in [(first_id,'reexport-first.json'),(second_id,'reexport-repaired.json')]:
            target.ask('verification.export',{'id':identity,'format':'json','path':str(assets/name)},True)
            assert (assets/name).read_bytes()==original_bytes;preserve(inspect())
        rows=target.ask('operation.samples',{'id':second_id,'limit':100})['rows'];assert len(rows)==7
        saved=assets/'imported-project.elab';target.ask('project.save',{'path':str(saved)},True)
        assert digest(manual)==manual_hash and not inspect()['document']['dirty']
        fresh=helper.Client(command_preconditions=True);fresh.ask('project.load',{'path':str(saved)},True);reopened=fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count']==3
        assert reopened['transient']['sample']['time_s']==0 and reopened['recording']['viewing']
        reopened_id=reopened['recording']['id'];assert fresh.ask('operation.inspect',{'id':reopened_id})['metadata']==original['metadata']
        assert fresh.ask('operation.samples',{'id':reopened_id,'limit':100})['rows']==rows
        fresh.ask('verification.export',{'id':reopened_id,'format':'json','path':str(assets/'reexport-reopened.json')},True)
        assert (assets/'reexport-reopened.json').read_bytes()==original_bytes
        report=dict(passed=True,transport='native-socket' if args.connect else 'headless-stdio',contracts=contracts,admission_refusals=admission_refusals,
                    work_failures=failures,retry=dict(failure='invalid-recording',duplicate_failure=True,same_original_job=True,new_key_record_count=3),
                    source_sha256=original_hash,manual_sha256=manual_hash,metadata=original['metadata'],rows=rows,first_work=work,
                    first_id=first_id,second_id=second_id,live=baseline_live,reopened=dict(paused=reopened['paused'],time_s=reopened['transient']['sample']['time_s'],recording_count=3,dirty=reopened['document']['dirty']),
                    limits=['Viewer/capacity/cancel/replacement races use controlled core tests; no nondeterministic public-race claim',
                            'No decoder/model/numerical/runtime/format change or complete physical picker/AX/device/storage qualification'])
        args.output.parent.mkdir(parents=True,exist_ok=True)
        with args.output.open('x') as f:json.dump(report,f,indent=2);f.write('\n')
        print(json.dumps(dict(passed=True,transport=report['transport'],contracts=len(contracts),admission_refusals=len(admission_refusals),work_failures=len(failures),rows=len(rows))))
    finally:
        try:
            if fresh:fresh.close()
            if source:source.close()
            if args.connect:target.ask('app.close',{'discard_changes':True},True)
        finally:target.close()


if __name__=='__main__':main()

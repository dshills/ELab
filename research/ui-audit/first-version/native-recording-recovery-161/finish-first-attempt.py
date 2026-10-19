import gzip,hashlib,importlib.util,json,math,shutil,subprocess,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).resolve().parent;out=base/'observations'
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
r=json.loads((out/'report.json').read_text());assert r['passed'] and 'error' not in r
for host in r['hosts']:
    status=subprocess.run(['/bin/ps','-p',str(host['pid']),'-o','pid=,comm='],capture_output=True,text=True);assert status.returncode==1 and not status.stdout.strip()
assert [h['exit_code'] for h in r['hosts']]==[-9,0]
assert not [line for line in subprocess.check_output(['/bin/ps','-axo','pid=,comm='],text=True).splitlines() if 'ELab-value-recovery' in line]
identity=json.loads((base/'identity.json').read_text());bundle=root/'build/native-files-156-ui/ELab-value-recovery.app';restored={}
for relative,digest in identity['harness_before'].items():
    backup=base/'harness-before'/Path(relative).name;assert hashlib.sha256(backup.read_bytes()).hexdigest()==digest;shutil.copyfile(backup,bundle/relative);restored[relative]=hashlib.sha256((bundle/relative).read_bytes()).hexdigest();assert restored[relative]==digest
manual=out/'Manual RC.elab';snapshot=base/'recovery'/(r['choice']['id']+'.elab-recovery')
assert hashlib.sha256(manual.read_bytes()).hexdigest()==r['manual_sha256'];assert hashlib.sha256(snapshot.read_bytes()).hexdigest()==r['capsule_sha256'];assert snapshot.read_bytes()==(out/'Original recorded snapshot.elab-recovery').read_bytes()
original_entry=json.loads(snapshot.read_text())['document']['recordings']['entries'][0]
events={e['name']:e['state'] for e in r['events']};states={};laws=[]
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
for label,filename,has_recording in [('native baseline','Manual RC.elab',False),('agent baseline','Agent baseline.elab',False),('native recovered','Recovered RC.elab',True),('agent recovered','Agent recovered RC.elab',True)]:
    client=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']))
    try:
        blank=client.ask('simulation.wait');assert not blank['components'];client.ask('project.load',dict(path=str(out/filename)),True);state=client.ask('simulation.wait');assert state['paused'] and not state['document']['dirty'];assert state['transient']['sample']['time_s']==0 and state['transient']['sample']['capacitor_voltage_v']==0
        expected=events['restarted']
        for key in ['components','wires','nets','transient_settings']:assert state[key]==expected[key],(label,key)
        for key in ['result','transient','npn','mixed','scope']:p.compare(state.get(key),expected.get(key),label+'.'+key)
        assert state['document']['recording_count']==int(has_recording)
        if has_recording:
            view=state['recording'];assert not view['viewing'] and view['sample_index']==0 and view['sample_count']==21
            operation=client.ask('operation.inspect',dict(id=state['last_operation']));assert operation['metadata']==r['original_capture']['metadata']
            page=client.ask('operation.samples',dict(id=state['last_operation'],offset=0,limit=100));assert page['rows']==r['original_page']['rows'] and page['probes']==r['original_page']['probes']
            saved=json.loads((out/filename).read_text());assert saved['recordings']['entries']==[original_entry]
            maximum=0
            for row in page['rows']:
                for probe,value in zip(page['probes'],row['values']):
                    assert probe['id'] in ['v:C1.1','v:V1.positive'];reference=5*(1-math.exp(-row['time_s']/.01)) if probe['id']=='v:C1.1' else 5;error=abs(value-reference);assert error<=max(.001,.001*abs(reference));maximum=max(maximum,error)
            laws.append(dict(case=label,rows=21,maximum_error_v=maximum,passed=True))
        states[label]=state
    finally:client.close()
with (out/'fresh-agent-reopen.json.gz').open('wb') as f:
    with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write((json.dumps(dict(passed=True,fresh_reopen_processes=4,states=states,independent_rc=laws),indent=2)+'\n').encode())
cleanup=dict(passed=True,native_pids_absent=[h['pid'] for h in r['hosts']],native_exit_codes=[h['exit_code'] for h in r['hosts']],stopped_bundle_restored=restored,manual_save_preserved=True,original_capsule_preserved=True,embedded_recording_preserved_exactly=True,fresh_agent_reopens=4,release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json'))
(out/'cleanup.json').write_text(json.dumps(cleanup,indent=2)+'\n');print(json.dumps(cleanup))

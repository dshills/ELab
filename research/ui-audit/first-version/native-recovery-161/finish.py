import gzip,hashlib,importlib.util,json,shutil,subprocess,sys
from decimal import Decimal
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).resolve().parent;out=base/'observations'
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
r=json.loads((out/'report.json').read_text());reopen=json.loads((out/'reopen/report.json').read_text());assert not r['passed'] and r['saved_fields_except_view_match'];assert reopen['passed'] and reopen['native_exit_code']==0
assert 'TimeoutExpired' in r['error'] and 'allow-native-close' in [e['name'] for e in r['events']]
hosts=[row['pid'] for row in r['hosts']]+[reopen['pid']]
for pid in hosts:
    status=subprocess.run(['/bin/ps','-p',str(pid),'-o','pid=,comm='],capture_output=True,text=True);assert status.returncode==1 and not status.stdout.strip()
assert not [line for line in subprocess.check_output(['/bin/ps','-axo','pid=,comm='],text=True).splitlines() if 'ELab-value-recovery' in line]
bundle=root/'build/native-files-156-ui/ELab-value-recovery.app';identity=json.loads((base/'identity.json').read_text());restored={}
for relative,digest in identity['harness_before'].items():
    backup=base/'harness-before'/Path(relative).name;assert hashlib.sha256(backup.read_bytes()).hexdigest()==digest;shutil.copyfile(backup,bundle/relative);restored[relative]=hashlib.sha256((bundle/relative).read_bytes()).hexdigest();assert restored[relative]==digest
cleanup=dict(native_pids_absent=hosts,stopped_bundle_restored=restored,release=release,controller_failure_preserved=True,restored_host_exit_code=None,corrected_close=json.loads((out/'corrected-close.json').read_text()),fresh_native_reopen_exit_code=0)
(out/'cleanup.json').write_text(json.dumps(cleanup,indent=2)+'\n')

spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
spec=importlib.util.spec_from_file_location('frozen',root/'scripts/freeze-electrical-reference.py');frozen=importlib.util.module_from_spec(spec);spec.loader.exec_module(frozen)
events={e['name']:e['state'] for e in r['events']}
for name,expected in r['reference'].items():
    state=events[name]
    for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:assert state.get(key)==expected.get(key),(name,key)
    for key in ['result','npn','mixed','transient','scope','trace_focus']:p.compare(state.get(key),expected.get(key),name+'.'+key)
manual=out/'first/Manual LED.elab';capsule=base/'recovery'/(r['automatic_candidate']['id']+'.elab-recovery')
assert hashlib.sha256(manual.read_bytes()).hexdigest()==r['manual_save_sha256'];assert hashlib.sha256(capsule.read_bytes()).hexdigest()==r['automatic_capsule_sha256'];assert capsule.read_bytes()==(out/'Original automatic snapshot.elab-recovery').read_bytes()
states={};independent=[]
cases=[('native baseline',manual,events['loaded-paused'],1000),('agent baseline',out/'Agent baseline.elab',events['loaded-paused'],1000),('native recovered',out/'restored/Recovered LED.elab',reopen['reference'],2200),('agent recovered',out/'Agent recovered.elab',reopen['reference'],2200)]
for label,path,expected,resistance in cases:
    client=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']))
    try:
        blank=client.ask('simulation.wait');assert not blank['components'] and not blank['wires'];client.ask('project.load',dict(path=str(path)),True);state=client.ask('simulation.wait')
        assert state['paused'] and not state['document']['dirty'] and not state['result']['stale']
        for key in ['components','wires','nets','transient_settings']:assert state[key]==expected[key],(label,key)
        for key in ['result','scope','transient','npn','mixed']:p.compare(state.get(key),expected.get(key),label+'.'+key)
        current_reference=float(frozen.led(Decimal(resistance))['current_a']);actual=state['result']['current_a'];assert abs(actual-current_reference)<=.01*abs(current_reference)
        independent.append(dict(case=label,resistance_ohm=resistance,current_a=actual,independent_current_a=current_reference,relative_error=abs(actual-current_reference)/current_reference,passed=True));states[label]=state
    finally:client.close()
client=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']))
try:
    client.ask('recovery.configure',dict(directory=str(base/'recovery'),enabled=False,interval_s=2),True);client.ask('simulation.wait');choice=r['automatic_candidate'];inspection=client.ask('recovery.inspect',dict(id=choice['id'],sha256=choice['sha256']));assert inspection['candidate']['owner_status']=='inactive';assert inspection['document']['graph']['components']==events['edited-fresh']['components'];assert inspection['document']['paused']
finally:client.close()
with (out/'fresh-agent-reopen.json.gz').open('wb') as f:
    with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write((json.dumps(dict(passed=True,fresh_reopen_processes=4,states=states,independent_led_checks=independent,original_capsule_validated_after_native_closure=True),indent=2)+'\n').encode())
cleanup.update(fresh_reopen_processes=4,manual_save_sha256=r['manual_save_sha256'],original_capsule_sha256=r['automatic_capsule_sha256'],original_capsule_preserved=True,manual_save_preserved=True,release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json'))
(out/'cleanup.json').write_text(json.dumps(cleanup,indent=2)+'\n');(out/'independent-led-checks.json').write_text(json.dumps(independent,indent=2)+'\n')
print('Recovery graph/history/result proofs, four fresh reopens and independent LED references pass; all three native PIDs absent, original manual save/capsule unchanged, stopped bundle restored and /161 reverified. Original controller failure and missing second-host exit status remain explicit.')

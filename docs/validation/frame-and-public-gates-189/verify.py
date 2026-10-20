"""Read-only archived report/source replay; no subprocess, app, test or capture."""
from pathlib import Path,PurePosixPath
import gzip,hashlib,io,json,re,stat,tarfile
P=Path(__file__).resolve().parent;R=P.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
raw=lambda n:gzip.decompress((P/'raw'/n).read_bytes())
read=lambda n:json.loads(raw(n))
cp=json.loads((P/'checkpoint.json').read_text());actual={str(p.relative_to(P)) for p in P.rglob('*') if p.is_file() and p.name!='checkpoint.json' and '__pycache__' not in p.parts}
assert actual==set(cp['files']) and all(sha(P/n)==h for n,h in cp['files'].items())
m=json.loads((P/'manifest.json').read_text())
for n,v in m['files'].items():
 data=gzip.decompress((P/n).read_bytes());assert len(data)==v['bytes'] and hashlib.sha256(data).hexdigest()==v['sha256']
meta=m['public_archive'];archive=P/meta['path'];assert archive.stat().st_size==meta['bytes'] and sha(archive)==meta['sha256']
entries={}
with tarfile.open(archive,mode='r:gz') as tar:
 for member in tar:
  path=PurePosixPath(member.name);assert member.isfile() and not path.is_absolute() and '..' not in path.parts and str(path)==member.name and member.name not in entries
  e=meta['entries'][member.name];assert member.size==e['bytes'];data=tar.extractfile(member).read();assert len(data)==e['bytes'] and hashlib.sha256(data).hexdigest()==e['sha256'];entries[member.name]=data
assert set(entries)==set(meta['entries']) and len(entries)==713 and sum(len(v) for v in entries.values())==47785345
r=json.loads((P/'report.json').read_text());f=read('frame/report.json.gz');c=read('public/controller.json.gz');assert r['frame_smoke']==f and r['public_controller']==c
assert f['status']==c['status']=='completed' and f['exit_code']==c['exit_code']==0 and c['public_passed']
assert f['owned_process_absent'] and f['postprocess_ps_returncode']==1 and f['postprocess_command']=='' and int(raw('frame/native.pid.gz'))==f['owned_pid']
assert not any(r[k] for k in ('qualified_replacement','native_time_previous_closed','blank_window_cause_confirmed','external_capture_performed','new_version'))
assert not f['native_time_previous_closed'] and not c['native_time_previous_closed'] and not f['qualified_replacement'] and not c['qualified_replacement']
assert json.loads(raw('frame/stdout.txt.gz'))=={'event':'elab.native.smoke','passed':True}
log=raw('frame/stderr.txt.gz').decode();source=raw('sources/smoke.go.txt.gz').decode()
expected=['construction and first GPU frame']+re.findall(r'setStage\("([^"]+)"\)',source)
notes={'nativeSchematicKeyboardSmoke':'schematic keyboard refused agent selection and moved fresh LED one grid step','nativeSchematicPointerSmoke':'schematic pointer refused replacement and accepted fresh LED at 120,80'}
expected_log=[]
for stage in expected:
 expected_log.append(stage)
 if stage in notes:expected_log.append(notes[stage])
stages=re.findall(r'smoke: ([^\n]+)',log);assert stages==expected_log and len(expected)==48 and len(stages)==50 and stages[-1]=='nativeLifecycleSmoke'
assert 'if colored < 500' in source and 'image, err := g.Snapshot()' in source
for data,report in [('frame',f),('public',c)]:
 assert hashlib.sha256(raw(data+'/stdout.txt.gz')).hexdigest()==report['stdout_sha256'] and hashlib.sha256(raw(data+'/stderr.txt.gz')).hexdigest()==report['stderr_sha256']
 assert report['shared_geometry_before']==report['shared_geometry_after'] and all(report[k] for k in ('shared_geometry_unchanged','candidate_unchanged','copied_bundle_unchanged','prior_review_unchanged','canonical_unchanged'))
 assert all(v['denied'] for v in read(data+'/geometry-control.json.gz').values())
 assert hashlib.sha256(raw(data+'/runtime.sb.gz')).hexdigest()==report['policy_sha256']
public=json.loads(entries['report.json']);assert public['passed'] and [a['name'] for a in public['adapters']]==['in-process','headless-stdio','headless-socket','native-socket']
for a in public['adapters']:
 assert a['equivalent_to_in_process'] and len(a['workflows'])==3
 for k,count in [('electrical',294),('inductors',22),('opamps',931),('opamp_storage',752),('capacity',4900)]:assert len(a[k]['checks'])==count
assert hashlib.sha256(entries['report.json']).hexdigest()==c['public_report_sha256']
exits=[json.loads(v) for n,v in entries.items() if PurePosixPath(n).name=='process-exit.json'];assert len(exits)==102 and all(e['exit_code']==0 and not e['graceful_wait_timed_out'] for e in exits)
closes=[json.loads(v)['response'] for n,v in entries.items() if PurePosixPath(n).name=='shutdown-response.json'];assert len(closes)==112
for response in closes:
 assert 'error' not in response;j=response['result']['shutdown'];assert j['status']=='completed' and all(j[k] for k in ('workers_joined','exit_ready','recovery_finalized'))
assert not any(PurePosixPath(n).name=='endpoint.json' for n in entries)
labels=[]
for line in log.splitlines():
 at=line.find('{"event":')
 if at<0:continue
 event=json.loads(line[at:])
 if event['event']=='elab.inspector-width.smoke':assert event['passed'];labels=event['labels']
assert len(labels)==11 and all(l['complete_text'] and not l['truncated'] and l['actual_width_px']<=l['frame_width_px'] and l['shaped_right_px']<=l['frame_width_px'] and l['frame_width_px']<=l['group_width_px'] for l in labels)
q=json.loads((R/'docs/validation/recorded-time-first-settlement/report.json').read_text())['quality'];assert q['exit_code']==0 and q['sources_unchanged'] and len(q['source_sha256'])==670 and all(sha(R/n)==h for n,h in q['source_sha256'].items())
assert raw('sources/smoke.go.txt.gz')==(R/'desktop/smoke.go').read_bytes()
assert raw('sources/mesh_canvas.go.txt.gz')==(R/'build/.vellum-sdk-b22478c/drivers/cogent/mesh_canvas.go').read_bytes()
for key,path in [('source_proof_checkpoint_sha256','docs/validation/recorded-time-first-settlement/checkpoint.json'),('native_unavailable_checkpoint_sha256','docs/validation/native-recorded-transport-189/checkpoint.json'),('prior_release_receipt_sha256','research/ui-audit/first-version/recorded-step-review-checkpoint.json'),('sdk_lock_sha256','third_party/vellum/sdk.lock.json')]:assert sha(R/path)==r[key]
b=json.loads((R/'build/first-version-189-tools/build-result-signed.json').read_text());assert r['source_commit']==f['source_commit']==c['source_commit']==b['source_commit']
def tree(folder):
 out={}
 for p in [folder,*sorted(folder.rglob('*'))]:
  assert not p.is_symlink() and (p.is_dir() or p.is_file());out[str(p.relative_to(folder))]={'mode':stat.S_IMODE(p.stat().st_mode),'sha256':sha(p) if p.is_file() else None}
 return out
for path in [Path(f['candidate'])/'ELab.app',Path('/private/tmp/elab-frame-diagnostic-189/ELab.app'),Path('/private/tmp/elab-public-contracts-189/ELab.app')]:assert tree(path)==b['source_bundle']
assert sha(Path(f['candidate'])/'elab-agent')==b['agent_sha256']==c['agent_sha256']
assert sha(R/'build/first-version-189-tools/build/elab-conformance')==c['controller_sha256']
assert sha(R/'build/ELab.app/Contents/MacOS/ELab')==f['canonical_sha256']==c['canonical_sha256']
print(json.dumps(dict(evidence_verified=True,source_unchanged=True,internal_smoke_stages=48,diagnostic_log_messages=2,public_adapters=4,normal_child_exits=102,ready_shutdown_responses=112,inspector_labels=11,native_time_previous_closed=False,blank_window_cause_confirmed=False,qualified_replacement=False,app_started=False)))

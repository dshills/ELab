"""Read-only native-failure evidence replay; no app, test, capture or UI access."""
from pathlib import Path
import gzip,hashlib,json,math,stat,sys
if sys.flags.optimize:raise SystemExit('Assertions required')
P=Path(__file__).resolve().parent;R=P.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def raw(name):
 p=P/'raw'/(name+'.gz');return gzip.decompress(p.read_bytes())
def read(name):return json.loads(raw(name))
def members(folder):return {str(p.relative_to(folder)):dict(sha256=sha(p),mode=oct(stat.S_IMODE(p.stat().st_mode))) for p in sorted(folder.rglob('*')) if p.is_file()}
def stable(v):return {k:x for k,x in v.items() if k not in ('view','viewport','revision','command_revision','last_actor')}
cp=json.loads((P/'checkpoint.json').read_text())['files'];assert set(cp)=={str(p.relative_to(P)) for p in P.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
for n,h in cp.items():
 p=P/n;assert not p.is_symlink() and p.resolve().is_relative_to(P) and sha(p)==h,n
for n,info in json.loads((P/'manifest.json').read_text()).items():
 p=P/n;data=gzip.decompress(p.read_bytes()) if info['compressed'] else p.read_bytes();assert sha(p)==info['stored_sha256'] and len(data)==info['original_bytes'] and hashlib.sha256(data).hexdigest()==info['original_sha256'],n
report=json.loads((P/'report.json').read_text());assert report['issue']=='RECORD-NATIVE-ORDER-01' and report['native_failure_reproduced'] and not report['native_repaired'] and not report['planned_native_batch_completed'] and not report['source_changed'] and not report['new_version']
receipt_path=R/'research/ui-audit/first-version/recorded-step-review-checkpoint.json';assert sha(receipt_path)==report['receipt_sha256'];receipt=json.loads(receipt_path.read_text());assert receipt['source_commit']==report['base_source_commit'] and receipt['native_sha256']==report['native_sha256'] and receipt['agent_sha256']==report['agent_sha256']
review=Path(receipt['review_path']);actual=members(review);actual.pop('RELEASE_MANIFEST.json');assert actual==json.loads((review/'RELEASE_MANIFEST.json').read_text())['files'] and sha(review/'RELEASE_MANIFEST.json')==report['release_manifest_sha256']
assert len(report['source_go_sha256'])==666 and report['source_go_sha256']==receipt['quality']['source_sha256'] and all(sha(R/n)==h for n,h in report['source_go_sha256'].items())
quality=json.loads(gzip.decompress((R/'docs/validation/recorded-step-settlement/raw/stage/full-quality.json.gz').read_bytes()));assert quality['exit_code']==0 and quality['sources_unchanged'] and len(quality['source_sha256'])==669 and all(sha(R/n)==h for n,h in quality['source_sha256'].items())
assert sha(R/'build/ELab.app/Contents/MacOS/ELab')==report['canonical_sha256'] and sha(R/'third_party/vellum/sdk.lock.json')==report['sdk_lock_sha256'] and sha(R/receipt['sdk']['archive'])==report['sdk_archive_sha256']
for p in (P/'sources').rglob('*.txt'):assert p.read_bytes()==(R/str(p.relative_to(P/'sources'))[:-4]).read_bytes()
a,b,c,d,e,f=(read('original/'+n+'.json') for n in ('before-index','index-private','index-next','time-private','time-previous-refused','time-enter-control'))
g=read('original/time-enter-previous-control.json');assert a==b and c==d
assert c['recording']['sample_index']==4 and c['recording']['time_s']==.004 and c['last_actor']=='user' and c['command_revision']==a['command_revision']+2
assert {k:v for k,v in stable(a).items() if k!='recording'}=={k:v for k,v in stable(c).items() if k!='recording'}
assert stable(d)==stable(e) and e['recording']['sample_index']==4 and e['command_revision']==d['command_revision']+1
assert f['recording']['sample_index']==3 and f['recording']['time_s']==.003 and g['recording']['sample_index']==2 and g['recording']['time_s']==.002
assert not any(v['recording']['playing'] for v in (a,b,c,d,e,f,g)) and all(v['paused'] for v in (a,b,c,d,e,f,g))
assert b'Recorded playback entries changed. Review them and try again.' in raw('original/time-previous-ui.ax.txt')
assert b'Value: 3 ms' in raw('original/time-private.ax.txt') and b'sample 4 / 1000' in raw('original/time-private.ax.txt')
original_file=raw('original/fixture.elab');assert original_file==raw('original/fixture-final.elab')
rows=read('original/capture-rows.json');metadata=read('original/capture-metadata.json');assert len(rows)==1001 and metadata['input_mode']=='charge' and metadata['settings']['initial_voltage_v']==0
ve=ie=0
for row in rows:
 decay=math.exp(-row['time_s']/.01);ve=max(ve,abs(row['values'][0]-5*(1-decay)));ie=max(ie,abs(row['values'][1]-.0005*decay))
assert ve<1e-12 and ie<1e-15
reference=read('reference/report.json');assert reference['passed'] and reference['rows_restored_exact']==reference['independent_rc_rows']==1001 and reference['max_voltage_error_v']==ve and reference['max_current_error_a']==ie and reference['original_source_unchanged']
assert reference['native_matched_states']==['index-next','time-enter-control','time-enter-previous-control'] and reference['native_failed_time_previous_index']==4 and reference['reference_time_previous_index']==2
assert hashlib.sha256(original_file).hexdigest()==reference['source_sha256']
assert reference['shutdown']['status']=='completed' and reference['shutdown']['workers_joined'] and reference['shutdown']['exit_ready'] and read('reference/reference-bridge-exit.json')['exit_code']==0
original,expired,safe=(read(label+'/controller.json') for label in ('original','expired-diagnostic','safe-diagnostic'))
for ctl in (original,expired,safe):assert ctl['owned_process_absent'] and ctl['endpoint_removed'] and not ctl['forced_cleanup'] and ctl['shared_geometry_unchanged'] and ctl['review_unchanged'] and ctl['copied_app_unchanged'] and ctl['canonical_unchanged'] and ctl['open_exit_code']==0 and ctl['native_exit_status'] is None
assert expired['status']=='failed' and 'owned session ended or expired' in expired['failure'] and not expired.get('cleanup_error')
assert safe['default_entry_protected'] and safe['observation_ttl_removed'] and safe['duplicate_default_launch_refused']==73 and safe['base_runtime_unchanged']
extra=read('expired-diagnostic/unexpected-extra-cleanup.json');assert extra['pid']==61551 and extra['owned_extra_process_absent'] and extra['signal']=='SIGTERM' and extra['exit_status_unavailable']
lease=read('expired-diagnostic/unexpected-extra-lease-cleanup.json');assert lease['owned_pid']==61551 and lease['bytes']==50 and lease['reversible_move'] and not lease['remaining_recovery_entries']
assert hashlib.sha256(raw('expired-diagnostic/unexpected-extra-recovery-lease.json')).hexdigest()==lease['sha256']
jsonkey='window-geometry-0.3.6.json';lst='window-geometry-0.3.6.lst'
assert original['shared_geometry_before'][jsonkey]==safe['shared_geometry_after'][jsonkey] and original['shared_geometry_before'][lst]!=safe['shared_geometry_after'][lst]
assert hashlib.sha256(raw('expired-diagnostic/after-extra-'+lst)).hexdigest()==safe['shared_geometry_before'][lst]['sha256']
probe=read('safe-diagnostic/case-observations.json');assert probe['window_pixels_blank'] and not probe['native_ordering_conclusion'] and probe['native_coordinate_reproduction_not_attempted'] and not probe['source_guard_changed']
decommission=json.loads(gzip.decompress((P/'producers/diagnostic/decommission.json.gz').read_bytes()));assert decommission['default_safe_wrapper_stop_refusal_exit']==75 and decommission['reversible_permission_change']
for entry in decommission['files']:
 p=Path(entry['path']);assert sha(p)==entry['sha256'] and oct(stat.S_IMODE(p.stat().st_mode))==entry['mode_after'] and stat.S_IMODE(p.stat().st_mode)&0o111==0 and entry['bytes_unchanged']
print(json.dumps(dict(evidence_verified=True,native_failure_reproduced=True,native_repaired=False,matched_native_controls=3,independent_rc_rows=1001,application_go_sources_unchanged=666,diagnostic_rendered_reproduction_available=False,geometry_json_unchanged=True,last_save_marker_changed=True,app_started=False)))

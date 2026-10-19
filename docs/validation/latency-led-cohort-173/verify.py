"""Read-only complete-ledger, source, archive, image and cleanup replay."""
from pathlib import Path
import gzip,hashlib,json,statistics,subprocess,sys

folder=Path(__file__).resolve().parent;root=folder.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def read(p):return json.loads(p.read_bytes() if p.exists() else gzip.decompress(p.with_name(p.name+'.gz').read_bytes()))
checkpoint=read(folder/'checkpoint.json')
assert {p.relative_to(folder).as_posix():sha(p) for p in folder.rglob('*') if p.is_file() and p.name!='checkpoint.json'}==checkpoint['evidence_sha256']
assert all(sha(root/name)==digest for name,digest in checkpoint['recorder_source_sha256'].items())
assert all(sha(Path(name))==digest for name,digest in checkpoint['preserved_artifacts'].items())
quality=read(root/'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256'])==650
assert all(sha(root/name)==digest for name,digest in quality['source_sha256'].items())
plan=read(root/'research/37-visible-latency-cohort-plan.json')
assert len(plan['trials'])==60 and plan['cohort_trials_run']==0  # immutable precollection declaration
assert [x['id'] for x in plan['trials'] if x['reference']=='led']==[f'led-{i:02}' for i in range(1,21)]
summary=read(folder/'cohort-outcomes.json');outcomes=summary['outcomes']
assert [x['id'] for x in outcomes]==[f'led-{i:02}' for i in range(1,21)]
assert summary['scheduled']==summary['attempted']==20 and summary['available']==14 and summary['unavailable']==6
assert summary['replacements']==0 and summary['full_cohort_p95'] is summary['performance_gate_passed'] is None
available=[x for x in outcomes if x['available']]
assert [x['id'] for x in available]==[f'led-{i:02}' for i in range(7,21)]
for phase in ['initial','remaining']:
    report=read(folder/phase/'report.json')
    assert all(report[k] for k in ['owned_process_absent','endpoint_removed','baseline_file_unchanged',
        'shared_geometry_unchanged','review_unchanged','copied_bundle_unchanged','canonical_unchanged'])
    assert not report['forced_cleanup'] and report['open_exit_code']==0 and report['native_exit_status'] is None
    close=read(folder/phase/'close-response.json')
    assert close['closed'] and close['shutdown']['exit_ready'] and close['shutdown']['status']=='completed'
    assert all(read(folder/phase/'host-before.json')[key]==plan['expected_host'][key] for key in ['model','os_version','os_build'])
assert read(folder/'remaining/cua-postflight.json')['running_elab']==[]
revision=read(folder/'remaining/producer-revision.json')
for phase,key in [('initial','previous_producer_sha256'),('remaining','current_producer_sha256')]:
    assert hashlib.sha256(gzip.decompress((folder/phase/'producers/worker.py.gz').read_bytes())).hexdigest()==revision[key]
assert revision['replacement_attempts']==0 and not revision['app_or_recorder_changed']
assert revision['remaining_ids']==[f'led-{i:02}' for i in range(7,21)]
frames=images=0
for outcome in outcomes:
    trial=folder/outcome['phase']/'trials'/outcome['id'];original=read(trial/'outcome.json')
    assert original['status']==outcome['original_status']
    if outcome['analysis'] is None:
        assert outcome['id'] in [f'led-{i:02}' for i in range(2,7)] and not (trial/'capture').exists()
        assert 'component ID is already used in this session' in original['reason']
        continue
    analysis=read(trial/'analysis.json');assert analysis==outcome['analysis']
    assert (analysis['measurement_status']=='available')==outcome['available']
    assert sha(trial/(outcome['id']+'-baseline.elab'))==original['baseline_sha256']
    if outcome['id']=='led-01':assert original['reason']=='AssertionError: wires' and not outcome['available']
    else:
        assert (trial/(outcome['id']+'-final.elab')).exists()
    frames+=analysis['complete_frames'];images+=analysis['unique_images']
    subprocess.run([sys.executable,'-B',str(folder/'analyze.py'),str(trial),'--verify'],check=True,stdout=subprocess.DEVNULL)
assert frames==summary['complete_image_frames']==2462
assert images==summary['original_pngs']==2317
for key in ['marker_to_committed_ms','marker_to_fresh_ms']:
    values=[r['analysis'][key] for r in available]
    assert summary['available_only_descriptive'][key]==dict(min=min(values),median=statistics.median(values),max=max(values))
subprocess.run([sys.executable,'-B',str(folder/'check-marker-order.py'),str(folder/'remaining/trials/led-08')],check=True)
assert summary['eligible_latency_observations']==0
print(dict(passed=True,scheduled_outcomes=20,available_bounds=14,unavailable=6,
    complete_frames=frames,original_pngs=images,full_cohort_p95=None,go_sources_unchanged=650))

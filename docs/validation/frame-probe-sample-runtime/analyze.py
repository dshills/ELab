from pathlib import Path
import argparse
import gzip
import hashlib
import json
import struct

parser=argparse.ArgumentParser();parser.add_argument('--artifacts',type=Path,default=Path('/private/tmp/elab-frame-probe-sample-runtime-171'));parser.add_argument('--verify',action='store_true');args=parser.parse_args();out=args.artifacts;cap=out/'capture'
def contents(path):return path.read_bytes() if path.exists() else gzip.decompress(Path(str(path)+'.gz').read_bytes())
def read(path):return json.loads(contents(path))
report=read(cap/'report.json');action=read(out/'capture-action.json');rows=[json.loads(line) for line in contents(cap/'frames.jsonl').decode().splitlines()];frames=[row for row in rows if row['event']=='frame'];markers=[row for row in rows if row['event']=='marker']
assert len(frames)==report['frames_received']==report['complete_frames']==173 and len(markers)==report['marker_count']==1
assert [row['index'] for row in frames]==list(range(1,174))
assert report['collector_error'] is report['command_error'] is None
assert markers[0]==action['marker_acknowledgment'] and markers[0]['id']=='sample_runtime01'
assert action['input_call_invoked'] and action['input_call_completed'] and action['after_ax_has_expected_readout'] and not action.get('error')
assert [e['name'] for e in action['events']]==['capture_start_requested','baseline_image_received','marker_requested','complete_acknowledgment_validated','input_call_will_invoke','input_call_completed','after_input_ax_observed']
assert [e['sequence'] for e in action['events']]==list(range(1,8))
annotations=read(out/'image-annotations.json');assert set(annotations['images'])=={row['png_sha256'] for row in frames}
image_bytes={}
for row in frames:
    assert row['status']==0 and row['eligible_for_latency'] is False
    data=(cap/row['png']).read_bytes();digest=hashlib.sha256(data).hexdigest();assert digest==row['png_sha256'] and row['png']=='frame-'+digest+'.png'
    assert struct.unpack('>II',data[16:24])==(1053,802) and data[:8]==b'\x89PNG\r\n\x1a\n'
    image_bytes[row['png']]=len(data)
assert len(image_bytes)==report['unique_images']==5 and sum(image_bytes.values())==report['image_bytes']
display=[int(row['display_mach_ticks']) for row in frames];assert all(a<b for a,b in zip(display,display[1:]));assert all(int(row['received_mach_ticks'])>=int(row['display_mach_ticks']) for row in frames)
assert (report['timebase_numerator'],report['timebase_denominator'])==(125,3)
ratio=report['timebase_numerator']/report['timebase_denominator'];interval=lambda row:(int(row['display_mach_ticks'])-int(markers[0]['mach_ticks']))*ratio/1e6
stale=next(row for row in frames if annotations['images'][row['png_sha256']]['role']=='committed-value-stale-result')
fresh=next(row for row in frames if annotations['images'][row['png_sha256']]['role']=='committed-value-fresh-result')
assert stale['index']==11 and fresh['index']==14
before=read(out/'before-apply.json');after=read(out/'after-apply.json');expected=read(out/'agent-expected.json')
assert before['circuit']['resistance_ohm']==1000 and after['circuit']['resistance_ohm']==2000
assert after['components']==expected['components'] and after['wires']==expected['wires'] and after['result']==expected['result'] and not after['result']['stale']
assert after['run_id']==before['run_id']+1 and after['electrical_revision']==before['electrical_revision']+1
worker=read(out/'recording-worker.json');assert worker['exit_code']==0 and worker['recorder_sha256']=='dcf06ebefa9ec76002f22f0e67564682b1c429af2f19fc208c504c669ec505e7'
session=read(out/'report.json');postflight=read(out/'postflight.json');coordination=read(out/'cua-coordination.json')
assert report['executable_sha256']==session['native_sha256']=='a7a70f595a6cd117a16654592255ef0f893294c76f5dc52a2c56e08ce1427a03'
assert session['exact_target_identity_established'] and session['same_pid_after_exec']
assert session['registration']['bundle_identifier']=='com.dshills.elab' and len(session['target_windows'])==1
assert report['pid']==session['owned_pid']==session['registration']['pid']==session['target_windows'][0]['owner_pid']==coordination['owned_pid']
assert report['window_id']==session['target_windows'][0]['window_id']==coordination['owned_window_id']
assert session['controls']['read']['denied'] and session['controls']['write_without_truncate']['denied']
assert session['owned_process_absent'] and session['endpoint_removed'] and not session['forced_cleanup']
assert session['shared_geometry_unchanged'] and session['copied_bundle_unchanged'] and session['review_unchanged'] and session['canonical_unchanged']
assert session['open_exit_code']==0 and session['native_exit_status'] is None
assert hashlib.sha256(contents(out/'baseline.elab')).hexdigest()==session['baseline_file_sha256'] and session['baseline_file_unchanged']
assert postflight['all_owned_pids_absent'] and postflight['endpoint_removed'] and postflight['recorder_exit_code']==0
assert postflight['capture_directory_mode']=='0o700' and set(postflight['capture_file_modes'].values())=={'0o600'}
assert coordination['running_elab_inventory_count']==1 and coordination['final_running_elab_inventory_count']==0
assert coordination['observed_result']['input_call_completed'] and coordination['observed_result']['expected_readout'] and coordination['observed_result']['error'] is None
for digest,annotation in annotations['images'].items():
    assert next(row['index'] for row in frames if row['png_sha256']==digest)==annotation['first_frame']
result=dict(status='sample_checked_recorder_nominal_input_capture_calibrated_with_limits',frames=len(frames),unique_images=len(image_bytes),marker=markers[0],first_committed_value_with_stale_label_frame=stale['index'],first_fresh_value_and_result_frame=fresh['index'],marker_to_stale_label_ms=interval(stale),marker_to_fresh_value_and_result_ms=interval(fresh),interval_scope='Marker-to-Window-Server-frame arithmetic only; not OS input/accepted-edit latency or an eligible bound',controller_order_verified=True,one_cua_apply_call=True,headless_result_equal=True,single_electrical_revision_and_run_increment=True,display_times_strictly_increasing=True,display_time_no_later_than_receipt=True,presentation_to_mach_max_abs_difference_s=max(abs(row['presentation_seconds']-int(row['display_mach_ticks'])*ratio/1e9) for row in frames),all_frames_in_onscreen_stack_at_receipt=all(row['target_in_onscreen_stack_at_receipt'] for row in frames),recorder_exit_code=0,display_visibility_validated=False,physical_clock_calibration=False,eligible_latency_observations=0,latency_gate_passed=None)
assert result['presentation_to_mach_max_abs_difference_s']<1e-8
if args.verify:assert read(out/'analysis.json')==result
else:
    with (out/'analysis.json').open('x') as f:f.write(json.dumps(result,indent=2)+'\n')
print(json.dumps({key:result[key] for key in ['status','frames','unique_images','first_committed_value_with_stale_label_frame','first_fresh_value_and_result_frame','marker_to_fresh_value_and_result_ms','eligible_latency_observations']}))

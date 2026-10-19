from pathlib import Path
import argparse
import gzip
import hashlib
import json
import struct

parser=argparse.ArgumentParser();parser.add_argument('--artifacts',type=Path,default=Path('/private/tmp/elab-marker-input-calibration-170'));parser.add_argument('--verify',action='store_true');args=parser.parse_args()
out=args.artifacts;cap=out/'capture'
def contents(path):
    return path.read_bytes() if path.exists() else gzip.decompress(Path(str(path)+'.gz').read_bytes())
def read(path):return json.loads(contents(path))
report=read(cap/'report.json');action=read(out/'capture-action.json');rows=[json.loads(line) for line in contents(cap/'frames.jsonl').decode().splitlines()]
frames=[row for row in rows if row['event']=='frame'];markers=[row for row in rows if row['event']=='marker']
assert len(frames)==report['frames_received']==report['complete_frames']==174 and len(markers)==report['marker_count']==1
assert [row['index'] for row in frames]==list(range(1,175))
assert report['collector_error'] is None and report['command_error'] is None
marker=markers[0];assert marker==action['marker_acknowledgment'] and marker['id']=='led_apply_terminal01' and marker['frame_count_before']==2
assert action['input_call_invoked'] and action['input_call_completed'] and action['after_ax_has_expected_readout'] and not action.get('error')
assert [event['name'] for event in action['events']]==['baseline_image_received','marker_requested','complete_acknowledgment_validated','input_call_will_invoke','input_call_completed','after_input_ax_observed']
assert [event['sequence'] for event in action['events']]==list(range(1,7))
# Image semantics are manual observations of the four exact PNGs shown through
# view_image in this conversation, not an OCR classifier or automatic oracle.
annotations={
 'b509e41f704fac5203dc10d06a3fc941f01174744b027119749818f20bd143c1':dict(schematic_resistance_ohm=1000,led_current_label='2.917 mA',draft='2k',apply_button='enabled',accepted_result=False),
 '3a80081a77a6c3a88c2bedb8799fa50c60eb6be5ea0a45a826b2b180913cc75b':dict(schematic_resistance_ohm=1000,led_current_label='2.917 mA',draft='2k',apply_button='dimmed',accepted_result=False),
 'd429ccd028949a4ac2c1276d5cfd872be85824483df67635fa2776fa8db99d3e':dict(schematic_resistance_ohm=2000,led_current_label='1.477 mA',draft='2 kΩ',apply_button='enabled',accepted_result=True,caret='visible'),
 '41b31faa1980765c3c4d5345398a3e9893a9006b59969727aabbcc9f85f6b347':dict(schematic_resistance_ohm=2000,led_current_label='1.477 mA',draft='2 kΩ',apply_button='enabled',accepted_result=True,caret='hidden'),
}
assert set(annotations)=={row['png_sha256'] for row in frames}
image_files={}
for row in frames:
    assert row['status']==0 and not row['eligible_for_latency']
    path=cap/row['png'];data=path.read_bytes();digest=hashlib.sha256(data).hexdigest()
    assert digest==row['png_sha256'] and path.name=='frame-'+digest+'.png'
    assert data[:8]==b'\x89PNG\r\n\x1a\n' and struct.unpack('>II',data[16:24])==(1053,802)
    image_files[path.name]=len(data)
assert len(image_files)==report['unique_images']==4 and sum(image_files.values())==report['image_bytes']
display=[int(row['display_mach_ticks']) for row in frames]
assert all(left<right for left,right in zip(display,display[1:]))
assert all(int(row['display_mach_ticks'])<=int(row['received_mach_ticks']) for row in frames)
ratio=report['timebase_numerator']/report['timebase_denominator']
assert report['timebase_numerator']==125 and report['timebase_denominator']==3
presentation_error=max(abs(row['presentation_seconds']-int(row['display_mach_ticks'])*ratio/1e9) for row in frames)
assert presentation_error<1e-8
interval=lambda row:(int(row['display_mach_ticks'])-int(marker['mach_ticks']))*ratio/1e6
first_accepted=next(row for row in frames if annotations[row['png_sha256']]['accepted_result'])
assert first_accepted['index']==12
first_dimmed=next(row for row in frames if annotations[row['png_sha256']]['apply_button']=='dimmed')
before=read(out/'before-apply.json');after=read(out/'after-apply.json');expected=read(out/'agent-expected.json')
assert before['circuit']['resistance_ohm']==1000 and after['circuit']['resistance_ohm']==2000
assert after['components']==expected['components'] and after['wires']==expected['wires'] and after['result']==expected['result']
assert not after['result']['stale'] and after['run_id']==before['run_id']+1 and after['electrical_revision']==before['electrical_revision']+1
assert read(out/'recording-worker.json')['exit_code']==0
expired=out/'attempt-terminal-expired';old=read(expired/'capture/report.json');timing=read(expired/'timing.json')
assert old['marker_count']==0 and not read(expired/'capture-action.json')['input_call_invoked']
assert timing['marker_request_mtime_ns']>timing['report_mtime_ns']
result=dict(status='one_led_marker_input_causal_frame_sequence_calibrated_with_limits',frames=len(frames),unique_images=len(image_files),png_dimensions=[1053,802],marker=marker,controller_order_verified=True,one_cua_apply_call=True,source_of_image_annotations='Manual inspection of all four exact PNGs using view_image in this conversation',image_annotations=annotations,first_dimmed_button_frame=first_dimmed['index'],first_dimmed_button_marker_interval_ms=interval(first_dimmed),first_accepted_value_and_result_frame=first_accepted['index'],first_accepted_value_and_result_png=first_accepted['png'],marker_to_first_accepted_image_ms=interval(first_accepted),interval_scope='Marker-to-Window-Server-frame arithmetic only; not input-event/accepted-edit latency or an eligible upper bound',presentation_to_mach_max_abs_difference_s=presentation_error,display_times_strictly_increasing=True,display_time_no_later_than_receipt=True,all_frames_in_onscreen_stack_at_receipt=all(row['target_in_onscreen_stack_at_receipt'] for row in frames),display_visibility_validated=False,physical_clock_calibration=False,headless_result_equal=True,single_electrical_revision_and_run_increment=True,baseline_current_a=before['result']['current_a'],accepted_current_a=after['result']['current_a'],recorder_exit_code=0,eligible_latency_observations=0,latency_gate_passed=None,failures_retained=['Node-host Screen Recording preflight refusal and unavailable process global before input','Terminal recording expired before marker arrival; no input or acknowledgment'],remaining=['Unobscured display visibility at the recorded timestamp','Physical clock and pixel-scale calibration','Stream error, idle/blank/dropped frame and budget/fault calibration','Cohorts of at least 20 eligible observations per LED, RC and resistor-loaded NPN, plus difficult-solve interaction coverage'])
if args.verify:
    assert read(out/'analysis.json')==result,'Stored analysis differs from recomputed evidence'
else:
    with (out/'analysis.json').open('x') as f:f.write(json.dumps(result,indent=2)+'\n')
print(json.dumps({key:result[key] for key in ['status','frames','unique_images','first_accepted_value_and_result_frame','marker_to_first_accepted_image_ms','headless_result_equal','eligible_latency_observations']}))

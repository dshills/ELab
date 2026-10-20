"""Read-only composition/causality replay with in-memory corruption checks."""
from pathlib import Path
from collections import Counter
from fractions import Fraction
from functools import lru_cache
from PIL import Image
import copy,gzip,hashlib,io,json,math,struct,sys,tarfile

if sys.flags.optimize:raise SystemExit('Verification requires assertions')
folder=Path(sys.argv[1])
proof=Path('/Users/dshills/Development/projects/ELab/docs/validation/latency-followup-173/led')
sys.path.insert(0,str(Path(__file__).resolve().parent))
from frame_store import load_images
images=load_images(folder/'capture')
def contents(path):
    return path.read_bytes() if path.exists() else gzip.decompress(path.with_name(path.name+'.gz').read_bytes())
def read(name):return json.loads(contents(folder/name))
def png(name):
    assert Path(name).name==name and name.endswith('.png')
    return images[name]

marker_name='capture/marker-'+folder.name.replace('-','_')+'.json'
inputs={name:(json.loads((proof/name).read_bytes()) if name=='image-annotations.json' else read(name)) for name in ['capture/report.json','capture/configuration.json','capture-action.json',
    marker_name,'before-apply.json','after-apply.json','agent-expected.json',
    'image-annotations.json','inspection.json','outcome.json']}
assert inputs['inspection.json']['exit_code']==0
inputs['inspect-stdout.json']=json.loads(inputs['inspection.json']['stdout'])
inputs['rows']=[json.loads(line) for line in contents(folder/'capture/frames.jsonl').decode().splitlines()]

def independent(state):
    model=state['result']['model'];v=state['circuit']['voltage_v'];r=state['circuit']['resistance_ohm']
    vt=1.380649e-23*(273.15+model['temperature_c'])/1.602176634e-19
    lo=0.;hi=v/r
    for _ in range(160):
        mid=(lo+hi)/2
        drop=model['n']*vt*math.log1p(mid/model['is_a'])+mid*model['rs_ohm']
        if mid*r+drop<v:lo=mid
        else:hi=mid
    expected=(lo+hi)/2
    assert abs(state['result']['current_a']-expected)<1e-12
    assert abs(state['result']['kvl_residual_v'])<1e-10
    return dict(current_a=expected,formula='Independent bisection of declared Shockley LED plus series resistance and KVL')

def validate(data):
    report=data['capture/report.json'];config=data['capture/configuration.json']
    action=data['capture-action.json'];rows=data['rows'];annotation=data['image-annotations.json']
    frames=[r for r in rows if r['event']=='frame'];markers=[r for r in rows if r['event']=='marker']
    assert len(frames)==report['frames_received_within_deadline'] and len(frames)<=300
    assert [r['index'] for r in frames]==list(range(1,len(frames)+1))
    assert len(markers)==report['marker_count']==1
    marker=markers[0];tick=int(marker['mach_ticks'])
    assert marker==action['marker_acknowledgment']==data[marker_name]
    assert marker['id']==folder.name.replace('-','_') and tick>0
    assert marker['frame_count_before']==sum(r['event']=='frame' for r in rows[:rows.index(marker)])>=1
    assert action['input_call_invoked'] and action['input_call_completed'] and action['after_ax_has_expected_readout'] and not action.get('error')
    assert [e['name'] for e in action['events']]==['capture_start_requested','baseline_image_received','marker_requested',
        'complete_acknowledgment_validated','input_call_will_invoke','input_call_completed','after_input_ax_observed']
    assert [e['sequence'] for e in action['events']]==list(range(1,8))
    assert action['events'][1]['frame_index']<=marker['frame_count_before']
    assert report['collector_error'] is report['command_error'] is report['journal_close_error'] is None
    assert report['capture_executed'] and report['geometry_rechecked_after_stop']
    assert report['eligible_latency_observations']==0 and report['latency_gate_passed'] is None
    assert not any(report[k] for k in ['pixel_mapping_calibrated','display_visibility_calibrated','clock_calibrated_for_latency'])
    assert config['capture_plan']==data['inspect-stdout.json']['capture_plan']
    plan=config['capture_plan'];bounds=plan['window_global_points']
    assert bounds==dict(x=2034,y=303,width=1053,height=802) and plan['point_pixel_scale']==1
    assert plan['source_display_local_points']==bounds and config['seconds']==3
    assert config['tool_executable_sha256']=='a8cec4e8f9d16b2b5ad6b751f5657b537c5ed6ea956e7695403168b445bb923f'
    assert config['target_executable_sha256']=='ecd24e458e0db4a0e580747b37145c7d30e9c8f4be686731df78cb690b38a330'
    assert config['filter']=='display-excluding-no-windows' and not any(config[k] for k in ['audio','microphone','cursor','permission_requested'])
    assert data['outcome.json']['recorder_exit_code']==0
    numerator=report['timebase_numerator'];denominator=report['timebase_denominator']
    assert numerator>0 and denominator>0
    scale=Fraction(numerator,denominator);start=int(config['start_mach_ticks']);deadline=int(config['deadline_mach_ticks'])
    assert start<tick<deadline and deadline-start==int(Fraction(3_000_000_000,1)/scale)
    states={state['role']:state for state in annotation['states']}
    assert set(states)=={'private-draft-original-result','committed-value-stale-result','committed-value-fresh-result'}
    baseline=states['private-draft-original-result']['hashes'];stale=states['committed-value-stale-result']['hashes'];fresh=states['committed-value-fresh-result']['hashes']
    assert baseline['resistor']!=fresh['resistor']==stale['resistor']
    assert baseline['current']==stale['current']!=fresh['current']
    for key in ['header_status','reading_status']:assert baseline[key]==fresh[key]!=stale[key]
    cache={};first={};counts=Counter();last=0;future=0;outside=0;complete=0;image_sizes={}
    for row in frames:
        assert row['eligible_for_latency'] is False and row['target_geometry_matches_at_receipt']
        received=int(row['received_mach_ticks']);assert start<=received<=deadline
        assert row['status'] in [0,1,3]
        displayed=int(row['display_mach_ticks']) if row.get('display_mach_ticks') else None
        if displayed is not None:future+=int(displayed>received)
        if row['status']!=0:assert not row.get('png');continue
        assert displayed and displayed>=last;last=displayed
        assert row['presentation_timescale']==1_000_000_000
        assert int(row['presentation_value'])==math.floor(displayed*scale)
        assert row['presentation_matches_converted_display_time']
        assert row['screen_rect_points']==bounds and row['content_rect_points']==dict(x=0,y=0,width=1053,height=802)
        assert row['scale_factor']==row['content_scale']==1
        assert (row['width_pixels'],row['height_pixels'])==(1053,802)
        if displayed>deadline:assert row.get('outside_presentation_window') and not row.get('png');outside+=1;continue
        assert row['frame_geometry_validated']
        raw=png(row['png']);digest=hashlib.sha256(raw).hexdigest()
        assert digest==row['png_sha256'] and row['png']=='frame-'+digest+'.png'
        assert raw[:8]==b'\x89PNG\r\n\x1a\n' and struct.unpack('>II',raw[16:24])==(1053,802)
        image_sizes[row['png']]=len(raw)
        if digest not in cache:
            image=Image.open(io.BytesIO(raw)).convert('RGB')
            hashes={name:hashlib.sha256(image.crop(box).tobytes()).hexdigest() for name,box in annotation['regions'].items()}
            matches=[name for name,state in states.items() if hashes==state['hashes']]
            assert len(matches)==1,'Unclassified or ambiguous visible ROI state'
            cache[digest]=matches[0]
        role=cache[digest];counts[role]+=1;complete+=1
        if role not in first:first[role]=row
        # A pre-ack callback can carry a prospective display timestamp after
        # the marker. Only journal order establishes whether the collector
        # already knew that marker when processing this callback.
        if row['index']>marker['frame_count_before'] and displayed>tick:
            assert abs(row['since_acknowledged_marker_ms']-float((displayed-tick)*scale/1_000_000))<1e-9
        else:assert 'since_acknowledged_marker_ms' not in row
    assert complete==report['complete_frames']>0
    assert future==report['frames_with_display_time_after_receipt'] and outside==report['frames_outside_presentation_window']
    assert len(image_sizes)==report['unique_images'] and sum(image_sizes.values())==report['image_bytes']<=64*1024*1024
    assert 'private-draft-original-result' in first and 'committed-value-fresh-result' in first
    committed=min((r for name,r in first.items() if name!='private-draft-original-result'),key=lambda r:r['index'])
    valid=first['committed-value-fresh-result']
    assert tick<int(committed['display_mach_ticks'])<=int(valid['display_mach_ticks'])
    before=data['before-apply.json'];after=data['after-apply.json'];expected=data['agent-expected.json']
    assert before['circuit']['resistance_ohm']==1000 and after['circuit']['resistance_ohm']==2000
    assert after['electrical_revision']==before['electrical_revision']+1 and after['run_id']==before['run_id']+1
    for key in ['components','nets','model','circuit']:assert after[key]==expected[key]
    assert after['wires']==before['wires']
    wire_semantics=lambda values:[{k:v for k,v in wire.items() if k!='id'} for wire in values]
    assert wire_semantics(after['wires'])==wire_semantics(expected['wires'])
    actual=dict(after['result']);anticipated=dict(expected['result'])
    for key in ['run_id','electrical_revision']:actual.pop(key);anticipated.pop(key)
    assert actual==anticipated
    independent(before);independent(after)
    assert not after['result']['stale']
    return dict(trial=folder.name,measurement_status='available' if data['outcome.json']['status']=='collected_pending_pixel_analysis' else 'unavailable_original_harness_error',frames=len(frames),complete_frames=complete,unique_images=len(image_sizes),states=dict(counts),
        first_committed_parameter_frame=committed['index'],first_fresh_frame=valid['index'],
        marker_to_committed_ms=float((int(committed['display_mach_ticks'])-tick)*scale/1_000_000),
        marker_to_fresh_ms=float((int(valid['display_mach_ticks'])-tick)*scale/1_000_000),
        future_at_receipt=future,interval_scope='Marker to supplied composition-frame timestamp; no OS input-event or acceptance timestamp',
        eligible_latency_observations=0,latency_gate_passed=None)

result=validate(inputs)
negative=[]
first_image_index=next(i for i,r in enumerate(inputs['rows']) if r.get('png'))
def refuses(name,change):
    altered=copy.deepcopy(inputs);change(altered)
    try:validate(altered)
    except (AssertionError,KeyError,ValueError,FileNotFoundError):negative.append(name);return
    raise AssertionError('Corruption admitted: '+name)
refuses('shifted_display_clock',lambda x:x['rows'][first_image_index].__setitem__('display_mach_ticks',str(int(x['rows'][first_image_index]['display_mach_ticks'])+24_000_000)))
refuses('changed_crop',lambda x:x['capture/configuration.json']['capture_plan']['window_global_points'].__setitem__('x',2035))
refuses('unacknowledged_marker',lambda x:x['capture-action.json']['marker_acknowledgment'].__setitem__('mach_ticks','0'))
refuses('missing_image',lambda x:x['rows'][first_image_index].__setitem__('png','frame-missing.png'))
refuses('wrong_pixel_scale',lambda x:x['rows'][first_image_index].__setitem__('scale_factor',2))
refuses('forged_eligibility',lambda x:x['rows'][first_image_index].__setitem__('eligible_for_latency',True))
refuses('stale_pixels_labeled_fresh',lambda x:x['image-annotations.json']['states'][2].__setitem__('hashes',x['image-annotations.json']['states'][1]['hashes']))
refuses('unfinished_input',lambda x:x['capture-action.json'].__setitem__('input_call_completed',False))
result['rejected_mutations']=negative
if '--verify' in sys.argv:assert result==read('analysis.json')
else:
    with (folder/'analysis.json').open('x') as f:json.dump(result,f,indent=2);f.write('\n')
print(json.dumps(result,indent=2))

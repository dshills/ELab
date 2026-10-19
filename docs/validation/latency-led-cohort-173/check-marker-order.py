from pathlib import Path
import contextlib,copy,io,json,runpy,sys
stage=Path(__file__).resolve().parent;folder=Path(sys.argv[1]);saved=list(sys.argv)
sys.argv=[str(stage/'analyze.py'),str(folder),'--verify']
with contextlib.redirect_stdout(io.StringIO()):namespace=runpy.run_path(str(stage/'analyze.py'))
sys.argv=saved
inputs=namespace['inputs'];validate=namespace['validate'];marker=inputs['capture-action.json']['marker_acknowledgment'];tick=int(marker['mach_ticks'])
row=next(r for r in inputs['rows'] if r.get('png') and r['index']<=marker['frame_count_before'] and int(r['display_mach_ticks'])>tick)
assert row['index']==2
cases=[]
def refuse(name,alter):
 data=copy.deepcopy(inputs);alter(data)
 try:validate(data)
 except (AssertionError,KeyError):cases.append(name);return
 raise AssertionError('Incorrect marker association admitted: '+name)
refuse('prospective_pre_ack_frame_cannot_acquire_marker',lambda d:d['rows'][inputs['rows'].index(row)].__setitem__('since_acknowledged_marker_ms',(int(row['display_mach_ticks'])-tick)*125/3/1e6))
post=next(i for i,r in enumerate(inputs['rows']) if r.get('since_acknowledged_marker_ms') is not None)
refuse('post_ack_bound_cannot_disappear',lambda d:d['rows'][post].pop('since_acknowledged_marker_ms'))
print(json.dumps(dict(passed=True,observed_pre_ack_frame_index=row['index'],prospective_display_mach_ticks=row['display_mach_ticks'],marker_mach_ticks=marker['mach_ticks'],rejected_mutations=cases)))

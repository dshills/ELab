"""Declared resistor-loaded NPN fixture; reuse the pinned independent Decimal reference."""
from pathlib import Path
import json,math
ROOT=Path('/Users/dshills/Development/projects/ELab')
FROZEN=json.loads((ROOT/'internal/conformance/testdata/electrical-reference.json').read_text())
PARTS=[('Supply','dc-source','5 V',260,-100),('Drive','dc-source','0 V',-260,60),
       ('BaseR','resistor','10k',-100,60),('Limit','resistor','1k',100,-180),
       ('Transistor','npn','',100,60),('Earth','ground','',100,180)]
WIRES=[('Supply.positive','Limit.1'),('Limit.2','Transistor.collector'),('Drive.positive','BaseR.1'),
       ('BaseR.2','Transistor.base'),('Transistor.emitter','Earth.reference'),
       ('Supply.negative','Earth.reference'),('Drive.negative','Earth.reference')]
def seed(target,canonical=None):
    if canonical is not None and canonical.exists():
        target.ask('project.load',dict(path=str(canonical),discard_changes=True),True)
    else:
        target.ask('project.clear',dict(discard_changes=True),True);target.ask('simulation.pause',{},True)
        for name,kind,value,x,y in PARTS:target.ask('component.place',dict(id=name,kind=kind,value=value,position=dict(x=x,y=y)),True)
        for a,b in WIRES:target.ask('wire.connect',{'from':a,'to':b},True)
    target.ask('component.select',dict(id='Drive'),True);target.ask('simulation.resume',{},True)
    state=target.ask('simulation.wait');assert state['npn'] and not state['npn']['stale']
    assert next(p['value'] for p in state['components'] if p['id']=='Drive')==0
    independent(state)
    if canonical is not None and not canonical.exists():target.ask('project.save',dict(path=str(canonical)),True)
    return state

def independent(state):
    parts={p['id']:p for p in state['components']};result=state['npn'];model=result['model'];drive=parts['Drive']['value']
    assert parts['Supply']['value']==5 and parts['BaseR']['value']==10000 and parts['Limit']['value']==1000
    assert drive in [0,5] and result['engine']=='elab.npn-switch-dc.v1' and not result['stale']
    assert result.get('led_id','')=='' and not state.get('transient')
    assert model['id']==FROZEN['npn_model']['id']
    for key in ['saturation_current_a','forward_beta','reverse_beta','temperature_c']:
        assert model[key]==float(FROZEN['npn_model'][key])
    expected=next(r for r in FROZEN['npn'] if float(r['drive_v'])==drive)
    errors={}
    for actual,key,tolerance in [('vbe_v','base_v',1e-9),('vce_v','collector_v',1e-9),('base_current_a','base_current_a',1e-12),('collector_current_a','collector_current_a',1e-12)]:
        errors[actual]=abs(result[actual]-float(expected[key]));assert errors[actual]<tolerance,(actual,errors)
    b,c=result['vbe_v'],result['vce_v'];ib,ic=result['base_current_a'],result['collector_current_a']
    vt=1.380649e-23*(273.15+model['temperature_c'])/1.602176634e-19
    forward=model['saturation_current_a']*math.expm1(b/vt);reverse=model['saturation_current_a']*math.expm1((b-c)/vt)
    residuals=dict(base_kcl_a=ib-(forward/model['forward_beta']+reverse/model['reverse_beta']),
        collector_kcl_a=ic-(forward-(1+1/model['reverse_beta'])*reverse),base_loop_v=drive-b-10000*ib,
        load_loop_v=5-c-1000*ic,emitter_kcl_a=result['emitter_current_a']-ib-ic,
        power_w=5*ic+drive*ib-(result['transistor_power_w']+result['base_resistor_power_w']+result['load_resistor_power_w']))
    assert all(abs(v)<1e-10 for v in residuals.values()),residuals
    assert result['state']==('cutoff' if drive==0 else 'saturation')
    return dict(method=FROZEN['method'],generator_sha256=FROZEN['generator_sha256'],drive_v=drive,absolute_errors=errors,residuals=residuals)

def compare_states(native,expected):
 for key in ('components','nets'):assert native[key]==expected[key],key
 semantics=lambda wires:[{k:v for k,v in wire.items() if k!='id'} for wire in wires];assert semantics(native['wires'])==semantics(expected['wires'])
 actual=dict(native['npn']);anticipated=dict(expected['npn'])
 for key in ('run_id','electrical_revision'):actual.pop(key);anticipated.pop(key)
 assert actual==anticipated;independent(native);independent(expected)

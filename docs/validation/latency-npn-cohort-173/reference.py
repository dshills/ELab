"""Read-only pinned independent NPN law/reference checks; no app or solver import."""
from pathlib import Path
import json,math
ROOT=Path(__file__).resolve().parents[3]
FROZEN=json.loads((ROOT/'internal/conformance/testdata/electrical-reference.json').read_text())
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

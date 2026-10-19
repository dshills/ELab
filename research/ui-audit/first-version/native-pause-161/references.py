"""Independent frozen-model checks; never calls an ELab solver."""
from decimal import Decimal as D, localcontext
import hashlib, json, math
from pathlib import Path

ROOT = Path('/Users/dshills/Development/projects/ELab')

def npn_reference(drive, led):
    with localcontext() as context:
        context.prec = 70
        drive = D(str(drive))
        vt = D('1.380649e-23') * D('300.15') / D('1.602176634e-19')
        x = [D('.7'), D('.06') if led else D('.07'), D('.00286') if led else D('.00493')]
        if drive == 0:
            x = [D('1e-10'), D('4.25') if led else D('5'), D('2e-14')]
        elif drive < 1:
            x = [D('.663'), D('1.8') if led else D('3.63'), D('.00136')]

        def equations(x):
            vb, vc, current = x
            ef, er = (vb / vt).exp(), ((vb-vc) / vt).exp()
            forward, reverse = D('1e-14')*(ef-1), D('1e-14')*(er-1)
            drop = 2*vt*(1+current/D('1e-20')).ln()+current if led else D(0)
            residual = [forward/100+reverse-(drive-vb)/10000,
                        forward-2*reverse-current, vc+1000*current+drop-5]
            matrix = [[D('1e-14')*(ef/100+er)/vt+D('.0001'), -D('1e-14')*er/vt, D(0)],
                      [D('1e-14')*(ef-2*er)/vt, 2*D('1e-14')*er/vt, D(-1)],
                      [D(0), D(1), 1001+2*vt/(current+D('1e-20')) if led else D(1000)]]
            return residual, matrix

        for iteration in range(150):
            residual, matrix = equations(x)
            norm = max(abs(v) for v in residual)
            if norm < D('1e-50'):
                break
            augmented = [row[:] + [value] for row, value in zip(matrix, residual)]
            for column in range(3):
                pivot = max(range(column, 3), key=lambda row: abs(augmented[row][column]))
                augmented[column], augmented[pivot] = augmented[pivot], augmented[column]
                scale = augmented[column][column]
                augmented[column] = [value / scale for value in augmented[column]]
                for row in range(3):
                    if row != column:
                        factor = augmented[row][column]
                        augmented[row] = [a-factor*b for a, b in zip(augmented[row], augmented[column])]
            delta = [row[-1] for row in augmented]
            damping = D(1)
            for _ in range(100):
                trial = [a-damping*b for a, b in zip(x, delta)]
                if trial[2] >= 0 and max(abs(v) for v in equations(trial)[0]) < norm:
                    x = trial
                    break
                damping /= 2
            else:
                raise AssertionError('independent Newton line search')
        else:
            raise AssertionError('independent Newton convergence')
        vb, vc, current = x
        return dict(drive_v=str(drive), base_v=str(vb), collector_v=str(vc),
                    base_current_a=str((drive-vb)/10000), collector_current_a=str(current), iterations=iteration)

def mosfet_current(overdrive):
    # Same independent scalar load-line method as conformance/mosfet_references.go.
    if overdrive <= 0:
        return 1e-9
    lo, hi = 0., .005
    for _ in range(100):
        current = (lo+hi)/2
        led = 2*1.380649e-23*300.15/1.602176634e-19*math.log1p(current/1e-20)+current
        vds = 5-1000*current-led
        channel = 0.
        if vds > 0:
            channel = (.1*(overdrive*vds-.5*vds*vds) if vds < overdrive else .05*overdrive*overdrive)*(1+.02*vds)
        if channel+1e-9 > current:
            lo = current
        else:
            hi = current
    return (lo+hi)/2

def prepare():
    path = ROOT/'internal/conformance/testdata/electrical-reference.json'
    frozen = json.loads(path.read_text())
    resistor = [npn_reference(row['drive_v'], False) for row in frozen['npn']]
    for actual, expected in zip(resistor, frozen['npn']):
        for key in ['base_v', 'collector_v', 'base_current_a', 'collector_current_a']:
            assert abs(D(actual[key])-D(expected[key])) < D('1e-45'), key
    return dict(passed=True, method='70-digit simultaneous damped 3x3 Newton; frozen no-LED cases validate the independent equations before adding the declared LED law. MOSFET scalar load-line bisection omits at most 1 nA of reverse-diode detail.',
                frozen_sha256=hashlib.sha256(path.read_bytes()).hexdigest(), resistor_cases=resistor,
                npn_led=[npn_reference(drive, True) for drive in [0, .8, 5]],
                mosfet_high_abs_current_a=mosfet_current(5*100000/100100-1.5))

def npn_check(state, reference):
    assert state['analysis'] == 'NPN transient' and not state['npn']['stale'] and not state['transient']['stale']
    rows = state['transient']['samples']
    for row in rows:
        expected = next(x for x in reference['npn_led'] if float(x['drive_v']) == row['input_v'])
        for actual, key, tolerance in [(row['voltage_v'], 'collector_v', .001),
                                       (row['current_a'], 'collector_current_a', 1e-9),
                                       (row.get('base_current_a', 0), 'base_current_a', 1e-9),
                                       (row.get('vbe_v', 0), 'base_v', .001)]:
            target = float(expected[key])
            assert abs(actual-target) <= max(tolerance, .01*abs(target)), (key, actual, target)
        assert row['state'] == ('cutoff' if row['input_v'] == 0 else 'saturation')
    result = state['npn']
    assert result['kcl_residual_a'] < 1e-9 and abs(result['kvl_residual_v']) < 1e-8
    power = result['base_resistor_power_w']+result['load_resistor_power_w']+result['led_power_w']+result['transistor_power_w']
    assert abs(state['transient']['sample']['input_v']*result['base_current_a']+5*result['collector_current_a']-power) < 1e-8
    return dict(passed=True, sample_rows=len(rows), sample_time_s=state['transient']['sample']['time_s'], state=result['state'])

def mosfet_check(state, kind):
    mixed = state['mixed']
    assert not mixed['stale'] and not state['transient']['stale']
    result = mixed['mosfets'][0]
    assert result['id'] == 'M1' and result['model']['id'] == 'elab.generic-'+kind+'.v1'
    source = state['transient']['sample']['input_v']
    gate = source*100000/100100 if kind == 'nmos' else (source*100000+5*100)/100100-5
    sign = 1 if kind == 'nmos' else -1
    on = sign*gate > 1.5
    expected = sign*mosfet_current(sign*gate-1.5)
    assert abs(result['drain_current_a']-expected) < 2e-9
    assert abs(result['vgs_v']-gate) < 1e-9
    assert result['gate_current_a'] == 0 and result['source_current_a'] == result['drain_current_a']
    assert result['state'] == ('triode' if on else 'cutoff')
    assert abs(result['power_w']-result['vds_v']*result['drain_current_a']) < 1e-12 and result['power_w'] >= 0
    assert mixed['kcl_residual_a'] < 1e-9 and abs(mixed['power_residual_w']) < 1e-8
    traces = {t['probe']['id']: t['points'] for t in state['scope']['traces']}
    for probe, field in [('v:M1','vds_v'), ('v:M1.gate-source','vgs_v'), ('i:M1.drain','drain_current_a')]:
        if probe in traces:
            assert math.isclose(traces[probe][-1]['value'], result[field], rel_tol=1e-10, abs_tol=1e-12)
    return dict(passed=True, kind=kind, source_v=source, gate_source_v=gate,
                drain_current_a=result['drain_current_a'], reference_current_a=expected, state=result['state'])

def mosfet_trace_check(state, kind):
    traces = {t['probe']['id']: t['points'] for t in state['scope']['traces']}
    probes = ['v:M1', 'v:M1.gate-source', 'i:M1.drain', 'i:D1', 'v:VIN.positive', 'i:M1.gate']
    assert set(probes) == set(traces)
    count = len(traces[probes[0]])
    assert all(len(traces[probe]) == count for probe in probes)
    sign = 1 if kind == 'nmos' else -1
    observed_inputs = set()
    maximum_voltage_error = 0.
    for index in range(count):
        points = {probe: traces[probe][index] for probe in probes}
        assert len({point['time_s'] for point in points.values()}) == 1
        source = points['v:VIN.positive']['value']
        # Network node values may have sub-picovolt Newton residuals.
        nominal = 0 if abs(source) < 1e-9 else 5
        assert abs(source-nominal) < 1e-9
        observed_inputs.add(nominal)
        source = nominal
        gate = source*100000/100100 if kind == 'nmos' else (source*100000+5*100)/100100-5
        current = mosfet_current(sign*gate-1.5)
        led = 2*1.380649e-23*300.15/1.602176634e-19*math.log1p(current/1e-20)+current
        voltage_error = abs(points['v:M1']['value']-sign*(5-1000*current-led))
        maximum_voltage_error = max(maximum_voltage_error, voltage_error)
        # Current accuracy is bounded to 2 nA by the existing MOSFET reference.
        # Near cutoff the log LED law amplifies tiny current residuals into mV.
        # Retain the strict 1 mV outcome separately; bound VDS from the same
        # fixed current envelope rather than inventing a wider voltage tolerance.
        bounds = []
        for bound_current in [max(0., current-2e-9), current+2e-9]:
            drop = 2*1.380649e-23*300.15/1.602176634e-19*math.log1p(bound_current/1e-20)+bound_current
            bounds.append(sign*(5-1000*bound_current-drop))
        assert min(bounds)-1e-9 <= points['v:M1']['value'] <= max(bounds)+1e-9
        assert abs(points['v:M1.gate-source']['value']-gate) < 1e-9
        assert abs(points['i:M1.drain']['value']-sign*current) < 2e-9
        assert abs(points['i:D1']['value']-current) < 2e-9
        assert points['i:M1.gate']['value'] == 0
    return dict(passed=True, aligned_rows=count, phases=sorted(observed_inputs), per_row_values=6,
                drain_voltage_bounded_by_2na_current_envelope=True,
                maximum_drain_voltage_point_reference_error_v=maximum_voltage_error,
                strict_1mv_drain_voltage_reference_passed=maximum_voltage_error < .001,
                scope='Signed-current/gate/phase smoke with bounded load-line voltage. Strict 1 mV drain-voltage accuracy is reported separately, not required or claimed by the smoke.')

if __name__ == '__main__':
    output = Path(__file__).with_name('independent-references.json')
    output.write_text(json.dumps(prepare(), indent=2)+'\n')
    print('Independent frozen resistor NPN, LED NPN and MOSFET load-line references prepared')

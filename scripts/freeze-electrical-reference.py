#!/usr/bin/env python3
"""Independent pinned-model references; no ELab imports or solver invocation."""
import argparse
from decimal import Decimal as D, getcontext
import hashlib
import json
from pathlib import Path

getcontext().prec = 70
VT = D('1.380649e-23') * D('300.15') / D('1.602176634e-19')


def led(resistance):
    # Newton on current, unlike ELab's bounded bisection.
    current = D('.002')
    for _ in range(100):
        residual = current * (resistance + 1) + 2 * VT * (1 + current / D('1e-20')).ln() - 5
        delta = residual / (resistance + 1 + 2 * VT / (current + D('1e-20')))
        current -= delta
        if abs(delta) < D('1e-55'):
            break
    else:
        raise ValueError('LED reference did not converge')
    return dict(resistance_ohm=str(resistance), current_a=str(current), voltage_v=str(5-current*resistance))


def npn(drive):
    # Simultaneous damped 2x2 Newton on base/collector KCL, rather than
    # ELab's nested base/collector bisections. Reciprocal transport IS.
    vb, vc = (D('0'), D('5')) if drive == 0 else ((D('.65'), D('3')) if drive < 1 else (D('.7'), D('.07')))

    def equations(b, c):
        ef, er = (b / VT).exp(), ((b-c) / VT).exp()
        f, r = D('1e-14') * (ef-1), D('1e-14') * (er-1)
        ib, ic = f / 100 + r, f - 2*r
        return (ib-(drive-b)/10000, ic-(5-c)/1000,
                D('1e-14')*(ef/100+er)/VT+D('0.0001'), -D('1e-14')*er/VT,
                D('1e-14')*(ef-2*er)/VT, 2*D('1e-14')*er/VT+D('0.001'))

    for _ in range(100):
        f, g, a, b, c, d = equations(vb, vc)
        if max(abs(f), abs(g)) < D('1e-55'):
            break
        determinant = a*d-b*c
        db, dc = (d*f-b*g)/determinant, (a*g-c*f)/determinant
        damping = D(1)
        for _ in range(100):
            nb, nc = vb-damping*db, vc-damping*dc
            nf, ng, *_ = equations(nb, nc)
            if max(abs(nf), abs(ng)) < max(abs(f), abs(g)):
                vb, vc = nb, nc
                break
            damping /= 2
        else:
            raise ValueError('NPN reference line search failed')
    else:
        raise ValueError('NPN reference did not converge')
    return dict(drive_v=str(drive), base_v=str(vb), collector_v=str(vc),
                base_current_a=str((drive-vb)/10000), collector_current_a=str((5-vc)/1000))


def rc(at, finite, left=False):
    # Exact convolution of piecewise constant/linear forcing with exp(-t/RC).
    tau, t, voltage, source = D('.01'), D(0), D(0), D(0 if finite else 5)
    segments = [(D('.001'), D(0), D(5000)), (D('.049'), D(5), D(0)),
                (D('.001'), D(5), D(-5000)), (D('.049'), D(0), D(0))] if finite else [
                (D('.05'), D(5), D(0)), (D('.05'), D(0), D(0))]
    while t < at:
        for duration, start, slope in segments:
            dt = min(duration, at-t)
            decay = (-dt/tau).exp()
            voltage = voltage*decay + start*(1-decay) + slope*(dt-tau*(1-decay))
            source = start+slope*dt
            t += dt
            if t == at:
                break
    if not finite and not left:
        source = D(5) if (at % D('.1')) < D('.05') else D(0)
    return dict(time_s=str(at), source_v=str(source), voltage_v=str(voltage))


def reference():
    times = list(map(D, ['0', '.001', '.005', '.01', '.025', '.049', '.05', '.051', '.075', '.1', '.15', '.2', '.3', '.35', '.4']))
    ideal = []
    for at in times:
        if at and at % D('.05') == 0:
            ideal.append(rc(at, False, True))
        ideal.append(rc(at, False))
    return dict(version=1, method='70-digit Decimal Newton and closed-form convolution; same pinned educational models, independent code/algorithms',
                generator_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                led_model=dict(id='elab.generic-red-led.v1', is_a='1e-20', n='2', rs_ohm='1', temperature_c='27'),
                npn_model=dict(id='elab.generic-npn.v1', saturation_current_a='1e-14', forward_beta='100', reverse_beta='1', temperature_c='27'),
                rc_parameters=dict(resistance_ohm='10000', capacitance_f='0.000001', high_v='5', low_v='0', high_s='0.05', low_s='0.05', rise_s='0.001', fall_s='0.001'),
                led=[led(D(1000)), led(D(2000))], npn=[npn(D(v)) for v in ['0', '.8', '5']],
                rc_ideal=ideal, rc_finite=[rc(at, True) for at in times],
                rc_periodic_low_v=str(5*(-D(5)).exp()/(1+(-D(5)).exp())),
                rc_periodic_high_v=str(5/(1+(-D(5)).exp())))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--write', action='store_true')
    args = parser.parse_args()
    path = Path(__file__).resolve().parents[1]/'internal/conformance/testdata/electrical-reference.json'
    data = json.dumps(reference(), indent=2)+'\n'
    if args.write:
        path.write_text(data)
    else:
        assert path.read_text() == data, 'frozen reference differs; inspect the model/algorithm change before explicit regeneration'
    print(json.dumps(dict(passed=True, sha256=hashlib.sha256(data.encode()).hexdigest())))


if __name__ == '__main__':
    main()

#!/usr/bin/env python3
"""Freeze independent high-precision references for ELab's chosen op-amp law.

This does not import ELab or a vendor model. Values characterize the explicitly
chosen educational equations, not any physical device or SPICE implementation.
"""
import argparse
from decimal import Decimal as D, localcontext
import hashlib
import json
from pathlib import Path


def currents(values):
    vp, vn, out, high, low = values
    span = high-low
    enable = span*span/(span*span+D('.01')) if span > 0 else D(0)
    target = low
    active = D(0)
    if span > 0:
        z = D(200000)*(vp-vn)/span
        if abs(z) > 120:
            tangent = D(1).copy_sign(z)
        else:
            e = (-2*abs(z)).exp()
            tangent = (1-e)/(1+e)
            if z < 0:
                tangent = -tangent
        target = (high+low)/2+span*tangent/2
        demand = out-target  # Rout * Ilimit = 1 V for this fixed model.
        active = enable*D('.02')*demand/(1+demand*demand).sqrt()
    bleed = (out-low)/D(10)**9
    input_current = (vp-vn)/D(10)**7
    output_current = active+bleed
    positive_current = D('.001')*enable+max(-active,D(0))
    return [input_current,-input_current,output_current,positive_current,
            -output_current-positive_current], target, enable


def closed_loop(kind, vin, load, high, low):
    def state(out):
        if kind == 'follower':
            vplus, vminus = vin, out
            current, _, _ = currents([vplus,vminus,out,high,low])
            residual = current[2]+current[1]+out/load
        elif kind == 'noninverting':
            feedback, lower = D(90000),D(10000)
            vminus = (out/feedback+vin/D(10)**7)/(1/feedback+1/lower+1/D(10)**7)
            current, _, _ = currents([vin,vminus,out,high,low])
            residual = current[2]+out/load+(out-vminus)/feedback
        else:
            incoming, feedback = D(10000),D(20000)
            vminus = (vin/incoming+out/feedback)/(1/incoming+1/feedback+1/D(10)**7)
            current, _, _ = currents([D(0),vminus,out,high,low])
            residual = current[2]+out/load+(out-vminus)/feedback
        return residual
    lo,hi=D(-30),D(30)
    assert state(lo)<0<state(hi)
    for _ in range(300):
        middle=(lo+hi)/2
        if state(middle)>0:
            hi=middle
        else:
            lo=middle
    return (lo+hi)/2


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--output',type=Path,required=True,help='New reference JSON; never overwrite')
    args=parser.parse_args()
    cases=[('balanced',[0,0,0,5,-5]),('small-positive',[.000001,0,.2,5,-5]),
        ('small-negative',[-.000001,0,-.2,5,-5]),
        ('single-supply',[.10001,.1,1,5,0]),('positive-limit',[1,0,0,5,-5]),
        ('negative-limit',[-1,0,0,5,-5]),('forced-high',[0,0,30,5,0]),
        ('forced-low',[0,0,-30,5,0]),('unpowered',[1,0,1,0,0]),
        ('reversed',[1,0,4,-5,0]),('soft-start',[.000001,0,.01,1e-8,0]),
        ('full-span',[.00001,0,10,30,-30]),('tiny-span',[1e-300,0,1,1e-300,0]),
        ('subnormal-supply',[1,0,1,1e-320,0])]
    loops=[('follower','1','10000','5','-5'),('follower','2','1000','5','0'),
        ('follower','10','1000','12','2'),('noninverting','.1','1000','5','-5'),
        ('inverting','.5','10000','5','-5'),('noninverting','1','100','5','-5')]
    with localcontext() as ctx:
        ctx.prec=90
        rows=[]
        for name,values in cases:
            # Exact binary input values make the reference's rounding policy explicit.
            current,target,enable=currents([D.from_float(float(value)) for value in values])
            rows.append(dict(name=name,volts=values,current_a=list(map(float,current)),
                target_v=float(target),enable=float(enable)))
        amplifiers=[dict(kind=kind,vin_v=float(vin),load_ohm=float(load),high_v=float(high),low_v=float(low),
            output_v=float(closed_loop(kind,*map(D,(vin,load,high,low))))) for kind,vin,load,high,low in loops]
    result=dict(format='elab.opamp-foundation-reference',version=1,model='elab.generic-opamp.v1',
        precision_decimal_digits=90,generator_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        scope='Independent Decimal evaluation of ELab-selected generic equations and scalar amplifier KCL; not manufacturer or SPICE equivalence',
        current_reference_tolerance_a=1e-11,target_reference_tolerance_v=1e-10,
        pin_order=['noninverting','inverting','out','vplus','vminus'],cases=rows,amplifiers=amplifiers)
    args.output.parent.mkdir(parents=True,exist_ok=True)
    with args.output.open('x') as out:
        out.write(json.dumps(result,indent=2,allow_nan=False)+'\n')


if __name__=='__main__':
    main()

#!/usr/bin/env python3
"""Owned-session MOSFET construction, pulse/RC references, probes and persistence."""
import argparse
import csv
import importlib.util
import json
import math
from pathlib import Path
import time

from validation_artifacts import discovery_summary

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Requires a fresh owned blank native session; closes it afterward')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    ask = lambda command, params=None: client.ask(command, params, True)
    try:
        assert not client.ask('project.inspect')['components'], 'Use an owned blank session'
        parts = client.ask('parts.list')
        part = next(p for p in parts if p['kind'] == 'nmos')
        assert part['terminals'] == ['gate', 'drain', 'source'] and part['value_type'] == 'none'
        contract = discovery_summary(client.ask('capabilities.list', {'command': 'component.place'}), assets/'discovery.json')
        ask('simulation.pause')
        nodes = [('VCC', 'dc-source', -300, -120, '5 V'), ('VIN', 'pulse-source', -300, 60, '5 V'),
                 ('RG', 'resistor', -140, 60, '100 Ω'), ('RPD', 'resistor', 10, 190, '100k Ω'),
                 ('RL', 'resistor', 0, -200, '1k Ω'), ('D1', 'led', 150, -80, ''),
                 ('M1', 'nmos', 150, 60, ''), ('GND1', 'ground', 260, 220, '')]
        for identity, kind, x, y, value in nodes:
            p = dict(id=identity, kind=kind, position=dict(x=x, y=y))
            if value:
                p['value'] = value
            ask('component.place', p)
            if identity == 'RPD':
                ask('component.rotate', {'id': identity})
        wires = [('VCC.positive', 'RL.1'), ('RL.2', 'D1.anode'), ('D1.cathode', 'M1.drain'),
                 ('VIN.positive', 'RG.1'), ('RG.2', 'M1.gate'), ('M1.gate', 'RPD.1'),
                 ('RPD.2', 'GND1.reference'), ('M1.source', 'GND1.reference'),
                 ('VCC.negative', 'GND1.reference'), ('VIN.negative', 'GND1.reference')]
        for a, b in wires:
            ask('wire.connect', {'from': a, 'to': b})
        client.ask('simulation.wait')
        manual = client.ask('project.inspect')
        ask('example.load', {'id': 'nmos', 'discard_changes': True})
        ask('simulation.pause')
        client.ask('simulation.wait')
        ask('simulation.restart')
        initial = client.ask('simulation.wait')
        assert manual['components'] == initial['components'] and manual['wires'] == initial['wires']
        on = initial['mixed']['mosfets'][0]
        assert on['state'] == 'triode' and on['gate_current_a'] == 0 and .0028 < on['drain_current_a'] < .003
        assert abs(on['vgs_v']-5*100000/100100) < 1e-10 and on['vds_v'] < .01
        selected = ['v:M1', 'v:M1.gate-source', 'i:M1.drain', 'i:M1.gate', 'i:M1.source', 'i:D1']
        ask('scope.probes.set', {'ids': selected})
        before = client.ask('project.inspect')
        ask('component.move', {'id': 'M1', 'position': {'x': 120, 'y': 70}})
        ask('component.rotate', {'id': 'M1'})
        pose = client.ask('project.inspect')
        assert pose['mixed'] == before['mixed'] and pose['run_id'] == before['run_id']
        # Restore geometry for the exported beginner example.
        ask('edit.undo')
        ask('edit.undo')
        ask('export.schematic.svg', {'path': str(assets/'nmos.svg')})
        ask('export.schematic.png', {'path': str(assets/'nmos.png')})
        ask('simulation.step', {'duration_s': '200 ms'})
        falling = client.ask('simulation.wait')
        off = falling['mixed']['mosfets'][0]
        assert off['state'] == 'cutoff' and off['drain_current_a'] < 2e-9
        trace = next(t for t in falling['scope']['traces'] if t['probe']['id'] == 'v:M1.gate-source')
        points = trace['points']
        assert any(a['time_s'] == b['time_s'] == .2 and a['value'] > 4.9 and b['value'] == 0
                   for a, b in zip(points, points[1:]))
        ask('simulation.step', {'duration_s': '300 ms'})
        rising = client.ask('simulation.wait')
        assert rising['mixed']['mosfets'][0]['state'] == 'triode'
        assert abs(rising['mixed']['mosfets'][0]['drain_current_a']-on['drain_current_a']) < 1e-10
        ask('simulation.verify', {'duration_s': '500 ms', 'probe_ids': selected})
        identity = client.ask('project.inspect')['last_operation']
        deadline = time.monotonic()+30
        while True:
            operation = client.ask('operation.inspect', {'id': identity})
            if operation['operation']['status'] in ('completed', 'failed', 'canceled'):
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        assert operation['operation']['status'] == 'completed', operation
        assert operation['metadata']['nmos_model']['id'] == part['model']
        for fmt in ('json', 'csv'):
            ask('verification.export', {'id': identity, 'format': fmt, 'path': str(assets/f'nmos.{fmt}')})
        with (assets/'nmos.csv').open() as stream:
            rows = list(csv.DictReader(stream))
        assert rows and all(row['nmos_model'] == part['model'] and float(row['i:M1.gate [A]']) == 0 for row in rows)
        path = assets/'nmos.elab'
        ask('project.save', {'path': str(path)})
        ask('project.load', {'path': str(path)})
        loaded = client.ask('simulation.wait')
        assert loaded['document']['recording_count'] == 1 and loaded['transient']['sample']['time_s'] == 0
        assert loaded['nmos_model'] == operation['metadata']['nmos_model']
        assert loaded['mixed']['mosfets'][0]['model'] == loaded['nmos_model']
        broken = json.loads(path.read_text())
        broken['nmos_model']['beta_a_per_v2'] = 99
        bad = assets/'tampered-model.elab'
        bad.write_text(json.dumps(broken))
        client.ask('project.load', {'path': str(bad)}, True, 'invalid-document')
        ask('component.set', {'id': 'RG', 'value': '10k'})
        ask('component.place', {'id': 'CG', 'kind': 'capacitor', 'value': '1uF', 'position': {'x': -220, 'y': 220}})
        for pin, target in [('1', 'M1.gate'), ('2', 'GND1.reference')]:
            ask('wire.connect', {'from': 'CG.'+pin, 'to': target})
        ask('simulation.restart')
        start = client.ask('simulation.wait')
        assert start['mixed']['mosfets'][0]['state'] == 'cutoff'
        ask('simulation.step', {'duration_s': '10 ms'})
        charged = client.ask('simulation.wait')
        charge_v = charged['mixed']['mosfets'][0]['vgs_v']
        tau, target = 1e-6/(1/10000+1/100000), 5*100000/110000
        charge_ref = target*(1-math.exp(-.01/tau))
        assert abs(charge_v-charge_ref) < .003
        ask('simulation.step', {'duration_s': '190 ms'})
        edge = client.ask('simulation.wait')
        assert edge['mixed']['mosfets'][0]['vgs_v'] > 4.5
        ask('simulation.step', {'duration_s': '10 ms'})
        discharged = client.ask('simulation.wait')
        discharge_v = discharged['mixed']['mosfets'][0]['vgs_v']
        discharge_ref = target*(1-math.exp(-.2/tau))*math.exp(-.01/tau)
        assert abs(discharge_v-discharge_ref) < .003
        for state in [initial, falling, rising, charged, edge, discharged]:
            assert state['mixed']['kcl_residual_a'] < 1e-9 and abs(state['mixed']['power_residual_w']) < 1e-8
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', report_version=1,
                      part=part, place_contract=contract,
                      observations=dict(manual_matches_template=True, on=on, off=off, charge_10ms_v=charge_v,
                                        charge_reference_v=charge_ref, discharge_10ms_v=discharge_v,
                                        discharge_reference_v=discharge_ref, gate_rc_tau_s=tau,
                                        kept_recordings=loaded['document']['recording_count']),
                      checks=['Blank component placement and named wiring match the guided example',
                              'Pulse edges switch the LED with duplicate-time gate voltage samples and zero model gate current',
                              'Move/rotate preserve electrical result/run; selected MOSFET voltage and current probes available',
                              'External gate RC charge/discharge agree with analytic references within 3 mV; charge continuous at pulse edge',
                              'Verification/CSV/SVG/PNG preserve model and signed readings; saved recording reopens with fresh live time',
                              'Tampered model parameters refuse; KCL/power residuals remain bounded'],
                      limits=['Educational generic NMOS; no physical part identity, intrinsic gate charge, heating, ratings or hardware timing claim',
                              'Generic PMOS is available; configurable physical-device models remain future work'])
        with args.output.open('x') as stream:
            json.dump(report, stream, indent=2)
            stream.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], observations=report['observations'])))
    finally:
        if args.connect:
            client.ask('project.inspect')
            ask('app.close', {'discard_changes': True})
        client.close()


if __name__ == '__main__':
    main()

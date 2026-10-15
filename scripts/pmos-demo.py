#!/usr/bin/env python3
"""Fresh owned session: high-side PMOS, gate RC, captures and complementary inverter."""
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
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned blank native session; closes it afterward')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)
    def ask(command, params=None):
        return client.ask(command, params, True)
    def restart():
        ask('simulation.restart')
        return client.ask('simulation.wait')
    def step(duration):
        ask('simulation.step', {'duration_s': duration})
        return client.ask('simulation.wait')
    def build(nodes, wires):
        ask('simulation.pause')
        for identity, kind, x, y, value, rotation in nodes:
            p = dict(id=identity, kind=kind, position=dict(x=x, y=y))
            if value:
                p['value'] = value
            ask('component.place', p)
            for _ in range(rotation//90):
                ask('component.rotate', {'id': identity})
        for a, b in wires:
            ask('wire.connect', {'from': a, 'to': b})
    def verify(probes, duration=None):
        p = {'probe_ids': probes}
        if duration:
            p['duration_s'] = duration
        ask('simulation.verify', p)
        identity = client.ask('project.inspect')['last_operation']
        end = time.monotonic()+30
        while True:
            result = client.ask('operation.inspect', {'id': identity})
            if result['operation']['status'] in ('completed', 'failed', 'canceled'):
                break
            assert time.monotonic() < end
            time.sleep(.01)
        assert result['operation']['status'] == 'completed', result
        return identity, result['metadata']
    try:
        assert not client.ask('project.inspect')['components'], 'Requires a fresh owned blank session'
        part = next(p for p in client.ask('parts.list') if p['kind'] == 'pmos')
        assert part['terminals'] == ['gate', 'drain', 'source'] and part['value_type'] == 'none'
        contract = discovery_summary(client.ask('capabilities.list', {'command': 'component.place'}), assets/'discovery.json')
        build([('VCC', 'dc-source', -260, -150, '5 V', 0), ('VIN', 'pulse-source', -260, 80, '5 V', 0),
               ('RG', 'resistor', 260, -130, '100 Ω', 0), ('RPU', 'resistor', 230, -40, '100k Ω', 0),
               ('M1', 'pmos', 100, -130, '', 180), ('RL', 'resistor', 100, 0, '1k Ω', 90),
               ('D1', 'led', 100, 100, '', 0), ('GND1', 'ground', 100, 230, '', 0),
               ('J1', 'junction', 380, -180, '', 0)],
              [('VCC.positive', 'M1.source'), ('M1.drain', 'RL.1'), ('RL.2', 'D1.anode'), ('D1.cathode', 'GND1.reference'),
               ('VIN.positive', 'J1.node'), ('J1.node', 'RG.2'), ('RG.1', 'M1.gate'), ('M1.gate', 'RPU.1'), ('RPU.2', 'M1.source'),
               ('VCC.negative', 'GND1.reference'), ('VIN.negative', 'GND1.reference')])
        client.ask('simulation.wait')
        manual = client.ask('project.inspect')
        ask('example.load', {'id': 'pmos', 'discard_changes': True})
        ask('simulation.pause')
        client.ask('simulation.wait')
        initial = restart()
        assert initial['components'] == manual['components'] and initial['wires'] == manual['wires']
        off = initial['mixed']['mosfets'][0]
        assert off['state'] == 'cutoff' and abs(off['drain_current_a']) < 2e-9
        assert initial['pmos_model']['threshold_v'] == -1.5 and 'nmos_model' not in initial
        probes = ['v:M1', 'v:M1.gate-source', 'i:M1.drain', 'i:M1.gate', 'i:M1.source', 'i:D1']
        ask('scope.probes.set', {'ids': probes})
        ask('export.schematic.svg', {'path': str(assets/'pmos.svg')})
        ask('export.schematic.png', {'path': str(assets/'pmos.png')})
        falling = step('200 ms')
        on = falling['mixed']['mosfets'][0]
        assert on['state'] == 'triode' and -.003 < on['drain_current_a'] < -.0028
        assert abs(on['vgs_v']+5*100000/100100) < 1e-10 and on['vds_v'] < 0
        assert on['gate_current_a'] == 0 and on['source_current_a'] == on['drain_current_a']
        trace = next(t for t in falling['scope']['traces'] if t['probe']['id'] == 'v:M1.gate-source')
        assert any(a['time_s'] == b['time_s'] == .2 and abs(a['value']) < 1e-10 and b['value'] < -4.9
                   for a, b in zip(trace['points'], trace['points'][1:]))
        rising = step('300 ms')
        assert rising['mixed']['mosfets'][0]['state'] == 'cutoff'
        identity, metadata = verify(probes, '500 ms')
        assert metadata['pmos_model'] == initial['pmos_model']
        for fmt in ('json', 'csv'):
            ask('verification.export', {'id': identity, 'format': fmt, 'path': str(assets/f'pmos.{fmt}')})
        with (assets/'pmos.csv').open() as stream:
            rows = list(csv.DictReader(stream))
        assert rows and all(r['pmos_model'] == part['model'] and float(r['i:M1.gate [A]']) == 0 for r in rows)
        assert any(float(r['i:M1.drain [A]']) < -.0028 for r in rows)
        path = assets/'pmos.elab'
        ask('project.save', {'path': str(path)})
        ask('project.load', {'path': str(path)})
        loaded = client.ask('simulation.wait')
        assert loaded['document']['recording_count'] == 1 and loaded['transient']['sample']['time_s'] == 0
        assert loaded['pmos_model'] == metadata['pmos_model'] and loaded['mixed']['mosfets'][0]['state'] == 'cutoff'
        bad = json.loads(path.read_text())
        bad['pmos_model']['threshold_v'] = 1.5
        broken = assets/'wrong-polarity.elab'
        broken.write_text(json.dumps(bad))
        client.ask('project.load', {'path': str(broken)}, True, 'invalid-document')
        ask('component.set', {'id': 'RG', 'value': '10k'})
        ask('component.place', {'id': 'CG', 'kind': 'capacitor', 'value': '1uF', 'position': {'x': 520, 'y': -220}})
        for a, b in [('CG.1', 'M1.gate'), ('CG.2', 'M1.source')]:
            ask('wire.connect', {'from': a, 'to': b})
        restart()
        at_edge = step('200 ms')
        assert abs(at_edge['mixed']['mosfets'][0]['vgs_v']) < 1e-10
        charged = step('10 ms')
        charge_v = charged['mixed']['mosfets'][0]['vgs_v']
        tau, target = 1e-6/(1/10000+1/100000), -5*100000/110000
        charge_ref = target*(1-math.exp(-.01/tau))
        assert abs(charge_v-charge_ref) < .003
        at_rise = step('290 ms')
        assert at_rise['mixed']['mosfets'][0]['vgs_v'] < -4.5
        discharged = step('10 ms')
        discharge_v = discharged['mixed']['mosfets'][0]['vgs_v']
        discharge_ref = target*(1-math.exp(-.3/tau))*math.exp(-.01/tau)
        assert abs(discharge_v-discharge_ref) < .003
        # Two complementary devices share the same output; no fixed-current
        # surrogate or supply-only special-case is used for the high-side part.
        ask('project.clear', {'discard_changes': True})
        build([('CVCC', 'dc-source', -200, -100, '5 V', 0), ('CVIN', 'dc-source', -200, 100, '0 V', 0),
               ('MP', 'pmos', 100, -100, '', 0), ('MN', 'nmos', 100, 100, '', 0),
               ('ROUT', 'resistor', 250, 0, '10k', 0), ('CGND', 'ground', 300, 200, '', 0)],
              [('CVCC.positive', 'MP.source'), ('CVIN.positive', 'MP.gate'), ('MP.gate', 'MN.gate'),
               ('MP.drain', 'MN.drain'), ('MN.source', 'CGND.reference'), ('MN.drain', 'ROUT.1'),
               ('ROUT.2', 'CGND.reference'), ('CVCC.negative', 'CGND.reference'), ('CVIN.negative', 'CGND.reference')])
        inverter = []
        for voltage in [0, 2.5, 5]:
            ask('component.set', {'id': 'CVIN', 'value': f'{voltage} V'})
            state = restart()
            mp, mn = state['mixed']['mosfets']
            output = mn['vds_v']
            assert abs(mp['drain_current_a']+mn['drain_current_a']+output/10000) < 1e-9
            assert voltage != 0 or output > 4.99
            assert voltage != 5 or output < 1e-6
            inverter.append(dict(input_v=voltage, output_v=output, p_drain_a=mp['drain_current_a'], n_drain_a=mn['drain_current_a']))
        _, complementary = verify(['i:MP.drain', 'i:MN.drain', 'v:MN'])
        assert complementary['pmos_model']['id'] == part['model'] and complementary['nmos_model']['id'] == 'elab.generic-nmos.v1'
        ask('project.save', {'path': str(assets/'cmos.elab')})
        for state in [initial, falling, rising, charged, at_rise, discharged]:
            assert state['mixed']['kcl_residual_a'] < 1e-9 and abs(state['mixed']['power_residual_w']) < 1e-8
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', report_version=1,
                      part=part, place_contract=contract,
                      observations=dict(manual_matches_template=True, on=on, off=off, charge_10ms_v=charge_v,
                                        charge_reference_v=charge_ref, discharge_10ms_v=discharge_v,
                                        discharge_reference_v=discharge_ref, gate_rc_tau_s=tau, inverter=inverter,
                                        kept_recordings=loaded['document']['recording_count']),
                      checks=['Blank PMOS placement/named wiring matches guided example',
                              'Low gate turns the high-side LED on with negative signed drain/source currents and zero gate current',
                              'Pulse edge has both sides; external gate RC retains charge and matches analytic charge/discharge within 3 mV',
                              'Verification CSV/JSON and save/reopen preserve PMOS parameters; wrong threshold polarity refuses',
                              'Complementary NMOS/PMOS inverter switches across both rails with conservation and both captured model dependencies'],
                      limits=['Fixed generic square-law models; no physical matched-pair, intrinsic gate charge, breakdown, heating or ratings claim',
                              'Two MOSFETs total across both polarities; existing solver/global limits remain'])
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

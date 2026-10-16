#!/usr/bin/env python3
"""Exercise the complementary inverter through the public command interface."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import time

from validation_artifacts import discovery_summary

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def reference(input_voltage):
    # Separate scalar load-line calculation; <=1 nA reverse body leakage is
    # omitted. No calls to the application's solver or device helpers.
    def channel(gate, drain):
        overdrive = gate-1.5
        if overdrive <= 0:
            return 0.
        base = .05*overdrive**2
        if drain < overdrive:
            base = .1*(overdrive*drain-drain**2/2)
        return base*(1+.02*drain)
    low, high = 0., 5.
    for _ in range(70):
        output = (low+high)/2
        if channel(5-input_voltage, 5-output) > channel(input_voltage, output)+output/10000:
            low = output
        else:
            high = output
    output = (low+high)/2
    return output, -channel(5-input_voltage, 5-output), channel(input_voltage, output)


def physics(state):
    return {key: state.get(key) for key in ('run_id', 'electrical_revision', 'mixed', 'transient', 'result', 'npn')}


def invariant(state):
    return dict(physics(state), **{key: state.get(key) for key in ('components', 'wires', 'document', 'undo_name', 'redo_name')})


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint')
    parser.add_argument('--keep-open', action='store_true')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assert not args.keep_open or args.connect
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client, fresh, keep = helper.Client(args.connect), None, False
    mutate = lambda command, params=None: client.ask(command, params, True)
    try:
        assert not client.ask('project.inspect')['components'], 'requires a fresh owned blank app'
        catalog = client.ask('examples.list')
        assert any(item['id'] == 'cmos' and all(item[key] for key in ('question', 'expected', 'try', 'limitation')) for item in catalog)
        discovery = client.ask('capabilities.list', {'command': 'example.load'})
        assert 'cmos' in discovery['capabilities'][0]['input_schema']['properties']['id']['enum']
        contract = discovery_summary(discovery, assets/'example-load-discovery.json')
        mutate('example.load', {'id': 'cmos'})
        client.ask('simulation.wait')
        mutate('simulation.pause')
        mutate('simulation.restart')
        initial = client.ask('simulation.wait')
        probes = ['v:VIN.positive', 'v:MN', 'i:MP.drain', 'i:MN.drain']
        assert initial['selected'] == 'MN' and initial['scope']['selected'] == probes and not initial['example']['modified']
        before_capture = physics(initial)
        operation = mutate('simulation.verify', {'duration_s': .51, 'sample_step_s': .005, 'probe_ids': probes})['last_operation']
        deadline = time.monotonic()+20
        while True:
            view = client.ask('operation.inspect', {'id': operation})
            if view['operation']['status'] == 'completed':
                break
            assert view['operation']['status'] not in ('failed', 'canceled') and time.monotonic() < deadline, view
            time.sleep(.01)
        mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(assets/'inverter-capture.json')})
        capture = json.loads((assets/'inverter-capture.json').read_bytes())
        assert capture['metadata']['nmos_model'] and capture['metadata']['pmos_model']
        edges, voltage_errors, current_errors = {.2: set(), .5: set()}, [], []
        for row in capture['rows']:
            expected = reference(row['values'][0])
            voltage_errors.append(abs(row['values'][1]-expected[0]))
            current_errors.extend(abs(actual-want) for actual, want in zip(row['values'][2:], expected[1:]))
            for at, sides in edges.items():
                if abs(row['time_s']-at) < 1e-12:
                    sides.add(row['values'][0])
        assert max(voltage_errors) < 1e-5 and max(current_errors) < 1e-8
        assert all(sides == {0, 5} for sides in edges.values()) and physics(client.ask('project.inspect')) == before_capture
        levels = []
        for duration in (.2, .3):
            mutate('simulation.step', {'duration_s': duration})
            state = client.ask('project.inspect')
            expected = reference(state['transient']['sample']['input_v'])
            actual = state['mixed']['mosfets'][1]['vds_v']
            assert abs(actual-expected[0]) < 1e-5
            levels.append(dict(time_s=state['transient']['sample']['time_s'], input_v=state['transient']['sample']['input_v'], output_v=actual))
        mutate('component.set', {'id': 'VIN', 'value': '2.5 V'})
        assert client.ask('project.inspect')['mixed']['stale']
        mutate('simulation.restart')
        midpoint = client.ask('simulation.wait')
        mp, mn = midpoint['mixed']['mosfets']
        expected = reference(2.5)
        assert mp['drain_current_a'] < -.05 and mn['drain_current_a'] > .05
        assert abs(mn['vds_v']-expected[0]) < 1e-5 and midpoint['mixed']['kcl_residual_a'] < 1e-9 and abs(midpoint['mixed']['power_residual_w']) < 1e-8
        before_reset = invariant(midpoint)
        client.ask('example.current.reset', {}, True, 'unsaved-changes')
        assert invariant(client.ask('project.inspect')) == before_reset
        path = assets/'edited-inverter.elab'
        mutate('project.save', {'path': str(path)})
        saved_bytes = path.read_bytes()
        saved = json.loads(saved_bytes)
        fresh = helper.Client(None)
        fresh.ask('project.load', {'path': str(path)}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['components'] == saved['graph']['components'] and reopened['wires'] == saved['graph']['wires']
        assert reopened['example']['info']['id'] == 'cmos' and reopened['example']['modified'] and reopened['paused']
        assert reopened['scope']['selected'] == probes and reopened['document']['recording_count'] == 1 and reopened['transient']['sample']['time_s'] == 0
        assert saved['recordings']['entries'][0]['data']['rows'] == capture['rows']
        fresh.close()
        fresh = None
        mutate('example.current.reset')
        reset = client.ask('simulation.wait')
        assert not reset['example']['modified'] and reset['paused'] and reset['scope']['selected'] == probes
        assert next(part for part in reset['components'] if part['id'] == 'VIN')['value'] == 5
        mutate('edit.undo')
        assert client.ask('project.inspect')['example']['modified']
        mutate('edit.redo')
        mutate('simulation.restart')
        client.ask('simulation.wait')
        mutate('simulation.step', {'duration_s': .51})
        mutate('project.save', {'path': str(assets/'inverter.elab')})
        before_output = invariant(client.ask('project.inspect'))
        exports = {}
        for fmt in ('svg', 'png'):
            output = assets/f'inverter.{fmt}'
            mutate('export.schematic.'+fmt, {'path': str(output)})
            exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
        assert invariant(client.ask('project.inspect')) == before_output and path.read_bytes() == saved_bytes
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      default_probes=probes, capture_rows=len(capture['rows']),
                      maximum_reference_voltage_error_v=max(voltage_errors), maximum_reference_current_error_a=max(current_errors),
                      live_edge_levels=levels, midpoint=dict(output_v=mn['vds_v'], pmos_drain_a=mp['drain_current_a'], nmos_drain_a=mn['drain_current_a']),
                      rows_sha256=hashlib.sha256(json.dumps(capture['rows'], sort_keys=True).encode()).hexdigest(),
                      exports=exports, discovery=contract,
                      checks=['catalog/schema and selected output/four probes', 'independent load-line and both captured ideal-edge sides',
                              'live inversion and simultaneous conduction at intermediate input', 'Verify preserves live physics and captures both model dependencies',
                              'paused edit stales; dirty reset refuses; saved edited origin/capture reopens fresh',
                              'reset/undo/redo and pure SVG/PNG output'],
                      limitations=['generic pinned pair with zero gate charge/capacitance/delay; not a physical logic-gate or switching-loss qualification',
                                   'explicit 10k output load; independent reference omits <=1nA reverse body leakage',
                                   'public API/native callbacks are separate from full mounted keyboard/pointer/accessibility acceptance'])
        with args.output.open('x') as out:
            out.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({key: report[key] for key in ('passed', 'transport', 'capture_rows', 'midpoint')}, indent=2))
        if args.keep_open:
            mutate('scope.visibility.set', {'visible': True})
            mutate('workspace.layout.set', {'schematic_fraction': .5})
            mutate('view.fit', {'follow_viewport': True})
            keep = True
    finally:
        if fresh:
            fresh.close()
        if args.connect and not keep:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()

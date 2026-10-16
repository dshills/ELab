#!/usr/bin/env python3
"""Qualify the guided RC high-pass through public user/agent commands."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import time

from validation_artifacts import discovery_summary

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def physics(state):
    return {k: state.get(k) for k in ('run_id', 'electrical_revision', 'mixed', 'transient', 'result', 'npn')}


def invariant(state):
    return dict(physics(state), **{k: state.get(k) for k in ('components', 'wires', 'document', 'undo_name', 'redo_name')})


def reference(at, input_voltage):
    right_input, right_output = 0., 0.
    edge = 0
    while edge*.05 <= at+1e-12:
        delta = 5 if edge % 2 == 0 else -5
        right_input += delta
        right_output += delta*math.exp(-max(0, at-edge*.05)/.01)
        edge += 1
    capacitor = right_input-right_output
    return input_voltage-capacitor, capacitor


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closes unless --keep-open succeeds')
    parser.add_argument('--keep-open', action='store_true')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assert not args.keep_open or args.connect
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect)
    mutate = lambda name, params=None: client.ask(name, params, True)
    fresh = None
    keep = False

    def wait_capture(operation):
        deadline = time.monotonic()+20
        while True:
            view = client.ask('operation.inspect', {'id': operation})
            if view['operation']['status'] == 'completed':
                return
            assert view['operation']['status'] not in ('failed', 'canceled') and time.monotonic() < deadline
            time.sleep(.01)

    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh owned blank session'
        catalog = client.ask('examples.list')
        assert any(e['id'] == 'rc-high-pass' and all(e[k] for k in ('question', 'expected', 'try', 'limitation')) for e in catalog)
        discovery = client.ask('capabilities.list', {'command': 'example.load'})
        assert 'rc-high-pass' in discovery['capabilities'][0]['input_schema']['properties']['id']['enum']
        discovery_receipt = discovery_summary(discovery, assets/'example-load-discovery.json')
        mutate('example.load', {'id': 'rc-high-pass'})
        client.ask('simulation.wait')
        mutate('simulation.pause')
        mutate('simulation.restart')
        initial = client.ask('simulation.wait')
        probes = ['v:V1.positive', 'v:R1', 'v:C1']
        assert initial['selected'] == 'R1' and initial['scope']['selected'] == probes and not initial['example']['modified']
        before = physics(initial)
        operation = mutate('simulation.verify', {'duration_s': .12, 'probe_ids': probes})['last_operation']
        wait_capture(operation)
        mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(assets/'high-pass-capture.json')})
        captured = json.loads((assets/'high-pass-capture.json').read_bytes())
        errors, normalized_errors = [], []
        for row in captured['rows']:
            output, capacitor = reference(row['time_s'], row['values'][0])
            for want, actual in [(output, row['values'][1]), (capacitor, row['values'][2])]:
                error = abs(want-actual)
                errors.append(error)
                normalized_errors.append(error/max(.001, .001*abs(want)))
        assert max(normalized_errors) <= 1 and physics(client.ask('project.inspect')) == before
        mutate('simulation.step', {'duration_s': .05})
        edge = client.ask('project.inspect')
        falling = edge['mixed']['resistors'][0]['voltage_v']
        assert abs(falling-(5*math.exp(-5)-5)) < .003 and edge['transient']['sample']['input_v'] == 0
        mutate('simulation.step', {'duration_s': .01})
        negative = client.ask('project.inspect')['mixed']['resistors'][0]['voltage_v']
        assert abs(negative-(5*math.exp(-5)-5)*math.exp(-1)) < .003
        before_move = physics(client.ask('project.inspect'))
        mutate('component.move', {'id': 'C1', 'position': {'x': 20, 'y': -100}})
        assert physics(client.ask('project.inspect')) == before_move
        mutate('edit.undo')
        mutate('component.set', {'id': 'R1', 'value': '20k'})
        assert client.ask('project.inspect')['mixed']['stale']
        mutate('simulation.restart')
        client.ask('simulation.wait')
        mutate('simulation.step', {'duration_s': .01})
        slower = client.ask('project.inspect')['mixed']['resistors'][0]['voltage_v']
        assert abs(slower-5*math.exp(-.5)) < .003
        before_reset = invariant(client.ask('project.inspect'))
        client.ask('example.current.reset', {}, True, 'unsaved-changes')
        assert invariant(client.ask('project.inspect')) == before_reset
        path = assets/'edited-high-pass.elab'
        mutate('project.save', {'path': str(path)})
        saved_bytes = path.read_bytes()
        saved = json.loads(saved_bytes)
        assert saved['recordings']['entries'][0]['data']['rows'] == captured['rows']
        fresh = helper.Client(None)
        fresh.ask('project.load', {'path': str(path)}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['components'] == saved['graph']['components'] and reopened['wires'] == saved['graph']['wires']
        assert reopened['example']['info']['id'] == 'rc-high-pass' and reopened['example']['modified'] and reopened['paused']
        assert reopened['scope']['selected'] == probes and reopened['document']['recording_count'] == 1 and reopened['transient']['sample']['time_s'] == 0
        fresh.close()
        fresh = None
        mutate('project.load', {'path': str(path)})
        client.ask('simulation.wait')
        mutate('example.current.reset')
        reset = client.ask('simulation.wait')
        assert not reset['example']['modified'] and reset['paused'] and reset['scope']['selected'] == probes
        assert next(p for p in reset['components'] if p['id'] == 'R1')['value'] == 10000
        mutate('edit.undo')
        assert client.ask('project.inspect')['example']['modified']
        mutate('edit.redo')
        # Paused schematic history intentionally leaves readings stale.
        mutate('simulation.restart')
        assert not client.ask('simulation.wait')['mixed']['stale']
        mutate('project.save', {'path': str(assets/'high-pass.elab')})
        exports = {}
        before_export = invariant(client.ask('project.inspect'))
        for fmt in ('svg', 'png', 'pdf'):
            output = assets/f'high-pass.{fmt}'
            mutate('export.schematic.'+fmt, {'path': str(output)})
            if fmt == 'pdf':
                assert output.read_bytes().startswith(b'%PDF-')
            else:
                exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
        assert invariant(client.ask('project.inspect')) == before_export and path.read_bytes() == saved_bytes
        timing_before = invariant(client.ask('project.inspect'))
        elapsed_ms = []
        for _ in range(20):
            started = time.monotonic()
            timed = mutate('simulation.verify', {'duration_s': .2, 'probe_ids': probes})['last_operation']
            wait_capture(timed)
            elapsed_ms.append((time.monotonic()-started)*1000)
            mutate('recording.keep.set', {'id': timed, 'keep': False})
        assert max(elapsed_ms) <= 2000 and invariant(client.ask('project.inspect')) == timing_before
        timing = dict(repetitions=20, duration_s=.2, samples_ms=elapsed_ms,
                      p95_ms=sorted(elapsed_ms)[math.ceil(.95*len(elapsed_ms))-1], maximum_ms=max(elapsed_ms),
                      scope='Transport-inclusive backend completion on this workstation; not visible UI latency')
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      example_ids=[e['id'] for e in catalog], default_probes=probes,
                      capture_rows=len(captured['rows']), maximum_analytic_voltage_error_v=max(errors),
                      maximum_analytic_tolerance_ratio=max(normalized_errors),
                      falling_edge_output_v=falling, output_at_60ms_v=negative, doubled_resistance_output_at_10ms_v=slower,
                      rows_sha256=hashlib.sha256(json.dumps(captured['rows'], sort_keys=True).encode()).hexdigest(),
                      exports=exports, discovery=discovery_receipt, verify_200ms_timing=timing,
                      checks=['guided ID discoverable in catalog and command schema',
                              'live/captured positive and negative exponential responses agree with independent RC solution',
                              'geometry preserves physics; paused electrical edit stales; doubled R slows decay',
                              'kept baseline recording survives edited-origin fresh-process save/open',
                              'dirty reset refuses; clean reset restores template/probes; schematic undo/redo works',
                              'SVG/PNG/PDF exports preserve graph, physics, history and document'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: report[k] for k in ('passed', 'transport', 'maximum_analytic_voltage_error_v', 'falling_edge_output_v', 'output_at_60ms_v')}, indent=2))
        if args.keep_open:
            mutate('simulation.step', {'duration_s': .12})
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

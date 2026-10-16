#!/usr/bin/env python3
"""Validate LC sampling advice/fit through a fresh owned public session."""
import argparse
import importlib.util
import json
import math
from pathlib import Path
import time

from validation_artifacts import discovery_bundle

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def readings(state):
    result = state['mixed']
    return dict(time_s=result['time_s'], inductors=result['inductors'], capacitors=result['capacitors'])


def invariants(state):
    return {key: state.get(key) for key in ('components', 'wires', 'transient_settings', 'document',
        'run_id', 'electrical_revision', 'mixed', 'transient', 'undo_entry', 'redo_entry')}


def reference(at):
    damped = math.sqrt(10000**2-2500**2)
    return 5*(1-math.exp(-2500*at)*(math.cos(damped*at)+2500/damped*math.sin(damped*at)))


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closed afterward unless --keep-open')
    parser.add_argument('--keep-open', action='store_true')
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert not args.keep_open or args.connect
    assert not args.output.exists()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client, reopened = connect_demo_client(helper.Client, args.connect), None
    mutate = lambda command, params=None: client.ask(command, params, True)
    captures = {}
    admitted_session = False

    def capture(name, step=None):
        params = {'duration_s': '2 ms', 'probe_ids': ['v:C1', 'i:L1']}
        if step is not None:
            params['sample_step_s'] = step
        before = readings(client.ask('project.inspect'))
        operation = mutate('simulation.verify', params)['last_operation']
        deadline = time.monotonic()+20
        while True:
            view = client.ask('operation.inspect', {'id': operation})
            status = view['operation']['status']
            if status == 'completed':
                break
            assert status not in ('failed', 'canceled') and time.monotonic() < deadline, view
            time.sleep(.01)
        mutate('verification.export', {'id': operation, 'path': str(assets/(name+'.json')), 'format': 'json'})
        document = json.loads((assets/(name+'.json')).read_text())
        errors = [abs(row['values'][0]-reference(row['time_s'])) for row in document['rows']]
        assert max(errors) < .001
        assert readings(client.ask('project.inspect')) == before
        captures[name] = dict(rows=len(document['rows']), step_s=document['metadata']['settings']['step_s'],
            sampled_peak_v=max(row['values'][0] for row in document['rows']), max_reference_error_v=max(errors))
        return operation, document

    try:
        assert not client.ask('project.inspect')['components'], 'requires a fresh blank owned session'
        admitted_session = True
        discovery = discovery_bundle({cmd: client.ask('capabilities.list', {'command': cmd}) for cmd in
            ('simulation.sampling.fit', 'simulation.transient.set')}, assets/'discovery.json')
        mutate('example.load', {'id': 'rlc'})
        mutate('simulation.pause')
        for identity, value in [('L1', '10 mH'), ('C1', '1 µF')]:
            mutate('component.set', {'id': identity, 'value': value})
        settings = client.ask('project.inspect')['transient_settings']
        settings['step_s'] = .001
        mutate('simulation.transient.set', settings)
        mutate('simulation.restart')
        client.ask('simulation.wait')
        mutate('scope.probes.set', {'ids': ['v:C1', 'i:L1']})
        mutate('simulation.step', {'duration_s': '200 us'})
        coarse_id, coarse_document = capture('coarse')
        before = client.ask('project.inspect')
        advice = before['lc_sampling']
        assert advice['unresolved'] and advice['can_fit'] and not advice['minimum_limited']
        assert advice['inductor_id'] == 'L1' and advice['capacitor_id'] == 'C1'
        assert not before.get('source_warnings')
        assert abs(advice['target_step_s']-2*math.pi*math.sqrt(.01*1e-6)/32) < 1e-20
        assert 'not a measured period' in advice['note']

        # Exercise actual retry retention, including the exact old guards.
        request = dict(version=1, id='lc-fit', key='lc-fit', command='simulation.sampling.fit', params={})
        def dispatch():
            client.process.stdin.write(json.dumps(request)+'\n')
            client.process.stdin.flush()
            return json.loads(client.process.stdout.readline())
        for _ in range(8):
            guard = client.ask('project.inspect')
            request.update(expected_command_revision=guard['command_revision'],
                expected_document_generation=guard['document']['generation'])
            response = dispatch()
            if response.get('error', {}).get('code') != 'conflict':
                break
        assert 'error' not in response, response
        after = client.ask('project.inspect')
        assert readings(after) == readings(before) and after['mixed']['stale'] and after['paused']
        assert after['transient_settings']['step_s'] == advice['recommended_step_s']
        assert not after['lc_sampling']['unresolved'] and not after['lc_sampling']['can_fit']
        duplicate = dispatch()
        assert duplicate.get('duplicate') and 'error' not in duplicate
        assert invariants(client.ask('project.inspect')) == invariants(after)
        mutate('verification.export', {'id': coarse_id, 'path': str(assets/'coarse-after-fit.json'), 'format': 'json'})
        assert (assets/'coarse.json').read_bytes() == (assets/'coarse-after-fit.json').read_bytes()
        capture('fitted')
        capture('explicit-coarse', '1 ms')
        peak = reference(math.pi/math.sqrt(10000**2-2500**2))
        assert captures['coarse']['rows'] == captures['explicit-coarse']['rows'] == 3
        assert captures['fitted']['rows'] > 100
        assert captures['coarse']['sampled_peak_v'] < peak-1
        assert abs(captures['fitted']['sampled_peak_v']-peak) < .025
        fitted_settings = client.ask('project.inspect')['transient_settings']
        path = assets/'fitted.elab'
        mutate('project.save', {'path': str(path)})
        original = path.read_bytes()
        assert 'lc_sampling' not in json.loads(original)
        reopened = helper.Client(None)
        reopened.ask('project.load', {'path': str(path)}, True)
        opened = reopened.ask('simulation.wait')
        assert opened['paused'] and opened['mixed']['time_s'] == 0
        assert opened['transient_settings'] == fitted_settings and not opened['lc_sampling']['unresolved']
        assert opened['document']['recording_count'] == 3 and path.read_bytes() == original
        reopened.close()
        reopened = None
        mutate('edit.undo')
        assert client.ask('project.inspect')['transient_settings']['step_s'] == .001
        mutate('edit.redo')
        assert client.ask('project.inspect')['transient_settings'] == fitted_settings
        edits = [{'command': 'component.set', 'params': {'id': 'L1', 'value': '2.5 mH'}},
                 {'command': 'simulation.sampling.fit', 'params': {}}]
        bad = edits+[{'command': 'component.set', 'params': {'id': 'missing', 'value': '1k'}}]
        before = invariants(client.ask('project.inspect'))
        client.ask('edit.batch', {'name': 'Rejected fit', 'edits': bad}, True, 'invalid-params')
        assert invariants(client.ask('project.inspect')) == before
        mutate('edit.batch', {'name': 'Tune and fit', 'edits': edits})
        batch = client.ask('project.inspect')
        assert batch['undo_name'] == 'Tune and fit'
        assert abs(batch['transient_settings']['step_s']-2*math.pi*math.sqrt(.0025*1e-6)/32) < 1e-20
        mutate('edit.undo')
        assert client.ask('project.inspect')['transient_settings'] == fitted_settings
        for identity, value in [('L1', '1 µH'), ('C1', '1 nF')]:
            mutate('component.set', {'id': identity, 'value': value})
        mutate('simulation.sampling.fit')
        floor = client.ask('project.inspect')
        assert floor['transient_settings']['step_s'] == 1e-6 and floor['transient_settings']['seconds_per_wall_second'] == .001
        assert floor['lc_sampling']['minimum_limited'] and floor['lc_sampling']['unresolved'] and not floor['lc_sampling']['can_fit']
        assert 'cannot resolve' in floor['lc_sampling']['note']
        mutate('simulation.sampling.fit')
        unchanged = client.ask('project.inspect')
        assert unchanged['run_id'] == floor['run_id'] and unchanged['electrical_revision'] == floor['electrical_revision']
        assert path.read_bytes() == original
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
            discovery=discovery, analytic_peak_v=peak, captures=captures,
            fitted_step_s=fitted_settings['step_s'], fitted_playback=fitted_settings['seconds_per_wall_second'],
            retained_live_time_s=after['mixed']['time_s'], minimum_advice=floor['lc_sampling'],
            reopened_recordings=opened['document']['recording_count'], retry_retained=True,
            checks=['DC-powered LC advice', 'independent RLC rows and recovered sampled peak',
                'advanced paused physics retained stale', 'exact retry retention', 'existing captured samples unchanged',
                'explicit Verify override preserved', 'one undo/redo and fresh-process persistence',
                'staged-value atomic fit and rollback', 'unresolved supported-minimum limitation', 'original saved file unchanged'],
            limitations=['configured-value LC reference is not a measured period, circuit-pole estimate or resolution guarantee',
                'public/native API workflow is separate from physical typing, focus, accessibility and visible-latency acceptance'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as out:
            out.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({key: report[key] for key in ('passed', 'transport', 'analytic_peak_v', 'captures', 'fitted_step_s')}, indent=2))
    finally:
        if args.connect and admitted_session and not args.keep_open:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()
        if reopened:
            reopened.close()


if __name__ == '__main__':
    main()

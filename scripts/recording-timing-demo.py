#!/usr/bin/env python3
"""Qualify RC, pulse NPN and both flasher-side timing via public commands."""
import argparse
import importlib.util
import json
import math
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    cases = []

    def capture(example, duration, probes, **settings):
        client.ask(example, {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        state = client.ask('simulation.verify', dict(duration_s=duration, probe_ids=probes, **settings), True)
        identifier = state['last_operation']
        deadline = time.monotonic() + 15
        while True:
            view = client.ask('operation.inspect', {'id': identifier})
            assert view['operation']['status'] not in ('failed', 'canceled'), view
            if view['operation']['status'] == 'completed':
                return identifier
            assert time.monotonic() < deadline
            time.sleep(.01)

    def semantic(result):
        output = json.loads(json.dumps(result))
        output['settings'].pop('id')
        output.pop('origin_id')
        return output

    def measure(params):
        before = client.ask('project.inspect')
        result = client.ask('recording.timing', params)
        assert client.ask('project.inspect') == before
        configured = client.ask('recording.timing.set', params, True)
        assert configured['recording']['timing'] == result
        for key in ('components', 'wires', 'document', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert configured.get(key) == before.get(key), key
        return result

    try:
        details = client.ask('capabilities.list', {'command': 'recording.timing'})['capabilities'][0]['input_schema']
        assert details['properties']['limit']['minimum'] == 0 and details['properties']['limit']['maximum'] == 100
        assert details['properties']['level']['x-max-utf8-bytes'] == 128 and details['x-command-contract']
        identifier = capture('example.rc', '20 ms', ['v:C1'], input_mode='charge', sample_step_s='1 ms')
        params = dict(id=identifier, probe_id='v:C1', level='2.5 V', direction='rising', start_s=0, end_s='20 ms', limit=100)
        result = measure(params)
        assert result['total_crossings'] == 1
        crossing = result['crossings'][0]
        assert crossing['before']['time_s'] < .01 * math.log(2) < crossing['after']['time_s']
        cases.append(dict(case='rc-half-charge', result=semantic(result)))
        identifier = capture('example.npn-pulse', '250 ms', ['v:VIN.positive'])
        params = dict(id=identifier, probe_id='v:VIN.positive', level='2.5 V', direction='both', start_s=0, end_s='250 ms', limit=100)
        result = measure(params)
        assert result['total_crossings'] >= 4
        for crossing in result['crossings']:
            assert crossing['before']['time_s'] == crossing['after']['time_s']
        for side in ('rising', 'falling'):
            spacing = result['first_' + side + '_spacing']
            assert math.isclose(spacing['minimum_s'], .1, abs_tol=1e-12)
            assert math.isclose(spacing['maximum_s'], .1, abs_tol=1e-12)
        pages = []
        offset = 0
        while offset < result['total_crossings']:
            page = client.ask('recording.timing', dict(params, offset=offset, limit=1))
            pages += page['crossings']
            offset = page['next_offset']
        assert pages == result['crossings']
        cases.append(dict(case='ideal-npn-pulse', result=semantic(result)))
        identifier = capture('example.flasher', '1.2 s', ['i:D1', 'i:D2'])
        for probe in ('i:D1', 'i:D2'):
            params = dict(id=identifier, probe_id=probe, level='1 mA', direction='rising', start_s='300 ms', end_s='1.2 s', limit=100)
            result = measure(params)
            spacing = result['first_rising_spacing']
            assert .47 < spacing['minimum_s'] <= spacing['maximum_s'] < .51
            cases.append(dict(case='flasher-' + probe, result=semantic(result)))
        before = client.ask('project.inspect')
        for bad, code in [(dict(params, level='1 V'), 'invalid-params'), (dict(params, end_s='1.3 s'), 'invalid-params'),
                          (dict(params, limit=101), 'invalid-params'), (dict(params, probe_id='missing'), 'not-found')]:
            client.ask('recording.timing', bad, expect=code)
            assert client.ask('project.inspect') == before
        asset = directory / 'flasher.json'
        client.ask('verification.export', dict(id=identifier, path=str(asset), format='json'), True)
        client.ask('verification.import', dict(path=str(asset)), True)
        imported = client.ask('project.inspect')['recording']['id']
        imported_result = client.ask('recording.timing', dict(params, id=imported))
        assert semantic(imported_result) == semantic(result)
        project = directory / 'timed-captures.elab'
        client.ask('project.save', dict(path=str(project)), True)
        client.ask('project.load', dict(path=str(project)), True)
        client.ask('simulation.wait')
        state = client.ask('project.inspect')
        assert 'timing' not in state['recording']
        reopened = client.ask('recording.timing', dict(params, id=state['recording']['id']))
        assert semantic(reopened) == semantic(result)
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', cases=cases,
                      checks=['Discovered threshold/page/state contracts', 'Analytic RC half-charge bracket', 'Ideal pulse duplicate edge brackets and 100 ms spacing',
                              'Both flasher sides measured after startup', 'Paged arrivals equal whole-window data',
                              'Pure/refused queries preserve state; configuration preserves live/history/dirty state',
                              'Imported/reopened saved data yields identical timing'],
                      limits=['Sampled brackets only; hidden transitions may be missed', 'Startup spacing is not a steady period',
                              'No interpolation, time alignment or real-device certification'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], cases=len(cases))))
    finally:
        client.close()


if __name__ == '__main__':
    main()

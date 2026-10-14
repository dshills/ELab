#!/usr/bin/env python3
"""Public integration-limit guidance, atomic failure and recovery/retry checks.
--connect replaces only a designated owned development session.
"""
import argparse
import importlib.util
import json
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)

    def failed_request(id, key):
        client.sequence += 1
        request = {'version': 1, 'id': id, 'command': 'simulation.step', 'params': {'duration_s': .2},
                   'expected_command_revision': client.command_revision, 'key': key}
        client.process.stdin.write(json.dumps(request) + '\n')
        client.process.stdin.flush()
        response = json.loads(client.process.stdout.readline())
        client.revision = response['revision']
        if isinstance(response.get('result'), dict):
            client.command_revision = response['result']['command_revision']
        return response

    try:
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('edit.batch', {'name': 'Budget diagnostic fixture', 'edits': [
            {'command': 'component.place', 'params': {'id': 'Fast', 'kind': 'finite-pulse-source', 'position': {'x': -200, 'y': 0}}},
            {'command': 'component.place', 'params': {'id': 'Load', 'kind': 'resistor', 'value': '10k', 'position': {'x': 0, 'y': -100}}},
            {'command': 'component.place', 'params': {'id': 'Return', 'kind': 'ground', 'position': {'x': 200, 'y': 100}}},
            {'command': 'wire.connect', 'params': {'from': 'Fast.positive', 'to': 'Load.1'}},
            {'command': 'wire.connect', 'params': {'from': 'Load.2', 'to': 'Return.reference'}},
            {'command': 'wire.connect', 'params': {'from': 'Return.reference', 'to': 'Fast.negative'}},
        ]}, True)
        client.ask('source.pulse.set', {'id': 'Fast', 'waveform': {'high_v': 5, 'low_v': 0, 'high_s': .0005, 'low_s': .0005, 'rise_s': .0001, 'fall_s': .0001}}, True)
        cfg = {'step_s': .2, 'high_s': .05, 'low_s': .05, 'initial_voltage_v': 0, 'seconds_per_wall_second': .02}
        client.ask('simulation.transient.set', cfg, True)
        client.ask('simulation.restart', {}, True)
        before = client.ask('simulation.wait')
        assert before['limits']['integration_attempts_per_sample'] == 512
        assert before['limits']['integration_wall_ms_per_sample'] == 100
        key = 'coarse-budget-step'
        failure = failed_request('coarse-step', key)
        assert failure['error']['code'] == 'integration-attempt-limit', failure
        failed = client.ask('project.inspect')
        for field in ['transient', 'mixed', 'components', 'wires', 'run_id', 'electrical_revision', 'document', 'undo_entry']:
            assert before.get(field) == failed.get(field), field
        help = next(i for i in failed['diagnostic_presentation']['items'] if i['source'] == 'step')
        assert help['code'] == failure['error']['code'] and 'calculation attempts' in help['summary']
        assert 'sample interval' in help['action'] and 'maximum 512' in help['details']
        assert not help['object_ids'] and not help['terminal_refs'] and not help['wire_ids']
        client.ask('diagnostics.details.set', {'visible': True}, True)
        assert client.ask('project.inspect')['diagnostic_presentation']['details_visible']

        accepted = client.ask('simulation.verify', {'duration_s': .2, 'sample_step_s': .2, 'probe_ids': ['v:Fast.positive']}, True)
        op = accepted['last_operation']
        deadline = time.monotonic() + 5
        while True:
            record = client.ask('operation.inspect', {'id': op})
            if record['operation']['status'] not in ['queued', 'running']:
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        assert record['operation']['status'] == 'failed' and record['operation']['error']['code'] == 'integration-attempt-limit'
        assert not record['operation']['kept_in_project'] and record.get('metadata')
        client.ask('operation.samples', {'id': op, 'limit': 1}, expect='not-ready')
        assert client.ask('project.inspect')['document']['recording_count'] == 0

        cfg['step_s'] = .001
        client.ask('simulation.transient.set', cfg, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': .001}, True)
        recovered = client.ask('project.inspect')
        assert recovered['step_work']['status'] == 'completed' and not recovered['transient']['stale']
        assert recovered['transient']['sample']['time_s'] == .001
        assert not any(i['code'] == 'integration-attempt-limit' for i in recovered['diagnostic_presentation']['items'])
        # A known failure's retry key returns its original outcome without
        # reapplying a coarse step to the newer successful simulation.
        retry = failed_request('coarse-step-retry', key)
        assert retry.get('duplicate') and retry['error'] == failure['error']
        after_retry = client.ask('project.inspect')
        assert recovered['transient'] == after_retry['transient'] and recovered['run_id'] == after_retry['run_id']
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'limits': {'attempts_per_sample': 512, 'wall_ms_per_sample': 100},
                  'step_failure_code': failure['error']['code'], 'step_details': failure['error']['message'],
                  'recorded_failure_code': record['operation']['error']['code'], 'recovered_time_s': .001,
                  'checks': ['machine-readable limit discovery', 'real coarse-sample attempt rejection',
                             'source/history/physics preservation', 'readable shared guidance without guessed part',
                             'failed Verify retains input metadata but no partial rows/kept asset',
                             'smaller sample recovers and clears old failure', 'failed retry key preserves original outcome without reapplication'],
                  'limits_of_evidence': ['One millisecond recovers the work budget but still aliases this 1 kHz source; sampling warnings remain separate',
                                         'Time-limit pressure evidence is in the isolated Go audit; this public workflow checks attempt limits']}
        client.ask('project.clear', {'discard_changes': True}, True)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        client.close()


if __name__ == '__main__':
    main()

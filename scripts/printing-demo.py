#!/usr/bin/env python3
"""Validate public printer discovery/settings/refusals without submitting paper."""
import argparse
import importlib.util
import json
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def await_state(client, field):
    deadline = time.monotonic() + 35
    while True:
        state = client.ask('project.inspect')
        work = state[field] if field == 'print_preview' else state[field]['work']
        if work['status'] not in ('queued', 'running', 'canceling'):
            return state
        assert time.monotonic() < deadline, work
        time.sleep(.01)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        before = client.ask('simulation.wait')
        details = {}
        for command in ('print.printers.refresh', 'print.destination.set', 'print.submit',
                        'print.job.refresh', 'print.job.cancel', 'print.work.cancel'):
            cap = client.ask('capabilities.list', {'command': command})['capabilities'][0]
            assert cap['input_schema']['x-command-contract']['version'] == 1
            details[command] = cap['input_schema']
        assert before == client.ask('project.inspect')
        client.ask('print.preview.start', {}, True)
        state = await_state(client, 'print_preview')
        assert state['print_preview']['status'] == 'completed'
        preview = state['print_preview']
        client.ask('print.destination.set', {'printer_id': '', 'copies': 2}, True)
        client.ask('print.printers.refresh', {}, True)
        state = await_state(client, 'printing')
        # Read-only discovery may legitimately find no printers or a stopped service.
        discovery = {'status': state['printing']['work']['status'],
                     'destination_count': len(state['printing']['destinations']),
                     'error_code': state['printing']['work'].get('error', {}).get('code')}
        assert state['printing']['printer_id'] == '' and state['printing']['copies'] == 2
        for command, params, code in (
                ('print.submit', {'preview_id': preview['id'], 'printer_id': '__missing_elab_queue__', 'copies': 1}, 'unavailable'),
                ('print.submit', {'preview_id': preview['id'], 'printer_id': '', 'copies': 0}, 'invalid-params'),
                ('print.destination.set', {'printer_id': '', 'copies': 1.5}, 'invalid-params'),
                ('print.job.cancel', {'id': 'system-job-42'}, 'not-found'),
                ('print.work.cancel', {'id': 'retired'}, 'not-found')):
            before_refusal = client.ask('project.inspect')
            client.ask(command, params, True, code)
            assert before_refusal == client.ask('project.inspect')
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        after = client.ask('project.inspect')
        assert not after['print_preview']['source_current']
        assert after['print_preview']['sha256'] == preview['sha256']
        assert after['printing']['jobs'] == []
        for key in ('components', 'wires', 'run_id', 'electrical_revision', 'transient', 'undo_entry', 'document'):
            assert before.get(key) == state.get(key), key
        client.ask('print.preview.close', {'id': preview['id']}, True)
        client.ask('project.clear', {'discard_changes': True}, True)
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      detailed_schemas=details, discovery=discovery, preview_digest=preview['sha256'],
                      checks=['six detailed printing contracts', 'discovery and session copy settings',
                              'no automatic printer selection', 'missing queue, copy and foreign job refusals preserve state',
                              'captured preview remains intact during source edits', 'no spool submissions in this public workflow'],
                      limits=['Positive submit/refresh/cancel use service fakes and actual libcups against an isolated IPP fixture',
                              'No physical print, printer setup, device margins/scaling or actual-device cancellation acceptance'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], discovery=discovery, physical_jobs_submitted=0)))
    finally:
        client.close()


if __name__ == '__main__':
    main()

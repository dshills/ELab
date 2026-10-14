#!/usr/bin/env python3
"""Exercise trace/source association over the public user/agent commands."""
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
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    try:
        client.ask('example.flasher', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('scope.probes.set', {'ids': ['i:D1', 'v:C1']}, True)
        before = client.ask('simulation.step', {'duration_s': .01}, True)
        focused = client.ask('scope.trace.select', {'id': 'i:D1'}, True)
        assert focused['trace_focus']['object_ids'] == ['D1']
        assert focused['trace_focus']['terminal_refs'] == ['D1.anode', 'D1.cathode']
        assert len(focused['trace_focus']['net_ids']) == 2
        for field in ['transient', 'run_id', 'electrical_revision', 'selected', 'document']:
            assert focused[field] == before[field], field
        client.ask('scope.trace.select', {'id': 'i:D2'}, True, 'not-found')
        client.ask('object.delete', {'id': 'D1'}, True)
        missing = client.ask('project.inspect')['trace_focus']
        assert missing['unavailable'] and not missing['object_ids']
        client.ask('edit.undo', {}, True)
        restored = client.ask('simulation.wait')
        assert not restored['trace_focus']['unavailable']
        accepted = client.ask('simulation.verify', {'duration_s': .01, 'probe_ids': ['i:D1', 'v:C1']}, True)
        deadline = time.monotonic()+15
        while True:
            record = client.ask('operation.inspect', {'id': accepted['last_operation']})
            if record['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            assert time.monotonic() < deadline
            time.sleep(.005)
        assert record['operation']['status'] == 'completed'
        client.ask('operation.select', {'id': accepted['last_operation']}, True)
        recorded = client.ask('recording.view.set', {'mode': 'recorded'}, True)
        assert recorded['trace_focus']['unavailable'] and not recorded['trace_focus']['object_ids']
        live = client.ask('recording.view.set', {'mode': 'live'}, True)
        assert not live['trace_focus']['unavailable']
        removed = client.ask('scope.probe.remove', {'id': 'i:D1'}, True)
        assert 'trace_focus' not in removed
        client.ask('scope.trace.select', {'id': 'v:C1'}, True)
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'focused-flasher.elab')
        client.ask('project.save', {'path': path, 'overwrite': True}, True)
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert 'trace_focus' not in reopened and reopened['paused'] and not reopened['document']['dirty']
        cleared = client.ask('scope.trace.select', {'id': ''}, True)
        assert 'trace_focus' not in cleared and not cleared['document']['dirty'] and cleared['paused']
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      checks=['exact component/terminal/net associations', 'simulation/history/document preserved by focus',
                              'unselected trace refused', 'removed source unavailable and undo restores it',
                              'recorded plot never highlights current live schematic', 'trace removal and explicit clear',
                              'fresh paused reopen clears ephemeral focus', 'owned session ends clean and paused'],
                      limitations=['focus is session inspection state, not a saved preference',
                                   'recorded schematic interactive association is not implemented',
                                   'GPU and mounted control coverage are separate checks'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

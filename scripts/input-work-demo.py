#!/usr/bin/env python3
"""Public asynchronous wiper/input contract. --connect replaces its owned session."""
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
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('edit.batch', {'name': 'Pot capacitor experiment', 'edits': [
            {'command': 'object.delete', 'params': {'id': 'R1'}},
            {'command': 'component.place', 'params': {'id': 'Timing', 'kind': 'potentiometer', 'position': {'x': 0, 'y': -110}, 'value': '10k'}},
            {'command': 'wire.connect', 'params': {'from': 'V1.positive', 'to': 'Timing.1'}},
            {'command': 'wire.connect', 'params': {'from': 'Timing.2', 'to': 'GND.reference'}},
            {'command': 'wire.connect', 'params': {'from': 'Timing.wiper', 'to': 'C1.1'}},
        ]}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': .005}, True)
        before = client.ask('project.inspect')
        started = time.perf_counter()
        admitted = client.ask('potentiometer.wiper.set', {'id': 'Timing', 'position': .75}, True)
        admission_ms = (time.perf_counter()-started)*1000
        work = admitted['input_work']
        assert work['status'] == 'queued' and work['requested_wiper']['position'] == .75
        assert admitted['components'] == before['components'] and admitted['mixed'] == before['mixed']
        assert admitted['undo_entry'] == before['undo_entry']
        completed = client.ask('simulation.wait')
        assert completed['input_work']['id'] == work['id'] and completed['input_work']['status'] == 'completed'
        assert completed['run_id'] == before['run_id'] and completed['electrical_revision'] == before['electrical_revision']
        assert completed['transient']['sample']['time_s'] == before['transient']['sample']['time_s']
        assert abs(completed['mixed']['capacitors'][0]['voltage_v']-before['mixed']['capacitors'][0]['voltage_v']) < 1e-10
        assert completed['actuation_revision'] == before['actuation_revision']+2
        assert completed['undo_entry']['actor'] == work['actor']
        client.ask('simulation.input.set', {'mode': 'low'}, True)
        resumed = client.ask('simulation.resume', {}, True)
        assert resumed['input_work']['status'] == 'queued' and resumed['input_work']['requested_mode'] == 'low'
        applied = client.ask('simulation.wait')
        assert applied['input_work']['status'] == 'completed' and applied['transient']['input_mode'] == 'low'
        # Running changes also use the same algebraic worker; dependent commands wait.
        high = client.ask('simulation.input.set', {'mode': 'high'}, True)
        assert high['input_work']['status'] == 'queued'
        applied = client.ask('simulation.wait')
        assert applied['input_work']['status'] == 'completed' and applied['transient']['input_mode'] == 'high'
        client.ask('simulation.pause', {}, True)
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'input-rc.elab')
        saved = client.ask('project.save', {'path': path, 'overwrite': True}, True)
        assert not saved['document']['dirty']
        client.ask('export.schematic.svg', {'path': str(directory/'input-rc.svg'), 'overwrite': True}, True)
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and 'input_work' not in reopened
        assert reopened['transient']['sample']['time_s'] == 0
        assert next(p for p in reopened['components'] if p['id'] == 'Timing')['wiper_position'] == .75
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            wiper_admission_ms=admission_ms, work=work, capacitor_voltage_v=completed['mixed']['capacitors'][0]['voltage_v'],
            checks=['queued admission preserves committed graph/physics/history', 'worker publishes one actor-owned wiper edit',
                'charge/time/run/electrical identity retained', 'admission and publication have distinct actuation revisions',
                'queued Resume and running input changes use worker', 'saved paused circuit reopens fresh without job state'],
            limitations=['blocked-worker cancellation, failure and responsiveness are separately covered by controlled race tests',
                'backend timing excludes input-to-pixel latency', 'some file/recovery work remains synchronous'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

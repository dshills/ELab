#!/usr/bin/env python3
"""Shared time-unit inputs across live, recorded, source and recovery controls.
--connect replaces only a designated owned development session.
"""
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
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=True)
    client = helper.Client(args.connect, command_preconditions=True)
    try:
        catalog = client.ask('capabilities.list')
        assert catalog['schemas']['urn:elab:time:1']['maxLength'] == 128
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.transient.set', {'step_s': '1 ms', 'high_s': '50 ms', 'low_s': '.05 s',
            'initial_voltage_v': 0, 'seconds_per_wall_second': .02}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        stepped = client.ask('simulation.step', {'duration_s': '10 ms'}, True)
        assert abs(stepped['transient']['sample']['voltage_v']-5*(1-math.exp(-1))) < 1e-12
        client.ask('scope.cursor.set', {'cursor': 'a', 'time_s': '0 s'}, True)
        client.ask('scope.cursor.set', {'cursor': 'b', 'time_s': '10000 μs'}, True)
        scoped = client.ask('scope.window.set', {'mode': 'fixed', 'start_s': '0 ms', 'end_s': '.01 s'}, True)
        assert abs(scoped['scope']['delta_s']-.01) < 1e-12
        before = client.ask('project.inspect')
        for text in ['11 ms', '10 Hz', '-1 ms', '1e309 s', '1e-400 s']:
            client.ask('scope.cursor.set', {'cursor': 'b', 'time_s': text}, True, expect='invalid-params')
        assert client.ask('project.inspect') == before
        client.ask('verification.settings.set', {'duration_s': '200 ms', 'sample_step_s': '1 ms',
            'input_mode': 'auto', 'capacitor_initials': []}, True)
        admitted = client.ask('simulation.verify', {}, True)
        operation_id = admitted['last_operation']
        deadline = time.monotonic()+10
        while True:
            operation = client.ask('operation.inspect', {'id': operation_id})
            if operation['operation']['status'] == 'completed':
                break
            assert operation['operation']['status'] not in ['failed', 'canceled'] and time.monotonic() < deadline, operation
            time.sleep(.01)
        client.ask('operation.select', {'id': operation_id}, True)
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        client.ask('recording.seek', {'time_s': '10 ms'}, True)
        client.ask('recording.cursor.set', {'cursor': 'a', 'time_s': '10 ms'}, True)
        client.ask('recording.cursor.set', {'cursor': 'b', 'time_s': '20 ms'}, True)
        recorded = client.ask('recording.window.set', {'start_s': '10 ms', 'end_s': '200 ms'}, True)
        assert abs(recorded['recording']['scope']['delta_s']-.01) < 1e-12
        assert recorded['transient']['sample'] == before['transient']['sample']
        recovery = recorded['recovery']
        client.ask('recovery.configure', {'directory': recovery.get('directory') or str(directory/'recovery'),
            'enabled': False, 'interval_s': '2 s'}, True)
        rc_path = directory/'unit-rc.elab'
        client.ask('project.save', {'path': str(rc_path), 'overwrite': True}, True)
        rc_bytes = rc_path.read_bytes()
        document = json.loads(rc_bytes)
        assert document['transient_settings']['step_s'] == .001
        assert document['recordings']['entries'][0]['data']['metadata']['duration_s'] == .2
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('project.load', {'path': str(rc_path)}, True)
        reopened = client.ask('simulation.wait')
        assert reopened['paused'] and reopened['transient']['sample']['time_s'] == 0
        assert reopened['transient']['sample']['voltage_v'] == 0
        assert reopened['operations'] and reopened['operations'][0]['status'] == 'completed'
        client.ask('recording.view.set', {'mode': 'live'}, True)
        client.ask('object.delete', {'id': 'V1'}, True)
        client.ask('component.place', {'kind': 'finite-pulse-source', 'id': 'FiniteDrive', 'position': {'x': -200, 'y': 0}}, True)
        for source, target in [('FiniteDrive.positive', 'R1.1'), ('FiniteDrive.negative', 'GND.reference')]:
            client.ask('wire.connect', {'from': source, 'to': target}, True)
        client.ask('source.pulse.set', {'id': 'FiniteDrive', 'waveform': {'high_v': 5, 'low_v': 0,
            'high_s': '.05 s', 'low_s': '50 ms', 'rise_s': '1 ms', 'fall_s': '1000 us'}}, True)
        client.ask('simulation.sampling.fit', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        started = client.ask('simulation.step.start', {'duration_s': '5 ms'}, True)
        final = client.ask('simulation.wait')
        assert final['step_work']['status'] == 'completed'
        assert abs(final['transient']['sample']['time_s']-.005) < 1e-12
        client.ask('project.save', {'path': str(directory/'finite-unit-rc.elab'), 'overwrite': True}, True)
        assert not client.ask('project.inspect')['document']['dirty']
        assert rc_path.read_bytes() == rc_bytes
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            ideal_rc_10ms_v=stepped['transient']['sample']['voltage_v'],
            checks=['local schema advertises finite SI numbers or bounded unit strings',
                'ideal RC 10 ms sample and live cursor/window units',
                'invalid units/range/overflow/underflow preserve the complete paused snapshot',
                'Verify/recorded seek/cursors/window use the same grammar; live time/charge preserved',
                'saved numeric SI settings and recorded data reopen with fresh paused physics',
                'nested finite pulse units; immediate Step work accepts text',
                'recovery interval units; final project clean/paused; original RC bytes preserved'],
            limitations=['native GPU smoke uses real mounted callbacks; keyboard/IME/focus/screen-reader qualification remains separate',
                'bare time text is seconds; no unitless millisecond inference or silent native clamping',
                'model/numerical/device/work/file bounds unchanged'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()

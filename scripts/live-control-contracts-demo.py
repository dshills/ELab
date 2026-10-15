#!/usr/bin/env python3
"""Discover live controls and exercise input/step state and exact boundaries."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
COMMANDS = ['simulation.transient.set', 'simulation.input.set', 'simulation.pause', 'simulation.resume',
            'simulation.restart', 'simulation.step', 'simulation.step.start', 'simulation.work.cancel', 'simulation.wait']


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)
    cases = []

    def refuse(command, params, code):
        before = client.ask('project.inspect')
        client.ask(command, params, True, code)
        assert before == client.ask('project.inspect'), 'refusal changed state'

    def ready_example(command):
        client.ask(command, {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        return client.ask('simulation.wait')

    try:
        schemas = {command: client.ask('capabilities.list', {'command': command})['capabilities'][0]['input_schema'] for command in COMMANDS}
        assert all(value['x-command-contract']['scope'] == 'live-simulation-controls' for value in schemas.values())
        assert schemas['simulation.step.start']['x-max-output-steps'] == 1000
        for command in ('example.reset', 'example.rc', 'example.npn-pulse', 'example.flasher', 'example.rectifier'):
            state = ready_example(command)
            wanted = [] if command == 'example.reset' else ['auto', 'charge', 'discharge'] if command == 'example.rc' else ['auto', 'high', 'low']
            assert state['live_input_modes'] == wanted
            if wanted:
                before = state
                for mode in wanted:
                    state = client.ask('simulation.input.set', {'mode': mode}, True)
                    assert state['transient']['pending_input_mode'] == mode
                    assert state['transient']['sample'] == before['transient']['sample']
                    assert state['components'] == before['components'] and state['document'] == before['document']
                refuse('simulation.input.set', {'mode': 'invalid'}, 'invalid-params')
            else:
                refuse('simulation.input.set', {'mode': 'auto'}, 'unavailable')
            cases.append(dict(case=command, analysis=state['analysis'], live_input_modes=wanted, paused_input_check="queues without physical change" if wanted else "unavailable refused"))

        intact = ready_example('example.rc')
        broken = client.ask('object.delete', {'id': intact['wires'][0]['id']}, True)
        assert broken['analysis'] == 'RC transient' and broken['live_input_modes'] == []
        refuse('simulation.input.set', {'mode': 'charge'}, 'unavailable')
        restored = client.ask('edit.undo', {}, True)
        assert restored['live_input_modes'] == ['auto', 'charge', 'discharge']
        cases.append(dict(case='recognized-incomplete-rc-and-undo', unavailable_modes=[], restored_modes=restored['live_input_modes']))
        cfg = dict(step_s='1 us', high_s='50 ms', low_s='50 ms', initial_voltage_v=0, seconds_per_wall_second=.001)
        state = client.ask('simulation.transient.set', cfg, True)
        assert state['transient']['stale'] and state['live_input_modes'] == ['auto', 'charge', 'discharge']
        refuse('simulation.input.set', {'mode': 'charge'}, 'unavailable')
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        project = assets / 'microsecond-rc.elab'
        client.ask('project.save', {'path': str(project)}, True)
        digest = hashlib.sha256(project.read_bytes()).hexdigest()
        before = client.ask('project.inspect')
        client.ask('simulation.input.set', {'mode': 'charge'}, True)
        accepted = client.ask('simulation.step.start', {'duration_s': '1 ms'}, True)
        assert accepted['step_work']['status'] == 'queued' and accepted['step_work']['total_samples'] == 1000
        state = client.ask('simulation.wait')
        assert state['step_work']['status'] == 'completed' and state['step_work']['completed_samples'] == 1000
        assert state['transient']['input_mode'] == 'charge' and 'pending_input_mode' not in state['transient']
        expected = 5 * -math.expm1(-.001 / .01)
        assert abs(state['transient']['sample']['voltage_v'] - expected) < 1e-11
        assert abs(state['transient']['sample']['time_s'] - .001) < 1e-15
        for key in ('components', 'wires', 'document', 'undo_entry', 'electrical_revision', 'run_id'):
            assert state.get(key) == before.get(key), key
        refuse('simulation.work.cancel', {'id': state['step_work']['id']}, 'operation-unavailable')
        refuse('simulation.step.start', {'duration_s': math.nextafter(.001, math.inf)}, 'invalid-params')
        refuse('simulation.transient.set', dict(cfg, seconds_per_wall_second=math.nextafter(.001, math.inf)), 'invalid-params')
        waited = client.ask('simulation.step', {'duration_s': '1 ms'}, True)
        assert waited['step_work']['status'] == 'completed' and waited['step_work']['completed_samples'] == 1000
        assert abs(waited['transient']['sample']['voltage_v'] - 5 * -math.expm1(-.002 / .01)) < 1e-11
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest
        cases.append(dict(case='microsecond-boundaries-and-waiting-alias', step_s=1e-6, playback=.001, duration_s=.001,
                          samples=state['step_work']['completed_samples'], voltage_after_first_step_v=state['transient']['sample']['voltage_v'],
                          voltage_after_second_step_v=waited['transient']['sample']['voltage_v'], next_float_refusals_preserve_state=True))
        resumed = client.ask('simulation.resume', {}, True)
        again = client.ask('simulation.resume', {}, True)
        assert not resumed['paused'] and not again['paused'] and resumed['run_id'] == again['run_id'] == waited['run_id']
        client.ask('simulation.pause', {}, True)
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest

        state = ready_example('example.rectifier')
        refuse('simulation.step.start', {'duration_s': '50 ps'}, 'solve-failed')
        cases.append(dict(case='network-capacitor-minimum-step', analysis=state['analysis'], rejected_duration_s=5e-11, error='solve-failed'))
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schemas=schemas, cases=cases,
                      checks=['Shared live-mode inventory across DC/RC/NPN/mixed/waveform circuits', 'Paused mode queue and stale-result refusal',
                              'Exact 1 us /0.001 playback and 1000-output-step boundaries', 'Adjacent larger float refusal purity',
                              'Immediate Step admission versus completed waiting alias', 'Analytic RC charge at 1 ms and 2 ms',
                              'Terminal cancel refusal and Resume idempotence', 'No source/history/dirty/file mutation from actuation/Step',
                              'Published minimum network capacitor step refuses before work'],
                      limits=['Cooperative worker deadlines and unchanged integration budgets; admission/settled states do not imply success',
                              'Mode inventory does not replace freshness/readiness or lifecycle guards',
                              'Backend/native callback evidence does not establish physical keyboard/focus/IME/accessibility or visible latency'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], cases=len(cases))))
    finally:
        try:
            if args.connect:
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()

#!/usr/bin/env python3
"""Observe real capacity-work progress around shared backend commands.

No visible/input-to-pixel latency claim. --connect replaces only an owned test
session. Reports retain ineligible spans; no artificial solver block is used.
"""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import platform
import subprocess
import time


ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def stamp(state):
    return dict(run_id=state['run_id'], paused=state['paused'],
                time_s=state['transient']['sample']['time_s'],
                step_work=state.get('step_work'))


def manual_span(before, after, cancel=False):
    a, b = before.get('step_work', {}), after.get('step_work', {})
    return (before['paused'] and after['paused'] and before['time_s'] == after['time_s']
            and a.get('status') == 'running' and a.get('completed_samples', 0) > 0
            and b.get('completed_samples', 0) >= a['completed_samples']
            and bool(a.get('id')) and a.get('run_id') == before['run_id']
            and b.get('run_id') == after['run_id']
            and a.get('id') == b.get('id') and before['run_id'] == after['run_id']
            and b.get('status') in (('canceling', 'canceled') if cancel else ('running',)))


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--trials', type=int, default=20)
    args = parser.parse_args()
    if not 1 <= args.trials <= 20:
        parser.error('trials must be 1–20; 20 eligible spans are needed for a p95 summary')
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    fixture_path = ROOT/'internal/conformance/testdata/capacity-rc-mosfet.json'
    fixture = json.loads(fixture_path.read_text())
    hardware = subprocess.check_output(['sysctl', '-n', 'hw.model', 'hw.logicalcpu', 'hw.memsize', 'machdep.cpu.brand_string'], text=True, timeout=5).splitlines()
    report = dict(passed=False, scope='backend request-to-response and observed real work; no visible latency claim',
                  fixture_sha256=hashlib.sha256(fixture_path.read_bytes()).hexdigest(),
                  configuration=dict(os=platform.mac_ver()[0], architecture=platform.machine(),
                                     hardware=dict(model=hardware[0], logical_cpus=int(hardware[1]), memory_bytes=int(hardware[2]), cpu=hardware[3]),
                                     transport='native-private-socket' if args.connect else 'headless-stdio',
                                     agent_sha256=hashlib.sha256((ROOT/'build/elab-agent').read_bytes()).hexdigest()),
                  trials=[], observations=[], summaries={},
                  limitations=['Job-state bracketing does not time individual Newton iterations or visible frames',
                               'This capacity topology does not establish all difficult-topology behavior',
                               'No percentile is reported with fewer than 20 eligible observations'])

    def save_report():
        (directory/'report.json').write_text(json.dumps(report, indent=2)+'\n')

    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        for category in ('components', 'wires'):
            edits = []
            for item in fixture['graph'][category]:
                if category == 'components':
                    params = {k: item[k] for k in ('id', 'kind', 'position')}
                    if item['value']:
                        params['value'] = format(item['value'], '.17g')
                    assert item['rotation'] == 0
                    command = 'component.place'
                else:
                    params = {k: item[k] for k in ('id', 'from', 'to')}
                    command = 'wire.connect'
                edits.append(dict(command=command, params=params))
            client.ask('edit.batch', {'name': 'Capacity '+category, 'edits': edits}, True)
        settings = dict(fixture['settings'], seconds_per_wall_second=1)
        client.ask('simulation.transient.set', settings, True)
        client.ask('simulation.restart', {}, True)
        state = client.ask('simulation.wait')
        assert state['components'] == fixture['graph']['components']
        assert state['wires'] == fixture['graph']['wires']
        assert state['simulated_component_count'] == 64
        report['workload'] = dict(simulated_parts=64, objects=len(state['components']),
                                  wires=len(state['wires']), nets=len(state['nets']), settings=settings,
                                  manual_duration_s=1, manual_output_samples=1000)
        client.ask('project.save', {'path': str(directory/'capacity.elab')}, True)
        client.ask('simulation.wait')

        def inspect_until(ready):
            deadline = time.monotonic()+5
            while True:
                snapshot = client.ask('project.inspect')
                if ready(snapshot):
                    return snapshot
                assert time.monotonic() < deadline, stamp(snapshot)
                time.sleep(.001)

        def measured(command, params, eligible):
            before = client.ask('project.inspect')
            start = time.perf_counter()
            result = client.ask(command, params, True)
            elapsed = (time.perf_counter()-start)*1000
            after = client.ask('project.inspect')
            report['observations'].append(dict(command=command, wall_ms=elapsed,
                                               eligible=eligible(stamp(before), stamp(after)),
                                               before=stamp(before), after=stamp(after)))
            return result

        for index in range(args.trials):
            baseline = client.ask('project.inspect')
            accepted = client.ask('simulation.step.start', {'duration_s': 1}, True)
            work_id = accepted['step_work']['id']
            progress = inspect_until(lambda s: s['step_work']['status'] not in ('queued', 'running')
                                     or s['step_work']['completed_samples'] > 0)
            if progress['step_work']['status'] != 'running':
                report['trials'].append(dict(index=index, eligible=False, reason='work ended before progress observation', work=progress['step_work']))
                save_report()
                raise AssertionError('capacity work ended before the command overlap check')
            measured('view.pan', {'dx': 1, 'dy': 0}, manual_span)
            client.ask('drawing.tool.set', {'tool': 'wire'}, True)
            measured('drawing.cancel', {}, manual_span)
            measured('project.save', {'path': str(directory/'capacity.elab')}, manual_span)
            resistor = next(x for x in baseline['components'] if x['id'] == 'R1')
            moved = client.ask('component.move', {'id': 'R1', 'position': dict(resistor['position'], x=resistor['position']['x']+10)}, True)
            measured('edit.undo', {'expected_entry_revision': moved['undo_entry']['command_revision']}, manual_span)
            measured('simulation.work.cancel', {'id': work_id}, lambda a, b: manual_span(a, b, True))
            canceled = client.ask('simulation.wait')
            assert canceled['step_work']['status'] == 'canceled', canceled['step_work']
            for field in ('components', 'wires', 'mixed', 'transient', 'run_id', 'electrical_revision'):
                assert canceled.get(field) == baseline.get(field), field
            client.ask('simulation.resume', {}, True)
            active = inspect_until(lambda s: s['transient']['sample']['time_s'] > baseline['transient']['sample']['time_s']+.001)
            measured('simulation.pause', {}, lambda a, b: not a['paused'] and b['paused']
                     and a['run_id'] == b['run_id'] and a['time_s'] >= active['transient']['sample']['time_s'])
            paused = client.ask('simulation.wait')
            report['trials'].append(dict(index=index, eligible=True, staged_progress=progress['step_work'],
                                        canceled=canceled['step_work'], paused_time_s=paused['transient']['sample']['time_s']))
            save_report()
            print('Completed real capacity trial', index+1, flush=True)
        for command in sorted({x['command'] for x in report['observations']}):
            rows = [x for x in report['observations'] if x['command'] == command]
            eligible = sorted(x['wall_ms'] for x in rows if x['eligible'])
            report['summaries'][command] = dict(total=len(rows), eligible=len(eligible),
                                               maximum_ms=max(eligible) if eligible else None,
                                               p95_ms=eligible[math.ceil(.95*len(eligible))-1] if len(eligible) >= 20 else None)
        report['passed'] = True
        report['twenty_eligible_spans_per_command'] = all(x['eligible'] >= 20 for x in report['summaries'].values())
        save_report()
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
        save_report()
        raise
    finally:
        client.close()


if __name__ == '__main__':
    main()

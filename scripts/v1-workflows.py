#!/usr/bin/env python3
"""Run hobbyist workflows and a wall-clock soak against exact shipped binaries.

An optional native app is launched as an owned fixture, with private recovery
and appearance files and a policy denying writes to shared framework settings.
Commands use the public agent interface. This does not synthesize desktop input
or qualify native pointer, first-responder, or input-to-pixel behavior.
"""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import statistics
import subprocess
import time
import xml.etree.ElementTree as ET

from acceptance_client import AcceptanceClient, percentile
from demo_session import require_fresh_demo_session


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def graph(state):
    return {key: state[key] for key in ('components', 'wires')}


def physics_time(state):
    return state['transient']['sample']['time_s']


def await_operation(client, identifier):
    deadline = time.monotonic()+30
    while time.monotonic() < deadline:
        view = client.ask('operation.inspect', {'id': identifier})
        if view['operation']['status'] not in ('queued', 'running', 'canceling'):
            assert view['operation']['status'] == 'completed', view['operation']
            return view
        time.sleep(.02)
    raise TimeoutError('capture operation did not complete')


def save_reopen_export(client, agent, assets, name):
    client.ask('simulation.pause', mutate=True)
    original = client.ask('simulation.wait')
    path = assets/(name+'.elab')
    saved = client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
    assert not saved['document']['dirty']
    fresh = AcceptanceClient(agent)
    try:
        fresh.ask('project.load', {'path': str(path)}, True)
        reopened = fresh.ask('simulation.wait')
        assert graph(reopened) == graph(saved)
        assert not reopened['document']['dirty'] and reopened['paused']
        assert reopened['document']['recording_count'] == saved['document']['recording_count']
        if reopened.get('transient'):
            assert physics_time(reopened) == 0
    finally:
        fresh.close()
    exports = {}
    for fmt in ('svg', 'png', 'pdf'):
        dest = assets/(name+'.'+fmt)
        client.ask('export.schematic.'+fmt, {'path': str(dest), 'overwrite': True}, True)
        data = dest.read_bytes()
        assert len(data) > 100
        if fmt == 'svg':
            assert ET.fromstring(data).tag.endswith('svg')
        elif fmt == 'png':
            assert data.startswith(b'\x89PNG\r\n\x1a\n')
        else:
            assert data.startswith(b'%PDF-')
        exports[fmt] = dict(bytes=len(data), sha256=digest(dest))
    after = client.ask('project.inspect')
    assert graph(after) == graph(original)
    assert after['document'] == saved['document']
    assert after.get('transient') == original.get('transient')
    return dict(graph=graph(saved), document_sha256=digest(path),
                recording_count=saved['document']['recording_count'], exports=exports)


def workflows(client, agent, assets):
    require_fresh_demo_session(client.ask('project.inspect'))
    reports = []
    # Build the first circuit from individual parts and named terminals, with
    # the same mutation contracts used by the production drawing controls.
    for kind, identifier, value, x, y in [
        ('dc-source', 'V1', '5V', -200, 0),
        ('resistor', 'R1', '1k', 0, -120),
        ('led', 'D1', '', 200, 0), ('ground', 'GND', '', -200, 120)]:
        params = dict(kind=kind, id=identifier, position=dict(x=x, y=y))
        if value:
            params['value'] = value
        client.ask('component.place', params, True)
    for source, target in [('V1.positive', 'R1.1'), ('R1.2', 'D1.anode'),
                           ('D1.cathode', 'GND.reference'), ('GND.reference', 'V1.negative')]:
        client.ask('wire.connect', {'from': source, 'to': target}, True)
    state = client.ask('simulation.wait')
    assert len(state['components']) == 4 and len(state['wires']) == 4
    assert not state['result']['stale']
    assert abs(state['result']['current_a']-.002916796577441) < 5e-12
    current = state['result']['current_a']
    client.ask('component.set', {'id': 'R1', 'value': '2k'}, True)
    changed = client.ask('simulation.wait')
    assert 0 < changed['result']['current_a'] < current
    client.ask('edit.undo', mutate=True)
    restored = client.ask('simulation.wait')
    assert graph(restored) == graph(state)
    reports.append(dict(name='blank-led', current_a=current,
        checks=['individual placement', 'four terminal connections', 'value edit', 'undo', 'DC current'],
        artifacts=save_reopen_export(client, agent, assets, 'led')))

    client.ask('example.load', {'id': 'rc', 'discard_changes': True}, True)
    client.ask('simulation.pause', mutate=True)
    client.ask('simulation.restart', mutate=True)
    client.ask('simulation.wait')
    at10 = client.ask('simulation.step', {'duration_s': .01}, True)
    assert abs(at10['transient']['sample']['voltage_v']-5*(1-math.exp(-1))) < 1e-10
    client.ask('simulation.input.set', {'mode': 'discharge'}, True)
    down = client.ask('simulation.step', {'duration_s': .01}, True)
    assert abs(down['transient']['sample']['voltage_v']-at10['transient']['sample']['voltage_v']*math.exp(-1)) < 1e-10
    reports.append(dict(name='rc-charge-discharge', voltage_10ms_v=at10['transient']['sample']['voltage_v'],
        discharged_v=down['transient']['sample']['voltage_v'],
        artifacts=save_reopen_export(client, agent, assets, 'rc')))

    client.ask('example.load', {'id': 'display-counter', 'discard_changes': True}, True)
    client.ask('simulation.pause', mutate=True)
    client.ask('simulation.restart', mutate=True)
    client.ask('simulation.wait')
    seen, changes, previous = set(), 0, 0
    expected = ('abcdef', 'bc', 'abdeg', 'abcdg', 'bcfg', 'acdfg', 'acdefg', 'abc', 'abcdefg', 'abcdfg')
    for _ in range(100):
        client.ask('simulation.step', {'duration_s': .1}, True)
        state = client.ask('simulation.wait')
        display = next(x for x in state['mixed']['displays'] if x['id'] == 'DS1')
        count = state['mixed']['sequential'][0]['memory']['value']
        assert display['digit'] == count
        assert ''.join(c for c, lit in zip('abcdefg', display['lit']) if lit) == expected[count]
        assert state['mixed']['kcl_residual_a'] < 1e-8
        assert abs(state['mixed']['power_residual_w']) < 1e-7
        seen.add(count)
        if count != previous:
            assert count == (previous+1) % 10
            previous, changes = count, changes+1
    assert seen == set(range(10)) and changes >= 10
    accepted = client.ask('simulation.verify', dict(duration_s=.1, sample_step_s=.005,
        probe_ids=['v:C1.1', 'v:U1', 'i:DS1.a', 'i:DS1.g']), True)
    await_operation(client, accepted['last_operation'])
    client.ask('verification.export', dict(id=accepted['last_operation'], format='json',
        path=str(assets/'counter-capture.json')), True)
    reports.append(dict(name='555-counter', digits=sorted(seen), changes=changes,
        artifacts=save_reopen_export(client, agent, assets, 'counter')))
    return reports


def process_sample(pid):
    # Read only the owned process, not the rest of the user's process inventory.
    out = subprocess.check_output(['/bin/ps', '-p', str(pid), '-o', 'rss=,time='],
                                  text=True, timeout=3).strip().split()
    rss, cpu = out
    days, clock = (cpu.split('-', 1) if '-' in cpu else ('0', cpu))
    fields = [float(x) for x in clock.split(':')]
    seconds = sum(value*60**index for index, value in enumerate(reversed(fields)))
    return dict(rss_bytes=int(rss)*1024, cpu_s=seconds+int(days)*86400)


def soak(client, agent, assets, duration, pid):
    started = time.monotonic()
    deadline = started+duration
    samples, epochs = [], []
    cycle, current = 0, None
    while time.monotonic() < deadline:
        circuit = ('rc', 'flasher', 'display-counter')[(cycle//6) % 3]
        if circuit != current:
            client.ask('example.load', {'id': circuit, 'discard_changes': True}, True)
            client.ask('simulation.pause', mutate=True)
            client.ask('simulation.restart', mutate=True)
            client.ask('simulation.wait')
            client.ask('simulation.resume', mutate=True)
            current = circuit
        before = client.ask('project.inspect')
        reference = graph(before)
        tick = time.monotonic()
        client.ask('view.pan', {'dx': 1, 'dy': 0}, True)
        component = next(p for p in before['components'] if p['kind'] == 'resistor')
        position = dict(component['position'])
        position['x'] += 10
        client.ask('component.move', {'id': component['id'], 'position': position}, True)
        client.ask('edit.undo', mutate=True)
        assert graph(client.ask('project.inspect')) == reference
        # Continuous integration is real wall-clock work, rather than a tight
        # loop of paused simulation.step calls masquerading as an endurance run.
        time.sleep(min(2, max(0, deadline-time.monotonic())))
        live = client.ask('project.inspect')
        assert live['run_id'] == before['run_id']
        assert live['electrical_revision'] == before['electrical_revision']
        assert physics_time(live) > physics_time(before)
        assert not live['transient']['stale']
        client.ask('simulation.pause', mutate=True)
        frozen = client.ask('simulation.wait')
        accepted = client.ask('simulation.verify', {'duration_s': .02, 'sample_step_s': .005}, True)
        await_operation(client, accepted['last_operation'])
        after = client.ask('project.inspect')
        assert after['transient'] == frozen['transient']
        # New example lifetimes periodically retire retained records; each
        # lifetime is bounded to six captures and saves without raising limits.
        path = assets/'soak.elab'
        client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
        fresh = AcceptanceClient(agent)
        try:
            fresh.ask('project.load', {'path': str(path)}, True)
            restored = fresh.ask('simulation.wait')
            assert graph(restored) == reference and not restored['document']['dirty']
        finally:
            fresh.close()
        client.ask('simulation.resume', mutate=True)
        row = dict(cycle=cycle, circuit=circuit, elapsed_s=time.monotonic()-started,
                   accepted_time_s=physics_time(live), epoch_wall_s=time.monotonic()-tick)
        if pid:
            row.update(process_sample(pid))
        samples.append(row)
        with (assets/'soak-progress.jsonl').open('a') as output:
            output.write(json.dumps(row, allow_nan=False)+'\n')
        epochs.append(dict(circuit=circuit, elapsed_s=row['elapsed_s']))
        cycle += 1
        if cycle % 3 == 0:
            print(json.dumps(dict(soak_elapsed_s=round(time.monotonic()-started, 1), cycles=cycle)), flush=True)
        # Repeated activity is paced; the circuit keeps running during the gap.
        next_tick = min(started+cycle*10, deadline)
        while time.monotonic() < next_tick:
            time.sleep(min(1, next_tick-time.monotonic()))
    client.ask('simulation.pause', mutate=True)
    elapsed = time.monotonic()-started
    assert elapsed >= duration and samples
    rss = [x['rss_bytes'] for x in samples if 'rss_bytes' in x]
    summary = dict(passed=True, requested_wall_s=duration, observed_wall_s=elapsed,
                   cycles=cycle, circuits=sorted({x['circuit'] for x in samples}),
                   scope='public API against continuously integrating owned process; not visible latency',
                   samples=samples)
    if rss:
        summary.update(rss_min_bytes=min(rss), rss_max_bytes=max(rss), rss_final_bytes=rss[-1],
                       cpu_s=samples[-1]['cpu_s']-samples[0]['cpu_s'])
        # Report a descriptive post-warmup trend, never hide a new plateau as a
        # leak verdict or claim that a single workstation proves absence of leaks.
        warm = [x['rss_bytes'] for x in samples if x['elapsed_s'] >= 180]
        if len(warm) >= 12:
            n = max(3, len(warm)//4)
            summary['rss_late_minus_early_median_bytes'] = statistics.median(warm[-n:])-statistics.median(warm[:n])
            summary['rss_growth_under_128_mib'] = summary['rss_late_minus_early_median_bytes'] < 128*1024*1024
            assert summary['rss_growth_under_128_mib'], 'owned process RSS trend exceeds investigation threshold'
    return summary


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--agent-sha256', required=True)
    parser.add_argument('--app', type=Path)
    parser.add_argument('--app-sha256')
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--soak-seconds', type=float, default=0)
    args = parser.parse_args()
    if not math.isfinite(args.soak_seconds) or args.soak_seconds < 0:
        parser.error('soak-seconds must be finite and nonnegative')
    if args.app and not args.app_sha256:
        parser.error('native app needs its qualified executable checksum')
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    agent = args.agent.resolve(strict=True)
    assert digest(agent) == args.agent_sha256, 'agent checksum mismatch'
    report = dict(passed=False, transport='headless-stdio',
                  agent_sha256=args.agent_sha256, host=dict(os=platform.mac_ver()[0], architecture=platform.machine()),
                  controller_sha256={path.name: digest(path) for path in
                      (Path(__file__), Path(__file__).with_name('acceptance_client.py'))},
                  limitations=['Public command workflows are not physical desktop drawing or typing acceptance',
                      'CPU/RSS and command response timings do not measure visible input-to-pixel latency',
                      'One workstation and a bounded wall-clock run do not establish universal stability'])
    native, client, output = None, None, None
    try:
        endpoint = None
        if args.app:
            from runtime_isolation import bundle_executable
            executable = bundle_executable(args.app.resolve(strict=True), args.app_sha256)
            # Additional test-process restriction: never overwrite shared native
            # geometry/settings. This does not alter any OS security setting.
            shared = str(Path.home()/'Library/Cogent Core')
            profile = '(version 1) (allow default) (deny file-write* (subpath '+json.dumps(shared)+'))\n'
            policy = assets/'native.sb'
            policy.write_text(profile)
            endpoint = assets/'endpoint.json'
            appearance = assets/'appearance.json'
            appearance.write_text(json.dumps(dict(format='elab.appearance', version=1, theme='apollo'))+'\n')
            output = (assets/'native-output.txt').open('wb')
            started = time.monotonic()
            native = subprocess.Popen(['/usr/bin/sandbox-exec', '-f', str(policy), str(executable),
                '--agent-endpoint', str(endpoint), '--recovery-directory', str(assets/'recovery'),
                '--appearance-file', str(appearance)], stdout=output, stderr=subprocess.STDOUT)
            deadline = started+15
            while not endpoint.exists() and native.poll() is None and time.monotonic() < deadline:
                time.sleep(.05)
            assert endpoint.exists() and native.poll() is None, 'owned native fixture did not become ready'
            report.update(transport='exact-packaged-native-private-socket', native_pid=native.pid,
                          app_sha256=args.app_sha256, startup_s=time.monotonic()-started,
                          shared_geometry_writes_denied=True)
            print(json.dumps(dict(native_ready=True, pid=native.pid, startup_s=report['startup_s'])), flush=True)
        client = AcceptanceClient(agent, endpoint)
        report['workflows'] = workflows(client, agent, assets)
        print(json.dumps(dict(workflows_passed=True, cases=[x['name'] for x in report['workflows']])), flush=True)
        if args.soak_seconds:
            report['soak'] = soak(client, agent, assets, args.soak_seconds, native.pid if native else client.process.pid)
        metrics = client.metrics
        report['command_count'] = len(metrics)
        report['command_timings'] = {command: dict(count=len(values), p95_ms=percentile(values, 95), max_ms=max(values))
            for command in sorted({x['command'] for x in metrics})
            for values in [[x['wall_ms'] for x in metrics if x['command'] == command]]}
        report['backend_api_p95_ms'] = percentile([x['wall_ms'] for x in metrics], 95)
        report['passed'] = True
    except BaseException as error:
        report['failure'] = dict(type=type(error).__name__, message=str(error))
        raise
    finally:
        try:
            if client:
                if native and native.poll() is None:
                    client.ask('app.close', {'discard_changes': True}, True)
                client.close()
            if native:
                report['native_exit_code'] = native.wait(timeout=15)
                assert report['native_exit_code'] == 0
            report['normal_cleanup_passed'] = True
        except BaseException as error:
            report['passed'] = False
            report['cleanup_failure'] = dict(type=type(error).__name__, message=str(error))
            raise
        finally:
            if output:
                output.close()
            (assets/'report.json').write_text(json.dumps(report, indent=2, allow_nan=False)+'\n')
    print(json.dumps(dict(passed=report['passed'], report=str(assets/'report.json'))), flush=True)


if __name__ == '__main__':
    main()

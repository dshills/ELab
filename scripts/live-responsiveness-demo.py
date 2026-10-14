#!/usr/bin/env python3
"""Measure backend command latency while the actual flasher runs.
This is API/solver evidence, not visible UI latency or a full responsiveness gate.
--connect replaces only the designated owned development session.
"""
import argparse
import importlib.util
import json
import platform
from pathlib import Path
import subprocess
import tempfile
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    hardware = {'status': 'unavailable'}
    try:
        model, cpus, memory, cpu = subprocess.check_output(
            ['sysctl', '-n', 'hw.model', 'hw.logicalcpu', 'hw.memsize', 'machdep.cpu.brand_string'],
            text=True, stderr=subprocess.DEVNULL).strip().splitlines()
        hardware = dict(model=model, logical_cpus=int(cpus), memory_bytes=int(memory), cpu=cpu)
    except (OSError, ValueError, subprocess.CalledProcessError):
        pass
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    try:
        client.ask('example.flasher', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('scope.probes.set', {'ids': ['i:D1', 'i:D2', 'v:C1', 'v:C2']}, True)
        initial = client.ask('project.inspect')
        resistor = next(p for p in initial['components'] if p['kind'] == 'resistor')
        measurements = []

        def measured(command, params, mutate):
            started = time.perf_counter()
            result = client.ask(command, params, mutate)
            measurements.append(dict(command=command, wall_ms=(time.perf_counter()-started)*1000))
            return result

        with tempfile.TemporaryDirectory(prefix='elab-live-work-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            path = str(directory/'flasher.elab')
            client.ask('simulation.resume', {}, True)
            for i in range(80):
                measured('view.pan', {'dx': 1, 'dy': 0}, True)
                if i % 10 == 0:
                    measured('project.save', {'path': path, 'overwrite': True}, True)
                if i == 30:
                    position = dict(resistor['position'])
                    position['x'] += 20
                    moved = measured('component.move', {'id': resistor['id'], 'position': position}, True)
                    measured('edit.undo', {'expected_entry_revision': moved['undo_entry']['command_revision']}, True)
                if i % 5 == 0:
                    measured('project.inspect', {}, False)
                time.sleep(.005)
            measured('simulation.pause', {}, True)
            final = client.ask('project.inspect')
            assert final['run_id'] == initial['run_id']
            assert final['electrical_revision'] == initial['electrical_revision']
            assert final['transient']['sample']['time_s'] > 0 and not final['transient']['stale']
            assert len(final['scope']['traces']) == 4
            assert final['components'] == initial['components']
            verify_start = time.perf_counter()
            accepted = client.ask('simulation.verify', {'duration_s': .2, 'probe_ids': ['i:D1', 'i:D2', 'v:C1', 'v:C2']}, True)
            operation_id = accepted['last_operation']
            deadline = time.monotonic()+15
            while True:
                operation = client.ask('operation.inspect', {'id': operation_id})
                if operation['operation']['status'] not in ['queued', 'running', 'canceling']:
                    break
                assert time.monotonic() < deadline, 'Verify completion timeout'
                time.sleep(.005)
            verify_ms = (time.perf_counter()-verify_start)*1000
            assert operation['operation']['status'] == 'completed', operation
            assert client.ask('project.inspect')['transient']['sample'] == final['transient']['sample']
            client.ask('project.save', {'path': path, 'overwrite': True}, True)
            fresh = helper.Client(command_preconditions=True)
            fresh.ask('project.load', {'path': path}, True)
            restored = fresh.ask('simulation.wait')
            assert restored['paused'] and not restored['document']['dirty']
            assert restored['document']['recording_count'] == 1 and restored['transient']['sample']['time_s'] == 0
            ordered = sorted(m['wall_ms'] for m in measurements)
            p95 = ordered[(95*len(ordered)+99)//100-1]
            report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                          scope='Backend API acknowledgment and actual solver work; no visible latency or full acceptance claim',
                          reference_configuration=dict(os=platform.mac_ver()[0], architecture=platform.machine(),
                              hardware=hardware, go=subprocess.check_output(['go', 'version'], text=True).strip(), race_instrumented=False),
                          circuit=dict(example='capacitor-timed flasher', objects=len(initial['components']),
                              nets=len(initial['nets']), engine=final['transient']['engine'], settings=initial['transient_settings'], traces=4),
                          command_count=len(measurements), backend_api_p95_ms=p95, backend_api_max_ms=max(ordered),
                          live_accepted_time_s=final['transient']['sample']['time_s'], verify_200ms_wall_ms=verify_ms,
                          observed_targets=dict(backend_api_p95_under_100ms=p95<100, verify_200ms_under_2s=verify_ms<2000),
                          checks=['pan/save/inspect and geometry undo during actual live nonlinear integration',
                              'run/electrical identity and graph preserved', 'four traces remain valid',
                              'paused live time unaffected by independent Verify', 'saved experiment reopens fresh and clean'],
                          limitations=['some file/recovery operations still have synchronous work',
                              'physical capacity is bounded and topology-specific',
                              'controlled blocked-kernel race tests separately prove lock availability and obsolete result rejection',
                              'this fixture does not establish input-to-pixel p95 or all reference-circuit latency gates'],
                          measurements=measurements)
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2)+'\n')
            print(json.dumps({k:v for k,v in report.items() if k != 'measurements'}, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

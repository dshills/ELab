"""Launch one owned isolated /174 app; CUA supplies all physical input."""
from pathlib import Path
import datetime
import hashlib
import importlib.util
import json
import os
import subprocess
import sys
import time

root = Path('/Users/dshills/Development/projects/ELab')
stage = Path(__file__).resolve().parent
out = Path('/private/tmp/elab-recovery-error-174')
runtime = Path('/private/tmp/elab-first-version-174-signed-runtime')
binary = runtime / 'ELab.app/Contents/MacOS/ELab'
endpoint = out / 'endpoint.json'
receipt_path = root / 'research/ui-audit/first-version/export-progress-review-checkpoint.json'
receipt = json.loads(receipt_path.read_text())
sys.path.insert(0, str(root / 'scripts'))
from package_review import members
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
review = Path(receipt['review_path'])
before_review = members(review)
before_copy = members(runtime / 'ELab.app')
assert before_copy == members(review / 'ELab.app')
assert sha(binary) == receipt['native_sha256'] and sha(runtime / 'elab-agent') == receipt['agent_sha256']
canonical = root / 'build/ELab.app/Contents/MacOS/ELab'
canonical_before = sha(canonical)
shared = Path.home() / 'Library/Cogent Core'
geometry = lambda: {p.relative_to(shared).as_posix(): dict(sha256=sha(p), mode=oct(p.stat().st_mode & 0o777))
                    for p in shared.rglob('*') if p.is_file()}
before_geometry = geometry()
control = subprocess.run(['/usr/bin/sandbox-exec', '-f', str(runtime / 'runtime.sb'), sys.executable, '-I',
                          str(runtime / 'geometry-probe.py'), str(shared / 'window-geometry-0.3.6.json')],
                         capture_output=True, text=True, timeout=15, check=True)
controls = json.loads(control.stdout)
assert all(row['denied'] and row['errno'] in [1, 13] for row in controls.values())
os.umask(0o077)
spec = importlib.util.spec_from_file_location('owned_recovery_error_client', root / 'scripts/persistence-demo.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
command = ['/usr/bin/open', '-n', '-W', '-g', '-a', str(out / 'ELab-recovery-error.app'),
           '--stdout', str(out / 'native-stdout.txt'), '--stderr', str(out / 'native-stderr.txt')]
report = dict(status='starting', started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
              command=command, native_sha256=sha(binary), agent_sha256=sha(runtime / 'elab-agent'),
              receipt_sha256=sha(receipt_path), launcher_sha256=sha(out / 'ELab-recovery-error.app/Contents/MacOS/ELabIsolatedLauncher'),
              launcher_source_sha256=sha(stage / 'launcher.swift'), policy_sha256=sha(runtime / 'runtime.sb'),
              shared_geometry_before=before_geometry, controls=controls, forced_cleanup=False,
              native_exit_status=None, native_exit_status_reason='LaunchServices owns native exit; open -W is recorded separately')
def save():
    (out / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
pid = None
client = None
launch = None
def process_command():
    if pid is None:
        return None
    value = subprocess.run(['/bin/ps', '-p', str(pid), '-o', 'command='], capture_output=True, text=True, timeout=5)
    return value.stdout.strip() if value.returncode == 0 else None
try:
    with (out / 'open-output.txt').open('x') as log:
        launch = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT)
    report['open_pid'] = launch.pid
    save()
    deadline = time.monotonic() + 30
    while not ((out / 'launcher.pid').exists() and endpoint.exists()):
        assert launch.poll() is None and time.monotonic() < deadline, 'Owned endpoint readiness deadline'
        time.sleep(.05)
    pid = int((out / 'launcher.pid').read_text().strip())
    observed = process_command()
    assert observed and observed.startswith(str(binary) + ' ') and str(endpoint) in observed
    report.update(owned_pid=pid, observed_command=observed, same_pid_after_exec=True)
    client = module.Client(str(endpoint), command_preconditions=True, agent_binary=runtime / 'agent')
    state = client.ask('project.inspect')
    assert state['components'] == [] and state['wires'] == []
    client.ask('example.rc', {}, True)
    client.ask('simulation.pause', {}, True)
    client.ask('component.select', dict(id='R1'), True)
    client.ask('simulation.wait')
    accepted = client.ask('simulation.verify', dict(duration_s=.2, probe_ids=['v:C1', 'i:R1']), True)
    operation_id = accepted['last_operation']
    deadline = time.monotonic()+15
    while True:
        operation = client.ask('operation.inspect', dict(id=operation_id))
        assert operation['operation']['status'] not in ['failed', 'canceled']
        if operation['operation']['status'] == 'completed':
            break
        assert time.monotonic() < deadline
        time.sleep(.01)
    (out/'recording-baseline.json').write_text(json.dumps(operation, indent=2)+'\n')
    client.ask('simulation.step', dict(duration_s=.01), True)
    client.ask('view.pan', dict(dx=0, dy=0), True)
    baseline = out/'Manual-RC.elab'
    client.ask('project.save', dict(path=str(baseline)), True)
    report.update(baseline_sha256=sha(baseline), baseline_files={baseline.name: sha(baseline)}, operation_id=operation_id)
    client.ask('recovery.configure', dict(directory=str(out/'recovery'), enabled=True, interval_s=2), True)
    client.ask('component.set', dict(id='R1', value='20k'), True)
    client.ask('simulation.restart', {}, True)
    client.ask('simulation.wait')
    client.ask('simulation.step', dict(duration_s=.01), True)
    expected = client.ask('simulation.wait')
    deadline = time.monotonic()+8
    while True:
        state = client.ask('project.inspect')
        assert not state['recovery'].get('error'), state['recovery'].get('error')
        matching = None
        for candidate in state['recovery']['candidates']:
            path = out/'recovery'/(candidate['id']+'.elab-recovery')
            raw = path.read_bytes()
            capsule = json.loads(raw)
            if capsule['document']['graph'] == dict(components=expected['components'], wires=expected['wires']):
                assert hashlib.sha256(raw).hexdigest() == candidate['sha256']
                matching = candidate
                (out/'last-good.elab-recovery').write_bytes(raw)
                break
        if matching is not None:
            break
        assert time.monotonic() < deadline, 'Initial automatic capsule deadline'
        time.sleep(.025)
    assert state['paused'] and not state['transient']['stale'] and state['document']['recording_count'] == 1
    (out/'initial.json').write_text(json.dumps(state, indent=2)+'\n')
    report.update(status='ready_for_cua', good_candidate=matching,
                  good_capsule_sha256=sha(out/'last-good.elab-recovery'))
    save()
    print(json.dumps({key: report[key] for key in ['status', 'owned_pid', 'observed_command']}), flush=True)
    deadline = time.monotonic() + 1200
    while not (out / 'stop.request').exists():
        assert process_command() == observed, 'Owned process changed'
        assert time.monotonic() < deadline, 'Bounded input session expired'
        time.sleep(.1)
except BaseException as error:
    report.update(status='failed', error=type(error).__name__ + ': ' + str(error))
    save()
    raise
finally:
    try:
        moved = out/'recovery-preserved'
        recovery = out/'recovery'
        if moved.exists():
            assert moved.is_dir() and not moved.is_symlink()
            assert recovery.is_file() and not recovery.is_symlink()
            assert recovery.read_text() == 'owned ELab recovery storage fault\n'
            recovery.unlink()
            moved.rename(recovery)
            report['storage_restored_on_cleanup'] = True
        if client is not None:
            client.ask('simulation.pause', {}, True)
            final = client.ask('simulation.wait')
            (out / 'final.json').write_text(json.dumps(final, indent=2) + '\n')
            client.ask('project.save', dict(path=str(out / 'Final-Recovery-174.elab')), True)
            closed = client.ask('app.close', {}, True)
            (out / 'close-response.json').write_text(json.dumps(closed, indent=2) + '\n')
            assert closed['closed'] and closed['shutdown']['exit_ready'] and closed['shutdown']['status'] == 'completed'
            client.close()
            client = None
        if launch is not None:
            launch.wait(timeout=20)
            report['open_exit_code'] = launch.returncode
        report['owned_process_absent'] = pid is not None and process_command() is None
        report['endpoint_removed'] = not endpoint.exists()
        assert report['owned_process_absent'] and report['endpoint_removed']
    except BaseException as error:
        report['cleanup_error'] = type(error).__name__ + ': ' + str(error)
        # Preserve unexpected dirty work instead of forcing a process exit.
        if client is not None:
            client.close()
        raise
    finally:
        report['shared_geometry_after'] = geometry()
        report['shared_geometry_unchanged'] = geometry() == before_geometry
        report['review_unchanged'] = members(review) == before_review
        report['copied_bundle_unchanged'] = members(runtime / 'ELab.app') == before_copy
        report['canonical_unchanged'] = sha(canonical) == canonical_before
        report['baseline_file_unchanged'] = all(sha(out / name) == pin for name, pin in report['baseline_files'].items())
        report['ended_utc'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        assert all(report[k] for k in ['shared_geometry_unchanged', 'review_unchanged', 'copied_bundle_unchanged', 'canonical_unchanged', 'baseline_file_unchanged'])
        if report['status'] != 'failed' and 'cleanup_error' not in report:
            report['status'] = 'completed'
        save()

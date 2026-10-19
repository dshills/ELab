"""Fault/repair only the owned synthetic store; inspect through public commands."""
from datetime import datetime, timezone
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import time

root = Path('/Users/dshills/Development/projects/ELab')
out = Path('/private/tmp/elab-recovery-error-174')
runtime = Path('/private/tmp/elab-first-version-174-signed-runtime')
parser = argparse.ArgumentParser()
parser.add_argument('mode', choices=['break', 'repair'])
args = parser.parse_args()
report = json.loads((out/'report.json').read_text())
assert report['status'] == 'ready_for_cua'
actual = subprocess.check_output(['/bin/ps', '-p', str(report['owned_pid']), '-o', 'command='], text=True).strip()
assert actual == report['observed_command']
assert hashlib.sha256((runtime/'ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest() == report['native_sha256']
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
assert sha(out/'Manual-RC.elab') == report['baseline_sha256']
spec = importlib.util.spec_from_file_location('storage_fault_client', root/'scripts/persistence-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
client = helper.Client(str(out/'endpoint.json'), command_preconditions=True, agent_binary=runtime/'agent')
recovery, moved = out/'recovery', out/'recovery-preserved'
marker = 'owned ELab recovery storage fault\n'
record = {'mode': args.mode, 'started_utc': datetime.now(timezone.utc).isoformat(), 'passed': False}


def capture(name, value):
    with (out/(name+'.json')).open('x') as stream:
        json.dump(value, stream, indent=2)
        stream.write('\n')


try:
    before = client.ask('project.inspect')
    capture('before-storage-'+args.mode, before)
    assert before['recovery']['directory'] == str(recovery) and before['recovery']['enabled']
    assert before['paused'] and before['document']['recording_count'] == 1
    candidate = report['good_candidate']
    assert Path(candidate['id']).name == candidate['id'] and not candidate['id'].startswith('.')
    filename = candidate['id']+'.elab-recovery'
    if args.mode == 'break':
        assert recovery.is_dir() and not recovery.is_symlink() and not moved.exists()
        assert sha(recovery/filename) == report['good_capsule_sha256']
        recovery.rename(moved)
        with recovery.open('x') as stream:
            stream.write(marker)
        start = time.monotonic()
        polls = 0
        while True:
            state = client.ask('project.inspect')
            polls += 1
            if state['recovery'].get('error'):
                break
            assert time.monotonic()-start < 8, 'Automatic storage error not observed'
            time.sleep(.05)
        capture('automatic-failure', state)
        assert sha(moved/filename) == report['good_capsule_sha256']
        record.update(automatic_error=state['recovery']['error'], polls=polls,
                      observation_elapsed_s=time.monotonic()-start, last_good_preserved=True)
    else:
        assert moved.is_dir() and not moved.is_symlink()
        assert recovery.is_file() and not recovery.is_symlink() and recovery.read_text() == marker
        assert before['recovery'].get('error')
        assert next(p['value'] for p in before['components'] if p['id'] == 'R1') == 30000
        assert sha(moved/filename) == report['good_capsule_sha256']
        recovery.unlink()
        moved.rename(recovery)
        start = time.monotonic()
        polls = 0
        while True:
            state = client.ask('project.inspect')
            polls += 1
            matched = None
            if not state['recovery'].get('error'):
                for current in state['recovery']['candidates']:
                    assert Path(current['id']).name == current['id']
                    path = recovery/(current['id']+'.elab-recovery')
                    raw = path.read_bytes()
                    capsule = json.loads(raw)
                    if capsule['document']['graph'] == {'components': before['components'], 'wires': before['wires']}:
                        assert hashlib.sha256(raw).hexdigest() == current['sha256']
                        (out/'retry.elab-recovery').write_bytes(raw)
                        matched = current
                        break
            if matched is not None:
                break
            assert time.monotonic()-start < 8, 'Automatic complete retry not observed'
            time.sleep(.05)
        capture('automatic-retry', state)
        manual = json.loads((out/'Manual-RC.elab').read_text())
        assert capsule['document']['recordings'] == manual['recordings']
        assert sha(recovery/filename) == report['good_capsule_sha256']
        record.update(candidate=matched, retry_sha256=sha(out/'retry.elab-recovery'), polls=polls,
                      observation_elapsed_s=time.monotonic()-start, last_good_preserved=True,
                      full_recording_preserved=True)
    assert sha(out/'Manual-RC.elab') == report['baseline_sha256']
    record.update(passed=True, manual_source_preserved=True)
finally:
    client.close()
    record['forwarder_exit_code'] = client.process.returncode
    record['ended_utc'] = datetime.now(timezone.utc).isoformat()
    capture('storage-'+args.mode+'-result', record)
print(json.dumps(record))

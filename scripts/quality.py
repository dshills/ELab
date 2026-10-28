#!/usr/bin/env python3
"""Run the maintained portable/native gate plans with bounded owned processes."""
import argparse
from contextlib import contextmanager
import hashlib
import json
import os
from pathlib import Path
import platform
import signal
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]


def gate_plan(profile):
    steps = [
        ('sdk', [sys.executable, '-B', 'scripts/prepare-vellum.py']),
        ('python', [sys.executable, '-B', '-m', 'unittest', 'discover', '-s', 'scripts', '-p', 'test_*.py']),
        ('prd', [sys.executable, '-B', 'scripts/prd-audit.py', '--verify', 'research/11-prd-implementation-audit.json']),
        ('inventory', ['go', 'run', './cmd/elab-ui-audit', '--verify', 'research/13-native-command-inventory.json']),
        ('portable-race', ['go', 'test', '-race', './...', '-count=1', '-timeout=30m']),
        ('portable-vet', ['go', 'vet', './...']),
    ]
    if profile == 'native':
        steps += [
            ('lint', ['scripts/lint.sh']),
            ('desktop-race', ['go', '-C', 'desktop', 'test', '-race', './...', '-count=1', '-timeout=30m']),
            ('desktop-vet', ['go', '-C', 'desktop', 'vet', './...']),
            ('rendered-race', ['go', '-C', 'desktop', 'test', '-race', '-tags=offscreen', './...', '-count=1', '-timeout=30m']),
            ('rendered-vet', ['go', '-C', 'desktop', 'vet', '-tags=offscreen', './...']),
        ]
    elif profile != 'portable':
        raise ValueError('unknown quality profile')
    return steps


class GateInterrupted(KeyboardInterrupt):
    def __init__(self, signum):
        super().__init__("interrupted by signal %d" % signum)
        self.signum = signum


@contextmanager
def termination_signals():
    def interrupted(signum, frame):
        raise GateInterrupted(signum)
    previous = {number: signal.signal(number, interrupted) for number in (signal.SIGTERM, signal.SIGHUP)}
    try:
        yield
    finally:
        for number, handler in previous.items():
            signal.signal(number, handler)


def write_json(path, value):
    temporary = path.with_name('.'+path.name+'.tmp')
    created = False
    try:
        with temporary.open('x') as stream:
            created = True
            json.dump(value, stream, indent=2)
            stream.write('\n')
        os.replace(temporary, path)
    finally:
        if created and temporary.exists():
            temporary.unlink()


def go_environment(root=ROOT):
    env = dict(os.environ, GOWORK='off', GOCACHE=os.environ.get('ELAB_GOCACHE', os.environ.get('GOCACHE', str(root/'build/go-cache'))),
               CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native')
    env.setdefault('GOMAXPROCS', '4')
    env['GOFLAGS'] = (env.get('GOFLAGS', '')+' -p=1 -mod=readonly').strip()
    return env


def execute_owned(command, cwd, env, stream, timeout):
    """Only this newly created process group may be signaled on interruption."""
    process = subprocess.Popen(command, cwd=cwd, env=env, stdout=stream,
                               stderr=subprocess.STDOUT, start_new_session=True)
    try:
        return process.wait(timeout=timeout), False
    except (subprocess.TimeoutExpired, KeyboardInterrupt) as error:
        previous = {number: signal.signal(number, signal.SIG_IGN) for number in
                    (signal.SIGTERM, signal.SIGHUP, signal.SIGINT)}
        try:
            try:
                os.killpg(process.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                process.wait(timeout=5)
            error.owned_pid = process.pid
            error.owned_exit_code = process.returncode
        finally:
            for number, handler in previous.items():
                signal.signal(number, handler)
        raise


def run_plan(steps, artifacts, root, env, timeout, execute=None):
    if not steps:
        raise ValueError('quality plan cannot be empty')
    artifacts.mkdir(parents=True, exist_ok=False, mode=0o700)
    execute = execute or execute_owned
    result = dict(passed=False, gates_passed=False, platform=platform.platform(), steps=[])
    report = artifacts/'report.json'
    def save():
        write_json(report, result)
    save()
    try:
        for name, command in steps:
            entry = dict(name=name, command=command, status='running')
            result['steps'].append(entry)
            save()
            started = time.monotonic()
            try:
                with (artifacts/(name+'.log')).open('x') as stream:
                    code, _ = execute(command, root, env, stream, timeout)
                entry.update(exit_code=code, status='passed' if code == 0 else 'failed')
            except subprocess.TimeoutExpired as error:
                entry.update(exit_code=124, status='timed-out', owned_pid=getattr(error, 'owned_pid', None), owned_exit_code=getattr(error, 'owned_exit_code', None))
            except KeyboardInterrupt as error:
                entry.update(exit_code=128+getattr(error, 'signum', signal.SIGINT), status='interrupted', owned_pid=getattr(error, 'owned_pid', None), owned_exit_code=getattr(error, 'owned_exit_code', None))
            except (OSError, subprocess.SubprocessError) as error:
                entry.update(exit_code=1, status='failed', error_type=type(error).__name__, error=str(error))
            entry['seconds'] = round(time.monotonic()-started, 3)
            save()
            print(json.dumps(entry), flush=True)
            if entry['exit_code']:
                return result
    except KeyboardInterrupt as error:
        for running in result['steps']:
            if running['status'] == 'running':
                running.update(status='interrupted', exit_code=128+getattr(error, 'signum', signal.SIGINT))
        entry = dict(name='controller', status='interrupted', exit_code=128+getattr(error, 'signum', signal.SIGINT))
        result['steps'].append(entry)
        save()
        return result
    result['gates_passed'] = True
    save()  # passed remains false until main verifies immutable source inputs.
    return result


def source_inputs(root):
    names = subprocess.check_output(['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z'], cwd=root, timeout=30).split(b'\0')
    inputs = {}
    for raw in names:
        if not raw:
            continue
        name = os.fsdecode(raw)
        if not (name.endswith(('.go', '.py', '.sh', '.mod', '.sum', '.yml', '.yaml')) or name.startswith('third_party/') and name.endswith(('.json', '.gz'))):
            continue
        path = root / name
        inputs[name] = hashlib.sha256(path.read_bytes()).hexdigest() if path.is_file() else None
    return dict(sorted(inputs.items()))


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--profile', choices=['portable', 'native'], required=True)
    parser.add_argument('--artifacts', type=Path)
    parser.add_argument('--command-timeout', type=int, default=2400)
    parser.add_argument('--dry-run', action='store_true')
    args = parser.parse_args(argv)
    if args.command_timeout < 1:
        parser.error('command timeout must be positive')
    steps = gate_plan(args.profile)
    if args.dry_run:
        print(json.dumps(dict(profile=args.profile, steps=steps), indent=2))
        return 0
    if args.artifacts is None:
        parser.error('--artifacts must name a new output directory')
    if args.profile == 'native' and sys.platform != 'darwin':
        parser.error('native quality requires macOS; portable profile is available elsewhere')
    env = go_environment()
    result = None
    try:
        with termination_signals():
            before = source_inputs(ROOT)
            source = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True, timeout=30).strip()
            result = run_plan(steps, args.artifacts.resolve(), ROOT, env, args.command_timeout)
            unchanged = before == source_inputs(ROOT)
            result.update(profile=args.profile, source_commit=source, inputs_unchanged=unchanged)
            result['passed'] = result['gates_passed'] and unchanged
            write_json(args.artifacts/'inputs.json', before)
            write_json(args.artifacts/'report.json', result)
    except (OSError, subprocess.SubprocessError, KeyboardInterrupt) as error:
        if result is not None:
            result.update(passed=False, error_type=type(error).__name__, error=str(error))
            try:
                write_json(args.artifacts/'report.json', result)
            except OSError:
                pass
        if isinstance(error, KeyboardInterrupt):
            return 128+getattr(error, 'signum', signal.SIGINT)
        parser.error(str(error))
    if result['passed']:
        return 0
    interrupted = [step['exit_code'] for step in result['steps'] if step['status'] == 'interrupted']
    return interrupted[-1] if interrupted else 1


if __name__ == '__main__':
    raise SystemExit(main())

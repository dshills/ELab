#!/usr/bin/env python3
"""Check theme persistence across two owned process sessions and circuit changes."""
import argparse
import importlib.util
import json
from pathlib import Path
import subprocess
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def launch(profile, endpoint, native):
    if native:
        process = subprocess.Popen([str(native), '--no-recovery', '--appearance-file', str(profile),
            '--agent-endpoint', str(endpoint)], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    else:
        process = subprocess.Popen([str(helper.ROOT/'build/elab-agent'), '--appearance-file', str(profile),
            '--listen', str(endpoint)], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    try:
        deadline = time.monotonic()+10
        while not endpoint.exists():
            assert process.poll() is None, process.communicate()
            assert time.monotonic() < deadline, 'endpoint startup timeout'
            time.sleep(.01)
        return process, helper.Client(str(endpoint), command_preconditions=True)
    except BaseException:
        if process.poll() is None:
            process.terminate()
            process.wait(timeout=10)
        raise



def shutdown(process, client):
    try:
        client.ask('app.close', {'discard_changes': True}, True)
    finally:
        client.close()
        process.wait(timeout=10)
        assert process.returncode == 0, process.communicate()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--native-binary', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    root = args.artifacts.resolve()
    root.mkdir(parents=True, exist_ok=False)
    profile = root/'appearance.json'
    native = args.native_binary.resolve() if args.native_binary else None
    process, client = launch(profile, root/'first.endpoint.json', native)
    try:
        ready = client.ask('appearance.wait')
        assert ready['appearance']['status'] == 'ready' and ready['theme'] == 'light'
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        before = client.ask('simulation.step', {'duration_s': '10 ms'}, True)
        client.ask('appearance.theme.set', {'theme': 'dark'}, True)
        saved = client.ask('appearance.wait')
        for field in ['transient', 'document', 'run_id', 'electrical_revision', 'undo_name']:
            assert saved[field] == before[field], field
        assert saved['appearance']['saved_theme'] == 'dark'
        assert json.loads(profile.read_text())['theme'] == 'dark'
        project = root/'rc.elab'
        client.ask('project.save', {'path': str(project)}, True)
        assert 'theme' not in json.loads(project.read_text())
    finally:
        shutdown(process, client)
    process, client = launch(profile, root/'second.endpoint.json', native)
    try:
        loaded = client.ask('appearance.wait')
        assert loaded['theme'] == 'dark' and not loaded['components'] and not loaded['document']['dirty']
        client.ask('project.load', {'path': str(project)}, True)
        client.ask('simulation.wait')
        assert client.ask('project.inspect')['theme'] == 'dark'
        client.ask('appearance.configure', {'path': ''}, True)
        client.ask('appearance.theme.set', {'theme': 'light'}, True)
        assert json.loads(profile.read_text())['theme'] == 'dark'
        client.ask('appearance.configure', {'path': str(profile)}, True)
        assert client.ask('appearance.wait')['theme'] == 'dark'
        # Close immediately after a new selection; lifecycle cleanup flushes it.
        client.ask('appearance.theme.set', {'theme': 'system'}, True)
    finally:
        shutdown(process, client)
    assert json.loads(profile.read_text())['theme'] == 'system'
    assert not list(root.glob('.elab-appearance-*'))
    assert not list(root.glob('*.endpoint.json'))
    report = {'passed': True, 'transport': 'packaged-native-socket' if native else 'headless-socket',
        'checks': ['startup config and private preference file', 'theme save preserves paused RC state/history/document',
            'two fresh process sessions restore theme independently of circuit files',
            'project loading preserves application preference', 'session-only choice preserves stored file',
            'explicit reconfiguration reloads preference', 'immediate shutdown flushes final choice',
            'owned process/endpoints/temp files cleaned'],
        'rc_10ms_v': before['transient']['sample']['voltage_v']}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()

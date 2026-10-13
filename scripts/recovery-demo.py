#!/usr/bin/env python3
"""Kill an owned headless process, then restore its autosave through public commands."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]


class Client:
    def __init__(self, directory):
        self.process = subprocess.Popen(
            [str(ROOT / 'build/elab-agent'), '--recovery-directory', str(directory)],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
        self.revision = 0
        self.sequence = 0
        self.ask('project.inspect')

    def ask(self, command, params=None, mutate=False):
        self.sequence += 1
        request = dict(version=1, id=f'recovery-{self.sequence}', command=command, params=params or {})
        if mutate:
            request.update(expected_revision=self.revision, key=f'recovery-{time.time_ns()}')
        for attempt in range(3):
            self.process.stdin.write(json.dumps(request) + '\n')
            self.process.stdin.flush()
            line = self.process.stdout.readline()
            assert line, f'process exited during {command}'
            response = json.loads(line)
            self.revision = response['revision']
            error = response.get('error')
            if mutate and error and error['code'] == 'conflict' and attempt < 2:
                request['expected_revision'] = self.revision
                continue
            assert not error, response
            return response['result']
        raise AssertionError('admission conflict did not settle')

    def close(self):
        if self.process.poll() is None:
            self.process.stdin.close()
            self.process.wait(timeout=5)
        else:
            self.process.stdin.close()
        self.process.stdout.close()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='elab-crash-recovery-') as temporary:
        directory = Path(temporary)
        recovery = directory / 'recovery'
        source_path = directory / 'saved.elab'
        first = Client(recovery)
        second = None
        try:
            count = len(first.ask('capabilities.list')['capabilities'])
            first.ask('example.rc', {}, True)
            first.ask('simulation.pause', {}, True)
            first.ask('project.save', {'path': str(source_path)}, True)
            saved = source_path.read_bytes()
            first.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
            edited = first.ask('project.inspect')
            deadline = time.monotonic() + 5
            state = edited
            while not state['recovery']['candidates'] and time.monotonic() < deadline:
                time.sleep(.05)
                state = first.ask('project.inspect')
            assert state['recovery']['candidates'], 'automatic snapshot did not appear'
            assert state['document'] == edited['document']
            assert state['transient']['sample'] == edited['transient']['sample']
            assert source_path.read_bytes() == saved
            first.process.kill()
            first.process.wait(timeout=5)
            assert first.process.returncode != 0
            second = Client(recovery)
            scan = second.ask('recovery.scan', {}, True)['recovery']
            assert len(scan['candidates']) == 1 and not scan['problems']
            candidate = scan['candidates'][0]
            assert not candidate['active_session'] and not candidate['current_session']
            choice = {'id': candidate['id'], 'sha256': candidate['sha256']}
            inspected = second.ask('recovery.inspect', choice)
            assert inspected['document']['graph']['components'] == edited['components']
            second.ask('recovery.restore', choice, True)
            restored = second.ask('simulation.wait')
            assert restored['components'] == edited['components']
            assert restored['wires'] == edited['wires']
            assert restored['document']['dirty'] and not restored['document']['path']
            assert restored['document']['recovered_from'] == str(source_path)
            second.ask('simulation.pause', {}, True)
            second.ask('project.save', {'path': str(directory / 'recovered-copy.elab')}, True)
            assert source_path.read_bytes() == saved
            second.ask('recovery.dismiss', choice, True)
            report = {
                'passed': True, 'transport': 'headless-stdio', 'capability_count': count,
                'crash_exit_code': first.process.returncode,
                'source_sha256': hashlib.sha256(saved).hexdigest(),
                'recovered_components': len(restored['components']),
                'checks': ['automatic checkpoint', 'live state and explicit file preserved',
                           'SIGKILL recovery', 'inactive crashed owner', 'validated inspect',
                           'dirty unassociated copy', 'Save As without original overwrite',
                           'hash-confirmed dismissal'],
            }
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(report, indent=2))
        finally:
            first.close()
            if second:
                second.close()


if __name__ == '__main__':
    main()

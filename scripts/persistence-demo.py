#!/usr/bin/env python3
"""Verify portable schematic persistence across sessions and desktop/headless APIs."""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile
import time


ROOT = Path(__file__).resolve().parents[1]

class Client:
    def __init__(self, endpoint=None, command_preconditions=False):
        argv = [str(ROOT / 'build/elab-agent')]
        if endpoint:
            argv += ['--connect', endpoint]
        self.process = subprocess.Popen(argv, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
        self.revision = 0
        self.command_revision = 0
        self.command_preconditions = command_preconditions
        self.sequence = 0
        self.ask('project.inspect')

    def ask(self, command, params=None, mutate=False, expect=None):
        self.sequence += 1
        request = dict(version=1, id=f'persist-{self.sequence}', command=command, params=params or {})
        if mutate:
            guard = 'expected_command_revision' if self.command_preconditions else 'expected_revision'
            request.update({guard: self.command_revision if self.command_preconditions else self.revision, 'key': f'persist-{time.time_ns()}'})
        for attempt in range(8):
            self.process.stdin.write(json.dumps(request) + '\n')
            self.process.stdin.flush()
            line = self.process.stdout.readline()
            assert line, f'agent terminated during {command}'
            response = json.loads(line)
            self.revision = response['revision']
            result = response.get('result')
            snapshot = result.get('snapshot') if command == 'events.poll' and isinstance(result, dict) else result
            if isinstance(snapshot, dict) and 'components' in snapshot and 'document' in snapshot and 'command_revision' in snapshot:
                self.command_revision = snapshot['command_revision']
            # Native layout may legitimately advance the command revision.
            # Refresh only an unexpected conflict, keeping exact intent/key/ID;
            # an intentionally tested conflict must remain observable.
            if (mutate and response.get('error', {}).get('code') == 'conflict'
                    and expect != 'conflict' and attempt < 7):
                self.ask('project.inspect')
                request[guard] = self.command_revision if self.command_preconditions else self.revision
                continue
            if expect:
                assert response.get('error', {}).get('code') == expect, response
            else:
                assert 'error' not in response, response
            return response.get('result')

    def close(self):
        self.process.stdin.close()
        self.process.wait(timeout=5)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--fixture', type=Path)
    args = parser.parse_args()
    first = connect_demo_client(Client, args.connect)
    second = None
    try:
        first.ask('example.reset', {'discard_changes': True}, True)
        first.ask('simulation.wait')
        first.ask('component.set', {'id': 'R1', 'value': '2.2k'}, True)
        first.ask('component.move', {'id': 'R1', 'position': {'x': 40, 'y': -130}}, True)
        first.ask('component.rotate', {'id': 'R1'}, True)
        first.ask('view.pan', {'dx': 80, 'dy': 40}, True)
        original = first.ask('simulation.wait')
        with tempfile.TemporaryDirectory(prefix='elab-persistence-') as directory:
            path = str(Path(directory) / 'My LED.elab')
            saved = first.ask('project.save', {'path': path}, True)
            assert not saved['document']['dirty']
            document = json.loads(Path(path).read_text())
            assert document['format'] == 'elab.schematic' and document['version'] == 4
            first.ask('component.set', {'id': 'R1', 'value': '1k'}, True)
            first.ask('simulation.wait')
            first.ask('project.load', {'path': path}, True, 'unsaved-changes')
            first.ask('project.load', {'path': path, 'discard_changes': True}, True)
            loaded = first.ask('simulation.wait')
            assert loaded['components'] == original['components']
            assert loaded['wires'] == original['wires'] and loaded['view'] == original['view']
            assert loaded['result']['current_a'] == original['result']['current_a']
            assert not loaded['document']['dirty'] and not loaded['result']['stale']
            second = Client()
            second.ask('project.load', {'path': path}, True)
            restarted = second.ask('simulation.wait')
            assert restarted['components'] == original['components']
            assert restarted['result']['current_a'] == original['result']['current_a']
            Path(path).write_text('invalid outside edit')
            first.ask('project.save', {}, True, 'file-conflict')
            first.ask('project.load', {'path': path}, True, 'invalid-document')
            first.ask('project.save', {'overwrite': True}, True)
            if args.fixture:
                first.ask('project.save', {'path': str(args.fixture.resolve()), 'overwrite': True}, True)
            else:
                first.ask('project.clear', {}, True)
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      format='elab.schematic', version=4,
                      checks=['exact graph/value/rotation round trip', 'saved viewport', 'live recomputation',
                              'new process reload', 'unsaved change protection', 'external file conflict',
                              'invalid file refusal', 'explicit overwrite'],
                      resistance_ohm=2200, current_a=original['result']['current_a'])
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

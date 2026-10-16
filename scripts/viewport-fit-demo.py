#!/usr/bin/env python3
"""Verify viewport following through the public API in a fresh owned session.

Run native first, then replay its measured sizes headlessly with --sizes-from.
Native mode closes only the explicitly supplied owned session.
"""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile
import time

from validation_artifacts import discovery_summary

ROOT = Path(__file__).resolve().parents[1]


class Client:
    def __init__(self, endpoint):
        argv = [str(ROOT / 'build/elab-agent')]
        if endpoint:
            argv += ['--connect', endpoint]
        self.process = subprocess.Popen(argv, stdin=subprocess.PIPE,
                                        stdout=subprocess.PIPE, text=True)
        self.sequence = 0

    def request(self, command, params=None, guard=None):
        self.sequence += 1
        request = dict(version=1, id=f'viewport-{self.sequence}',
                       command=command, params=params or {})
        if guard:
            request.update(expected_command_revision=guard['command_revision'],
                           expected_document_generation=guard['document']['generation'],
                           key=f'viewport-{time.time_ns()}')
        self.process.stdin.write(json.dumps(request)+'\n')
        self.process.stdin.flush()
        line = self.process.stdout.readline()
        assert line, f'agent terminated during {command}'
        return json.loads(line)

    def ask(self, command, params=None, mutate=False, expect=None):
        # Native layout reports are legitimate competing presentation commands.
        # Refresh the complete guard and retry only explicit conflict refusals.
        for _ in range(8):
            guard = self.ask('project.inspect') if mutate else None
            response = self.request(command, params, guard)
            code = response.get('error', {}).get('code')
            if mutate and code == 'conflict' and expect != 'conflict':
                continue
            assert code == expect, response
            return response.get('result')
        raise AssertionError(f'could not admit {command} after layout conflicts')

    def close(self):
        self.process.stdin.close()
        assert self.process.wait(timeout=5) == 0


def invariant(state):
    return {key: state.get(key) for key in (
        'components', 'wires', 'document', 'undo_name', 'redo_name',
        'run_id', 'electrical_revision', 'transient', 'mixed', 'result')}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint, closed afterward')
    parser.add_argument('--sizes-from', type=Path, help='Replay native report sizes headlessly')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--discovery-output', type=Path, required=True)
    args = parser.parse_args()
    assert not (args.connect and args.sizes_from)
    replay = json.loads(args.sizes_from.read_text())['sizes'] if args.sizes_from else {
        'expanded': {'width_px': 594, 'height_px': 137},
        'collapsed': {'width_px': 594, 'height_px': 379}}
    client = Client(args.connect)
    mutate = lambda command, params=None: client.ask(command, params, True)
    sizes = {}

    def layout(name, visible):
        mutate('scope.visibility.set', {'visible': visible})
        if not args.connect:
            mutate('view.viewport.set', replay[name])
        deadline, stable_since, previous = time.monotonic()+8, None, None
        while time.monotonic() < deadline:
            state = client.ask('project.inspect')
            measured = state['viewport']
            fingerprint = (state['command_revision'], measured, state['view'])
            valid = measured['registered']
            if name == 'collapsed' and 'expanded' in sizes:
                valid = valid and measured['height_px'] > sizes['expanded']['height_px']
            if name == 'expanded' and 'collapsed' in sizes:
                valid = valid and measured['height_px'] < sizes['collapsed']['height_px']
            if valid and fingerprint == previous:
                if stable_since is not None and time.monotonic()-stable_since >= .3:
                    sizes[name] = {k: measured[k] for k in ('width_px', 'height_px')}
                    return state
            else:
                stable_since = time.monotonic() if valid else None
            previous = fingerprint
            time.sleep(.02)
        raise AssertionError(f'viewport did not settle at {name}')

    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh blank session'
        discovery = discovery_summary(client.ask('capabilities.list',
            {'command': 'view.viewport.set'}), args.discovery_output)
        mutate('example.load', {'id': 'nmos'})
        mutate('simulation.pause')
        mutate('simulation.restart')
        client.ask('simulation.wait')
        small = layout('expanded', True)
        assert small['viewport']['following'] and not small['viewport'].get('fit_failure')
        large = layout('collapsed', False)
        assert invariant(large) == invariant(small)
        assert large['view']['zoom'] >= small['view']['zoom']
        mutate('view.pan', {'dx': 20, 'dy': 0})
        manual = client.ask('project.inspect')['view']
        resized = layout('expanded', True)
        assert not resized['viewport']['following'] and resized['view'] == manual
        restored = mutate('view.fit', {'follow_viewport': True})
        assert restored['view'] == small['view'] and restored['viewport']['following']
        assert invariant(restored) == invariant(small)
        client.ask('view.viewport.set', {'width_px': 63, 'height_px': 137}, True,
                   expect='invalid-params')
        client.ask('view.fit', {'follow_viewport': None}, True, expect='invalid-params')
        assert client.ask('project.inspect')['view'] == restored['view']
        one_shot = mutate('view.fit')
        resized = layout('collapsed', False)
        assert not resized['viewport']['following'] and resized['view'] == one_shot['view']
        with tempfile.TemporaryDirectory(prefix='elab-viewport-') as directory:
            path = str(Path(directory)/'saved.elab')
            saved = mutate('project.save', {'path': path})
            mutate('view.fit', {'follow_viewport': True})
            opened = mutate('project.load', {'path': path})
            assert opened['view'] == saved['view'] and not opened['viewport']['following']
            assert opened['viewport']['width_px'] == sizes['collapsed']['width_px']
            assert opened['viewport']['height_px'] == sizes['collapsed']['height_px']
            document = json.loads(Path(path).read_text())
            assert 'viewport' not in document
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      discovery=discovery, sizes=sizes,
                      views=dict(expanded=small['view'], collapsed=large['view'],
                                 manual=manual, restored=restored['view'], one_shot=one_shot['view']),
                      checks=['measured canvas fitting', 'scope resize without electrical/document/history changes',
                              'manual pan survives resize', 'follow mode resumes',
                              'invalid size/null follow refuse', 'legacy one-shot fit',
                              'saved view survives open; following and measured size stay session-only'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as out:
            out.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({key: report[key] for key in ('passed', 'transport', 'sizes', 'views')}, indent=2))
    finally:
        if args.connect:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()

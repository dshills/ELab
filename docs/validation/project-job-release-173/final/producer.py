#!/usr/bin/env python3
"""Check retained project-job retry semantics through a pinned packaged agent.

Uses an owned headless stdio process and private temporary files only. No native
window, capture, preferences, recovery configuration or printer is involved.
"""
from pathlib import Path
import argparse
import copy
import gzip
import hashlib
import json
import os
import select
import subprocess
import tempfile
import time


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


class Client:
    def __init__(self, binary):
        self.process = subprocess.Popen([str(binary)], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        stderr=subprocess.PIPE)
        self.buffer = b''
        self.sequence = 0
        self.command_revision = 0
        self.transcript = []

    def request(self, command, params=None, mutate=False, key=None):
        self.sequence += 1
        request = dict(version=1, id='project-job-' + str(self.sequence), command=command,
                       params=params or {})
        if mutate:
            request.update(expected_command_revision=self.command_revision,
                           key=key or 'project-job-key-' + str(self.sequence))
        return request

    def send(self, request, error=None, duplicate=False):
        self.process.stdin.write(json.dumps(request).encode() + b'\n')
        self.process.stdin.flush()
        end = time.monotonic() + 10
        while b'\n' not in self.buffer:
            remaining = end - time.monotonic()
            require(remaining > 0, 'Response deadline exceeded')
            ready, _, _ = select.select([self.process.stdout], [], [], remaining)
            require(bool(ready), 'Response deadline exceeded')
            chunk = os.read(self.process.stdout.fileno(), 65536)
            require(bool(chunk), 'Agent exited before response')
            self.buffer += chunk
            require(len(self.buffer) <= 1 << 20, 'Response byte budget exceeded')
        line, self.buffer = self.buffer.split(b'\n', 1)
        response = json.loads(line)
        self.transcript.append(dict(request=copy.deepcopy(request), response=response))
        require(response.get('id') == request['id'], 'Response ID differs')
        require(response.get('error', {}).get('code') == error, 'Unexpected command outcome: ' + request['command'])
        require(response.get('duplicate', False) is duplicate, 'Unexpected retry flag')
        result = response.get('result')
        if (not duplicate and isinstance(result, dict) and 'components' in result
                and 'document' in result and 'command_revision' in result):
            self.command_revision = result['command_revision']
        return response

    def ask(self, command, params=None, mutate=False):
        return self.send(self.request(command, params, mutate))['result']

    def close(self):
        if not self.process.stdin.closed:
            self.process.stdin.close()
        code = self.process.wait(timeout=15)
        stderr = self.process.stderr.read(64 << 10)
        require(code == 0, 'Owned agent failed on normal EOF: ' + str(code))
        require(not stderr, 'Unexpected agent stderr')
        return code


def protected(state):
    # Work metadata, response/event revisions and diagnostic messages legitimately
    # change after a failed job. Committed drawing/physics/document/history do not.
    for key in ['components', 'wires', 'document', 'run_id', 'electrical_revision',
                'actuation_revision', 'paused', 'scope', 'can_undo', 'can_redo']:
        require(key in state, 'Snapshot is missing protected field: ' + key)
    require(any(key in state for key in ['result', 'transient', 'npn', 'mixed']), 'Snapshot has no physical result')
    keys = ['components', 'wires', 'document', 'run_id', 'electrical_revision',
            'actuation_revision', 'undo_entry', 'redo_entry', 'paused', 'result',
            'transient', 'npn', 'mixed', 'transient_settings', 'scope', 'recording',
            'can_undo', 'can_redo', 'undo_name', 'redo_name']
    return {key: state.get(key) for key in keys}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--sha256', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    require(sha(args.agent.read_bytes()) == args.sha256, 'Agent identity differs')
    args.output.mkdir(parents=True, exist_ok=False)
    declaration = dict(agent=str(args.agent.resolve()), agent_sha256=args.sha256,
                       producer_sha256=sha(Path(__file__).read_bytes()),
                       scope='Headless stdio only; real owned files; no controlled in-flight cancellation',
                       actual_native_ui=False, actual_capture=False, actual_printing=False)
    (args.output / 'declaration.json').write_text(json.dumps(declaration, indent=2) + '\n')
    client = Client(args.agent.resolve())
    saved_transcript = client.transcript
    checks = []
    try:
        with tempfile.TemporaryDirectory(prefix='elab-release-project-jobs-') as temp:
            folder = Path(temp)
            rc_path, damaged_path = folder / 'RC.elab', folder / 'Repaired.elab'
            client.ask('project.inspect')
            client.ask('example.rc', {}, True)
            client.ask('simulation.pause', {}, True)
            client.ask('simulation.wait')
            client.ask('project.save', dict(path=str(rc_path)), True)
            rc_bytes = rc_path.read_bytes()
            client.ask('example.reset', dict(discard_changes=True), True)
            client.ask('simulation.pause', {}, True)
            before = client.ask('simulation.wait')
            damaged_path.write_bytes(b'invalid JSON\n')
            load = client.request('project.load', dict(path=str(damaged_path), discard_changes=True), True)
            failed = client.send(load, error='invalid-document')
            current = client.ask('project.inspect')
            work = current['project_work']
            require(work['status'] == 'failed' and work['error']['code'] == 'invalid-document', 'Failed job metadata lost')
            require(protected(before) == protected(current), 'Failed Open changed source')
            checks.append('failed-open-preserves-committed-source')
            original_work = client.ask('project.work.inspect', dict(id=work['id']))
            damaged_path.write_bytes(rc_bytes)
            retry = copy.deepcopy(load)
            retry['id'] += '-retry'
            repeated = client.send(retry, error='invalid-document', duplicate=True)
            require(repeated.get('error') == failed.get('error'), 'Same key lost original failure')
            require(client.ask('project.work.inspect', dict(id=work['id'])) == original_work, 'Same key replaced failed job')
            require(protected(client.ask('project.inspect')) == protected(before), 'Same key reread repaired file')
            checks.append('same-key-after-repair-retains-original-failed-job')
            opened = client.ask('project.load', dict(path=str(damaged_path), discard_changes=True), True)
            require(opened['project_work']['id'] != work['id'] and opened['project_work']['status'] == 'completed', 'New key did not admit a new job')
            require(opened['document']['path'] == str(damaged_path) and not opened['document']['dirty'], 'New Open failed association')
            graph = json.loads(rc_bytes)['graph']
            require(opened['components'] == graph['components'] and opened['wires'] == graph['wires'], 'Repaired graph differs')
            require(damaged_path.read_bytes() == rc_bytes and rc_path.read_bytes() == rc_bytes, 'Open rewrote source files')
            require(client.ask('project.work.inspect', dict(id=work['id'])) == original_work, 'New job overwrote retained failure')
            checks.append('new-key-loads-repaired-file-and-retains-original-failure')
            start = client.request('project.load.start', dict(path=str(rc_path)), True)
            admitted = client.send(start)['result']['project_work']
            require(admitted['status'] == 'queued', 'Immediate command lost admission snapshot')
            settled = client.ask('simulation.wait')
            require(settled['project_work']['id'] == admitted['id'] and settled['project_work']['status'] == 'completed', 'Open did not settle')
            retry = copy.deepcopy(start)
            retry['id'] += '-retry'
            duplicate = client.send(retry, duplicate=True)['result']['project_work']
            require(duplicate == admitted, 'Immediate retry changed original admission')
            completed = client.ask('project.work.inspect', dict(id=admitted['id']))
            require(completed['status'] == 'completed', 'Exact inspection lost terminal outcome')
            checks.append('start-retry-keeps-admission-exact-inspection-keeps-completion')
            before_cancel = client.ask('project.inspect')
            for target in [admitted['id'], 'unknown-project-job']:
                client.send(client.request('project.work.cancel', dict(id=target), True), error='operation-unavailable')
                after_cancel = client.ask('project.inspect')
                require(protected(after_cancel) == protected(before_cancel), 'Refused cancel changed source')
                require(after_cancel['command_revision'] == before_cancel['command_revision'], 'Refused cancel advanced command revision')
            checks.append('terminal-and-unknown-cancel-preserve-source-and-command-revision')
            final = client.ask('app.close', dict(discard_changes=True), True)
            require(final['closed'] and final['shutdown']['exit_ready'] and final['shutdown']['workers_joined'], 'Close reported early readiness')
            checks.append('normal-shutdown-joins-owned-workers')
        code = client.close()
        client = None
        report = dict(passed=True, checks=checks, exit_code=code, **declaration)
        (args.output / 'result.json').write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        if client is not None:
            # EOF is the normal owned-process cleanup route even on a check failure.
            try:
                client.close()
            finally:
                (args.output / 'transcript.json.gz').write_bytes(gzip.compress(json.dumps(client.transcript, indent=2).encode(), mtime=0))
        else:
            (args.output / 'transcript.json.gz').write_bytes(gzip.compress(json.dumps(saved_transcript, indent=2).encode(), mtime=0))


if __name__ == '__main__':
    main()

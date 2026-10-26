"""Bounded JSON-lines client for owned release-acceptance processes.

Uses the shipped agent transport. This is software-test orchestration, not OS
input injection; a native endpoint does not imply physical UI acceptance.
"""
import json
import os
from pathlib import Path
import selectors
import subprocess
import time


class AcceptanceClient:
    def __init__(self, agent, endpoint=None, timeout_s=30):
        self.agent = Path(agent).resolve(strict=True)
        self.timeout_s = timeout_s
        self.metrics = []
        self.sequence = 0
        self.buffer = b''
        self.guard = None
        argv = [str(self.agent)]
        if endpoint:
            argv += ['--connect', str(endpoint)]
        self.process = subprocess.Popen(argv, stdin=subprocess.PIPE,
                                        stdout=subprocess.PIPE, bufsize=0)
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.process.stdout, selectors.EVENT_READ)

    def _receive(self):
        deadline = time.monotonic() + self.timeout_s
        while b'\n' not in self.buffer:
            remaining = deadline - time.monotonic()
            if remaining <= 0 or not self.selector.select(remaining):
                raise TimeoutError('agent response deadline exceeded')
            block = os.read(self.process.stdout.fileno(), 65536)
            if not block:
                raise RuntimeError('agent exited before its response')
            self.buffer += block
            if len(self.buffer) > 64 * 1024 * 1024:
                raise RuntimeError('agent response exceeds acceptance-controller bound')
        line, self.buffer = self.buffer.split(b'\n', 1)
        return json.loads(line)

    def ask(self, command, params=None, mutate=False, expect=None):
        if mutate:
            self.ask('project.inspect')
        started = time.perf_counter()
        self.sequence += 1
        request = dict(version=1, id=f'acceptance-{self.sequence}',
                       command=command, params=params or {})
        if mutate:
            request.update(expected_command_revision=self.guard['command_revision'],
                           expected_document_generation=self.guard['document']['generation'],
                           key=f'acceptance-{os.getpid()}-{time.time_ns()}')
        for attempt in range(8):
            data = (json.dumps(request, allow_nan=False) + '\n').encode()
            # Requests are small; stdout reads, the potentially slow side, have
            # an explicit wall-clock deadline. No retry of ambiguous outcomes.
            self.process.stdin.write(data)
            response = self._receive()
            if response.get('id') != request['id']:
                raise RuntimeError('agent response identity mismatch')
            code = response.get('error', {}).get('code')
            if mutate and code == 'conflict' and expect != 'conflict' and attempt < 7:
                state = self.ask('project.inspect')
                request['expected_command_revision'] = state['command_revision']
                request['expected_document_generation'] = state['document']['generation']
                continue
            self.metrics.append(dict(command=command,
                wall_ms=(time.perf_counter()-started)*1000, error=code))
            if code != expect:
                raise AssertionError(f'{command}: expected {expect}, got {response.get("error")}')
            result = response.get('result')
            if isinstance(result, dict) and 'command_revision' in result and 'document' in result:
                self.guard = result
            return result
        raise AssertionError('unreachable admission retry exhaustion')

    def close(self):
        self.process.stdin.close()
        try:
            code = self.process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            # Only this owned forwarding/headless child, never the native host.
            self.process.terminate()
            self.process.wait(timeout=5)
            raise RuntimeError('owned agent needed forced cleanup')
        finally:
            self.selector.close()
            self.process.stdout.close()
        if code != 0:
            raise RuntimeError(f'owned agent exit {code}')


def percentile(values, percent):
    if not values:
        raise ValueError('percentile requires observations')
    import math
    return sorted(values)[max(0, math.ceil(len(values)*percent/100)-1)]

"""Small bounded reference client for ELab demos/tests, not a general MCP SDK.

Uses the 2025-11-25 stdio handshake. The product server also supports the current
protocol through its official SDK. Owns only its launched child, never a host.
"""
import json
from pathlib import Path
import selectors
import subprocess
import time
import uuid

from acceptance_client import AcceptanceClient


class MCPClient(AcceptanceClient):
    def __init__(self, agent, endpoint=None, cwd=None, timeout_s=20):
        self.agent = Path(agent).resolve(strict=True)
        config_args = [str(self.agent), '--mcp-config']
        if endpoint:
            config_args += ['--connect', str(Path(endpoint).resolve())]
        config = subprocess.run(config_args, check=True, capture_output=True,
                                text=True, timeout=5, cwd=cwd)
        server = json.loads(config.stdout)['mcpServers']['elab']
        self.sequence = 0
        self.buffer = b''
        self.guard = None
        self.timeout_s = timeout_s
        self.metrics = []
        self.process = subprocess.Popen([server['command'], *server['args']],
                                        stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        bufsize=0, cwd=cwd)
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.process.stdout, selectors.EVENT_READ)
        try:
            result = self.rpc('initialize', dict(protocolVersion='2025-11-25', capabilities={},
                              clientInfo=dict(name='ELab reference demo', version='1')))
            if result['protocolVersion'] != '2025-11-25':
                raise RuntimeError('unexpected protocol negotiation')
            self.process.stdin.write(b'{"jsonrpc":"2.0","method":"notifications/initialized"}\n')
            self.tools = self.rpc('tools/list', {})['tools']
            self.inspect()
        except BaseException:
            self.close()
            raise

    def rpc(self, method, params):
        self.sequence += 1
        identifier = self.sequence
        request = dict(jsonrpc='2.0', id=identifier, method=method, params=params)
        self.process.stdin.write((json.dumps(request, allow_nan=False)+'\n').encode())
        for _ in range(64):
            response = self._receive()
            if 'id' not in response:
                continue  # Bounded unsolicited protocol notifications.
            if response.get('jsonrpc') != '2.0' or response['id'] != identifier:
                raise RuntimeError('MCP response identity mismatch')
            if 'error' in response:
                raise RuntimeError('MCP protocol error: '+json.dumps(response['error']))
            return response['result']
        raise RuntimeError('too many unsolicited MCP messages')

    def tool(self, name, arguments):
        result = self.rpc('tools/call', dict(name=name, arguments=arguments))
        response = json.loads(result['content'][0]['text'])
        if response != result['structuredContent'] or bool(response.get('error')) != result.get('isError', False):
            raise RuntimeError('MCP structured/fallback/error mismatch')
        if response.get('version') != 1:
            raise RuntimeError('invalid ELab response')
        return response

    def inspect(self):
        response = self.tool('elab_inspect', {})
        if 'error' in response:
            raise RuntimeError('inspect failed: '+json.dumps(response['error']))
        self.guard = response['result']
        return self.guard

    def ask(self, command, params=None, mutate=False, expect=None):
        # This owned demo explicitly inspects before each NEW planned mutation.
        # It never refreshes a refused command or retries an uncertain outcome.
        if mutate:
            self.inspect()
        request = dict(version=1, id='demo-'+uuid.uuid4().hex,
                       command=command, params=params or {})
        if mutate:
            request.update(key=uuid.uuid4().hex,
                           expected_command_revision=self.guard['command_revision'],
                           expected_document_generation=self.guard['document']['generation'])
        started = time.perf_counter()
        response = self.tool('elab_command', request)
        self.metrics.append(dict(command=command, wall_ms=(time.perf_counter()-started)*1000))
        if expect is not None:
            if response.get('error', {}).get('code') != expect:
                raise AssertionError('unexpected refusal: '+json.dumps(response))
        elif 'error' in response:
            raise RuntimeError(command+': '+json.dumps(response['error']))
        result = response.get('result')
        if isinstance(result, dict) and 'command_revision' in result and 'document' in result:
            self.guard = result
        return result

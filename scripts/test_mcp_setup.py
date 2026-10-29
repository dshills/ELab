"""Exercise the actual agent executable and generated MCP config outside the repo."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

from mcp_client import MCPClient

ROOT=Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('elab_mcp_demo',ROOT/'scripts/mcp-demo.py')
demo=importlib.util.module_from_spec(spec);spec.loader.exec_module(demo)


class MCPSetupTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.directory=tempfile.TemporaryDirectory(prefix='elab-mcp-setup-')
        cls.addClassCleanup(cls.directory.cleanup)
        cls.agent=Path(cls.directory.name)/'agent with spaces'
        env=dict(os.environ,GOWORK='off',GOTOOLCHAIN='local')
        env.setdefault('GOCACHE',str(ROOT/'build/go-cache'))
        subprocess.run(['go','build','-mod=readonly','-o',str(cls.agent),'./cmd/elab-agent'],
                       cwd=ROOT,env=env,check=True,capture_output=True,timeout=180)

    def test_generated_configuration_builds_measures_compares_and_pages(self):
        with tempfile.TemporaryDirectory() as cwd:
            c=MCPClient(self.agent,cwd=cwd)
            try:
                report=demo.exercise(c)
                self.assertEqual(report['commands'],228)
                self.assertEqual(report['bounded_rows'],5)
                self.assertEqual(report['automatic_mutation_retries'],0)
                self.assertLess(report['compared_voltages_v'][1],report['compared_voltages_v'][0])
                closed=c.ask('app.close',{'discard_changes':True},True)
                self.assertTrue(closed['closed'])
            finally:c.close()
            self.assertEqual(c.process.returncode,0)

    def test_raw_json_mode_is_unchanged(self):
        response=subprocess.run([str(self.agent)],input='{"version":1,"id":"inspect","command":"project.inspect","params":{}}\n',
                                text=True,capture_output=True,check=True,timeout=8)
        value=json.loads(response.stdout)
        self.assertEqual(value['version'],1)
        self.assertNotIn('jsonrpc',value)
        self.assertEqual(value['result']['components'],[])

    def test_bad_explicit_connection_does_not_create_a_headless_session(self):
        path=Path(self.directory.name)/'missing.endpoint.json'
        response=subprocess.run([str(self.agent),'--mcp','--connect',str(path)],
                                input='',text=True,capture_output=True,timeout=8)
        self.assertNotEqual(response.returncode,0)
        self.assertEqual(response.stdout,'')
        self.assertFalse(path.exists())

    def test_oversized_mcp_frame_closes_within_the_process_deadline(self):
        request=dict(jsonrpc='2.0',id=1,method='tools/call',params=dict(name='elab_command',arguments=dict(padding='x'*70000)))
        response=subprocess.run([str(self.agent),'--mcp'],input=json.dumps(request)+'\n',
                                text=True,capture_output=True,timeout=8)
        self.assertNotEqual(response.returncode,0)
        for line in response.stdout.splitlines():
            self.assertEqual(json.loads(line)['jsonrpc'],'2.0')

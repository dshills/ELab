import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

import quality


class QualityTests(unittest.TestCase):
    def test_gate_profiles_preserve_full_suites_and_timeouts(self):
        portable = dict(quality.gate_plan('portable'))
        native = dict(quality.gate_plan('native'))
        self.assertIn('portable-race', portable)
        self.assertEqual(portable['web-ui'], ['node', 'scripts/check-web.mjs'])
        self.assertNotIn('desktop-race', portable)
        for name in ['portable-race', 'desktop-race', 'rendered-race']:
            self.assertIn('./...', native[name])
            self.assertIn('-race', native[name])
            self.assertIn('-timeout=30m', native[name])
        self.assertIn('-tags=offscreen', native['rendered-race'])
        self.assertIn('lint', native)
        with self.assertRaises(ValueError):
            quality.gate_plan('unknown')

    def test_failure_timeout_and_interrupt_are_terminal_and_never_pass(self):
        for failure in [3, subprocess.TimeoutExpired(['owned'], 1), KeyboardInterrupt()]:
            with self.subTest(failure=type(failure).__name__), tempfile.TemporaryDirectory() as tmp:
                calls = []
                def execute(command, *args):
                    calls.append(command)
                    if isinstance(failure, BaseException):
                        raise failure
                    return failure, False
                path = Path(tmp)/'owned'
                result = quality.run_plan([('first', ['first']), ('second', ['second'])], path, Path(tmp), {}, 1, execute)
                self.assertFalse(result['passed'])
                self.assertEqual(len(calls), 1)
                self.assertNotEqual(result['steps'][0]['exit_code'], 0)
                self.assertEqual(json.loads((path/'report.json').read_text()), result)

    def test_exclusive_output_and_complete_success(self):
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp)/'owned'
            result = quality.run_plan([('one', ['one']), ('two', ['two'])], out, Path(tmp), {}, 1,
                                      lambda *args: (0, False))
            self.assertTrue(result['gates_passed'])
            self.assertFalse(result['passed'])
            before = (out/'report.json').read_bytes()
            with self.assertRaises(FileExistsError):
                quality.run_plan([('one', ['one'])], out, Path(tmp), {}, 1)
            self.assertEqual(before, (out/'report.json').read_bytes())

    def test_input_verification_failure_cannot_leave_a_passed_report(self):
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp)/'quality'
            with patch.object(quality, 'gate_plan', return_value=[('gate', ['unused'])]), \
                 patch.object(quality, 'source_inputs', side_effect=[{}, subprocess.CalledProcessError(1, ['git'])]), \
                 patch.object(quality.subprocess, 'check_output', return_value='a'*40), \
                 patch.object(quality, 'execute_owned', return_value=(0, False)), \
                 self.assertRaises(SystemExit):
                quality.main(['--profile', 'portable', '--artifacts', str(out)])
            saved = json.loads((out/'report.json').read_text())
            self.assertFalse(saved['passed'])
            self.assertTrue(saved['gates_passed'])
            self.assertEqual(saved['error_type'], 'CalledProcessError')

    def test_cache_override_and_readonly_flags_survive_existing_environment(self):
        from unittest.mock import patch
        with patch.dict(quality.os.environ, {'ELAB_GOCACHE': '/tmp/owned-cache', 'GOFLAGS': '-tags=example -mod=mod'}):
            env = quality.go_environment()
        self.assertEqual(env['GOCACHE'], '/tmp/owned-cache')
        self.assertTrue(env['GOFLAGS'].endswith('-p=1 -mod=readonly'))
        self.assertIn('-tags=example', env['GOFLAGS'])

    @unittest.skipIf(os.name != 'posix', 'requires POSIX process groups')
    def test_sigterm_reaps_owned_process_and_records_interruption(self):
        import os
        import signal
        import sys
        import time
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            pidfile, out = root/'child.pid', root/'quality'
            child = 'import os,time;from pathlib import Path;Path('+repr(str(pidfile))+').write_text(str(os.getpid()));time.sleep(30)'
            controller = ('import sys;sys.path.insert(0,'+repr(str(quality.ROOT/'scripts'))+');import quality;'
                          'quality.gate_plan=lambda profile:[("owned",[sys.executable,"-c",'+repr(child)+'])];'
                          'quality.source_inputs=lambda root:{};'
                          'raise SystemExit(quality.main(["--profile","portable","--artifacts",'+repr(str(out))+']))')
            process = subprocess.Popen([sys.executable, '-c', controller], stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
            try:
                deadline = time.monotonic()+8
                while not pidfile.exists() and process.poll() is None and time.monotonic()<deadline:
                    time.sleep(.01)
                self.assertTrue(pidfile.exists())
                process.send_signal(signal.SIGTERM)
                process.communicate(timeout=10)
                self.assertEqual(process.returncode, 143)
                saved = json.loads((out/'report.json').read_text())
                self.assertFalse(saved['passed'])
                self.assertEqual(saved['steps'][0]['status'], 'interrupted')
                self.assertEqual(saved['steps'][0]['owned_pid'], int(pidfile.read_text()))
                self.assertIsNotNone(saved['steps'][0]['owned_exit_code'])
                with self.assertRaises(ProcessLookupError):
                    os.kill(int(pidfile.read_text()), 0)
            finally:
                if process.poll() is None:
                    process.terminate()
                    try:
                        process.communicate(timeout=10)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.communicate(timeout=5)
                if pidfile.exists():
                    saved = json.loads((out/'report.json').read_text()) if (out/'report.json').exists() else {}
                    reaped = any(step.get('owned_exit_code') is not None for step in saved.get('steps', []))
                    if not reaped:
                        try:
                            os.killpg(int(pidfile.read_text()), signal.SIGTERM)
                        except ProcessLookupError:
                            pass

    def test_main_passes_only_with_stable_inputs(self):
        from unittest.mock import patch
        for changed in [False, True]:
            with self.subTest(changed=changed), tempfile.TemporaryDirectory() as tmp:
                out = Path(tmp)/'quality'
                with patch.object(quality, 'gate_plan', return_value=[('gate', ['unused'])]), \
                     patch.object(quality, 'source_inputs', side_effect=[{'input': 'a'}, {'input': 'b' if changed else 'a'}]), \
                     patch.object(quality.subprocess, 'check_output', return_value='a'*40), \
                     patch.object(quality, 'execute_owned', return_value=(0, False)):
                    code = quality.main(['--profile', 'portable', '--artifacts', str(out)])
                saved = json.loads((out/'report.json').read_text())
                self.assertEqual(code, 1 if changed else 0)
                self.assertEqual(saved['passed'], not changed)
                self.assertEqual(saved['inputs_unchanged'], not changed)

    def test_interrupt_during_report_update_leaves_no_running_step(self):
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp)/'quality'
            original = quality.write_json
            raised = False
            def write(path, value):
                nonlocal raised
                if value['steps'] and value['steps'][-1]['status'] == 'running' and not raised:
                    raised = True
                    raise quality.GateInterrupted(15)
                original(path, value)
            with patch.object(quality, 'write_json', side_effect=write):
                result = quality.run_plan([('gate', ['unused'])], out, Path(tmp), {}, 1)
            self.assertFalse(result['passed'])
            self.assertFalse(any(step['status'] == 'running' for step in result['steps']))

"""Reject misleading work-overlap witnesses before computing timing summaries."""
import copy
import importlib.util
from pathlib import Path
import unittest


spec = importlib.util.spec_from_file_location('capacity_timing', Path(__file__).with_name('capacity-responsiveness-demo.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class WorkOverlapTests(unittest.TestCase):
    def setUp(self):
        self.before = dict(paused=True, time_s=0, run_id=7,
                           step_work=dict(id='step-3', run_id=7, status='running', completed_samples=1))
        self.after = copy.deepcopy(self.before)
        self.after['step_work']['completed_samples'] = 2

    def test_progressed_running_work_is_eligible(self):
        self.assertTrue(module.manual_span(self.before, self.after))

    def test_queued_or_unstarted_work_is_not_numerical_overlap(self):
        for field, value in [('status', 'queued'), ('completed_samples', 0)]:
            before = copy.deepcopy(self.before)
            before['step_work'][field] = value
            self.assertFalse(module.manual_span(before, self.after))

    def test_terminal_work_is_not_an_ordinary_command_span(self):
        for status in ['completed', 'failed', 'canceling', 'canceled']:
            after = copy.deepcopy(self.after)
            after['step_work']['status'] = status
            self.assertFalse(module.manual_span(self.before, after))
            self.assertEqual(module.manual_span(self.before, after, cancel=True), status in ('canceling', 'canceled'))

    def test_changed_identity_or_regressed_progress_is_rejected(self):
        for field, value in [('id', 'step-4'), ('run_id', 8), ('completed_samples', 0)]:
            after = copy.deepcopy(self.after)
            after['step_work'][field] = value
            self.assertFalse(module.manual_span(self.before, after))

    def test_committed_clock_or_run_change_is_not_staged_work(self):
        for field, value in [('time_s', .001), ('paused', False), ('run_id', 8)]:
            after = copy.deepcopy(self.after)
            after[field] = value
            self.assertFalse(module.manual_span(self.before, after))


if __name__ == '__main__':
    unittest.main()

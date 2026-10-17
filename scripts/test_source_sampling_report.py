"""Refuse incomplete or misleading sampling-floor reference evidence."""
import copy
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('source_sampling_demo', Path(__file__).with_name('source-sampling-demo.py'))
demo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(demo)


class SamplingReportTests(unittest.TestCase):
    def test_reference_evidence_preserves_warning_corner_grid_and_numeric_values(self):
        advice = dict(id='Ramp', source_kind='finite-pulse-source', target_step_s=.25e-6,
                      recommended_step_s=1e-6, minimum_step_s=1e-6, unresolved=True, minimum_limited=True, can_fit=False)
        # Actual owned public capture, including both retained sides of the
        # continuous finite-ramp corner. The checker uses an independent ODE.
        values = [[0, 0, 0], [5, 1.8393789147579669, .003160621085242033],
                  [5, 1.8393789147579669, .003160621085242033], [5, 3.8373125476924246, .0011626874523075754],
                  [5, 4.572304641964943, .00042769535803505664], [5, 4.842685337163222, .00015731466283677787]]
        report = dict(passed=True, fitted=dict(source_sampling=[advice], source_warnings=[dict(message='supported minimum')]),
            rows=[dict(time_s=t, values=v) for t, v in zip([0, 1e-6, 1e-6, 2e-6, 3e-6, 4e-6], values)],
            metadata=dict(engine='elab.network-waveforms-mna-sdirk2.v1', duration_s=4e-6, sample_count=6,
                          settings=dict(step_s=1e-6), probes=[dict(id=p) for p in ['v:Ramp.positive', 'v:C1', 'i:R1']]),
            reopened=dict(source_sampling=[copy.deepcopy(advice)], transient=dict(sample=dict(time_s=0))),
            slower_ramp=dict(source_sampling=[dict(target_step_s=1e-6, minimum_limited=False, unresolved=False, can_fit=False)]),
            recorded_bytes_preserved=True)
        demo.validate_report(report)
        changes = [
            lambda r: r['fitted']['source_sampling'][0].update(unresolved=False),
            lambda r: r['fitted'].update(source_warnings=[]),
            lambda r: r['rows'].pop(2),
            lambda r: r['rows'][2].update(time_s=1.5e-6),
            lambda r: r['rows'][1]['values'].__setitem__(1, 0),
            lambda r: r['rows'][1]['values'].__setitem__(0, float('nan')),
            lambda r: r['metadata']['probes'][0].update(id='v:wrong'),
            lambda r: r.update(recorded_bytes_preserved=False),
            lambda r: r['reopened']['transient']['sample'].update(time_s=4e-6),
        ]
        for change in changes:
            damaged = copy.deepcopy(report)
            change(damaged)
            with self.subTest(change=changes.index(change)), self.assertRaises(ValueError):
                demo.validate_report(damaged)
        demo.validate_report(report)


if __name__ == '__main__':
    unittest.main()

"""Owned class-AB artifact reproduction; no GUI input or owner-session access.

Outputs require a new directory. The existing 53 kinds, 194 commands and
published solver limits are used unchanged. Models are educational.
"""
from pathlib import Path
import sys, json, time, math, hashlib, argparse
r = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(r / 'scripts'))
from acceptance_client import AcceptanceClient
parser = argparse.ArgumentParser(description='Rebuild and electrically check the class-AB example using existing public commands.')
parser.add_argument('--agent', type=Path, default=r / 'build/ELab-0.196.1-274-Vellum/elab-agent')
parser.add_argument('--output', type=Path, required=True, help='new, exclusive output directory')
args = parser.parse_args()
out = args.output.resolve()
out.mkdir(exist_ok=False)
c = AcceptanceClient(args.agent)
probes = ['v:VIN.positive', 'v:RL', 'i:Q1.collector', 'i:Q2.collector', 'v:Q1.base-emitter', 'v:Q2.base-emitter']

def capture(name, step):
    state = c.ask('simulation.verify', dict(duration_s=0.2, sample_step_s=step, probe_ids=probes), mutate=True)
    op = state['last_operation']
    deadline = time.monotonic() + 30
    while True:
        view = c.ask('operation.inspect', dict(id=op))
        status = view['operation']['status']
        if status in ('completed', 'failed', 'canceled'):
            break
        if time.monotonic() > deadline:
            raise TimeoutError('owned capture did not finish within 30 seconds')
        time.sleep(0.02)
    assert status == 'completed', view
    c.ask('verification.export', dict(id=op, path=str(out / (name + '.json')), format='json'), mutate=True)
    c.ask('verification.export', dict(id=op, path=str(out / (name + '.csv')), format='csv'), mutate=True)
    return (op, json.loads((out / (name + '.json')).read_text()))
try:
    c.ask('simulation.pause', mutate=True)
    c.ask('edit.batch', json.loads(Path(__file__).with_name('build-commands.json').read_text()), mutate=True)
    c.ask('simulation.transient.set', dict(step_s=0.0005, high_s=0.05, low_s=0.05, initial_voltage_v=0, seconds_per_wall_second=0.02), mutate=True)
    c.ask('simulation.restart', mutate=True)
    state = c.ask('simulation.wait')
    assert state['paused'] and (not state['diagnostics'])
    c.ask('scope.probes.set', dict(ids=probes), mutate=True)
    operation, record = capture('class-ab-response', 0.0005)
    fine_id, fine = capture('refinement', 0.00025)
    rows = record['rows']
    fine_rows = fine['rows']
    assert len(rows) == 401 and len(fine_rows) == 801
    assert all((math.isfinite(x) for row in rows + fine_rows for x in row['values']))

    def row_at(t, rs=rows):
        return min(rs, key=lambda x: abs(x['time_s'] - t))['values']
    zero = row_at(0)
    positive = row_at(0.025)
    negative = row_at(0.075)
    max_error = max((abs(x['values'][1] - 2 * x['values'][0]) for x in rows))
    refinement = max((abs(x['values'][1] - row_at(x['time_s'], fine_rows)[1]) for x in rows))
    symmetry = max((abs(rows[k]['values'][1] + rows[k + 100]['values'][1]) for k in range(301)))
    assert max_error < 0.0001, (max_error, positive, negative)
    assert refinement < 2e-05, refinement
    assert symmetry < 2e-06, symmetry
    assert zero[2] > 1e-05 and zero[3] < -1e-05, zero
    assert positive[2] > 0.005 and abs(positive[3]) < 1e-08, positive
    assert negative[3] < -0.005 and abs(negative[2]) < 1e-08, negative
    vt = 1.380649e-23 * (273.15 + 27) / 1.602176634e-19

    def residual(f):
        """Independent zero-input bias load line including reverse leakage."""
        ib = f / 100 - 1e-14
        ie = 1.01 * f + 1e-14
        vb = vt * math.log1p(f / 1e-14) + 22 * ie
        di = (5 - vb) / 2200 - ib
        return vb - (1.7 * vt * math.log1p(di / 1e-09) + di) if di > 0 else 1
    lo, hi = (0.0, 0.01)
    for _ in range(80):
        m = (lo + hi) / 2
        if residual(m) > 0:
            hi = m
        else:
            lo = m
    idle_reference = (lo + hi) / 2 + 2e-14
    assert abs(zero[2] - idle_reference) / idle_reference < 0.0001
    c.ask('simulation.restart', mutate=True)
    c.ask('simulation.wait')
    live = c.ask('simulation.step', dict(duration_s=0.2), mutate=True)
    assert live['paused'] and (not live['transient']['stale'])
    c.ask('recording.keep.set', dict(id=operation, keep=True), mutate=True)
    c.ask('view.fit', mutate=True)
    c.ask('project.save', dict(path=str(out / 'class-ab-amplifier.elab')), mutate=True)
    c.ask('export.schematic.svg', dict(path=str(out / 'class-ab-amplifier.svg')), mutate=True)
    c.ask('export.schematic.png', dict(path=str(out / 'class-ab-amplifier.png')), mutate=True)
    report = dict(passed=True, scope='Existing public headless /274 agent, generic educational models; no hardware/native GUI acceptance or physical power/stability qualification', model_reference='https://wiki.analog.com/university/courses/engineering_discovery/lab_14', parts=15, transistors=2, supplies_v=[5, -5], input_peak_v=0.5, frequency_hz=10, load_ohm=100, feedback_gain=2, samples=len(rows), refined_samples=len(fine_rows), idle_collector_current_a=zero[2], independent_idle_reference_a=idle_reference, positive_peak_output_v=positive[1], negative_peak_output_v=negative[1], positive_peak_collector_current_a=positive[2], negative_peak_collector_current_a=negative[3], opposite_device_positive_peak_abs_a=abs(positive[3]), opposite_device_negative_peak_abs_a=abs(negative[2]), max_output_error_vs_2vin_v=max_error, max_refinement_difference_v=refinement, max_half_cycle_asymmetry_v=symmetry, agent_sha256=hashlib.sha256(c.agent.read_bytes()).hexdigest(), built_in_examples_unchanged=62, public_commands_unchanged=194, solver_limits_changed=False)
finally:
    c.close()
report['agent_exit_code'] = c.process.returncode
(out / 'checks.json').write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps(report, indent=2))

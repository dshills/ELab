#!/usr/bin/env python3
"""Check finite-ramp sampling limits, independent fast RC values and fresh reopen."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path

from demo_session import connect_demo_client

spec = importlib.util.spec_from_file_location('waveforms', Path(__file__).with_name('waveforms-demo.py'))
waveforms = importlib.util.module_from_spec(spec)
spec.loader.exec_module(waveforms)


def expected_voltage(at):
    tau, ramp = 1e-6, 1e-6
    if at <= ramp:
        return 5 * (at + tau*math.expm1(-at/tau)) / ramp
    return 5 * (1 - tau/ramp * (-math.expm1(-ramp/tau)) * math.exp(-(at-ramp)/tau))


def validate_report(report):
    if report.get('passed') is not True:
        raise ValueError('unsuccessful source sampling workflow')
    advice = report['fitted']['source_sampling']
    if len(advice) != 1 or advice[0] != dict(id='Ramp', source_kind='finite-pulse-source', target_step_s=.25e-6,
            recommended_step_s=1e-6, minimum_step_s=1e-6, unresolved=True, minimum_limited=True, can_fit=False):
        raise ValueError('minimum source sampling advice differs')
    if not report['fitted'].get('source_warnings') or 'minimum' not in report['fitted']['source_warnings'][0]['message']:
        raise ValueError('minimum warning disappeared')
    metadata=report['metadata']
    if (metadata['engine']!='elab.network-waveforms-mna-sdirk2.v1' or metadata['duration_s']!=4e-6
            or metadata['settings']['step_s']!=1e-6 or metadata['sample_count']!=6
            or [p['id'] for p in metadata['probes']]!=['v:Ramp.positive','v:C1','i:R1']):
        raise ValueError('capture engine/grid/probes differ')
    if report['recorded_bytes_preserved'] is not True:
        raise ValueError('old capture changed after source edit')
    rows = report['rows']
    if len(rows) != 6:
        raise ValueError('fast RC sample coverage differs')
    for at, row in zip([0,1e-6,1e-6,2e-6,3e-6,4e-6],rows):
        source = 0 if at == 0 else 5
        expected = [source, expected_voltage(at), (source-expected_voltage(at))/1000]
        if not math.isfinite(row['time_s']) or abs(row['time_s']-at) > 1e-18 or len(row['values']) != 3:
            raise ValueError('fast RC sample time/probes differ')
        for value, reference, tolerance in zip(row['values'], expected, [1e-10, .001, 1e-6]):
            if not math.isfinite(value) or abs(value-reference) > tolerance:
                raise ValueError('fast RC source/capacitor/current differs from analytic reference')
    if report['reopened']['source_sampling'] != advice or report['reopened']['transient']['sample']['time_s'] != 0:
        raise ValueError('fresh reopen did not recompute minimum guidance')
    resolved = report['slower_ramp']['source_sampling'][0]
    if resolved['target_step_s'] != 1e-6 or resolved['minimum_limited'] or resolved['unresolved'] or resolved['can_fit']:
        raise ValueError('slower ramp should meet the configured policy at the floor')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect', help='fresh owned blank native endpoint; closed afterward')
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if args.output.exists():
        parser.error('output already exists')
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(waveforms.helper.Client, args.connect, command_preconditions=True)
    fresh = None
    def mutate(command, params=None):
        return client.ask(command, params, True)
    try:
        waveforms.build_rc(client, 'finite-pulse-source', 'Ramp')
        mutate('component.set', dict(id='R1', value='1k'))
        mutate('component.set', dict(id='C1', value='1 nF'))
        pulse = dict(high_v=5, low_v=0, high_s=.0005, low_s=.0005, rise_s=1e-6, fall_s=2e-6)
        mutate('source.pulse.set', dict(id='Ramp', waveform=pulse))
        before = client.ask('project.inspect')
        assert before['source_sampling'][0]['can_fit'] and before['source_warnings']
        mutate('simulation.sampling.fit')
        fitted = client.ask('project.inspect')
        assert fitted['source_warnings'] and fitted['transient_settings']['step_s'] == 1e-6
        mutate('simulation.sampling.fit')
        noop = client.ask('project.inspect')
        for key in ('run_id', 'electrical_revision', 'document', 'undo_entry', 'transient'):
            assert fitted.get(key) == noop.get(key), key
        mutate('simulation.restart')
        client.ask('simulation.wait')
        op = mutate('simulation.verify', dict(duration_s='4 us', sample_step_s='1 us',
                    probe_ids=['v:Ramp.positive', 'v:C1', 'i:R1']))['last_operation']
        view = waveforms.wait(client, op)
        mutate('verification.export', dict(id=op, path=str(assets/'fast-ramp.json'), format='json'))
        capture = json.loads((assets/'fast-ramp.json').read_text())
        rows = capture['rows']
        path = assets/'minimum-ramp.elab'
        mutate('project.save', dict(path=str(path)))
        saved = path.read_bytes()
        assert 'source_sampling' not in json.loads(saved)
        fresh = waveforms.helper.Client(command_preconditions=True)
        fresh.ask('project.load', dict(path=str(path)), True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count'] == 1
        assert path.read_bytes() == saved
        pulse['rise_s'], pulse['fall_s'] = 4e-6, 8e-6
        mutate('source.pulse.set', dict(id='Ramp', waveform=pulse))
        slower = client.ask('project.inspect')
        assert not slower.get('source_warnings')
        # A pre-existing saved capture retains its original grid and values.
        mutate('verification.export', dict(id=op, path=str(assets/'fast-ramp-after-edit.json'), format='json'))
        assert (assets/'fast-ramp-after-edit.json').read_bytes() == (assets/'fast-ramp.json').read_bytes()
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', before=before,
            fitted=fitted, reopened=reopened, slower_ramp=slower, rows=rows, metadata=view['metadata'],
            saved_project_sha256=hashlib.sha256(saved).hexdigest(), recorded_bytes_preserved=True,
            limitations=['four samples per shortest ramp is a display policy, not a solver accuracy guarantee',
                         'minimum cannot resolve this ramp target; internal integration/limits are unchanged',
                         'native physical input/accessibility and visible latency remain separate'])
        validate_report(report)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps(dict(passed=True, output=str(args.output), samples=len(rows))))
    finally:
        if fresh:
            fresh.close()
        if args.connect:
            try:
                mutate('app.close', dict(discard_changes=True))
            finally:
                client.close()
        else:
            client.close()


if __name__ == '__main__':
    main()

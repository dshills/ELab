#!/usr/bin/env python3
"""Check captured interval measurements; --connect replaces only an owned test session."""
import argparse
import importlib.util
import json
import math
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)

    def capture(duration, probes, step='1 ms', mode='auto'):
        admitted = client.ask('simulation.verify', dict(duration_s=duration, sample_step_s=step,
                                                       probe_ids=probes, input_mode=mode), True)
        identity = admitted['last_operation']
        deadline = time.monotonic() + 35
        while time.monotonic() < deadline:
            status = client.ask('operation.inspect', {'id': identity})['operation']['status']
            if status == 'completed':
                return identity
            assert status not in ('failed', 'canceled'), status
            time.sleep(.01)
        raise TimeoutError('owned capture did not finish')

    def measure(identity, probe, start, end):
        return client.ask('recording.statistics', dict(id=identity, probe_id=probe, start_s=start, end_s=end))

    try:
        schemas = {name: client.ask('capabilities.list', {'command': name})['capabilities'][0]
                   for name in ('recording.statistics', 'recording.statistics.set')}
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        rc = capture('20.5 ms', ['v:C1', 'i:R1'], mode='charge')
        before = client.ask('project.inspect')
        result = measure(rc, 'v:C1', '1.5 ms', '20.5 ms')
        assert client.ask('project.inspect') == before
        lo, hi, tau = .0015, .0205, .01
        e1, e2 = math.exp(-lo/tau)-math.exp(-hi/tau), math.exp(-2*lo/tau)-math.exp(-2*hi/tau)
        mean = 5*(1-tau/(hi-lo)*e1)
        rms = 5*math.sqrt(1-2*tau/(hi-lo)*e1+tau/(2*(hi-lo))*e2)
        assert abs(result['time_mean']-mean) < .003
        assert abs(result['time_rms']-rms) < .003
        assert abs(result['ac_rms']-math.sqrt(rms*rms-mean*mean)) < .003
        cfg = result['settings']
        client.ask('recording.statistics.set', cfg, True)
        selected = client.ask('project.inspect')
        assert selected['recording']['interval_statistics'] == result
        for key in ('document', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert selected.get(key) == before.get(key), key
        client.ask('recording.window.set', {'start_s': '10 ms', 'end_s': '20 ms'}, True)
        assert client.ask('project.inspect')['recording']['interval_statistics'] == result
        client.ask('recording.statistics', dict(cfg, end_s=cfg['start_s']), expect='invalid-params')
        client.ask('recording.statistics', dict(cfg, probe_id='v:missing'), expect='not-found')
        current = measure(rc, 'i:R1', '1.5 ms', '20.5 ms')
        assert current['probe']['unit'] == 'A'
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        assert measure(rc, 'v:C1', '1.5 ms', '20.5 ms') == result
        saved = assets/'captured-rc.elab'
        client.ask('project.save', {'path': str(saved)}, True)
        client.ask('project.load', {'path': str(saved)}, True)
        loaded = client.ask('simulation.wait')
        assert not loaded['recording'].get('interval_statistics')
        restored = measure(loaded['recording']['id'], 'v:C1', '1.5 ms', '20.5 ms')
        for key in ('time_mean', 'time_rms', 'ac_rms', 'minimum', 'maximum', 'peak_to_peak'):
            assert restored[key] == result[key], key
        client.ask('example.npn-pulse', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        pulse = capture('100 ms', ['v:VIN.positive'])
        pulse_stats = measure(pulse, 'v:VIN.positive', 0, '100 ms')
        for key, expected in [('time_mean', 2.5), ('time_rms', 5/math.sqrt(2)), ('ac_rms', 2.5), ('peak_to_peak', 5)]:
            assert abs(pulse_stats[key]-expected) < 1e-12, (key, pulse_stats[key])
        client.ask('example.rectifier', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        rectifier = []
        for capacitance in ('10uF', '47uF'):
            client.ask('component.set', {'id': 'C1', 'value': capacitance}, True)
            identity = capture('2 s', ['v:C1'], step='2 ms')
            stats = measure(identity, 'v:C1', '1.9 s', '2 s')
            # Separate Simpson integration of each linear saved segment.
            rows, offset = [], 0
            while True:
                page = client.ask('operation.samples', {'id': identity, 'offset': offset, 'limit': 200})
                rows.extend(page['rows'])
                offset = page['next_offset']
                if offset >= page['total']:
                    break
            segments = [(b['time_s']-a['time_s'], a['values'][0], b['values'][0])
                        for a, b in zip(rows, rows[1:]) if a['time_s'] >= 1.9 and b['time_s'] <= 2]
            span = math.fsum(dt for dt, _, _ in segments)
            avg = math.fsum(dt*(a+b)/2 for dt, a, b in segments)/span
            rms_ref = math.sqrt(math.fsum(dt*(a*a+4*((a+b)/2)**2+b*b)/6 for dt, a, b in segments)/span)
            ac_ref = math.sqrt(math.fsum(dt*((a-avg)**2+4*((a+b)/2-avg)**2+(b-avg)**2)/6
                                        for dt, a, b in segments)/span)
            assert abs(avg-stats['time_mean']) < 1e-12
            assert abs(rms_ref-stats['time_rms']) < 1e-12
            assert abs(ac_ref-stats['ac_rms']) < 1e-12
            rectifier.append(dict(capacitance=capacitance, measurements=stats))
        assert rectifier[1]['measurements']['ac_rms'] < rectifier[0]['measurements']['ac_rms']
        assert rectifier[1]['measurements']['peak_to_peak'] < rectifier[0]['measurements']['peak_to_peak']
        cfg = rectifier[1]['measurements']['settings']
        client.ask('recording.statistics.set', cfg, True)
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      schemas=schemas, rc=result, rc_mean_reference=mean, rc_rms_reference=rms,
                      current=current, pulse=pulse_stats, rectifier=rectifier,
                      checks=['RC analytic interval references within 3 mV linear-reconstruction envelope',
                              'Query purity, shared set result, independent plot interval and invalid-input refusal',
                              'Captured measurements survive later live edits and recompute after save/reopen; session setting clears',
                              'Ideal pulse duplicate edge rows yield exact configured mean/RMS/AC RMS',
                              'Rectifier last-cycle metrics agree with independent saved-segment Simpson integration',
                              '47 uF reduces captured last-cycle ripple RMS and peak-to-peak versus 10 uF'],
                      limits=['Linear reconstruction of retained samples, not continuous waveform or unseen-peak certification',
                              'No period/steady-state detection or physical device validation; no model/kernel/sampling changes'])
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], rc_mean=result['time_mean'],
                              ripple_rms=[r['measurements']['ac_rms'] for r in rectifier])))
    finally:
        if args.connect:
            client.ask('project.inspect')
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()

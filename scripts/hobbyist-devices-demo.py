#!/usr/bin/env python3
"""Replay new hobbyist devices through only public ELab commands."""
import argparse
import base64
import importlib.util
import json
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('controls', Path(__file__).with_name('controls-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def run(agent, assets):
    cases = [('buck', ['v:C1', 'i:L1', 'v:PWM'], .00002),
             ('opamp-slew', ['v:VIN', 'v:U1'], .00002),
             ('opamp-bandwidth', ['v:VIN', 'v:U1'], .00001),
             ('shift-register', ['v:U1.clk', 'v:U1.q0', 'v:U1.serial'], .0025),
             ('555-tone', ['v:BZ1', 'i:BZ1', 'v:C1'], .00001),
             ('light-alarm', ['v:BZ1', 'i:BZ1'], .0001)]
    reports = []
    for example, probes, step in cases:
        client = helper.Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            client.mutate('simulation.restart')
            state = client.ask('simulation.wait')
            assert state['device_model']['id'] == 'elab.hobbyist-devices.v1'
            assert not state['document']['dirty']
            state = client.step(step * 20)
            assert state['mixed']['kcl_residual_a'] < 1e-7
            assert abs(state['mixed']['power_residual_w']) < 2e-6
            exports = helper.capture_and_reopen(client, agent, assets, example, probes,
                                               duration=step*20, sample_step=step)
            audio = None
            if example in ('555-tone', 'light-alarm'):
                before = client.ask('project.inspect')['mixed']
                client.mutate('audio.render', {'id': 'BZ1', 'start_s': .02,
                                              'duration_s': .02, 'gain': .1})
                deadline = time.monotonic() + 30
                while True:
                    audio = client.ask('project.inspect')['audio_preview']
                    if audio['status'] != 'running':
                        break
                    assert time.monotonic() < deadline
                    time.sleep(.02)
                assert audio['status'] == 'completed', audio
                data = client.ask('audio.read', {'id': audio['id']})
                wav = base64.b64decode(data['wav_base64'])
                assert wav[:4] == b'RIFF' and len(wav) == 524
                assert not audio['playing']
                assert before == client.ask('project.inspect')['mixed']
                client.mutate('export.start', {'kind': 'audio-wav', 'id': audio['id'],
                                              'path': str(assets/(example+'.wav'))})
                while True:
                    work = client.ask('project.inspect')['project_work']
                    if work['status'] not in ('queued', 'running'):
                        break
                    assert time.monotonic() < deadline
                    time.sleep(.01)
                assert work['status'] == 'completed', work
                assert (assets/(example+'.wav')).read_bytes() == wav
            reports.append(dict(example=example, capture_fresh_reopen=True, exports=exports,
                                audio=audio))
        finally:
            client.close()
    return dict(passed=True, transport='headless-stdio', examples=reports)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.artifacts.mkdir(parents=True, exist_ok=False)
    report = run(args.agent, args.artifacts.resolve())
    args.output.write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()

"""One fresh packaged headless replay of the captured native slider cases."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sys

ROOT = Path('/Users/dshills/Development/projects/ELab')
OUT = Path('/private/tmp/elab-slider-input-174')
RUNTIME = Path('/private/tmp/elab-first-version-174-signed-runtime')
PIN = 'a8636912f15394e647a773d36395fd7f0380d9315f4c16a3e599a51390e6aa54'
assert sys.flags.optimize == 0
assert hashlib.sha256((RUNTIME/'elab-agent').read_bytes()).hexdigest() == PIN
sys.path.insert(0, str(ROOT/'scripts'))
spec = importlib.util.spec_from_file_location('slider_reference_client', ROOT/'scripts/persistence-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
dest = OUT/'reference'
dest.mkdir(exist_ok=False)
client = helper.Client(command_preconditions=True, agent_binary=RUNTIME/'agent')
report = {'agent_sha256': PIN, 'transport': 'fresh packaged headless stdio', 'passed': False}


def capture(name):
    state = client.ask('simulation.wait')
    (dest/(name+'.json')).write_text(json.dumps(state, indent=2)+'\n')
    return state


def reset():
    client.ask('project.load', {'path': str(OUT/'Baseline-Pot-174.elab'), 'discard_changes': True}, True)
    client.ask('simulation.wait')


try:
    reset()
    client.ask('simulation.step', {'duration_s': .005}, True)
    capture('before-paused-drag')
    native = json.loads((OUT/'after-paused-drag.json').read_text())
    wiper = next(p['wiper_position'] for p in native['components'] if p['id'] == 'Timing')
    client.ask('potentiometer.wiper.set', {'id': 'Timing', 'position': wiper}, True)
    capture('after-paused-drag')
    client.ask('edit.undo', {}, True)
    capture('after-paused-undo')
    reset()
    running = json.loads((OUT/'after-running-pause.json').read_text())
    event = next(e for e in running['transient']['events'] if e['mode'].startswith('wiper:'))
    client.ask('simulation.step', {'duration_s': event['time_s']}, True)
    capture('before-running-actuation')
    client.ask('potentiometer.wiper.set', {'id': 'Timing', 'position': wiper}, True)
    capture('after-running-actuation')
    client.ask('simulation.step', {'duration_s': running['mixed']['time_s']-event['time_s']}, True)
    capture('after-running-pause')
    reset()
    client.ask('simulation.step', {'duration_s': .005}, True)
    client.ask('potentiometer.wiper.set', {'id': 'Timing', 'position': .8}, True)
    capture('after-agent-hold')
    client.ask('project.clear', {'discard_changes': True}, True)
    closed = client.ask('app.close', {}, True)
    assert closed['closed'] and closed['shutdown']['exit_ready']
    (dest/'close-response.json').write_text(json.dumps(closed, indent=2)+'\n')
    report['passed'] = True
finally:
    client.close()
    report['process_exit_code'] = client.process.returncode
    assert client.process.returncode == 0
    (dest/'report.json').write_text(json.dumps(report, indent=2)+'\n')
print(json.dumps(report))

"""Replay the captured startup input cases through one fresh packaged agent."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sys

root = Path('/Users/dshills/Development/projects/ELab')
out = Path('/private/tmp/elab-startup-input-174')
runtime = Path('/private/tmp/elab-first-version-174-signed-runtime')
pin = 'a8636912f15394e647a773d36395fd7f0380d9315f4c16a3e599a51390e6aa54'
assert sys.flags.optimize == 0
assert hashlib.sha256((runtime/'elab-agent').read_bytes()).hexdigest() == pin
dest = out/'reference'
dest.mkdir(exist_ok=False)
spec = importlib.util.spec_from_file_location('startup_reference_helper', root/'scripts/persistence-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
client = helper.Client(command_preconditions=True, agent_binary=runtime/'agent')
report = {'passed': False, 'agent_sha256': pin, 'transport': 'fresh packaged headless stdio', 'cases': []}


def capture(name):
    state = client.ask('simulation.wait')
    (dest/(name+'.json')).write_text(json.dumps(state, indent=2)+'\n')


try:
    cases = [('rc-restart', 'RC', 'C1', 'capacitor.initial.set', '3 V', 'restart'),
             ('rc-default', 'RC', 'C1', 'capacitor.initial.set', '4 V', 'default'),
             ('rl-restart', 'RL', 'L1', 'inductor.initial.set', '30 mA', 'restart'),
             ('rl-default', 'RL', 'L1', 'inductor.initial.set', '40 mA', 'default'),
             ('rl-agent-held-reset', 'RL', 'L1', 'inductor.initial.set', '40 mA', 'agent')]
    for name, example, part, command, draft, action in cases:
        client.ask('project.load', {'path': str(out/('Baseline-'+example+'-174.elab')), 'discard_changes': True}, True)
        client.ask('component.select', {'id': part}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': .005}, True)
        capture('before-'+name)
        if action == 'agent':
            record = json.loads((out/'agent-interruption-result.json').read_text())
            assert record['status'] == 'agent_edit_completed'
            baseline = json.loads((out/'draft-rl-agent-held-reset.json').read_text())
            delta = record['before']['command_revision']-baseline['command_revision']
            assert delta in [0, 1]
            if delta:
                client.ask(command, {'id': part, 'value': '30 mA'}, True)
        client.ask(command, {'id': part, 'value': draft}, True)
        if action == 'restart':
            client.ask('simulation.restart', {}, True)
        elif action == 'default':
            client.ask(command, {'id': part, 'value': 'default'}, True)
        capture('after-'+name)
        report['cases'].append(name)
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

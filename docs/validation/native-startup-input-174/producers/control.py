from pathlib import Path
import argparse
import datetime
import hashlib
import importlib.util
import json
import subprocess

root = Path('/Users/dshills/Development/projects/ELab')
out = Path('/private/tmp/elab-startup-input-174')
runtime = Path('/private/tmp/elab-first-version-174-signed-runtime')
parser = argparse.ArgumentParser()
parser.add_argument('--name', required=True)
parser.add_argument('--actions', default='[]')
args = parser.parse_args()
assert args.name.isascii() and all(c.isalnum() or c in '-_' for c in args.name)
report = json.loads((out / 'report.json').read_text())
assert report['status'] == 'ready_for_cua'
process = subprocess.check_output(['/bin/ps', '-p', str(report['owned_pid']), '-o', 'command='], text=True).strip()
assert process == report['observed_command']
assert hashlib.sha256((runtime / 'ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest() == report['native_sha256']
spec = importlib.util.spec_from_file_location('owned_startup_helper', root / 'scripts/persistence-demo.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
client = module.Client(str(out / 'endpoint.json'), command_preconditions=True, agent_binary=runtime / 'agent')
try:
    before = client.ask('project.inspect')
    actions = json.loads(args.actions)
    for action in actions:
        client.ask(action['command'], action.get('params', {}), True)
    state = client.ask('simulation.wait')
    with (out / (args.name + '.json')).open('x') as stream:
        json.dump(state, stream, indent=2)
        stream.write('\n')
    with (out / (args.name + '-actions.json')).open('x') as stream:
        json.dump(dict(before_command_revision=before['command_revision'], actions=actions,
                       observed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat()), stream, indent=2)
    selected = next((p for p in state['components'] if p['id'] == state['selected']), {})
    print(json.dumps(dict(name=args.name, command_revision=state['command_revision'], paused=state['paused'],
                         selected=selected, time_s=state.get('transient', {}).get('sample', {}).get('time_s'),
                         undo_entry=state.get('undo_entry'), stale=state.get('mixed', {}).get('stale'))))
finally:
    client.close()

"""Read-only replay of exact /179 junction observations; never starts an app."""
from pathlib import Path
import gzip
import hashlib
import json
import sys

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def digest(data):
    return hashlib.sha256(data).hexdigest()


def raw(name):
    return gzip.decompress((HERE/'raw'/(name+'.gz')).read_bytes())


def read(name):
    return json.loads(raw(name))


def graph(state):
    return dict(components=state['components'], wires=state['wires'])


def net_groups(state):
    # Independent connected-component calculation over explicit endpoint pairs.
    groups = [{p['id']+'.'+terminal} for p in state['components'] for terminal in p['terminals']]
    for wire in state['wires']:
        if not wire['connected']:
            continue
        joined = set()
        for group in groups:
            if wire['from'] in group or wire['to'] in group:
                joined |= group
        assert wire['from'] in joined and wire['to'] in joined
        groups = [group for group in groups if not group & joined]+[joined]
    result = sorted(sorted(group) for group in groups)
    assert result == sorted(sorted(net['terminals']) for net in state['nets'])
    return result


def main():
    checkpoint = json.loads((HERE/'checkpoint.json').read_text())
    files = {str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
    assert files == set(checkpoint['files'])
    for relative, sha in checkpoint['files'].items():
        p = HERE/relative
        assert not p.is_symlink() and p.resolve().is_relative_to(HERE) and digest(p.read_bytes()) == sha, relative
    for relative, entry in json.loads((HERE/'manifest.json').read_text()).items():
        data = (HERE/relative).read_bytes(); original = gzip.decompress(data)
        assert digest(data) == entry['stored_sha256'] and digest(original) == entry['original_sha256']
        assert len(original) == entry['original_bytes']
    plan, collected = read('plan.json'), read('report.json')
    report = json.loads((HERE/'report.json').read_text())
    receipt_path = ROOT/'research/ui-audit/first-version/geometry-action-review-checkpoint.json'
    assert digest(receipt_path.read_bytes()) == plan['receipt_sha256'] == report['receipt_sha256']
    receipt = json.loads(receipt_path.read_text())
    assert receipt['agent_sha256'] == plan['agent_sha256'] == report['agent_sha256']
    assert len(receipt['quality']['source_sha256']) == plan['source_count'] == 654
    for relative, sha in receipt['quality']['source_sha256'].items():
        assert digest((ROOT/relative).read_bytes()) == sha, relative
    assert collected['passed'] and collected['failure'] is None and collected['agent_unchanged']
    assert len(collected['checks']) == 20 and all(check['passed'] for check in collected['checks'])
    assert len(collected['process_exits']) == 2
    total, refused = 0, 0
    for outcome in collected['process_exits']:
        rows = [json.loads(line) for line in raw(outcome['name']+'-protocol.jsonl').splitlines()]
        assert len(rows) == outcome['requests']*2
        commands = [(rows[i]['request'], rows[i+1]['response']) for i in range(0, len(rows), 2)]
        command_revision = 0
        for i, (request, response) in enumerate(commands):
            assert request['id'] == response['id'] == f'junction-{i+1}' and request['version'] == 1
            if 'key' in request:
                assert request['expected_command_revision'] == command_revision
                assert request['key'] == outcome['name']+'-'+str(i+1)
            if response.get('error'):
                assert response['error']['code'] == 'invalid-params'
                assert request['command'] in ('wire.junction.insert', 'edit.batch')
                before, after = commands[i-1], commands[i+1]
                assert before[0]['command'] == after[0]['command'] == 'project.inspect'
                assert before[1]['result'] == after[1]['result']
                refused += 1
            state = response.get('result')
            if isinstance(state, dict) and 'components' in state and 'command_revision' in state:
                command_revision = state['command_revision']
                net_groups(state)
        assert commands[-1][0]['command'] == 'app.close'
        assert commands[-1][1]['result']['closed']
        assert commands[-1][1]['result']['shutdown'] == outcome['shutdown']
        work = outcome['shutdown']
        assert outcome['exit_code'] == 0 and work['status'] == 'completed'
        assert all(work[k] for k in ('workers_joined', 'recovery_finalized', 'exit_ready'))
        assert not raw(outcome['name']+'-stderr.txt')
        total += len(commands)
    assert total == report['requests'] == 78 and refused == report['refusals'] == 18
    baseline, joined, undone, redone, deleted, restored, reopened = (
        read(name+'.json') for name in ('baseline', 'joined', 'undone', 'redone', 'deleted', 'restored', 'reopened'))
    assert baseline['limits']['drawing_objects'] == 100 and baseline['limits']['wires'] == 300
    assert len(baseline['components']) == 18 and len(baseline['wires']) == 9
    assert len(net_groups(baseline)) == 9
    assert joined['components'][:-1] == baseline['components']
    junction = joined['components'][-1]
    assert junction['id'] == 'W1' and junction['kind'] == 'junction'
    assert junction['position'] == dict(x=0, y=0) and junction['terminals'] == ['node']
    assert joined['selected'] == 'W1' and joined['drawing']['tool'] == 'select'
    expected_wires = [dict(w, to='W1.node') if w['id'] != 'Route9' else w for w in baseline['wires']]
    expected_wires += [dict(id=f'W{i+1}', **{'from':'W1.node', 'to':f'End{i}b.node', 'connected':True}) for i in range(1,9)]
    assert joined['wires'] == expected_wires
    groups = net_groups(joined)
    assert groups == sorted([sorted(['W1.node']+[f'End{i}{suffix}.node' for i in range(1,9) for suffix in 'ab']),
                             ['End9a.node','End9b.node']])
    assert graph(undone) == graph(baseline) and net_groups(undone) == net_groups(baseline)
    for state in (redone, restored, reopened):
        assert graph(state) == graph(joined) and net_groups(state) == groups
    assert deleted['components'] == baseline['components']
    assert deleted['wires'] == [baseline['wires'][-1]] and len(net_groups(deleted)) == 17
    assert raw('junction.elab') == raw('resaved.elab')
    stored = read('junction.elab')
    assert stored['graph'] == graph(joined)
    assert reopened['paused'] and not reopened['document']['dirty']
    # Observe the actual staged failure and subsequent ID allocation, rather than
    # treating a hash or a passing collector flag as proof of rollback.
    commands = [json.loads(line) for line in raw('writer-protocol.jsonl').splitlines()]
    requests = [row['request'] for row in commands if 'request' in row]
    batches = [request for request in requests if request['command'] == 'edit.batch']
    assert len(batches) == 2 and len(batches[1]['params']['edits']) == 2
    assert batches[1]['params']['edits'][0] == dict(command='wire.junction.insert', params=dict(id='W1',wire_ids=[f'Route{i}' for i in range(1,9)],position=dict(x=0,y=0)))
    assert batches[1]['params']['edits'][1]['params']['from'] == 'missing.node'
    assert [request['params']['id'] for request in requests if request['command'] == 'wire.junction.insert'][-3:] == ['W1','W2','W9']
    assert report['application_changed'] is False and report['new_release'] is False
    assert report['native_input'] is False and report['owner_acceptance'] is False
    sys.path.insert(0, str(ROOT/'scripts'))
    from package_review import members
    folder = Path(receipt['review_path']); actual = members(folder); actual.pop('RELEASE_MANIFEST.json')
    assert actual == json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual) == 418
    assert digest((ROOT/receipt['sdk']['archive']).read_bytes()) == receipt['sdk']['sha256']
    assert digest((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()) == receipt['canonical']['native_sha256']
    print(json.dumps(dict(checks=20, requests=78, whole_state_refusals=18,
                          named_routes_joined=8, unrelated_routes_preserved=1,
                          byte_identical_fresh_reopens=1, normal_process_exits=2,
                          application_changed=False)))


if __name__ == '__main__':
    main()

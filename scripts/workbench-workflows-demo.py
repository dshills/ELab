#!/usr/bin/env python3
"""Exercise drawing/explanation/plot/experiment/proposal parity on an owned session.

Uses public service commands, not physical mouse/keyboard input. A connected
native session must be an explicitly disposable blank fixture.
"""
import argparse
import hashlib
import json
from pathlib import Path

from acceptance_client import AcceptanceClient
from demo_session import require_fresh_demo_session


def exercise(client):
    start = client.ask('project.inspect')
    require_fresh_demo_session(start)
    catalog = client.ask('capabilities.list')['capabilities']
    required = {'drawing.preview', 'component.explain', 'scope.probe.plot',
                'edit.batch.preview', 'edit.proposal.set',
                'edit.proposal.apply', 'edit.proposal.dismiss'}
    assert required <= {entry['id'] for entry in catalog}
    client.ask('simulation.pause', mutate=True)
    parts = {}
    for kind, x, y in [('dc-source', -200, 0), ('resistor', 0, -110),
                       ('led', 200, 0), ('ground', 0, 110)]:
        client.ask('drawing.tool.set', dict(tool='place', kind=kind), mutate=True)
        before = client.ask('project.inspect')
        preview = client.ask('drawing.preview', dict(position=dict(x=x, y=y)))
        assert preview['valid'] and preview['command'] == 'component.place'
        assert before == client.ask('project.inspect'), 'preview mutated accepted state'
        placed = client.ask(preview['command'], preview['params'], mutate=True)
        parts[kind] = placed['selected']
    for a, b in [('V1.positive', 'R1.1'), ('R1.2', 'D1.anode'),
                 ('D1.cathode', 'GND1.reference'), ('GND1.reference', 'V1.negative')]:
        client.ask('wire.connect', {'from': a, 'to': b}, mutate=True)
    client.ask('simulation.resume', mutate=True)
    baseline = client.ask('simulation.wait')
    assert abs(baseline['result']['current_a'] - 0.002916796577) < 2e-8
    client.ask('simulation.pause', mutate=True)
    explanation = client.ask('component.explain', dict(id=parts['led']))
    assert explanation['reading_state'] == 'live' and explanation['limitations']
    assert any('conducting' in fact for fact in explanation['facts'])
    client.ask('example.load', dict(id='rc', discard_changes=True), mutate=True)
    client.ask('simulation.pause', mutate=True)
    client.ask('simulation.restart', mutate=True)
    client.ask('simulation.wait')
    client.ask('scope.probes.set', dict(ids=[]), mutate=True)
    client.ask('scope.visibility.set', dict(visible=False), mutate=True)
    plotted = client.ask('scope.probe.plot', dict(id='v:C1'), mutate=True)
    assert plotted['scope_visible'] and plotted['trace_focus']['probe_id'] == 'v:C1'
    batch = dict(name='Double RC capacitance', edits=[dict(
        command='component.set', params=dict(id='C1', value='2uF'))])
    before = client.ask('project.inspect')
    preview = client.ask('edit.batch.preview', batch)
    assert len(preview['changes']) == 1 and preview['changes'][0]['id'] == 'C1'
    assert before == client.ask('project.inspect')
    staged = client.ask('edit.proposal.set', batch, mutate=True)
    assert staged['components'] == before['components']
    original_id = staged['edit_proposal']['id']
    client.ask('component.set', dict(id='R1', value='20k'), mutate=True)
    stable = client.ask('project.inspect')
    assert stable['edit_proposal']['stale']
    client.ask('edit.proposal.apply', dict(id=original_id), mutate=True, expect='conflict')
    assert stable == client.ask('project.inspect')
    refreshed = client.ask('edit.proposal.set', batch, mutate=True)
    current_id = refreshed['edit_proposal']['id']
    assert current_id != original_id
    applied = client.ask('edit.proposal.apply', dict(id=current_id), mutate=True)
    assert applied['undo_name'] == batch['name'] and 'edit_proposal' not in applied
    assert len(applied['example']['changes']) == 2
    client.ask('edit.undo', mutate=True)
    restored = client.ask('project.inspect')
    assert len(restored['example']['changes']) == 1
    assert restored['example']['changes'][0]['id'] == 'R1'
    proposed = client.ask('edit.proposal.set', batch, mutate=True)
    client.ask('edit.proposal.dismiss', dict(id=proposed['edit_proposal']['id']), mutate=True)
    assert 'edit_proposal' not in client.ask('project.inspect')
    return dict(passed=True, scope=__doc__.strip(), component_kinds=len(client.ask('parts.list')),
                public_commands=len(catalog), example_count=len(client.ask('examples.list')),
                led_current_a=baseline['result']['current_a'], final_paused=restored['paused'],
                checks=['pure snapped placement', 'LED electrical reference', 'accepted explanation',
                        'plot reveals live scope', 'pure atomic preview', 'stale proposal refusal',
                        'refresh/apply', 'one-operation Undo', 'template diff', 'dismissal'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--connect', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if args.output.exists():
        raise FileExistsError('refusing to replace a previous receipt')
    client = AcceptanceClient(args.agent, args.connect)
    try:
        result = exercise(client)
        result['agent_sha256'] = hashlib.sha256(args.agent.read_bytes()).hexdigest()
    finally:
        client.close()
    result['agent_exit_code'] = client.process.returncode
    args.output.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result))


if __name__ == '__main__':
    main()

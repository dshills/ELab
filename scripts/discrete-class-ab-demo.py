#!/usr/bin/env python3
"""Check the all-discrete amplifier through public commands in an owned session.

A connected endpoint must be a fresh disposable blank fixture. This is semantic
software testing, not physical mouse/keyboard or hardware acceptance.
"""
import argparse
import hashlib
import json
import math
from pathlib import Path
import time

from acceptance_client import AcceptanceClient
from demo_session import require_fresh_demo_session

PROBES = ['v:VIN.positive', 'v:RL', 'i:Q6.collector', 'i:Q7.collector',
          'i:Q8.collector', 'v:C1']


def exercise(client, output):
    require_fresh_demo_session(client.ask('project.inspect'))
    client.ask('simulation.pause', mutate=True)
    client.ask('example.load', {'id': 'discrete-class-ab'}, mutate=True)
    client.ask('simulation.pause', mutate=True)
    client.ask('simulation.restart', mutate=True)
    initial = client.ask('simulation.wait')
    assert not initial['diagnostics'] and not initial['document']['dirty']
    assert len(initial['mixed']['transistors']) == 8
    assert not initial['mixed'].get('opamps')
    assert initial['limits']['bipolar_transistors_total'] == 16
    records = []
    for name, step in [('response', .0005), ('refinement', .00025)]:
        state = client.ask('simulation.verify', dict(duration_s=.2,
            sample_step_s=step, probe_ids=PROBES), mutate=True)
        identifier = state['last_operation']
        deadline = time.monotonic() + 30
        while True:
            operation = client.ask('operation.inspect', dict(id=identifier))
            status = operation['operation']['status']
            if status in ('completed', 'failed', 'canceled'):
                break
            if time.monotonic() > deadline:
                raise TimeoutError('amplifier capture exceeded 30 seconds')
            time.sleep(.02)
        assert status == 'completed', operation
        for fmt in ('json', 'csv'):
            client.ask('verification.export', dict(id=identifier,
                path=str(output / (name+'.'+fmt)), format=fmt), mutate=True)
        records.append(json.loads((output / (name+'.json')).read_text()))
    assert [len(x['rows']) for x in records] == [401, 801]
    assert initial['mixed'] == client.ask('project.inspect')['mixed']
    rows, refined = [x['rows'] for x in records]
    steady = [x for x in rows if x['time_s'] >= .05]
    assert all(math.isfinite(v) for x in rows + refined for v in x['values'])
    max_error = max(abs(x['values'][1]-11*x['values'][0]) for x in steady)
    refinement = max(abs(x['values'][1]-min(refined,
        key=lambda y: abs(y['time_s']-x['time_s']))['values'][1]) for x in steady)
    assert max_error < .055 and refinement < .002, (max_error, refinement)
    positive = min(rows, key=lambda x: abs(x['time_s']-.125))['values']
    negative = min(rows, key=lambda x: abs(x['time_s']-.175))['values']
    zero = rows[0]['values']
    assert zero[2] > 1e-5 and zero[3] < -1e-5
    assert positive[2] > .009 and abs(positive[3]) < 1e-6
    assert negative[3] < -.009 and abs(negative[2]) < 1e-6
    assert abs(zero[4]-.0009853264853) < 1e-9
    client.ask('simulation.step', dict(duration_s=.2), mutate=True)
    live = client.ask('simulation.wait')
    assert not live['mixed']['stale'] and abs(live['mixed']['time_s']-.2) < 1e-10
    assert live['mixed']['kcl_residual_a'] < 1e-9
    assert abs(live['mixed']['power_residual_w']) < 1e-8
    client.ask('view.fit', mutate=True)
    project = output / 'discrete-class-ab.elab'
    client.ask('project.save', dict(path=str(project)), mutate=True)
    for fmt in ('svg', 'png'):
        client.ask('export.schematic.'+fmt,
            dict(path=str(output / ('discrete-class-ab.'+fmt))), mutate=True)
    # Reconstruct the exact authored graph through existing editing commands,
    # rather than relying solely on template loading.
    graph = json.loads(project.read_text())['graph']
    edits = []
    for part in graph['components']:
        params = dict(id='BUILT_'+part['id'], kind=part['kind'], position=part['position'])
        if part.get('unit'):
            params['value'] = str(part['value'])+' '+part['unit']
        if part['kind'] == 'net-label':
            params['value'] = part['label']
        edits.append(dict(command='component.place', params=params))
        for _ in range(part.get('rotation', 0)//90):
            edits.append(dict(command='component.rotate', params=dict(id='BUILT_'+part['id'])))
        if part.get('sine'):
            edits.append(dict(command='source.sine.set', params=dict(id='BUILT_'+part['id'], waveform=part['sine'])))
        if part.get('initial_voltage_v') is not None:
            edits.append(dict(command='capacitor.initial.set', params=dict(id='BUILT_'+part['id'], value=str(part['initial_voltage_v'])+' V')))
    for wire in graph['wires']:
        params = {key: 'BUILT_'+wire[key] for key in ('from', 'to')}
        if wire.get('points'):
            params['points'] = wire['points']
        edits.append(dict(command='wire.connect', params=params))
    recipe = dict(name='Build an all-discrete class-AB amplifier', edits=edits)
    (output / 'build-commands.json').write_text(json.dumps(recipe, indent=2)+'\n')
    client.ask('project.clear', dict(discard_changes=True), mutate=True)
    for offset in range(0, len(edits), 80):
        client.ask('edit.batch', dict(name=recipe['name'], edits=edits[offset:offset+80]), mutate=True)
    client.ask('simulation.restart', mutate=True)
    rebuilt = client.ask('simulation.wait')
    assert not rebuilt['diagnostics'], rebuilt['diagnostics']
    # Public connection commands generate IDs; electrical graph identity ignores
    # those incidental wire IDs while retaining all authored terminal pairs.
    normalized = [dict(p, id=p['id'].removeprefix('BUILT_')) for p in rebuilt['components']]
    assert normalized == initial['components']
    assert [(w['from'].removeprefix('BUILT_'), w['to'].removeprefix('BUILT_')) for w in rebuilt['wires']] == [(w['from'], w['to']) for w in initial['wires']]
    client.ask('project.load', dict(path=str(project), discard_changes=True), mutate=True)
    loaded = client.ask('simulation.wait')
    assert loaded['components'] == initial['components']
    assert loaded['document']['recording_count'] == 2 and not loaded['document']['dirty']
    return dict(passed=True, scope=__doc__.strip(), parts=initial['simulated_component_count'],
        drawing_objects=len(initial['components']), nodes=len(initial['nets']),
        bipolar_capacity=16, transistors=8, component_kinds=len(client.ask('parts.list')),
        public_commands=len(client.ask('capabilities.list')['capabilities']),
        examples=len(client.ask('examples.list')), samples=[401, 801],
        zero_input_output_v=zero[1], idle_upper_collector_a=zero[2],
        idle_lower_collector_a=zero[3], tail_current_a=zero[4],
        positive_peak_output_v=positive[1], negative_peak_output_v=negative[1],
        max_output_error_vs_11vin_after_50ms_v=max_error,
        max_step_refinement_after_50ms_v=refinement,
        live_kcl_residual_a=live['mixed']['kcl_residual_a'],
        live_power_residual_w=live['mixed']['power_residual_w'],
        reconstructed_via_existing_edit_commands=True, clean_recorded_reopen=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--connect', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(exist_ok=False, parents=True)
    client = AcceptanceClient(args.agent, args.connect)
    try:
        report = exercise(client, output)
    finally:
        client.close()
    report.update(agent_exit_code=client.process.returncode,
        agent_sha256=hashlib.sha256(args.agent.read_bytes()).hexdigest())
    (output / 'checks.json').write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()

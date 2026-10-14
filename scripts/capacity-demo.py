#!/usr/bin/env python3
"""Build and validate the physical capacity boundary over public commands."""
import argparse
import importlib.util
import json
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def build_chain(client, source_kind, part_kind, count, tie=False):
    client.ask('project.clear', {'discard_changes': True}, True)
    client.ask('simulation.resume', {}, True)
    edits = [{'command': 'component.place', 'params': {'id': 'V1', 'kind': source_kind, 'position': {'x': -800, 'y': 0}}},
             {'command': 'component.place', 'params': {'id': 'GND1', 'kind': 'ground', 'position': {'x': 800, 'y': 0}}}]
    # Session IDs remain reserved; use fresh IDs discovered from auto placement.
    for edit in edits:
        edit['params'].pop('id')
    for i in range(count):
        edits.append({'command': 'component.place', 'params': {'kind': part_kind,
                      'position': {'x': (i % 8)*160-560, 'y': (i // 8)*160-560}}})
    placed = client.ask('edit.batch', {'name': 'Capacity components', 'edits': edits}, True)
    source = next(p['id'] for p in placed['components'] if p['kind'] == source_kind)
    ground = next(p['id'] for p in placed['components'] if p['kind'] == 'ground')
    parts = [p['id'] for p in placed['components'] if p['kind'] == part_kind]
    connections = []
    previous = source+'.positive'
    for part in parts:
        connections.append({'command': 'wire.connect', 'params': {'from': previous, 'to': part+'.1'}})
        previous = part+'.2'
    connections += [{'command': 'wire.connect', 'params': {'from': previous, 'to': ground+'.reference'}},
                    {'command': 'wire.connect', 'params': {'from': source+'.negative', 'to': ground+'.reference'}}]
    if tie:
        connections.append({'command': 'wire.connect', 'params': {'from': parts[-1]+'.wiper', 'to': parts[-1]+'.1'}})
    started = time.perf_counter()
    client.ask('edit.batch', {'name': 'Capacity connections', 'edits': connections}, True)
    ready = client.ask('simulation.wait')
    return ready, source, parts, (time.perf_counter()-started)*1000


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    try:
        state, source, parts, startup_ms = build_chain(client, 'pulse-source', 'potentiometer', 50, True)
        assert state['simulated_component_count'] == 51 and len(state['nets']) == 100
        assert state['limits']['simulated_components'] == 64 and state['limits']['electrical_nodes_including_ground'] == 100
        current = 5/495000
        for i, p in enumerate(state['mixed']['potentiometers']):
            expected = 5-current*(i*10000+5000) if i < 49 else current*5000
            assert abs(p['wiper_voltage_v']-expected) < 1e-7
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        probes = ['v:'+source, 'v:'+parts[0]+'.wiper', 'v:'+parts[24]+'.wiper', 'i:'+parts[48]+'.1']
        client.ask('scope.probes.set', {'ids': probes}, True)
        started = time.perf_counter()
        client.ask('simulation.step', {'duration_s': .06}, True)
        step_ms = (time.perf_counter()-started)*1000
        low = client.ask('project.inspect')
        assert len(low['scope']['traces']) == 4 and not low['transient']['stale']
        assert max(abs(p['wiper_voltage_v']) for p in low['mixed']['potentiometers']) < 1e-7
        accepted = client.ask('simulation.verify', {'duration_s': .06, 'probe_ids': probes}, True)
        deadline = time.monotonic()+15
        while True:
            record = client.ask('operation.inspect', {'id': accepted['last_operation']})
            if record['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            assert time.monotonic() < deadline
            time.sleep(.005)
        assert record['operation']['status'] == 'completed', record
        assert len(record['metadata']['mixed']['nodes_v']) == 100
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'hundred-node.elab')
        client.ask('project.save', {'path': path, 'overwrite': True}, True)
        client.ask('export.schematic.svg', {'path': str(directory/'hundred-node.svg'), 'overwrite': True}, True)
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count'] == 1
        assert len(reopened['nets']) == 100 and reopened['simulated_component_count'] == 51
        fresh.close()
        fresh = None
        boundary, _, _, _ = build_chain(client, 'dc-source', 'resistor', 63)
        assert boundary['simulated_component_count'] == 64
        assert all(abs(p['current_a']-5/63000) < 1e-10 for p in boundary['mixed']['resistors'])
        excess, _, _, _ = build_chain(client, 'dc-source', 'resistor', 64)
        assert excess['status'] == 'unsupported' and excess['diagnostics'][0]['code'] == 'mixed-limit'
        client.ask('project.save', {'path': str(directory/'excess-parts.elab'), 'overwrite': True}, True)
        client.ask('project.load', {'path': path, 'discard_changes': True}, True)
        client.ask('simulation.wait')
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      simulated_part_boundary=64, hundred_node_parts=51, hundred_node_drawing_objects=52,
                      electrical_nodes_including_ground=100, traces=4, startup_wall_ms=startup_ms, step_60ms_wall_ms=step_ms,
                      divider_current_a=current, checks=['public atomic placement/connection batches', 'independent divider references',
                      '64-part resistor reference', '100 nodes and four live traces', 'pulse edge/high-low behavior',
                      '100-node Verify and embedded record import', 'save/reopen with fresh paused startup',
                      'SVG at capacity', 'excess physical parts diagnosed but saved', 'final owned project clean and paused'],
                      limitations=['per-device source/nonlinear/capacitor bounds still apply',
                      'capacity fixtures do not prove convergence for every topology or input-to-pixel latency',
                      'private unsigned local build'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

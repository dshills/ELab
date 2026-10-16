#!/usr/bin/env python3
"""Check shared workspace layout in a fresh native or headless session.

Run native first, then replay measured sizes headlessly using --sizes-from.
"""
import argparse
import importlib.util
import json
from pathlib import Path
import tempfile
import time

from validation_artifacts import discovery_summary

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closed afterward')
    parser.add_argument('--sizes-from', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--discovery-output', type=Path, required=True)
    args = parser.parse_args()
    assert not (args.connect and args.sizes_from)
    replay = json.loads(args.sizes_from.read_text())['sizes'] if args.sizes_from else {
        'default': {'width_px': 594, 'height_px': 219},
        'large': {'width_px': 594, 'height_px': 306},
        'small': {'width_px': 594, 'height_px': 80},
        'restored': {'width_px': 594, 'height_px': 277}}
    client = helper.Client(args.connect)
    mutate = lambda name, params=None: client.ask(name, params, True)
    sizes, views = {}, {}

    def settled(name=None):
        if name and not args.connect:
            mutate('view.viewport.set', replay[name])
        deadline, previous, stable = time.monotonic()+8, None, time.monotonic()
        while time.monotonic() < deadline:
            state = client.ask('project.inspect')
            stamp = (state['command_revision'], state['viewport'], state['view'])
            if stamp != previous:
                stable = time.monotonic()
            elif state['viewport']['registered'] and time.monotonic()-stable > .3:
                if name:
                    sizes[name] = {k: state['viewport'][k] for k in ('width_px', 'height_px')}
                    views[name] = state['view']
                return state
            previous = stamp
            time.sleep(.02)
        raise AssertionError('layout did not settle')

    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh blank session'
        discovery = discovery_summary(client.ask('capabilities.list',
            {'command': 'workspace.layout.set'}), args.discovery_output)
        mutate('example.load', {'id': 'nmos'})
        mutate('simulation.pause')
        mutate('simulation.restart')
        client.ask('simulation.wait')
        default = settled('default')
        changed = mutate('workspace.layout.set', {'schematic_fraction': .8})
        assert helper.invariant(changed) == helper.invariant(default)
        assert changed['view'] == default['view']
        large = settled('large')
        assert sizes['large']['height_px'] > sizes['default']['height_px']
        assert large['view']['zoom'] >= default['view']['zoom']
        mutate('view.pan', {'dx': 20, 'dy': 0})
        manual = client.ask('project.inspect')['view']
        mutate('workspace.layout.set', {'schematic_fraction': .4})
        small = settled('small')
        assert small['view'] == manual and not small['viewport']['following']
        assert sizes['small']['height_px'] < sizes['large']['height_px']
        for invalid in ({}, {'schematic_fraction': None}, {'schematic_fraction': .349},
                        {'schematic_fraction': .851}, {'schematic_fraction': '0.8'}):
            client.ask('workspace.layout.set', invalid, True, expect='invalid-params')
        assert client.ask('project.inspect')['workspace_layout']['schematic_fraction'] == .4
        mutate('scope.visibility.set', {'visible': False})
        mutate('workspace.layout.set', {'schematic_fraction': .75})
        mutate('scope.visibility.set', {'visible': True})
        restored = settled('restored')
        assert restored['workspace_layout']['schematic_fraction'] == .75
        assert restored['view'] == manual
        admitted = mutate('simulation.verify', {'duration_s': '20 ms', 'sample_step_s': '1 ms'})
        identity, deadline = admitted['last_operation'], time.monotonic()+20
        while True:
            operation = client.ask('operation.inspect', {'id': identity})['operation']
            if operation['status'] == 'completed':
                break
            assert operation['status'] not in ('failed', 'canceled') and time.monotonic() < deadline
            time.sleep(.01)
        mutate('operation.select', {'id': identity})
        mutate('recording.view.set', {'mode': 'recorded'})
        recorded = mutate('recording.cursor.set', {'cursor': 'a', 'time_s': '10 ms'})
        changed = mutate('workspace.layout.set', {'schematic_fraction': .6})
        assert recorded['recording'] == changed['recording']
        assert recorded['transient'] == changed['transient']
        with tempfile.TemporaryDirectory(prefix='elab-layout-') as directory:
            path = str(Path(directory)/'layout.elab')
            mutate('project.save', {'path': path})
            mutate('workspace.layout.set', {'schematic_fraction': .7})
            opened = mutate('project.load', {'path': path})
            assert opened['workspace_layout']['schematic_fraction'] == .7
            second = helper.Client(None)
            try:
                other = second.ask('project.load', {'path': path}, True)
                assert other['workspace_layout']['schematic_fraction'] == .65
            finally:
                second.close()
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      discovery=discovery, sizes=sizes, views=views,
                      checks=['native allocation/headless size replay', 'ratio preserves physics/document/history/view before allocation',
                              'following refits on measured resize', 'manual view survives divider resize',
                              'invalid fractions refuse', 'collapse/expand retains accepted fraction',
                              'recorded cursor/live state preserved', 'layout remains session-only across file open'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as out:
            out.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: report[k] for k in ('passed', 'transport', 'sizes', 'views')}, indent=2))
    finally:
        if args.connect:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()

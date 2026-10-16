#!/usr/bin/env python3
"""Exercise the immutable example catalog, protected current reset and origin files."""
import argparse
import importlib.util
import json
from pathlib import Path

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    try:
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        before = client.ask('project.inspect')
        catalog = client.ask('examples.list')
        expected_ids = {'led', 'rc', 'npn', 'switch-npn', 'blink', 'flasher',
                        'rectifier', 'led-fade', 'nmos', 'pmos', 'split-supply'}
        assert expected_ids == {e['id'] for e in catalog} and client.ask('project.inspect') == before
        tested = []
        for info in catalog:
            assert all(info[k] for k in ['question', 'expected', 'try', 'limitation'])
            client.ask('example.load', {'id': info['id'], 'discard_changes': True}, True)
            original = client.ask('simulation.wait')
            assert not original['paused'] and original['example']['info'] == info and not original['example']['modified']
            client.ask('simulation.pause', {}, True)
            part = next(p for p in original['components'] if p['kind'] == 'resistor')
            client.ask('component.move', {'id': part['id'], 'position': {'x': 900, 'y': 700}}, True)
            edited = client.ask('project.inspect')
            assert edited['example']['modified']
            client.ask('example.current.reset', {}, True, 'unsaved-changes')
            assert client.ask('project.inspect') == edited
            client.ask('example.current.reset', {'expected_example_id': info['id'],
                                                'expected_document_generation': edited['document']['generation'],
                                                'discard_changes': True}, True)
            reset = client.ask('simulation.wait')
            assert reset['example']['info']['id'] == info['id'] and not reset['example']['modified'] and reset['paused']
            assert reset['components'] == original['components'] and reset['wires'] == original['wires']
            assert reset['transient_settings'] == original['transient_settings']
            if 'transient' in reset:
                assert reset['transient']['sample']['time_s'] == 0 and not reset['transient']['stale']
            tested.append(info['id'])
        client.ask('example.load', {'id': 'rc', 'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'edited-guided-rc.elab')
        client.ask('project.save', {'path': path, 'overwrite': True}, True)
        saved = client.ask('project.inspect')
        assert saved['example']['modified'] and not saved['document']['dirty']
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['example']['info']['id'] == 'rc' and reopened['example']['modified'] and not reopened['document']['dirty']
        generation = saved['document']['generation']
        client.ask('example.load', {'id': 'npn'}, True)
        client.ask('simulation.wait')
        client.ask('example.current.reset', {'expected_example_id': 'rc', 'discard_changes': True}, True, 'conflict')
        client.ask('example.current.reset', {'expected_example_id': 'npn', 'expected_document_generation': generation,
                                            'discard_changes': True}, True, 'conflict')
        client.ask('project.load', {'path': path, 'discard_changes': True}, True)
        final = client.ask('simulation.wait')
        assert final['paused'] and not final['document']['dirty'] and final['example']['info']['id'] == 'rc'
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      templates=tested, checks=['read-only immutable guidance catalog', 'all shipped examples open running',
                                              'edited circuit labeled against template expectation', 'unsaved reset refuses without mutation',
                                              'current reset restores exact template and fresh paused startup',
                                              'saved edited origin survives fresh reopen without trusted readings',
                                              'stale example/document reset guards', 'owned project ends clean and paused'],
                      limitations=['template guidance describes expected template behavior, not guaranteed behavior of an edited circuit',
                                   'origin is optional version-4 metadata; older readers lacking it refuse it',
                                   'engineering guide/reset checks do not establish novice or accessibility qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()

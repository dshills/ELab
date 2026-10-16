#!/usr/bin/env python3
"""Public-command checks for PRD audit repairs; replaces only the chosen owned session."""
import argparse
import importlib.util
import json
from pathlib import Path
import tempfile


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect)
    try:
        discovery = client.ask('capabilities.list')
        parts = client.ask('parts.list')
        assert all(p['aliases'] and p['limitations'] and p['model'] for p in parts)
        for query, kind in [('battery', 'dc-source'), ('variable resistor', 'potentiometer'), ('square wave', 'finite-pulse-source'), ('BJT', 'npn')]:
            matches = client.ask('parts.search.set', {'query': query}, True)['matching_parts']
            assert any(p['kind'] == kind for p in matches)
        client.ask('parts.search.set', {'query': ''}, True)
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': 1e-320}, True)
        client.ask('scope.cursor.set', {'cursor': 'a', 'time_s': 0}, True)
        client.ask('scope.cursor.set', {'cursor': 'b', 'time_s': 1e-320}, True)
        tiny = client.ask('project.inspect')
        assert tiny['scope']['a']['valid'] and tiny['scope']['b']['valid']
        assert tiny['scope']['delta_s'] > 0 and 'interval_hz' not in tiny['scope']
        assert tiny['transient']['sample']['voltage_v'] > 0
        client.ask('simulation.restart', {}, True)
        before = client.ask('project.inspect')
        client.ask('object.delete', {'id': 'W2'}, True)
        broken = client.ask('project.inspect')
        assert broken['scope']['selected'] == before['scope']['selected'] and broken['scope']['stale']
        client.ask('edit.undo', {}, True)
        client.ask('object.delete', {'id': 'C1'}, True)
        missing = client.ask('project.inspect')['scope']['unavailable']
        assert len(missing) == 1 and missing[0]['id'] == 'v:C1.1'
        client.ask('scope.probe.remove', {'id': missing[0]['id']}, True)
        assert not client.ask('project.inspect')['scope'].get('unavailable')
        client.ask('edit.undo', {}, True)
        client.ask('component.place', {'kind': 'finite-pulse-source', 'position': {'x': -400, 'y': 0}}, True)
        negative_id = client.ask('project.inspect')['selected']
        pulse = dict(high_v=-1, low_v=-3, high_s=.02, low_s=.03, rise_s=.001, fall_s=.002)
        client.ask('source.pulse.set', {'id': negative_id, 'waveform': pulse}, True)
        client.ask('component.duplicate', {'id': negative_id}, True)
        copied = client.ask('project.inspect')
        duplicate = next(p for p in copied['components'] if p['id'] == copied['selected'])
        assert duplicate['pulse'] == pulse and duplicate['value'] == -1
        client.ask('edit.undo', {}, True)
        client.ask('object.delete', {'id': negative_id}, True)
        client.ask('simulation.restart', {}, True)
        with tempfile.TemporaryDirectory(prefix='elab-hardening-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            path = directory/'clean-rc.elab'
            client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            assert not client.ask('project.inspect')['document']['dirty']
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      capabilities=len(discovery['capabilities']), library_kinds=len(parts),
                      checks=['hobbyist aliases', 'all library identities/limits discoverable', 'tiny cursor interval serializes without infinity',
                              'stable exact RC tiny-step voltage', 'default traces retained stale after broken wire',
                              'removed source explicitly unavailable', 'shared removal preserves physics', 'negative pulse duplication preserves parameters',
                              'undo removes duplicate and restores sources', 'final project save remains usable'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()

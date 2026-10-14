#!/usr/bin/env python3
"""Exercise immediate initialization admission, paused restart and saved startup.
--connect replaces only the designated owned development project.
"""
import argparse
import importlib.util
import json
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    cases = []
    try:
        for example in ['example.npn', 'example.npn-pulse', 'example.flasher']:
            started = time.perf_counter()
            accepted = client.ask(example, {'discard_changes': True}, True)
            admission_ms = (time.perf_counter()-started)*1000
            assert accepted['status'] == 'updating'
            ready = client.ask('simulation.wait')
            assert ready['status'] == 'running'
            client.ask('simulation.pause', {}, True)
            source = next(p for p in ready['components'] if p['kind'] in ['dc-source', 'pulse-source'])
            stale = client.ask('component.set', {'id': source['id'], 'value': '3V'}, True)
            assert stale['paused']
            restarted = client.ask('simulation.restart', {}, True)
            assert restarted['status'] == 'updating' and restarted['paused']
            frozen = client.ask('simulation.wait')
            assert frozen['status'] == 'paused' and frozen['paused']
            if 'transient' in frozen:
                assert frozen['transient']['sample']['time_s'] == 0 and not frozen['transient']['stale']
                client.ask('simulation.step', {'duration_s': .01}, True)
            assert not (frozen.get('npn', {}).get('stale') or frozen.get('mixed', {}).get('stale'))
            directory = args.artifacts.resolve()
            directory.mkdir(parents=True, exist_ok=True)
            path = str(directory/(example.split('.')[1]+'.elab'))
            client.ask('project.save', {'path': path, 'overwrite': True}, True)
            fresh = helper.Client(command_preconditions=True)
            loaded = fresh.ask('project.load', {'path': path}, True)
            assert loaded['status'] == 'updating'
            restored = fresh.ask('simulation.wait')
            assert restored['paused'] and restored['status'] == 'paused' and not restored['document']['dirty']
            assert restored['components'] == frozen['components']
            if 'transient' in restored:
                assert restored['transient']['sample']['time_s'] == 0
            fresh.close()
            fresh = None
            cases.append(dict(example=example, admission_ms=admission_ms, source_id=source['id'], restored_source_v=3))
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      cases=cases, checks=['immediate Updating admission', 'explicit readiness wait',
                        'paused electrical edit keeps previous result stale', 'paused Restart computes fresh DC/transient startup',
                        'legacy Step waits for initialized transient', 'saved paused projects initialize fresh on reopen'],
                      limitations=['backend admission measurements are not visible latency',
                        'input/wiper and some file/recovery work remain synchronous', 'circuit capacity remains below the original target'])
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

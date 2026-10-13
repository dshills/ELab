#!/usr/bin/env python3
"""Save/reopen recorded experiments in a fresh process or an owned desktop session.
--connect replaces that session's schematic with a paused RC example.
"""
import argparse
import importlib.util
import json
from pathlib import Path
import tempfile
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    first = helper.Client(args.connect)
    second = None
    try:
        catalog = first.ask('capabilities.list')
        first.ask('example.rc', {'discard_changes': True}, True)
        first.ask('simulation.pause', {}, True)
        first.ask('simulation.restart', {}, True)
        first.ask('simulation.step', {'duration_s': .03}, True)
        with tempfile.TemporaryDirectory(prefix='elab-project-recordings-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            path = directory / 'experiment.elab'
            first.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            accepted = first.ask('simulation.verify', {'duration_s': .02, 'probe_ids': ['v:C1', 'i:C1']}, True)
            identifier = accepted['last_operation']
            deadline = time.monotonic() + 5
            while time.monotonic() < deadline:
                captured = first.ask('operation.inspect', {'id': identifier})
                if captured['operation']['status'] == 'completed':
                    break
                assert captured['operation']['status'] not in ['failed', 'canceled']
                time.sleep(.01)
            assert captured['operation']['status'] == 'completed'
            dirty = first.ask('project.inspect')['document']
            assert dirty['recordings_dirty'] and not dirty['schematic_dirty'] and dirty['dirty']
            first.ask('project.clear', {}, True, expect='unsaved-changes')
            first.ask('recording.view.set', {'mode': 'recorded'}, True)
            original_rows = first.ask('operation.samples', {'id': identifier, 'limit': 100})['rows']
            first.ask('project.save', {}, True)
            encoded = json.loads(path.read_text())
            assert encoded['version'] == 4 and encoded['paused']
            entry = encoded['recordings']['entries'][0]
            assert entry['data']['metadata'] == captured['metadata'] and entry['data']['rows'] == original_rows
            assert all(len(digest) == 64 for digest in entry['model_sha256'].values())
            assert str(directory) not in path.read_text()
            # A new headless process proves portability without session IDs/history.
            second = helper.Client()
            second.ask('project.load', {'path': str(path)}, True)
            fresh = second.ask('simulation.wait')
            assert fresh['paused'] and fresh['transient']['sample']['time_s'] == 0
            assert not fresh['document']['dirty'] and fresh['document']['recording_count'] == 1
            restored_id = fresh['recording']['id']
            assert fresh['recording']['viewing'] and not fresh['recording']['playing']
            assert second.ask('operation.inspect', {'id': restored_id})['metadata'] == captured['metadata']
            assert second.ask('operation.samples', {'id': restored_id, 'limit': 100})['rows'] == original_rows
            assert not second.ask('operation.inspect', {'id': restored_id})['operation']['source_unchanged']
            second.ask('recording.keep.set', {'id': restored_id, 'keep': False}, True)
            assert second.ask('project.inspect')['document']['recordings_dirty']
            second.ask('recording.keep.set', {'id': restored_id, 'keep': True}, True)
            assert not second.ask('project.inspect')['document']['dirty']
            # The original desktop/headless session gets the same fresh startup.
            first.ask('project.load', {'path': str(path)}, True)
            loaded = first.ask('simulation.wait')
            assert loaded['paused'] and loaded['transient']['sample']['time_s'] == 0
            assert loaded['recording']['sample_count'] == len(original_rows)
            tampered = directory / 'tampered-model.elab'
            broken = json.loads(path.read_text())
            broken['recordings']['entries'][0]['model_sha256']['ideal-resistor.v1'] = 'tampered'
            tampered.write_text(json.dumps(broken))
            before_bad = first.ask('project.inspect')
            first.ask('project.load', {'path': str(tampered)}, True, expect='invalid-document')
            after_bad = first.ask('project.inspect')
            assert after_bad['document'] == before_bad['document'] and after_bad['recording'] == before_bad['recording']
            report = {
                'passed': True, 'transport': 'packaged-native-private-socket' if args.connect else 'headless-stdio',
                'capabilities': len(catalog['capabilities']), 'project_version': 4,
                'embedded_samples': len(original_rows), 'live_restart_time_s': loaded['transient']['sample']['time_s'],
                'saved_pause_restored': loaded['paused'], 'checks': [
                    'completed recording makes saved project dirty', 'record-only unsaved guard',
                    'embedded full captured metadata/rows', 'canonical dependency hashes',
                    'no portable host-path leakage', 'fresh-process record portability',
                    'fresh paused live startup', 'explicit recorded viewer',
                    'include/exclude membership restores clean save point',
                    'tampered dependency refused without replacing current work'],
            }
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(report, indent=2))
    finally:
        first.close()
        if second:
            second.close()


if __name__ == '__main__':
    main()

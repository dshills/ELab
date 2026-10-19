#!/usr/bin/env python3
"""Replay the preserved source/layout proof without running apps or writing files."""
import gzip
import hashlib
import json
from pathlib import Path
import re
import struct
import sys

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def load(name):
    data = (HERE/name).read_bytes()
    return json.loads(gzip.decompress(data) if name.endswith('.gz') else data)


def geometry(name):
    text = (HERE/name).read_text()
    pattern = r'scene=\(0,0\)-\((\d+),(\d+)\) button=\((\d+),(\d+)\)-\((\d+),(\d+)\) visible=true'
    return [tuple(map(int, match)) for match in re.findall(pattern, text)]


def verify():
    assert sys.flags.optimize == 0, 'Run without Python optimization'
    manifest = load('manifest.json')
    for name, entry in manifest['files'].items():
        relative = Path(name)
        assert not relative.is_absolute() and '..' not in relative.parts
        path = HERE/relative
        assert path.is_file() and not path.is_symlink(), name
        assert path.stat().st_size == entry['bytes'] and sha(path) == entry['sha256'], name
    initial_manifest = load('diagnosis/initial-manifest.json')
    patch = gzip.decompress((HERE/'source.diff.gz').read_bytes())
    assert hashlib.sha256(patch).hexdigest() == initial_manifest['files']['source.diff']['sha256']
    check_log = gzip.decompress((HERE/'diagnosis/raw-patch-whitespace-check.txt.gz').read_bytes()).decode()
    assert 'source.diff:' in check_log and 'trailing whitespace.' in check_log
    base = load('base-release.json.gz')
    assert base['version'] == '0.158.0' and base['build'] == '175'
    old, new = base['quality']['source_sha256'], load('quality/source-pins.json')['source_sha256']
    assert len(old) == len(new) == 650 and old.keys() == new.keys()
    changed = [n for n in old if old[n] != new[n]]
    assert changed == ['desktop/button_context_offscreen_test.go', 'desktop/shutdown.go', 'desktop/slider_invalidation_offscreen_test.go']
    for name in changed:
        for prefix, pins in [('original', old), ('final', new)]:
            assert sha(HERE/prefix/(Path(name).name+'.txt')) == pins[name]
    for name in ['original/shutdown.go.txt', 'final/shutdown.go.txt']:
        assert '"Close window anyway", "app.close.acknowledge", workbench.IDParams{ID: work.ID}' in (HERE/name).read_text()
    original, fixed = geometry('original/final-regression.txt'), geometry('final/focused-race.txt')
    assert len(original) == len(fixed) == 3
    assert [g[:2] for g in original] == [g[:2] for g in fixed] == [(1040,720),(1040,720),(1280,800)]
    assert [g[5]-g[3] for g in original] == [6,6,49]
    assert [g[5]-g[3] for g in fixed] == [49,49,49]
    for width, height, left, top, right, bottom in fixed:
        assert 0 <= left < right <= width and 0 <= top < bottom <= height
    original_log = (HERE/'original/final-regression.txt').read_text()
    final_log = (HERE/'final/focused-race.txt').read_text()
    for theme in ['light','dark']:
        assert '--- FAIL: TestMountedButtonOwnerShutdownLongErrorsKeepAcknowledgementVisible/'+theme+'-1040x720' in original_log
    assert '--- PASS: TestMountedButtonOwnerShutdownLongErrorsKeepAcknowledgementVisible/light-1280x800' in original_log
    assert '\nFAIL' in original_log and '\nPASS\n' in final_log and '\nFAIL' not in final_log
    assert final_log.count('error scroll: (32,419)-(522,623) -> (32,186)-(522,623); footer=(32,639)-(208,688)') == 2
    for name, size in [('light-1040x720',(1040,720)), ('dark-1040x720',(1040,720)), ('light-1280x800',(1280,800))]:
        data = (HERE/'final'/(name+'.png')).read_bytes()
        assert data[:8] == b'\x89PNG\r\n\x1a\n' and data[12:16] == b'IHDR'
        assert struct.unpack('>II', data[16:24]) == size
    differences = load('native-ledger-diff.json')
    assert len(differences) == 2
    assert differences[0] == dict(path='/source_sha256/desktop/shutdown.go', before=old['desktop/shutdown.go'], after=new['desktop/shutdown.go'])
    assert differences[1] == dict(path='/sites/320/line', before=15, after=22)
    before, after = load('original/native-command-inventory.json'), load('final/native-command-inventory.json')
    before['source_sha256']['desktop/shutdown.go'] = after['source_sha256']['desktop/shutdown.go']
    before['sites'][320]['line'] = 22
    assert before == after
    assert len(after['source_sha256']) == 72 and len(after['sites']) == 397 and len(after['public_commands']) == 164
    status = load('quality/status.json')
    assert status['exit_code'] == 0 and status['elapsed_seconds'] > 0
    quality_log = (HERE/'quality/full-quality.txt').read_text()
    assert '\nFAIL' not in quality_log and 'Ran 54 tests' in quality_log
    checkpoint = load('checkpoint.json')
    assert checkpoint['source_repair_qualified'] and not checkpoint['replacement_package_qualified']
    assert not checkpoint['canonical_promoted'] and not checkpoint['full_mvp_complete']
    assert sha(HERE/'manifest.json') == checkpoint['manifest_sha256']
    for name, digest in checkpoint['preserved'].items():
        assert sha(ROOT/name) == digest, name
    return dict(passed=True, finding='SHUTDOWN-CUTOFF-01', old_minimum_button_height=6,
                fixed_button_height=49, final_scenarios=3, full_quality_exit=0,
                replacement_packaging='separate gate', physical_native_input=False)


if __name__ == '__main__':
    print(json.dumps(verify(), indent=2))

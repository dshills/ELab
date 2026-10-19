"""Read-only verification of the recorded timing source checkpoint."""
import hashlib
import json
from pathlib import Path
import re
import stat

ROOT = Path(__file__).resolve().parents[3]
BASE = Path(__file__).parent


def read(name):
    return json.loads((BASE/name).read_text())


def main():
    checkpoint = read('checkpoint.json')
    for name,expected in checkpoint['evidence_sha256'].items():
        assert hashlib.sha256((BASE/name).read_bytes()).hexdigest()==expected,name
    identity = read('source-identity.json')
    for name,expected in identity['production_and_test_sha256'].items():
        assert hashlib.sha256((ROOT/name).read_bytes()).hexdigest()==expected,name
    raw = (ROOT/'research/13-native-command-inventory.json').read_bytes()
    assert hashlib.sha256(raw).hexdigest()==identity['native_inventory_sha256']
    assert raw==(BASE/'inventory.json').read_bytes()
    current,previous = json.loads(raw),read('previous-inventory.json')
    assert len(current['source_sha256'])==69 and len(current['sites'])==392
    for name,digest in current['source_sha256'].items():
        assert hashlib.sha256((ROOT/name).read_bytes()).hexdigest()==digest,name
    for key in ['public_commands','native_alias_targets','commands_with_source_routes','commands_without_source_routes','issues']:
        assert current[key]==previous[key],key
    assert len(current['public_commands'])==162 and len(current['commands_with_source_routes'])==132
    for name in ['main.go','recording_timing.go']:
        assert hashlib.sha256((BASE/'original-overlay'/(name+'.txt')).read_bytes()).hexdigest()==previous['source_sha256']['desktop/'+name]
    before = (BASE/'original-final-regression.txt').read_text()
    assert 'one Next click failed after private threshold' in before and 'invalid threshold advanced page or cleared validation' in before
    assert before.count('--- FAIL: TestMountedRecordingTiming')==2
    after = (BASE/'focused-complete.txt').read_text()
    for name in ['TestMountedRecordingTimingTypedThresholdThenNextMatchesAgent','TestMountedRecordingTimingInvalidThresholdDoesNotAdvancePage','TestRecordedTimingPageWaitsForAllFieldsAndRunsOnce','TestRecordedTimingPageRefusesInterveningWork','TestRecordedTimingPageSettlementFailurePreservesState','TestRecordedTimingPagePreviousUsesAcceptedFieldSettings','TestRecordedTimingPageStopsWhenEditedThresholdHasNoMoreArrivals']:
        assert '--- PASS: '+name in after,name
    assert 'FAIL' not in after and read('quality.json')['passed'] and read('quality.json')['exit_code']==0
    assert 'FAIL' not in (BASE/'quality.txt').read_text()
    assert 'RecordingTiming|' in (ROOT/'scripts/check.sh').read_text()
    assert read('native-smoke.json')['passed'] and read('native-smoke.json')['exit_code']==0
    preservation = read('preservation.json')
    assert preservation['passed'] and len(preservation['review_manifests'])==23
    for name,record in preservation['review_manifests'].items():
        folder = ROOT/'build'/name
        path = folder/'RELEASE_MANIFEST.json'
        assert hashlib.sha256(path.read_bytes()).hexdigest()==record['manifest_sha256']
        files = json.loads(path.read_text())['files']
        assert len(files)==record['entries']
        assert {str(p.relative_to(folder)) for p in folder.rglob('*') if p.is_file()}==set(files)|{'RELEASE_MANIFEST.json'}
        for relative,expected in files.items():
            p = folder/relative
            assert not p.is_symlink() and hashlib.sha256(p.read_bytes()).hexdigest()==expected['sha256']
            assert oct(stat.S_IMODE(p.stat().st_mode))==expected['mode']
    assert hashlib.sha256((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest()==preservation['canonical_native_sha256']
    assert hashlib.sha256((ROOT/'third_party/vellum/sdk.lock.json').read_bytes()).hexdigest()==preservation['sdk_lock_sha256']
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)',(BASE/'README.md').read_text()):
        assert (BASE/target.strip('<>')).exists(),target
    print(json.dumps(dict(passed=True,original_source_reproduces_both_failures=True,focused_and_full_quality_passed=True,native_sources=69,sites=392,public_commands=162,preserved_reviews=23,clean_replacement_qualified=False,evidence_files=len(checkpoint['evidence_sha256']))))


if __name__ == '__main__':
    main()

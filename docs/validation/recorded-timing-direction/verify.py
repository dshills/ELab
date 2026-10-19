"""Read-only verification of source repair and original native failure scopes."""
import gzip
import hashlib
import json
from pathlib import Path
import re
import stat

ROOT=Path(__file__).resolve().parents[3]
BASE=Path(__file__).parent

def read(name):
    data=(BASE/name).read_bytes()
    return json.loads(gzip.decompress(data) if name.endswith('.gz') else data)
def digest(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def main():
    checkpoint=read('checkpoint.json')
    for name,expected in checkpoint['evidence_sha256'].items():assert digest(BASE/name)==expected,name
    identity=read('source-identity.json')
    for name,expected in identity['production_and_test_sha256'].items():
        assert digest(ROOT/name)==expected,name
        assert digest(BASE/'source'/(Path(name).name+'.txt'))==expected,name
    raw=(ROOT/'research/13-native-command-inventory.json').read_bytes()
    assert hashlib.sha256(raw).hexdigest()==identity['native_inventory_sha256'] and raw==gzip.decompress((BASE/'inventory.json.gz').read_bytes())
    current,previous=json.loads(raw),read('previous-inventory.json.gz')
    assert len(current['source_sha256'])==69 and len(current['sites'])==391
    for name,expected in current['source_sha256'].items():assert digest(ROOT/name)==expected,name
    for key in ['public_commands','native_alias_targets','commands_with_source_routes','commands_without_source_routes','issues']:assert current[key]==previous[key],key
    for name in ['recorded_timing_actions.go','recording_timing.go']:assert digest(BASE/'source-original'/(name+'.txt'))==previous['source_sha256']['desktop/'+name]
    before=gzip.decompress((BASE/'original-regression.txt.gz').read_bytes()).decode()
    assert before.count('--- FAIL: TestMountedRecordingTimingDirection')==2 and 'direction lost private threshold: level=2.5 V' in before and 'direction accepted invalid threshold or hid its error' in before
    focused=(BASE/'focused-final.txt').read_text();assert 'FAIL' not in focused
    for name in ['TestMountedRecordingTimingDirectionCommitsPrivateThreshold','TestMountedRecordingTimingDirectionRefusesInvalidThresholdAndCorrects','TestNativeRecordedTimingDirectionWaitsForFieldsAndRunsOnce','TestNativeRecordedTimingDirectionRefusesInterveningWork','TestNativeRecordedTimingDirectionSettlementFailurePreservesState','TestMountedRecordingTimingTypedThresholdThenNextMatchesAgent','TestMountedRecordingTimingInvalidThresholdDoesNotAdvancePage','TestNativeRecordedTimingMenuRefusesInterveningContext']:assert '--- PASS: '+name in focused,name
    for name in ['quality','native-smoke']:assert read(name+'.json')['passed'] and read(name+'.json')['exit_code']==0
    assert 'FAIL' not in (BASE/'quality.txt').read_text()
    assert not read('native-smoke-first-attempt/native-smoke.json')['passed']
    assert 'native text client is stale' in (BASE/'native-smoke-first-attempt/native-smoke.txt').read_text()
    for record in read('compression.json')['files']:
        assert record['lossless'] and hashlib.sha256(gzip.decompress((BASE/record['path']).read_bytes())).hexdigest()==record['original_sha256']
    native=json.loads(gzip.decompress((BASE/'native-original/report.json.gz').read_bytes()))
    assert not native['passed'] and native['normal_closure'] and native['exit_code']==native['final_host_exit_code']==0
    assert native['capture_metadata_pages_unchanged'] and native['saved_fields_match_except_view_and_independent_capture_provenance']
    stages={stage['name']:stage for stage in native['stages']}
    for name in ['direction-chosen','pointer-direction']:
        stage=stages[name];assert not stage['passed'] and stage['state']['recording']['timing']['settings']['level']=='2.5 V'
    assert stages['direction-chosen']['reference']['recording']['timing']['settings']['level']=='3 V'
    assert stages['pointer-direction']['reference']['recording']['timing']['settings']['level']=='4 V'
    ui=read('native-original/ui-conclusion.json');assert not ui['passed'] and ui['native_interaction_completed'] and ui['pointer_first_menu_open_observed']
    cleanup=read('native-original/cleanup.json');assert cleanup['passed'] and cleanup['original_process_absent'] and cleanup['normal_closure'] and cleanup['restored_from_verified_backups']
    for relative,expected in cleanup['restored_sha256'].items():assert digest(ROOT/'build/native-files-156-ui/ELab-value-recovery.app'/relative)==expected
    interrupted=read('approval-interrupted/cleanup.json');assert interrupted['passed'] and interrupted['native_host_exit_code']==-15 and not interrupted['normal_closure']
    for label in ['Native','Agent']:
        document=json.loads(gzip.decompress((BASE/'native-original'/(label+' timing.elab.gz')).read_bytes()))
        assert len(document['recordings']['entries'][0]['data']['rows'])==521
    for picture in (BASE/'native-original').glob('*.jpg'):assert picture.read_bytes().startswith(b'\xff\xd8'),picture
    preserved=read('preservation.json');assert preserved['passed'] and preserved['qualified_reviews']==24 and len(preserved['review_manifests'])==25
    for name,record in preserved['review_manifests'].items():
        folder=ROOT/'build'/name;manifest_name=record.get('manifest_name','RELEASE_MANIFEST.json');path=folder/manifest_name;assert digest(path)==record['manifest_sha256'];files=json.loads(path.read_text())['files'];assert len(files)==record['entries']
        assert {str(p.relative_to(folder)) for p in folder.rglob('*') if p.is_file()}==set(files)|{manifest_name}
        for relative,expected in files.items():
            path=folder/relative;assert not path.is_symlink() and digest(path)==expected['sha256'];assert oct(stat.S_IMODE(path.stat().st_mode))==expected['mode']
    assert digest(ROOT/'build/ELab.app/Contents/MacOS/ELab')==preserved['canonical_native_sha256']
    assert digest(ROOT/'third_party/vellum/sdk.lock.json')==preserved['sdk_lock_sha256']
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)',(BASE/'README.md').read_text()):assert (BASE/target.strip('<>')).exists(),target
    print(json.dumps(dict(passed=True,original_native_failure_reproduced=True,original_source_reproduces_valid_and_invalid_failures=True,focused_and_full_quality_passed=True,general_development_native_smoke_passed=True,fixed_physical_input_replay_completed=False,clean_replacement_qualified=False,preserved_qualified_reviews=24,canonical_promoted=False)))

if __name__=='__main__':main()

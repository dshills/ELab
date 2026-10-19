"""Read-only evidence verification; fixed native correction remains pending."""
import gzip,hashlib,json,re,stat
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3];BASE=Path(__file__).parent

def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def raw(name):
    data=(BASE/name).read_bytes()
    return gzip.decompress(data) if name.endswith('.gz') else data
def read(name):return json.loads(raw(name))
def main():
    cp=read('checkpoint.json')
    for name,expected in cp['evidence_sha256'].items():assert sha(BASE/name)==expected,name
    identity=read('source-identity.json')
    for name,expected in identity['production_and_test_sha256'].items():
        assert sha(ROOT/name)==expected,name
        assert sha(BASE/'source'/(Path(name).name+'.txt'))==expected,name
    ledger=(ROOT/'research/13-native-command-inventory.json').read_bytes()
    assert hashlib.sha256(ledger).hexdigest()==identity['native_inventory_sha256'] and ledger==raw('inventory.json.gz')
    current=json.loads(ledger);previous=read('previous-inventory.json.gz')
    assert len(current['source_sha256'])==69 and len(current['sites'])==391
    for name,expected in current['source_sha256'].items():assert sha(ROOT/name)==expected,name
    for key in ['public_commands','native_alias_targets','commands_with_source_routes','commands_without_source_routes','issues']:assert current[key]==previous[key],key
    assert sha(BASE/'source-original/recorded_timing_actions.go.txt')==previous['source_sha256']['desktop/recorded_timing_actions.go']
    before=raw('original-final-regression.txt.gz').decode();assert before.count('corrected field hid its refusal before the pending direction finished')==2 and '--- FAIL: TestNativeRecordedTimingCorrectionRetainsFeedbackUntilDirectionCompletes' in before
    focused=raw('focused-final.txt').decode();assert 'FAIL' not in focused
    for name in ['TestNativeRecordedTimingCorrectionRetainsFeedbackUntilDirectionCompletes','TestNativeRecordedTimingCorrectionDoesNotRetainOtherRequestHelp','TestNativeRecordedTimingCorrectionSettlementErrorReplacesHeldFeedback','TestNativeRecordedTimingDirectionRefusesInterveningWork','TestMountedRecordingTimingDirectionRefusesInvalidThresholdAndCorrects','TestNativeViewportRetainsValueRefusalAcrossItsOwnLayout']:
        assert '--- PASS: '+name in focused,name
    for name in ['quality','native-smoke']:assert read(name+'.json')['passed'] and read(name+'.json')['exit_code']==0
    assert 'FAIL' not in raw('quality.txt').decode()
    native=read('native-original/report.json.gz');stages={r['name']:r for r in native['stages']}
    assert not native['passed'] and native['normal_closure'] and native['exit_code']==native['final_host_exit_code']==0 and len(stages)==10
    assert all(stages[name]['passed'] for name in stages if name!='corrected-3.5V-rising')
    failed=stages['corrected-3.5V-rising'];assert not failed['passed']
    assert failed['state']['recording']['timing']['settings']['level']=='3.5 V' and failed['state']['recording']['timing']['settings']['direction']=='falling'
    assert failed['reference']['recording']['timing']['settings']['direction']=='rising'
    assert 'expected command revision 16; current command revision is 17' in raw('native-original/corrected-3.5V-rising.ax.txt.gz').decode()
    assert native['capture_metadata_pages_unchanged'] and native['saved_fields_match_except_view_and_independent_capture_provenance']
    conclusion=read('native-original/ui-conclusion.json');assert not conclusion['passed'] and conclusion['valid_AX_3V_rising_passed'] and conclusion['valid_pointer_4V_falling_passed'] and conclusion['invalid_threshold_refusal_passed'] and not conclusion['corrected_threshold_and_direction_first_attempt_passed']
    cleanup=read('native-original/cleanup.json');assert cleanup['passed'] and cleanup['normal_closure'] and cleanup['original_process_absent'] and cleanup['restored_from_verified_backups']
    for name,expected in cleanup['restored_sha256'].items():assert sha(ROOT/'build/native-files-156-ui/ELab-value-recovery.app'/name)==expected
    for label in ['Native','Agent']:assert len(read('native-original/'+label+' timing.elab.gz')['recordings']['entries'][0]['data']['rows'])==521
    for picture in (BASE/'native-original').glob('*.jpg'):assert picture.read_bytes().startswith(b'\xff\xd8'),picture
    candidate=read('candidate/build-result.json');assert candidate['clean_build_vcs'] and candidate['source_commit']==identity['source_base']
    manifest=read('candidate/CANDIDATE_MANIFEST.json');assert manifest['source_build_clean'] and manifest['source_record_commit']==candidate['source_commit'] and manifest['native_sha256']==candidate['native_sha256'] and not manifest['canonical_promoted']
    prior=read('prior-source-proof.json');path=ROOT/prior['checkpoint'];assert sha(path)==prior['checkpoint_sha256']
    for name,expected in json.loads(path.read_text())['evidence_sha256'].items():assert sha(path.parent/name)==expected,name
    preservation=read('preservation.json');assert preservation['qualified_reviews']==24 and preservation['unqualified_candidates']==2 and len(preservation['review_manifests'])==26
    for name,record in preservation['review_manifests'].items():
        folder=ROOT/'build'/name;manifest_name=record.get('manifest_name','RELEASE_MANIFEST.json');p=folder/manifest_name;assert sha(p)==record['manifest_sha256'];files=json.loads(p.read_text())['files'];assert len(files)==record['entries']
        assert {str(p.relative_to(folder)) for p in folder.rglob('*') if p.is_file()}==set(files)|{manifest_name}
        for relative,expected in files.items():
            p=folder/relative;assert not p.is_symlink() and sha(p)==expected['sha256'] and oct(stat.S_IMODE(p.stat().st_mode))==expected['mode'],str(p)
    assert sha(ROOT/'build/ELab.app/Contents/MacOS/ELab')==preservation['canonical_native_sha256']
    assert sha(ROOT/'third_party/vellum/sdk.lock.json')==preservation['sdk_lock_sha256']
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)',raw('README.md').decode()):assert (BASE/target.strip('<>')).exists(),target
    print(json.dumps(dict(passed=True,clean_164_native_valid_cases_pass=True,clean_164_native_correction_failure_preserved=True,original_source_feedback_failure_reproduced=True,source_quality_and_general_native_smoke_passed=True,fixed_physical_correction_replay_completed=False,replacement_qualified=False,qualified_reviews_preserved=24,unqualified_candidates_preserved=2,canonical_promoted=False)))
if __name__=='__main__':main()

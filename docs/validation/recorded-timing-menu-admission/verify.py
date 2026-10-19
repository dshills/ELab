"""Read-only verification of the menu-callback evidence and unchanged review."""
import hashlib
import json
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[3]
BASE = Path(__file__).parent
sys.path.insert(0,str(ROOT/'scripts'))
from wiring_replay import validate_release


def main():
    checkpoint=json.loads((BASE/'checkpoint.json').read_text())
    for name,expected in checkpoint['evidence_sha256'].items():
        assert hashlib.sha256((BASE/name).read_bytes()).hexdigest()==expected,name
    identity=json.loads((BASE/'identity.json').read_text())
    assert hashlib.sha256((ROOT/identity['test_file']).read_bytes()).hexdigest()==identity['test_sha256']
    receipt=ROOT/identity['release_receipt']
    assert hashlib.sha256(receipt.read_bytes()).hexdigest()==identity['release_receipt_sha256']
    release=validate_release(ROOT,receipt)
    assert release['release']=='0.146.0 /163' and release['production_source_files_verified']==69 and release['runtime_inputs_verified']==344
    output=(BASE/'race.txt').read_text()
    assert 'FAIL' not in output
    for name in ['TestNativeRecordedTimingMenuFreshChoiceMatchesAgent','TestNativeRecordedTimingMenuRefusesInterveningContext']:
        assert '--- PASS: '+name in output
    for name in ['agent-settings','user-field','other-recording','new-document','menu-reopened','file-busy']:
        assert '--- PASS: TestNativeRecordedTimingMenuRefusesInterveningContext/'+name in output
    assert (BASE/'vet.txt').read_bytes()==b'' and all(c['passed'] and c['exit_code']==0 for c in identity['commands'])
    approval=json.loads((BASE/'approval-review.json').read_text())
    assert not approval['native_setup_executed'] and not approval['missing_user_authorization']
    assert [f['reason'] for f in approval['failures']]==['review_model_at_capacity','review_deadline_timeout','review_deadline_timeout']
    preservation=json.loads((BASE/'preservation.json').read_text())
    assert preservation['passed'] and not preservation['native_setup_directory_created']
    bundle=ROOT/'build/native-files-156-ui/ELab-value-recovery.app'
    for name,expected in preservation['disposable_bundle_unchanged'].items():
        assert hashlib.sha256((bundle/name).read_bytes()).hexdigest()==expected
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)',(BASE/'README.md').read_text()):
        assert (BASE/target.strip('<>')).exists(),target
    print(json.dumps(dict(passed=True,release=release['release'],source_callback_cases=7,race_and_vet_passed=True,native_interaction_completed=False,approval_review_unavailable=True,runtime_unchanged=True)))


if __name__=='__main__':
    main()

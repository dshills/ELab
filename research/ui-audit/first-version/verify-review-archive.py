"""Read-only verification of the /165 ZIP and retained extraction/runtime evidence."""
import gzip
import hashlib
import importlib.util
import json
from pathlib import Path
import sys
import zipfile

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'scripts'))
import package_review
from wiring_replay import validate_release


def read(path):
    return json.loads(gzip.decompress(path.read_bytes()) if path.suffix == '.gz' else path.read_text())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    proof = ROOT/'docs/validation/review-archive-165'
    checkpoint = read(proof/'checkpoint.json')
    assert checkpoint['passed'] and not checkpoint['canonical_promoted'] and not checkpoint['owner_acceptance']
    receipt_path = ROOT/'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json'
    identity = validate_release(ROOT, receipt_path)
    receipt = read(receipt_path)
    assert identity['receipt_sha256'] == checkpoint['review_receipt_sha256']
    assert identity['manifest_sha256'] == checkpoint['release_manifest_sha256']
    for name, expected in checkpoint['source_sha256'].items():
        assert sha(ROOT/name) == expected, name
    names = {str(path.relative_to(proof)) for path in proof.rglob('*') if path.is_file()}
    assert names == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert sha(proof/name) == expected, name
    packaging = read(proof/'package-report.json.gz')
    runtime = read(proof/'runtime-report.json.gz')
    archive = Path(checkpoint['archive'])
    assert sha(archive) == checkpoint['archive_sha256'] == packaging['archive_sha256'] == runtime['archive_sha256']
    assert archive.stat().st_size == checkpoint['archive_bytes'] == packaging['archive_bytes']
    assert archive.with_name(archive.name+'.sha256').read_text() == checkpoint['archive_sha256']+'  '+archive.name+'\n'
    review = Path(receipt['review_path'])
    expected = package_review.members(review)
    assert expected == packaging['expected_files'] and len(expected) == checkpoint['files_verified'] == 789
    package_review.verify_archive(archive, review.name, expected)
    assert packaging['passed'] and packaging['ditto_exit_code'] == 0
    assert packaging['source_review_unchanged'] and packaging['extraction_bytes_and_modes_match']
    assert runtime['passed'] and runtime['native_exit_code'] == runtime['final_native_exit_code'] == 0
    assert runtime['normal_native_closure'] and runtime['endpoint_removed'] and runtime['extracted_review_unchanged']
    assert runtime['native_sha256'] == receipt['native_sha256'] and runtime['agent_sha256'] == receipt['agent_sha256']
    assert runtime['example_count'] == 3 and runtime['fresh_agent_reopens'] == 6
    assert sha(proof/'runtime.sb') == runtime['same_denial_policy_sha256']
    assert runtime['qualified_runtime_report_sha256'] == sha(ROOT/'docs/validation/first-version-165-runtime.json')
    specification = importlib.util.spec_from_file_location('parity', ROOT/'scripts/first-version-parity.py')
    parity = importlib.util.module_from_spec(specification)
    specification.loader.exec_module(parity)
    assert set(runtime['cases']) == {'LED-2k.elab', 'LED-flasher.elab', 'RC-charge.elab'}
    with zipfile.ZipFile(archive) as zipped:
        for name, case in runtime['cases'].items():
            source = zipped.read(review.name+'/Examples/'+name)
            original = json.loads(source)
            assert set(case) == {'native', 'headless'}
            for surface, observed in case.items():
                assert observed['source_example_sha256'] == hashlib.sha256(source).hexdigest()
                saved_path = proof/(surface+' '+name+'.gz')
                saved_bytes = gzip.decompress(saved_path.read_bytes())
                assert hashlib.sha256(saved_bytes).hexdigest() == observed['saved_sha256']
                saved = json.loads(saved_bytes)
                state, reopened = observed['state'], observed['reopened']
                assert state['paused'] == reopened['paused'] == saved['paused'] == original['paused']
                assert not state['document']['dirty'] and not reopened['document']['dirty']
                for key in ['components', 'wires']:
                    assert state[key] == reopened[key] == saved['graph'][key] == original['graph'][key]
                assert state['nets'] == reopened['nets']
                for key in ['result', 'transient', 'mixed', 'npn', 'scope']:
                    parity.compare(state.get(key), reopened.get(key), name+'.'+key)
            a, b = case['native']['state'], case['headless']['state']
            for key in ['components', 'wires', 'nets', 'paused']:
                assert a[key] == b[key], (name, key)
            for key in ['result', 'transient', 'mixed', 'npn', 'scope']:
                parity.compare(a.get(key), b.get(key), name+'.'+key)
    failed = read(proof/'first-runtime-attempt/report.json')
    assert failed['passed'] is False and failed['final_native_exit_code'] == -15
    cleanup = read(proof/'cleanup.json')
    assert cleanup['passed'] and cleanup['normal_retry_closure'] and cleanup['successful_retry_endpoint_removed']
    assert cleanup['owned_processes'] == {str(r['native_pid']): 'absent' for r in [failed, runtime]}
    assert 'Ran 54 tests' in (proof/'python-tests.txt').read_text() and (proof/'python-tests.txt').read_text().strip().endswith('OK')
    print(json.dumps(dict(passed=True,release=identity['release'],archive_bytes=archive.stat().st_size,
                          files_verified=789,examples=3,fresh_agent_reopens=6,
                          clean_account_and_distribution_qualified=False,owner_acceptance=False)))


if __name__ == '__main__':
    main()

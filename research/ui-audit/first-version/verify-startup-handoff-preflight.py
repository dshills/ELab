"""Verify historical startup/signing preflight and the current README handoff."""
import gzip
import hashlib
import importlib.util
import json
from pathlib import Path
import plistlib
import sys
import tarfile

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'scripts'))
from wiring_replay import validate_release


def sha(data):
    return hashlib.sha256(data).hexdigest()


def read(path):
    return json.loads(path.read_text())


def main():
    proof = ROOT/'docs/validation/startup-handoff-preflight-165'
    checkpoint = read(proof/'checkpoint.json')
    receipt_path = ROOT/'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json'
    identity = validate_release(ROOT, receipt_path)
    receipt = read(receipt_path)
    assert checkpoint['passed'] and checkpoint['review_receipt_sha256'] == identity['receipt_sha256']
    assert checkpoint['native_sha256'] == receipt['native_sha256']
    names = {str(path.relative_to(proof)) for path in proof.rglob('*') if path.is_file()}
    assert names == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert sha((proof/name).read_bytes()) == expected, name
    before = gzip.decompress((proof/'README-before.md.gz').read_bytes()).decode()
    after_bytes = gzip.decompress((proof/'README-after.md.gz').read_bytes())
    assert sha(after_bytes) == checkpoint['root_readme_sha256']
    assert sha(gzip.decompress((proof/'review-verifier.py.gz').read_bytes())) == checkpoint['review_verifier_sha256']
    specification = importlib.util.spec_from_file_location('review', ROOT/'research/ui-audit/first-version/verify-recorded-timing-feedback-review.py')
    review = importlib.util.module_from_spec(specification)
    specification.loader.exec_module(review)
    refused = False
    try:
        review.verify_readme(receipt, before)
    except AssertionError as error:
        refused = True
        assert str(error) == read(proof/'readme-regression.json')['reason']
    assert refused
    review.verify_readme(receipt, after_bytes.decode())
    review.verify_readme(receipt, (ROOT/'README.md').read_text())
    assert 'build/.vellum-sdk-ade66eb/scripts/package-app.py' in before
    packaging = read(proof/'packaging-report.json')
    pin_path = ROOT/'third_party/vellum/sdk.lock.json'
    pin = read(pin_path)
    assert sha(pin_path.read_bytes()) == checkpoint['sdk_lock_sha256']
    assert packaging['passed'] and packaging['exit_code'] == 0 and packaging['sdk_reverified_after_packaging']
    assert packaging['sdk_source'] == pin['source_commit'] and packaging['verified_sdk'] == str(ROOT/pin['cache'])
    assert packaging['binary_sha256'] == receipt['native_sha256'] and packaging['development_version'] == '0.0.0 /1'
    info = plistlib.loads((proof/'packaged-Info.plist').read_bytes())
    manifest = read(proof/'packaged-APP_MANIFEST.json')
    assert info['CFBundleShortVersionString'] == '0.0.0' and info['CFBundleVersion'] == '1'
    assert manifest['private_build'] and manifest['source_binary_sha256'] == receipt['native_sha256']
    facts = read(proof/'startup-paths.json')
    assert facts['passed'] and sha((ROOT/pin['archive']).read_bytes()) == pin['sha256']
    with tarfile.open(ROOT/pin['archive'], 'r:gz') as archive:
        for name, expected in facts['source_sha256'].items():
            if name.startswith(pin['cache']+'/'):
                relative = name.removeprefix(pin['cache']+'/')
                data = archive.extractfile(pin['prefix']+'/'+relative).read()
            else:
                data = (ROOT/name).read_bytes()
            assert sha(data) == expected, name
    preflight = read(proof/'identity-preflight.json')
    query = preflight['signing_identity_query']
    assert query['exit_code'] == 0 and query['summary_available'] and query['valid_identity_count'] == 0
    assert not query['identity_names_or_fingerprints_stored']
    signature = preflight['app_signature']
    assert signature['exit_code'] == 0 and signature['fields']['Signature'] == 'adhoc'
    assert signature['fields']['TeamIdentifier'] == 'not set' and not signature['authority_present']
    assert preflight['native_sha256'] == receipt['native_sha256']
    assert checkpoint['separate_account_or_vm_needed'] and checkpoint['separate_account_or_vm_available'] is None
    assert not checkpoint['separate_session_access_provided']
    assert not checkpoint['ordinary_startup_qualified'] and not checkpoint['owner_acceptance'] and not checkpoint['canonical_promoted']
    print(json.dumps(dict(passed=True, release=identity['release'], old_readme_refused=True,
                          current_readme_passed=True, current_sdk_packaging_passed=True,
                          observed_valid_signing_identities=0, ordinary_startup_qualified=False)))


if __name__ == '__main__':
    main()

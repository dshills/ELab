"""Read-only verification for the exact signed /171 private review."""
from pathlib import Path
import gzip
import hashlib
import json
import plistlib
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'scripts'))
from package_review import members, verify_archive
from wiring_replay import validate_release


def read(path):
    data = path.read_bytes()
    return json.loads(gzip.decompress(data) if path.suffix == '.gz' else data)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    folder = Path(__file__).resolve().parent
    checkpoint = read(folder/'checkpoint.json')
    actual = {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
    assert actual == checkpoint['evidence_sha256']
    receipt_path = ROOT/'research/ui-audit/first-version/appearance-work-review-checkpoint.json'
    receipt = read(receipt_path)
    assert sha(receipt_path) == checkpoint['receipt_sha256']
    identity = validate_release(ROOT, receipt_path)
    assert identity['release'] == '0.154.0 /171' and identity['source_commit'] == 'ba295b9317d93f21d7f9862bdb3b6d4675d300c7'
    assert identity['runtime_inputs_verified'] == receipt['runtime_source_input_count'] == 356
    assert identity['production_source_files_verified'] == 71
    assert identity['manifest_file_entries_verified'] == receipt['manifest_file_entries'] == 309
    assert identity['manifest_sha256'] == checkpoint['manifest_sha256']
    review = Path(receipt['review_path'])
    assert (review/'Evidence/Acceptance.json').read_bytes() == receipt_path.read_bytes()
    info = plistlib.loads((review/'ELab.app/Contents/Info.plist').read_bytes())
    assert info['CFBundleIdentifier'] == 'com.dshills.elab' and info['CFBundleVersion'] == '171' and info['CFBundleShortVersionString'] == '0.154.0'
    manifest = read(review/'ELab.app/Contents/Resources/APP_MANIFEST.json')
    assert manifest['source_binary_sha256'] == receipt['source_native_sha256'] == receipt['signature']['source_input_sha256']
    assert manifest['signed_identity'] == '-' and sha(review/'ELab.app/Contents/MacOS/ELab') == receipt['signature']['packaged_sha256']
    subprocess.run(['/usr/bin/codesign', '--verify', '--deep', '--strict', str(review/'ELab.app')], capture_output=True, check=True)
    assert read(folder/'Signature-admission.json') == receipt['signature']['exact_artifact_admission']
    assert len(read(folder/'Signature-admission.json')['refusals']) == 5
    quality = read(ROOT/'docs/validation/appearance-work/quality.json')
    assert quality['passed'] and quality['environment'] == {'GOFLAGS': '-p=1'}
    assert all(sha(ROOT/name) == digest for name, digest in quality['source_sha256'].items())
    proof = ROOT/'docs/validation/appearance-work'
    source_checkpoint = read(proof/'checkpoint.json')
    assert sha(proof/'checkpoint.json') == receipt['source_proof_checkpoint_sha256'] == checkpoint['source_proof_checkpoint_sha256']
    assert all(sha(proof/name) == digest for name, digest in source_checkpoint['evidence_sha256'].items())
    rr = read(folder/'Runtime.json')
    ar = read(folder/'acceptance/report.json')
    assert rr['passed'] and ar['passed'] and rr['shared_geometry_unchanged'] and ar['shared_geometry_unchanged']
    assert ar['discovery_commands']['commands'] == ar['discovery_commands']['semantic_annotations'] == 164
    assert ar['discovery_commands']['appearance_job_checks']['passed']
    public = read(folder/'Public-report.json.gz')
    assert public['passed'] and len(public['adapters']) == 4
    exits = read(folder/'Process-exits.json.gz')
    shutdowns = read(folder/'Shutdown-responses.json.gz')
    assert len(exits) == 102 and len(shutdowns) == 112
    assert all(row['exit_code'] == 0 and not row['graceful_wait_timed_out'] and not row.get('close_error') and not row.get('wait_error') for row in exits.values())
    for row in shutdowns.values():
        state = row['response']['result']
        assert state['closed'] and state['shutdown']['status'] == 'completed' and state['shutdown']['workers_joined'] and state['shutdown']['exit_ready']
    archive = read(folder/'archive/package-report.json.gz')
    zip_path = ROOT/'build/ELab-0.154.0-171-review.zip'
    assert archive['passed'] and sha(zip_path) == archive['archive_sha256'] == checkpoint['archive_sha256']
    assert archive['files_verified'] == receipt['manifest_file_entries']+1
    verify_archive(zip_path, review.name, members(review))
    checksum = (ROOT/'build/ELab-0.154.0-171-review.zip.sha256').read_text()
    assert checksum == archive['archive_sha256']+'  '+zip_path.name+'\n'
    assert read(folder/'archive/example-report.json.gz')['passed'] and read(folder/'archive/signature-report.json.gz')['passed']
    preserved = read(folder/'preservation.json')
    for name, expected in preserved['preserved_bundles'].items():
        older = ROOT/'build'/('ELab-'+name+'-review')
        files = members(older)
        assert files.pop('RELEASE_MANIFEST.json')['sha256'] == expected['manifest_sha256']
        assert files == read(older/'RELEASE_MANIFEST.json')['files'] and len(files) == expected['files_verified']
    assert len(preserved['preserved_bundles']) == 30
    for path, key in [('ELab-0.130.0-147-review', 'original_147_candidate'),
                      ('ELab-0.147.0-164-review', 'failed_164_candidate'),
                      ('ELab-0.154.0-171-unsigned-candidate', 'unsigned_171_candidate')]:
        candidate = ROOT/'build'/path
        files = members(candidate)
        assert files.pop('CANDIDATE_MANIFEST.json')['sha256'] == preserved[key]['manifest_sha256']
        assert files == read(candidate/'CANDIDATE_MANIFEST.json')['files'] and len(files) == preserved[key]['files_verified']
    assert sha(ROOT/'build/ELab.app/Contents/MacOS/ELab') == receipt['canonical']['native_sha256']
    assert not receipt['canonical_promoted'] and receipt['owner_acceptance'].startswith('Pending:')
    assert receipt['p0_counts'] == dict(implemented=17, partial=10)
    print(dict(passed=True, release=identity['release'], manifest_entries=receipt['manifest_file_entries'],
               archive_files=archive['files_verified'], strict_signature_valid=True, preserved_reviews=30,
               owner_acceptance=False, canonical_promoted=False))


if __name__ == '__main__':
    main()

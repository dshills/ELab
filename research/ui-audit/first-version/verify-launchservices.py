"""Read-only verification of exact /165 LaunchServices smoke evidence."""
import gzip
import hashlib
import json
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'scripts'))
from wiring_replay import validate_release


def sha(data):
    return hashlib.sha256(data).hexdigest()


def stages(text):
    return [match.group(1) for line in text.splitlines()
            if (match := re.search(r'smoke: (.*)$', line))]


def main():
    proof = ROOT/'docs/validation/launchservices-165'
    checkpoint = json.loads((proof/'checkpoint.json').read_text())
    receipt_path = ROOT/'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json'
    identity = validate_release(ROOT, receipt_path)
    receipt = json.loads(receipt_path.read_text())
    assert checkpoint['passed'] and checkpoint['review_receipt_sha256'] == identity['receipt_sha256']
    assert checkpoint['release_manifest_sha256'] == identity['manifest_sha256']
    assert checkpoint['native_sha256'] == receipt['native_sha256']
    names = {str(path.relative_to(proof)) for path in proof.rglob('*') if path.is_file()}
    assert names == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert sha((proof/name).read_bytes()) == expected, name
    for name, expected in checkpoint['source_sha256'].items():
        assert sha((ROOT/name).read_bytes()) == expected, name
    archive = json.loads((ROOT/'docs/validation/review-archive-165/checkpoint.json').read_text())
    assert checkpoint['archive_sha256'] == archive['archive_sha256'] == sha(Path(archive['archive']).read_bytes())
    report = json.loads((proof/'report.json').read_text())
    assert report['passed'] and not report['forced_cleanup']
    assert report['archive_sha256'] == checkpoint['archive_sha256']
    assert report['native_sha256'] == receipt['native_sha256']
    assert report['launcher_exit_code'] == checkpoint['launcher_exit_code'] == 0
    assert report['native_process_exit_status'] is checkpoint['target_native_exit_status'] is None
    assert report['owned_native_process_absent'] and report['endpoint_removed'] and report['private_endpoint_observed']
    assert report['extracted_review_unchanged'] and report['native_smoke_success_event']
    assert len(report['owned_pids']) == 1 and set(report['observed_commands']) == {str(report['owned_pids'][0])}
    args = report['argv']
    assert args[:5] == ['/usr/bin/open', '-n', '-W', '-g', '-a']
    assert args[6:11:2] == ['--stdout', '--stderr', '--args']
    assert args[11:15] == ['--smoke', '--smoke-timeout', '2m', '--agent-endpoint'] and len(args) == 16
    binary = str(Path(args[5])/'Contents/MacOS/ELab')
    assert report['observed_commands'][str(report['owned_pids'][0])] == binary+' '+' '.join(args[11:])
    output = gzip.decompress((proof/'native-stdout.txt.gz').read_bytes()).decode()
    errors = gzip.decompress((proof/'native-stderr.txt.gz').read_bytes()).decode()
    events = [json.loads(line) for line in output.splitlines() if line.startswith('{"event":"elab.native.smoke"')]
    assert events == [dict(event='elab.native.smoke', passed=True)]
    reference = gzip.decompress((Path(receipt['review_path'])/'Evidence/Native-smoke.txt.gz').read_bytes())
    assert sha(reference) == checkpoint['reference_smoke_log_sha256']
    assert stages(errors) == report['stages'] == report['qualified_reference_stages'] == stages(reference.decode())
    assert len(report['stages']) == checkpoint['stage_journal_markers'] == 50
    assert not any(message in errors for message in
                   ['shutdown cleanup failed:', 'Recovery unavailable:', 'Appearance unavailable:', 'Application menus unavailable:'])
    assert not checkpoint['ordinary_startup_qualified'] and not checkpoint['clean_account_and_distribution_qualified']
    assert not checkpoint['owner_acceptance'] and not checkpoint['canonical_promoted']
    print(json.dumps(dict(passed=True, release=identity['release'], stage_journal_markers=50,
                          launcher_exit_code=0, target_native_exit_status=None,
                          ordinary_startup_qualified=False, owner_acceptance=False)))


if __name__ == '__main__':
    main()

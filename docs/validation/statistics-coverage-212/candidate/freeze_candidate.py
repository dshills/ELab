"""Record a clean candidate without launching or promoting its native bundle."""
import hashlib, json, shutil, subprocess, sys
from pathlib import Path

root = Path('/Users/dshills/Development/projects/ELab')
stage = Path(__file__).resolve().parent
out = root/'docs/validation/statistics-coverage-212'
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
build = json.loads((stage/'build-result-signed.json').read_text())
protocol = json.loads((stage/'protocol-report.json').read_text())
assert protocol['passed'] and protocol['refusal_cases'] == 8 and protocol['covered_cases'] == 4
assert protocol['source_commit'] == build['source_commit'] and protocol['agent_sha256'] == build['agent_sha256']
assert protocol['agent_exit_code'] == 0 and protocol['shutdown']['workers_joined'] and protocol['shutdown']['exit_ready']
assert build['clean_build_vcs'] and build['strict_bundle_signature_valid']
inputs = json.loads((out/'source-inputs.json').read_text())
assert all(sha(root/p) == h for p, h in inputs.items())
sys.path.insert(0, str(root/'scripts'))
from runtime_isolation import bundle_tree
review = Path(build['review'])
assert build['source_bundle'] == bundle_tree(review/'ELab.app')
assert sha(review/'ELab.app/Contents/MacOS/ELab') == build['native_sha256']
assert sha(review/'elab-agent') == build['agent_sha256']
old = json.loads((root/'docs/validation/recorded-position-211/candidate-report.json').read_text())
old_review = Path(old['candidate'])
assert sha(old_review/'elab-agent') == old['agent_sha256']
assert sha(old_review/'ELab.app/Contents/MacOS/ELab') == old['native_sha256']
assert sha(root/'build/releases/ELab-0.181.4-macos-arm64.zip') == 'd13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b'
subprocess.run(['/usr/bin/codesign', '--verify', '--deep', '--strict', str(review/'ELab.app')], check=True)
destination = out/'candidate'
assert not destination.exists()
destination.mkdir()
for name in ['build_candidate.py', 'protocol_check.py', 'freeze_candidate.py', 'build-check.txt', 'build.txt',
             'package.txt', 'build-result-signed.json', 'signature-signed-verify.txt', 'signature-signed-details.txt',
             'protocol-check.txt', 'protocol-report.json', 'roundtrip.json', 'source-commit.txt']:
    shutil.copyfile(stage/name, destination/name)
for p in (stage/'build').glob('*-build-info.txt'):
    shutil.copyfile(p, destination/p.name)
report = dict(version=build['version'], build=build['build'], source_commit=build['source_commit'],
              candidate=str(review), clean_build_vcs=True, source_inputs_still_match=len(inputs),
              native_sha256=build['native_sha256'], agent_sha256=build['agent_sha256'],
              strict_ad_hoc_signature_valid=True, bundle_unchanged_after_agent_test=True,
              matching_headless_agent_passed=True, refusal_cases=protocol['refusal_cases'],
              covered_cases=protocol['covered_cases'], imported_rows_and_metadata_preserved=True,
              public_sample_reads_discover_actual_bound=True, constant_reference_v=5,
              headless_shutdown=protocol['shutdown'], agent_exit_code=0,
              previous_candidate_preserved=True, native_app_launched=False,
              native_acceptance_passed=False, promotion_status='held', release_ready=False,
              published=False, owner_review_required=False, download_unchanged='0.181.4 /205',
              limitations='Native measurement default and accumulated /207–/211 native checks remain pending; no fresh launch prompt is queued.')
(out/'candidate-report.json').write_text(json.dumps(report, indent=2)+'\n')
pending = json.loads((root/'docs/validation/recorded-position-211/native-acceptance-pending.json').read_text())
pending['required_checks'].append('Imported capture measurement selector defaults to the actual saved endpoint, displays 5 V constant-source mean/RMS and reports unsupported typed intervals without losing prior results')
pending.update(source_commit=build['source_commit'], native_sha256=build['native_sha256'], agent_sha256=build['agent_sha256'])
(out/'native-acceptance-pending.json').write_text(json.dumps(pending, indent=2)+'\n')
with (out/'README.md').open('a') as f:
    f.write(f'''

The [clean /212 candidate receipt](candidate-report.json) identifies source
`{build['source_commit']}` and both exact binary hashes. All three Go binaries
report that revision with `vcs.modified=false`; strict ad-hoc bundle validation
passes. Its matching packaged agent refuses eight unsupported query/set cases,
passes four covered constant-source measurements (including a one-step-wide
interval), discovers the endpoint through public row reads, and preserves rows
and metadata through export/reimport. The agent closes normally with joined
workers and exit code 0. The signed bundle is unchanged by these checks.

The candidate is held for [native acceptance](native-acceptance-pending.json).
No graphical launch, download replacement, release promotion or publication
occurred. Previous /211 binaries and the /205 ZIP remain exact.
''')
with (root/'research/89-recorded-statistics-coverage.md').open('a') as f:
    f.write('''

The [clean /212 matching packaged agent](../docs/validation/statistics-coverage-212/candidate-report.json)
passes eight unsupported query/set cases and four covered measurements. Exact
saved rows and metadata survive export/reimport; public sample reads discover
the usable endpoint. Normal shutdown joins workers and exits 0. All captured
source inputs and the signed bundle remain exact. This is a held candidate,
with native acceptance pending and /205 still the current download.
''')
manifest = {p.relative_to(out).as_posix(): sha(p) for p in sorted(out.rglob('*')) if p.is_file() and p.name != 'MANIFEST.json'}
(out/'MANIFEST.json').write_text(json.dumps(manifest, indent=2)+'\n')
print(json.dumps({k: report[k] for k in ['version', 'build', 'clean_build_vcs', 'matching_headless_agent_passed', 'refusal_cases', 'covered_cases', 'agent_exit_code']}))

"""Record package checks separately from pending native event acceptance."""
import hashlib, json, shutil, subprocess, sys
from pathlib import Path

root = Path('/Users/dshills/Development/projects/ELab')
stage = Path(__file__).resolve().parent
out = root/'docs/validation/analysis-field-context-213'
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
build = json.loads((stage/'build-result-signed.json').read_text())
protocol = json.loads((stage/'protocol-report.json').read_text())
assert protocol['passed'] and protocol['fresh_refusal_cases'] == 9 and protocol['stale_valid_cases'] == 3
assert protocol['fresh_corrections'] == 3 and protocol['scalar_readings_checked'] == 123
assert protocol['source_commit'] == build['source_commit'] and protocol['agent_sha256'] == build['agent_sha256']
assert protocol['agent_exit_code'] == 0 and protocol['shutdown']['workers_joined'] and protocol['shutdown']['exit_ready']
assert build['clean_build_vcs'] and build['strict_bundle_signature_valid']
inputs = json.loads((out/'source-inputs.json').read_text())
assert all(sha(root/p) == h for p,h in inputs.items())
sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import bundle_tree
review = Path(build['review'])
assert build['source_bundle'] == bundle_tree(review/'ELab.app')
assert sha(review/'elab-agent') == build['agent_sha256']
previous = json.loads((root/'docs/validation/statistics-coverage-212/candidate-report.json').read_text())
old = Path(previous['candidate'])
assert sha(old/'elab-agent') == previous['agent_sha256']
assert sha(old/'ELab.app/Contents/MacOS/ELab') == previous['native_sha256']
assert sha(root/'build/releases/ELab-0.181.4-macos-arm64.zip') == 'd13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b'
subprocess.run(['/usr/bin/codesign','--verify','--deep','--strict',str(review/'ELab.app')],check=True)
destination = out/'candidate'
assert not destination.exists()
destination.mkdir()
for name in ['build_candidate.py','protocol_check.py','freeze_candidate.py','build-check.txt','build.txt',
             'package.txt','build-result-signed.json','signature-signed-verify.txt','signature-signed-details.txt',
             'protocol-check.txt','protocol-report.json','source-commit.txt']:
    shutil.copyfile(stage/name,destination/name)
for p in (stage/'build').glob('*-build-info.txt'):
    shutil.copyfile(p,destination/p.name)
report = dict(version=build['version'],build=build['build'],source_commit=build['source_commit'],candidate=str(review),
              clean_build_vcs=True,source_inputs_still_match=len(inputs),native_sha256=build['native_sha256'],
              agent_sha256=build['agent_sha256'],strict_ad_hoc_signature_valid=True,bundle_unchanged_after_agent_test=True,
              matching_headless_agent_passed=True,fresh_refusal_cases=9,stale_valid_cases=3,fresh_corrections=3,
              scalar_readings_checked=123,max_voltage_error_v=protocol['max_voltage_error_v'],
              max_current_error_a=protocol['max_current_error_a'],captured_metadata_preserved=True,
              crossing_brackets_independent_rc_reference=True,headless_shutdown=protocol['shutdown'],agent_exit_code=0,
              previous_candidate_preserved=True,native_app_launched=False,native_acceptance_passed=False,
              promotion_status='held',release_ready=False,published=False,owner_review_required=False,
              download_unchanged='0.181.4 /205',
              limitations='Native late-field/refusal correction and accumulated /207–/212 native checks remain pending; no fresh launch prompt is queued.')
(out/'candidate-report.json').write_text(json.dumps(report,indent=2)+'\n')
pending = json.loads((root/'docs/validation/statistics-coverage-212/native-acceptance-pending.json').read_text())
pending['required_checks'].append('Queued pre-agent verification/time/sweep-count parser failures cannot install text in the newer form; currently owned refusals still accept a fresh correction after unrelated agent work')
pending.update(source_commit=build['source_commit'],native_sha256=build['native_sha256'],agent_sha256=build['agent_sha256'])
(out/'native-acceptance-pending.json').write_text(json.dumps(pending,indent=2)+'\n')
with (out/'README.md').open('a') as f:
    f.write(f'''

The [clean /213 candidate](candidate-report.json) identifies exact source
`{build['source_commit']}` and native/agent hashes. All three binaries report
that commit with `vcs.modified=false`; strict ad-hoc signature checks pass.
The matching headless agent preserves state across nine fresh invalid entries
and three stale valid requests, then applies three fresh form corrections.
Its 41-row RC capture passes 123 independent voltage/current comparisons and
brackets the analytic 3 V capacitor crossing. Captured metadata remains exact.
Normal shutdown joins workers and exits 0; the signed bundle stays unchanged.

These shared-command checks do not exercise private native refusal stores.
The candidate is held for [native acceptance](native-acceptance-pending.json).
The previous /212 binaries and /205 download remain exact; nothing is promoted
or published, and no graphical launch or owner review is requested.
''')
with (root/'research/90-analysis-field-context.md').open('a') as f:
    f.write('''

The [clean /213 matching agent](../docs/validation/analysis-field-context-213/candidate-report.json)
passes nine current invalid entries, three stale valid requests and three fresh
form corrections, preserving the agent's draft. A 41-row RC capture passes
123 independent scalar references and brackets the analytic 3 V crossing.
Workers join and the process exits 0. The signed bundle, captured inputs,
previous candidate and /205 download remain exact. Private native refusal
behavior remains a separate acceptance check; the candidate is held.
''')
manifest = {p.relative_to(out).as_posix():sha(p) for p in sorted(out.rglob('*')) if p.is_file() and p.name!='MANIFEST.json'}
(out/'MANIFEST.json').write_text(json.dumps(manifest,indent=2)+'\n')
print(json.dumps({k:report[k] for k in ['version','build','matching_headless_agent_passed','fresh_refusal_cases','stale_valid_cases','fresh_corrections','scalar_readings_checked','agent_exit_code']}))

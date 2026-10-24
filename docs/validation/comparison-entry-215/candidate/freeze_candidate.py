"""Record clean-package comparison checks separately from native acceptance."""
import hashlib,json,shutil,subprocess,sys
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');stage=Path(__file__).resolve().parent
out=root/'docs/validation/comparison-entry-215';sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
build=json.loads((stage/'build-result-signed.json').read_text());protocol=json.loads((stage/'protocol-report.json').read_text())
assert protocol['passed'] and protocol['queries_checked']==23 and protocol['scalar_readings_checked']==138
assert protocol['refusal_cases']==6 and protocol['different_sample_times_queries']>0
assert protocol['captured_metadata_preserved_after_live_edit'] and protocol['clear_preserves_scope_window_and_cursors']
assert protocol['source_commit']==build['source_commit'] and protocol['agent_sha256']==build['agent_sha256']
assert protocol['agent_exit_code']==0 and protocol['shutdown']['workers_joined'] and protocol['shutdown']['exit_ready']
assert build['clean_build_vcs'] and build['strict_bundle_signature_valid']
inputs=json.loads((out/'source-inputs.json').read_text());assert all(sha(root/p)==h for p,h in inputs.items())
sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import bundle_tree
review=Path(build['review']);assert build['source_bundle']==bundle_tree(review/'ELab.app')
assert sha(review/'elab-agent')==build['agent_sha256']
previous=json.loads((root/'docs/validation/statistics-entry-214/candidate-report.json').read_text())
old=Path(previous['candidate'])
assert sha(old/'elab-agent')==previous['agent_sha256'] and sha(old/'ELab.app/Contents/MacOS/ELab')==previous['native_sha256']
assert sha(root/'build/releases/ELab-0.181.4-macos-arm64.zip')=='d13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b'
subprocess.run(['/usr/bin/codesign','--verify','--deep','--strict',str(review/'ELab.app')],check=True)
destination=out/'candidate';assert not destination.exists();destination.mkdir()
for name in ['build_candidate.py','protocol_check.py','freeze_candidate.py','build-check.txt','build.txt','package.txt',
             'build-result-signed.json','signature-signed-verify.txt','signature-signed-details.txt','protocol-check.txt',
             'protocol-report.json','captured-rc-comparisons.json','source-commit.txt']:
    shutil.copyfile(stage/name,destination/name)
for p in (stage/'build').glob('*-build-info.txt'):shutil.copyfile(p,destination/p.name)
report=dict(version=build['version'],build=build['build'],source_commit=build['source_commit'],candidate=str(review),
    clean_build_vcs=True,source_inputs_still_match=len(inputs),native_sha256=build['native_sha256'],agent_sha256=build['agent_sha256'],
    strict_ad_hoc_signature_valid=True,bundle_unchanged_after_agent_test=True,matching_headless_agent_passed=True,
    queries_checked=23,scalar_readings_checked=138,different_sample_times_queries=protocol['different_sample_times_queries'],
    refusal_cases=6,max_voltage_error_v=protocol['max_voltage_error_v'],max_current_error_a=protocol['max_current_error_a'],
    captured_metadata_preserved_after_live_edit=True,live_resistance_ohm=30000,captured_a_resistance_ohm=20000,
    captured_b_resistance_ohm=10000,unit_time_correction_succeeded=True,preference_and_reference_changes_succeeded=True,
    clear_preserves_scope_window_and_cursors=True,headless_shutdown=protocol['shutdown'],agent_exit_code=0,
    previous_candidate_preserved=True,native_app_launched=False,native_acceptance_passed=False,promotion_status='held',
    release_ready=False,published=False,owner_review_required=False,download_unchanged='0.181.4 /205',
    limitations='Changed comparison-time retention/reference ownership and accumulated /207–/214 native checks remain pending; no new launch prompt is queued.')
(out/'candidate-report.json').write_text(json.dumps(report,indent=2)+'\n')
pending=json.loads((root/'docs/validation/statistics-entry-214/native-acceptance-pending.json').read_text())
pending['required_checks'].append('Captured comparison retains invalid times, distinguishes cached results, preserves independent preferences and clears old private/rejected time when the reference capture changes')
pending.update(source_commit=build['source_commit'],native_sha256=build['native_sha256'],agent_sha256=build['agent_sha256'])
(out/'native-acceptance-pending.json').write_text(json.dumps(pending,indent=2)+'\n')
with (out/'README.md').open('a') as f:
    f.write(f'''

The [clean /215 candidate](candidate-report.json) identifies exact source
`{build['source_commit']}` and native/agent hashes. All three binaries report
that commit with `vcs.modified=false`; strict ad-hoc signature validation passes.
The matching headless agent checks 23 pure comparisons and 138 independent
voltage/current/delta references. A captures 20 kohm at 1 ms; B captures 10 kohm
at 2 ms. The external checker selects nearest actual saved rows with later
ties, then evaluates ideal RC laws separately at A and B's timestamps.

After the live resistor changes to 30 kohm, all three captured metadata records
remain exact and captured reference values still pass. Six invalid/stale
requests preserve the complete accepted state. Unit-time correction, preference
changes, a shorter reference and clearing succeed without changing A's scope,
window or cursors. Workers join and the agent exits 0. The signed bundle and all
captured inputs remain exact. [Raw captured data](candidate/captured-rc-comparisons.json)
and [runtime report](candidate/protocol-report.json) are retained.

These headless checks do not exercise private native widget text. The candidate
is held for [native acceptance](native-acceptance-pending.json); /214 and /205
remain exact. No graphical app is launched, promoted or published, and no owner
review is requested.
''')
with (root/'research/92-comparison-entry-recovery.md').open('a') as f:
    f.write('''

The [clean /215 matching agent](../docs/validation/comparison-entry-215/candidate-report.json)
passes 23 pure comparisons and 138 independent RC voltage/current/delta readings
across different saved rates and resistances. It preserves captured metadata
after a live 30 kohm edit, refuses six invalid/stale requests without changing
state, and passes unit-time correction, preference/reference changes and clear.
Scope windows/cursors remain exact; workers join and the process exits 0.
All 886 source/check inputs and the signed bundle remain exact. The candidate
is held for fresh native acceptance; /205 remains the current download.
''')
manifest={p.relative_to(out).as_posix():sha(p) for p in sorted(out.rglob('*')) if p.is_file() and p.name!='MANIFEST.json'}
(out/'MANIFEST.json').write_text(json.dumps(manifest,indent=2)+'\n')
print(json.dumps({k:report[k] for k in ['version','build','matching_headless_agent_passed','queries_checked','scalar_readings_checked','different_sample_times_queries','refusal_cases','agent_exit_code']}))

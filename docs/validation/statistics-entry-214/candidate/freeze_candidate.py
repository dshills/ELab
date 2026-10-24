"""Record the clean measurement-entry candidate and its unchanged bundle."""
import hashlib, json, shutil, subprocess, sys
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');stage=Path(__file__).resolve().parent
out=root/'docs/validation/statistics-entry-214';sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
build=json.loads((stage/'build-result-signed.json').read_text());protocol=json.loads((stage/'protocol-report.json').read_text())
assert protocol['passed'] and protocol['refusal_cases']==8 and protocol['covered_cases']==4
assert protocol['quantity_sets']==2 and protocol['quantity_refusal_cases']==2 and protocol['measurement_clear_succeeded']
assert protocol['quantity_query_set_equality'] and protocol['measurement_edits_preserve_live_document_history']
assert protocol['source_commit']==build['source_commit'] and protocol['agent_sha256']==build['agent_sha256']
assert protocol['agent_exit_code']==0 and protocol['shutdown']['workers_joined'] and protocol['shutdown']['exit_ready']
assert build['clean_build_vcs'] and build['strict_bundle_signature_valid']
inputs=json.loads((out/'source-inputs.json').read_text());assert all(sha(root/p)==h for p,h in inputs.items())
sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import bundle_tree
review=Path(build['review']);assert build['source_bundle']==bundle_tree(review/'ELab.app')
assert sha(review/'elab-agent')==build['agent_sha256']
previous=json.loads((root/'docs/validation/analysis-field-context-213/candidate-report.json').read_text())
old=Path(previous['candidate'])
assert sha(old/'elab-agent')==previous['agent_sha256'] and sha(old/'ELab.app/Contents/MacOS/ELab')==previous['native_sha256']
assert sha(root/'build/releases/ELab-0.181.4-macos-arm64.zip')=='d13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b'
subprocess.run(['/usr/bin/codesign','--verify','--deep','--strict',str(review/'ELab.app')],check=True)
destination=out/'candidate';assert not destination.exists();destination.mkdir()
for name in ['build_candidate.py','protocol_check.py','freeze_candidate.py','build-check.txt','build.txt','package.txt',
             'build-result-signed.json','signature-signed-verify.txt','signature-signed-details.txt','protocol-check.txt',
             'protocol-report.json','roundtrip.json','source-commit.txt']:
    shutil.copyfile(stage/name,destination/name)
for p in (stage/'build').glob('*-build-info.txt'):shutil.copyfile(p,destination/p.name)
report=dict(version=build['version'],build=build['build'],source_commit=build['source_commit'],candidate=str(review),
    clean_build_vcs=True,source_inputs_still_match=len(inputs),native_sha256=build['native_sha256'],agent_sha256=build['agent_sha256'],
    strict_ad_hoc_signature_valid=True,bundle_unchanged_after_agent_test=True,matching_headless_agent_passed=True,
    uncovered_query_set_refusals=8,covered_constant_source_queries=4,quantity_sets=2,quantity_refusal_cases=2,
    measurement_clear_succeeded=True,quantity_query_set_equality=True,imported_rows_and_metadata_preserved=True,
    measurement_edits_preserve_live_document_history=True,headless_shutdown=protocol['shutdown'],agent_exit_code=0,
    previous_candidate_preserved=True,native_app_launched=False,native_acceptance_passed=False,promotion_status='held',
    release_ready=False,published=False,owner_review_required=False,download_unchanged='0.181.4 /205',
    limitations='Changed-candidate native retained-entry and channel-switch behavior plus accumulated /207–/213 checks remain pending. The old native /206 baseline was restored; no new launch prompt is queued.')
(out/'candidate-report.json').write_text(json.dumps(report,indent=2)+'\n')
pending=json.loads((root/'docs/validation/analysis-field-context-213/native-acceptance-pending.json').read_text())
pending['required_checks'].append('Captured statistics retain invalid and unsupported start/end text, distinguish cached results, recover on the first correction and discard old private/rejected entries when the measured channel changes')
pending.update(source_commit=build['source_commit'],native_sha256=build['native_sha256'],agent_sha256=build['agent_sha256'])
(out/'native-acceptance-pending.json').write_text(json.dumps(pending,indent=2)+'\n')
with (out/'README.md').open('a') as f:
    f.write(f'''

The [clean /214 candidate](candidate-report.json) identifies exact source
`{build['source_commit']}` and native/agent hashes. All three binaries report
that revision with `vcs.modified=false`; strict ad-hoc signature validation
passes. Its matching headless agent refuses eight uncovered query/set cases,
passes four constant-5-V covered measurements (including a one-step-wide span),
and preserves rows/metadata through export/reimport. Two fresh voltage/current
quantity edits match pure queries, two ordinary invalid entries preserve cached
state, and clearing measurements preserves live physics, document and history.
Normal shutdown joins workers and exits 0; the signed bundle stays unchanged.

The candidate is held for [native acceptance](native-acceptance-pending.json).
Headless checks do not exercise private widget text. The old native /206 baseline
and its restored state are separately documented. The /213 binaries and current
/205 ZIP remain exact; nothing is promoted, published or graphically launched.
''')
with (root/'research/91-statistics-entry-recovery.md').open('a') as f:
    f.write('''

The [clean /214 candidate](../docs/validation/statistics-entry-214/candidate-report.json)
passes matching-agent imported coverage, exact round-trip and constant-source
checks, plus voltage/current unit-time edits, pure query/set equality, two
ordinary invalid-entry refusals and measurement clearing. Workers join and the
agent exits 0. All 883 source/check inputs and the signed bundle remain exact.
The candidate is held for fresh native acceptance; /205 remains the download.
''')
manifest={p.relative_to(out).as_posix():sha(p) for p in sorted(out.rglob('*')) if p.is_file() and p.name!='MANIFEST.json'}
(out/'MANIFEST.json').write_text(json.dumps(manifest,indent=2)+'\n')
print(json.dumps({k:report[k] for k in ['version','build','matching_headless_agent_passed','uncovered_query_set_refusals','covered_constant_source_queries','quantity_sets','quantity_refusal_cases','agent_exit_code']}))

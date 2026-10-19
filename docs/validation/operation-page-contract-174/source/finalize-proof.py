from pathlib import Path
import hashlib
import json
import shutil

root = Path('/Users/dshills/Development/projects/ELab')
stage = root / 'build/operation-pages-174-work'
proof = root / 'docs/validation/operation-page-contract-174'
status = json.loads((stage / 'final-quality-status.json').read_text())
assert status['exit_code'] == 0
pins = json.loads((stage / 'final-quality-source-pins.json').read_text())['source_sha256']
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
assert len(pins) == 650 and all(sha(root/n) == digest for n, digest in pins.items())
for name in ['final-quality.txt', 'final-quality-status.json']:
    shutil.copyfile(stage/name, proof/'source'/name)
shutil.copyfile(__file__, proof/'source/finalize-proof.py')
elapsed = status['elapsed_seconds']
readme = proof/'README.md'
text = readme.read_text()
assert '## Final source gate' not in text
text += f'''
## Final source gate

The final default `GOPROXY=off ./scripts/check.sh` exits 0 in {elapsed:.3f} s:
SDK admission, imports/format/Staticcheck, requirement-ledger consistency,
54 Python tests, both Go modules' race/vet checks, and selected mounted offscreen
controls/vet pass. Its exact 650 Go hashes are archived and checked before this
checkpoint. The earlier full-gate failure remains a failure, with its unmodified
log and original test source. This passing gate follows the test-adapter repair;
it does not retroactively qualify that earlier source configuration.

Read-only replay: `python3 -B docs/validation/operation-page-contract-174/verify.py`.
Clean replacement packaging is the next separate gate. No new physical UI,
latency, owner walkthrough or broader MVP completion is established.
'''
readme.write_text(text)
plan = root/'research/34-first-version-plan.md'
text = plan.read_text()
marker = '## Current position\n\n'
entry = f'''- **Recorded paging discovery repaired in source.** Frozen /174 accepts null offset/limit defaults while both discovery forms advertise integers only. The final regression fails against original metadata and passes after the two input types/default descriptions are corrected. Thirty development-agent cases/95 requests preserve full snapshots and all 209 captured rows; ordinary catalog projection changes only those two input properties. Final full default quality passes ({elapsed:.3f} s). Its earlier race-instrumented coil deadline failure remains preserved; the existing ordinary-agent fixture method passes all four unchanged coil captures with production guards enabled. [Proof](../docs/validation/operation-page-contract-174/README.md). Commands/decoder/output schemas/models/format/SDK remain unchanged. Next: one clean private app/agent replacement and exact packaged paging checks; preserve /174 and canonical /120. AGENT-02, owner walkthrough and broader gates remain open.\n\n'''
assert text.count(marker) == 1
plan.write_text(text.replace(marker, marker+entry, 1))
audit = root/'research/41-release-gate-reconciliation.md'
with audit.open('a') as stream:
    stream.write('\nPAGE-NULL-SCHEMA-01 [is now repaired and source-qualified](../docs/validation/operation-page-contract-174/README.md): both agent catalogs describe the already accepted null paging defaults, with exact public behavior and query-purity checks. This finite AGENT-02 slice does not change the historical reconciliation/source pins, 17 implemented/10 partial P0 counts, pending owner walkthrough or Gates C/D. Clean replacement packaging remains separate.\n')
preserve = {
    'build/ELab.app/Contents/MacOS/ELab': '914c1ba7547d9d1608cc1c8a0ff1bdeca42918e9343108b712e933c43e5d5b7c',
    'build/ELab-0.157.0-174-review/ELab.app/Contents/MacOS/ELab': '46f78038929c6bce7ed6b55da201d26cda0bb6279c80cea31abc188ca12dec2a',
    'build/ELab-0.157.0-174-review/elab-agent': 'a8636912f15394e647a773d36395fd7f0380d9315f4c16a3e599a51390e6aa54',
    'build/ELab-0.157.0-174-review.zip': 'dcc743fc95459d7c46f218f5bf4e5ab7b22f8471c6b0a72c64c7299e237c68ff',
    'third_party/vellum/vellum-b22478c.tar.gz': '22e38dbb24179292d69353421197ac867aa90c883eed85f1cbcdff999eb57968',
}
assert all(sha(root/n) == digest for n, digest in preserve.items())
files = {p.relative_to(proof).as_posix(): {'bytes': p.stat().st_size, 'sha256': sha(p)}
         for p in sorted(proof.rglob('*')) if p.is_file() and p.name not in ['manifest.json', 'checkpoint.json']}
(proof/'manifest.json').write_text(json.dumps({'files': files}, indent=2)+'\n')
checkpoint = dict(finding='PAGE-NULL-SCHEMA-01', source_repair_qualified=True,
    original_release='0.157.0 /174 affected', clean_replacement_qualified=False,
    quality=status, original_partial_cases=29, development_pure_cases=30,
    preserved=preserve, manifest_sha256=sha(proof/'manifest.json'),
    full_mvp_complete=False, canonical_promoted=False, owner_walkthrough='Not tried yet')
(proof/'checkpoint.json').write_text(json.dumps(checkpoint, indent=2)+'\n')
print(json.dumps({'passed': True, 'source_quality_seconds': elapsed, 'evidence_files': len(files)}))

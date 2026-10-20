# Correcting rejected recording positions

Source fix only; the owner's running /189 alpha and the preserved /190 candidate are unchanged.

At sample 8, entering an invalid index (`1.5`) or time (`bad`), then correcting back to the accepted value (`8` or `8 ms`), could leave Play, Previous and Next blocked when activated without focus loss. Pointer activation and correction to a different value were passing controls. The old view restored the accepted number/text after refusal, so the framework treated a fresh correction to that same value as unchanged and the old refusal survived.

The view now retains rejected input while its document, capture and accepted field value still match. A fresh correction reaches the existing guarded public seek command; transport can then continue through the existing commands. Changing the accepted position or capture clears the old draft; changing another field does not erase its rejection. No command, solver, file format or Vellum changes.

The final regressions against the original two production files produce eight failing and 34 passing cases: six same-value corrections without focus loss, plus two rejected-draft retention assertions. All 42 cases pass with the fix under the race detector. They cover Play/Previous/Next, index/time, pointer/activation without blur, unchanged invalid entries, same/different corrections and accepted-position/capture/unrelated-speed changes. Refusals preserve whole snapshots; successful correction and transport preserve physics, document and undo history. They do not establish actual OS typing or native accessibility behavior.

Full `scripts/check.sh` passes in 261.851 s: lint, script tests, Go race/vet and mounted offscreen suites. All 670 Go inputs stay unchanged during the gate. The reviewed inventory retains 73 native sources, 402 sites, 164 public commands and 132 source routes; only two source hashes, line positions and three callback expressions change.

[Report and raw hashes](report.json). Gzipped records retain original production files, the original-source overlay and failures, final focused results, inventory review and complete quality output. Packaging, actual native input, owner task acceptance and full MVP acceptance remain open.

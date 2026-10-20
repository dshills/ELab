# Reject malformed terminal arrays before probe validation

**FILE-TERMINALS-01** is reproduced in the preserved **0.163.0 /180** headless
agent: opening a schematic with an empty or shortened resistor terminal array
panics in `probeInventory` before the later component-metadata check can refuse
it. The same ordering can affect validation of a recovery document. These are
malformed inputs; ordinary saved circuit metadata is not shown to become corrupt.

The repair validates each component's kind and ordered terminal names against
the existing library immediately after decoding its required fields. Probe and
pinned-reading validation can then safely derive references. The existing full
metadata/model/graph checks still run. There is no guessed pin, silent repair,
blanket panic recovery, model/solver/budget change, format bump, new command or
native binding. The shared loader returns an ordinary validation error before
publishing a document.

## Reproduction and corrected behavior

- The exact final direct-decoder regression, overlaid on original production
  `persistence.go`, reproduces **24 panics**: six current-probed two-pin kinds,
  empty/one-element arrays and scope/pinned-preference paths. The corrected
  regression passes. Its valid control includes all **18 library kinds**.
- Two disposable /180 agents each load a private RC example, make a synthetic
  unsaved 20 kΩ edit, then Open a malformed file with explicit discard consent.
  Both produce EOF and **exit 2**, with `probeInventory`/`decodeDocument` panic
  stacks. These are intentional owned crash reproductions, not normal cleanup.
  Their pre-Open dirty snapshots and original/malformed files remain available;
  original file bytes and the review example stay unchanged. No owner data is
  involved.
- The repaired development agent opens those **exact malformed bytes** and
  returns `invalid-document`. The entire before/after snapshot agrees except
  project-job/revision/actor bookkeeping. Unsaved guards still refuse a later
  unconsented Open. Each surviving edit saves separately and reopens fresh;
  both startup currents match **5 V /20 kΩ =250 µA**. A later valid Open succeeds.
- Two checksum-valid recovery capsules containing those malformed documents
  become visible recovery problems. Scanning preserves graph, physics, document,
  history and capsule bytes. A source test also calls the stored-capsule reader
  directly and verifies the scanner does not delete/rewrite the bad capsule.
- Standalone recording import is a **noncrashing control** on both /180 and the
  repaired agent. Its version-1 validation envelope already reaches the full
  component check before building a probe inventory. Both reject the malformed
  recording as `invalid-recording` and preserve state. No original recording-
  import crash is claimed.

Four repaired file/reopen agents and the two import-control agents end with
completed, joined, recovery-finalized, exit-ready shutdowns and exit 0. Together
with the two declared crash reproductions, **64 protocol requests** are retained.
The initial setup attempt is separate: it used a nonexistent R1 instead of the
example's actual R2, received a correct refusal and closed normally. Its raw
protocol, failed producer and diagnostic remain preserved. A first recovery
unit-test fixture also passed a parent directory that did not meet the required
private permissions; the final fixture creates its own private child. That
failure and earlier test source/log are retained; no permission guard is relaxed.

The full default `./scripts/check.sh` passes in **355.380 seconds**, recorded in
`raw/stage/full-quality.json.gz`, on unchanged **656 application Go inputs**;
the runner additionally freezes three existing separate-tool sources
(659 hashes). The gate includes lint, 65 Python tests, root/desktop race/vet and
registered mounted-control/offscreen checks. Original failures remain separate.

## Evidence and release boundary

```sh
python3 -B docs/validation/terminal-shape-validation/verify.py
```

Replay starts no application. It checks file/source hashes, the 24 original
panic regressions, process/protocol outcomes, exact malformed bytes, whole-state
refusals, recovery checksums/preservation, independently derived startup current,
fresh saved edits, import controls, full quality and preserved /180/SDK/canonical
artifacts. Retain the referenced build/private binary files for this strict local
replay. Optimized Python refuses before evidence reads.

`raw/` contains compressed observations, protocols, files and logs; `producers/`
preserves the historical scripts as text. Private paths describe this run and
are not an unattended reproduction recipe. Both original and repaired public
agents use the existing policy denying development-file, outbound-IP and shared
window-settings access. No physical UI input, display recording, privacy grant,
printer operation or owner-project edit is performed.

**This is source/headless qualification; /180 remains affected and preserved.**
A clean replacement and exact copied native/agent qualification are next. Native
physical input, automatic startup recovery/crash, broader corruption/fuzzing,
power-loss, environment, owner and participant gates retain their separate
scopes. P0 stays 17 implemented/10 partial. No promotion or full-MVP acceptance.

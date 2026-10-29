# First-version candidate freeze

This is the historical October 26 closeout for **0.195.9 /270**, not a freeze
on later owner-authorized development. Subsequent owner-authorized releases extend that baseline; use the
[current README](../README.md) and [release tooling](release-tooling.md) for
current source and qualification procedures. Original release evidence and
outstanding acceptance gates below retain their historical scope.

This checklist is the single closeout for the October 26 completion batch.
Scope is frozen at **53 component kinds, 62 examples and 187 public commands**.
The same application commands serve users and agents. No additional components,
model families, public commands or solver-capacity/precision changes are in scope.
Fix demonstrated defects and complete qualification; do not begin another
feature batch to postpone release.

## Candidate and release channel

Preserve unsigned 0.195.8 /269 and every previously qualified package. Prepare
a separate 0.195.9 /270 candidate for the reproduced live-layout repair. Its
binary source, checksums, archive contents, matching agent and exact-runtime
checks must be recorded before promoting it. Current qualification status and
machine-readable receipts are in [the completion evidence](validation/v1-acceptance/README.md).

This remains an **unsigned private macOS arm64 release**. Ad-hoc signing proves
integrity, not Developer ID identity or notarization. No public upload, push,
hosting, security-setting change or owner-review wait is part of this batch.

## Must fix before promoting a candidate

- Lost/overwritten user work, stale user input borrowing newer agent intent,
  dirty-document protection failures or saves falsely reported as successful.
- Incorrect accepted electrical results in the reference circuits, corrupted
  saved graphs/recordings or changed model/solver limits without authorization.
- Blocked drawing, wiring, value edits, measurement, save/reopen or export in
  the existing supported workflow. Controls must retain their shared commands.
- Reproduced unstable/clipped controls during ordinary simulation. Time digits
  and device states may update; their readout regions and buttons must stay put.
- Failed required race/lint/vet/build, matching-binary runtime, archive identity,
  normal owned-process cleanup or preservation checks.

## Five acceptance workstreams

1. **Native input acceptance.** Preserve the original four open checks, with
   current evidence and the exact tool limit for each. A permanent Edit-menu
   AppKit control narrows the Save investigation. Never convert source,
   rendered-driver or service evidence into OS queue/hardware qualification.
2. **Systematic interactions.** Run the 15-family [interaction matrix](validation/v1-acceptance/interaction-matrix.json).
   It covers actual queued/held input, agent interleavings, document/control
   ownership, fresh retry, scrolling, selection, drawing and modal actions.
   Every named Go test must actually pass; missing, skipped or failed subtests
   cannot qualify a family. Preserve intentional SDK contracts absent a real
   reachable ELab defect.
3. **Packaged hobbyist workflows.** Build the four-part LED loop from blank,
   wire it, edit/undo resistance and check current. Charge/discharge RC against
   its equation. Observe all ten digits and wrap in the 555 counter, check
   segment truth and KCL/power, retain a capture. Save/reopen each in a fresh
   matching process and validate SVG/PNG/PDF without changing graph or physics.
   Public socket operation against a native app is distinct from physical UI
   drawing/typing; the latter retains its separately recorded evidence/limits.
4. **Sustained stability.** Run at least 1,800 real wall-clock seconds on the
   final exact candidate. Continuously integrate RC/flasher/counter circuits,
   interleave geometry edit/undo, pan, capture and save/reopen, and sample owned
   process CPU/RSS. Report latency by command and memory after warmup. An RSS
   late-minus-early median increase of 128 MiB triggers investigation; it is a
   workstation investigation threshold, not a universal product guarantee.
   API timing must not be called visible input-to-pixel latency.
5. **Freeze and package.** Require complete application gates after production
   changes, exact matching copied runtime under developer-file/IP denial,
   all 62 extracted examples paused/clean with exact graphs, archive byte/mode
   and integrity checks, preserved prior releases/SDK and a clean reviewed commit.
   Record explicit native and distribution limitations beside the candidate.

## Reproducible focused acceptance

From the repository root, run:

```sh
python3 scripts/v1-interactions.py --artifacts build/v1-interactions-NEW
python3 scripts/v1-workflows.py \
  --app /absolute/path/ELab.app --app-sha256 QUALIFIED_EXECUTABLE_SHA256 \
  --agent /absolute/path/elab-agent --agent-sha256 MATCHING_AGENT_SHA256 \
  --artifacts build/v1-workflows-NEW --soak-seconds 1800
```

Destinations must be new. The workflow runner launches its own native process,
uses private appearance/recovery/project paths, denies writes to shared framework
settings, bounds agent response waits and closes owned sessions normally. It
does not attach to or replace an owner session. Native UI input remains through
documented CUA APIs only. The full quality/package recipes and receipts are
retained in the completion evidence; unchanged framework gates are carried only
when the SDK and its portable/driver inputs are byte-identical.

## Explicit limits and deferred acceptance

Native checks **3, 6, 9 and 11** remain independent gates: Save-panel editing,
controlled divider hardware rates, and precise native pre-agent parser origin
for recorded position and Verify/time/sweep fields. The current CUA attachment
route can exceed its requested timeout by over an hour; repeating it unattended
is not a viable acceptance method. Its drag API exposes no rate/duration control.
Report these checks as open until their actual native scope is demonstrated.

The owner previously deferred review and separate-account startup/installation
for the private alpha. Screen-reader/IME participant, Gatekeeper/quarantine,
physical hardware/model fitting and general performance across machines remain
outside these receipts. A candidate can be packaged with documented limits;
it must not be described as fully native-qualified or publicly signed.

The offline heartbeat is paused. This directly requested batch does not resume
it or authorize another unattended feature batch after closeout.

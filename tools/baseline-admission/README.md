# Recorder baseline admission

Data-only validation tooling for the frozen /173 timing method. It reads an
existing authorized recorder PNG with Go's standard PNG decoder and compares
opaque RGB pixels in pinned regions. It never captures the desktop, opens ELab,
changes a project, or supplies an input operation.

The profiles preserve the already inspected LED/RC annotations in
`docs/validation/latency-followup-173/{led,rc}/image-annotations.json`. They require
the original accepted component value and recognizable live-view regions.
They are specific to the declared 1053×802 geometry, theme, fonts and fixtures.
A mismatch refuses admission; do not retune a profile to admit an unknown frame.

Build from the tools module:

```sh
cd tools
go build -trimpath -o ../build/baseline-admission-tools/baseline-admission ./baseline-admission
```

Inspect an existing frame using its recorder-journal SHA-256:

```sh
build/baseline-admission-tools/baseline-admission \
  --profile tools/baseline-admission/profiles/rc.json \
  --image /absolute/path/to/existing/frame-<sha256>.png --sha256 <sha256>
```

Exit 0 plus a version-1 `admitted: true` JSON decision is required. Exit 2 means
content/data refusal; exit 1 means invocation/read/write failure. Help exits 0
without a decision, so an exit status alone never authorizes input.

## Prospective worker/controller integration

No existing sealed producer, declaration or trial is changed. Before any future
native attempt, preserve and declare the actual revised worker/controller and
pin the tool, profiles and sources. Offline replay is not native qualification.

1. The controller creates a unique attempt ID in the capture-start handshake.
   Reuse neither a directory nor an attempt ID. Preserve existing target identity,
   geometry, private-draft and public before-state checks.
2. Immediately after starting the existing fixed-duration recorder, the worker
   calls `prepare.wait_and_admit` with the current trial/attempt IDs and pinned
   tool/profile hashes. The helper allows a positive decision only within a
   1.5-second deadline for the first complete journal row and its original PNG.
   File I/O and process startup are not hard real-time operations. It writes one exclusive positive or negative
   receipt. Missing, malformed, changed, mistimed or mismatched data fails closed.
3. The controller independently reads that first journal row, validates its
   recorder metadata and builds the `expected` bindings from that row, the current
   handshake and pinned declaration. Never construct expectations from a receipt.
   Pass the receipt through `controller.withAdmittedBaseline` **before requesting
   the marker or sending Apply**. The first callback retains existing marker/ack
   validation; the second retains the existing CUA Apply call. An invalid receipt
   enters neither callback; a failed acknowledgment never enters Apply.
4. On refusal, retain an unavailable outcome and complete the recorder's original
   bounded duration. Do not retry, shorten, extend or replace the trial. Preserve
   the existing seven controller events and add the receipt as separate metadata.
5. After input, retain complete-recording geometry, visibility, clocks, causal
   ordering, old/stale/fresh state and independent-physics analysis. A valid
   baseline does not establish focus, ownership, private draft, continued
   visibility, latency eligibility or an eventual valid result.

Composition timestamps may precede or follow callback receipt; this tool checks
the recorder's flag consistency and leaves timing interpretation to the existing
qualified method. `rc2-03` is an explicit counterexample: its valid initial frame
passes, but later coverage still makes the recording unavailable. Changes outside
the selected regions may also pass. This is a conservative state-template check,
not a general foreground detector or a security/authentication boundary.

## Offline checks

```sh
cd tools
go test -race ./baseline-admission
go vet ./baseline-admission
cd ..
node --test tools/baseline-admission/controller.test.mjs
python3 -B -m unittest discover -s tools/baseline-admission -p test_prepare.py
```

The replay additionally needs Pillow and the exact ignored raw archives declared
by the existing proof; missing files must be restored, never recaptured:

```sh
python3 -B tools/baseline-admission/replay.py \
  --tool build/baseline-admission-tools/baseline-admission \
  --output build/a-new-offline-replay-directory
```

The output contains decisions and hashes, not screenshots. Each saved Go region
hash is cross-checked with Pillow before the same real decision enters fake
marker/input callbacks. No UI or capture API participates in this replay.

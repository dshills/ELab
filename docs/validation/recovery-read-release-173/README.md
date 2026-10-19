# Packaged recovery read jobs

October 19, 2026. The exact frozen /173 agent passes the expanded
[public recovery workflow](../../../scripts/recovery-read-demo.py) over headless
stdio, using a private directory and synthetic RC project/recording only.

Ten public checks cover captured identity, stored inspection versus pure query,
exact original Verify sample pages, digest-checked 512-byte chunks, empty EOF,
invalid chunk bounds, stale-hash/unknown-job refusals, source/history/file purity,
retention after directory reconfiguration and eviction after eight terminal jobs.
The two recorded rows independently satisfy the RC charging law within 1e-12 V.
Final normal EOF exits 0; the process creates no native window, capture, shared
preference/recovery configuration or printer job.

The observed job inventory has two successful reads and nine failures: one wrong
hash, then eight requests for the old candidate in the new empty private directory.
Every failure has zero result bytes and a structured-result `not-ready` refusal;
the stale-hash case also refuses a chunk. Three old-job query forms refuse
`not-found` after eviction. No
in-flight cancellation race is claimed: terminal/unknown cancellation refusals
are checked here; blocked worker/slot/join tests retain their existing source scope.
AGENT-04, native/accessibility/environment and broader release gates remain open.

The workflow now accepts an explicit binary plus required matching SHA-256,
checks chunk progress to prevent a nonadvancing loop, retains job/chunk evidence,
and writes a passing report only after normal client exit. Wrong hash, incomplete
pin arguments and Python optimization each refuse before creating artifacts or
starting a session. The last guard prevents disabled assertions from producing a
false validation pass. Ordinary default binary selection and the fresh-owned
native endpoint admission remain available.

`final/` retains the prelaunch source/binary declaration, exact producer, report,
raw reconstructed JSON result bytes, chunk ranges/EOF metadata and synthetic
saved project/capsule. `preflight/` retains three no-launch refusals and that
producer's exact hash-checked bytes; the final producer adds only the raw-result
evidence field afterward. The app's 650 prior Go source hashes, frozen app/agent,
SDK, recorders and prior validation checkpoints are unchanged. No app rebuild or
behavioral repair was needed.

The initial offline verifier compared the inspection's metadata list to the
capsule's recording container and refused. Its bytes and refusal are retained;
the corrected verifier compares embedded metadata and the exact stored row slice.
This was an evidence-parser error, not a failed public workflow or app repair.

Replay source, artifact, digest, stored graph, sample-law, inventory and refusal
evidence without launching ELab:

```sh
python3 -B docs/validation/recovery-read-release-173/verify.py
```

Repeat the public workflow in new output/artifact directories:

```sh
python3 -B scripts/recovery-read-demo.py \
  --agent-binary /private/tmp/elab-first-version-173-signed-runtime/elab-agent \
  --agent-sha256 134f9863e2885e384035e30550adfc5fe0bea1bdee9f3d3cd42feaf802bb7de5 \
  --artifacts build/a-new-private-recovery-check \
  --output build/a-new-recovery-result.json
```

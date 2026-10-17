# Automatic recovery timing protocol

The PRD proposes complete committed autosaves within **five seconds while storage is available**. The public measurement protocol uses the ordinary **two-second** configuration. It creates every tested headless/native process and private directory itself; it never accepts an existing user endpoint. Only those owned hosts are forcibly killed.

Five named fixtures cover the LED reference, a recorded RC circuit, the pulse-controlled NPN LED, a 100-node/51-part potentiometer chain with four probes, and a 64-part resistor chain with four probes. Recorded fixtures retain a full 200 ms Verify capture, including both sides of source edges. Capacity boundaries remain separate fixtures rather than a claim of simultaneous maxima or every permitted topology.

Each series alternates resistor/resistance edits with kept-record membership changes where a recording exists. Every sample must produce a distinct committed document, and the latest capsule must contain that exact document: graph, model dependencies, settings, view, probes, pause state, kept records and captured rows. Record membership is restored before the final crash. Manual source bytes must remain unchanged; retention must keep at most the two expected owned capsules.

Timing starts **before dispatching the edit** and stops only after finding a matching committed capsule, reading/hashing it from disk and validating it through `recovery.inspect`. The monotonic wall-clock result therefore includes command admission, settling, 25 ms polling and validation overhead. It is a conservative completion upper bound. It does not measure an input event reaching a displayed frame. First samples and warm repetitions remain identified individually; no filesystem cache eviction or clean installation is inferred.

After each series, the controller kills the owned source host with SIGKILL, waits for its observed exit and uses a fresh agent process to inspect/restore the newest complete capsule. Restore must make a dirty unassociated copy. Saving that copy must reproduce the complete expected document, including the original kept capture; the explicit manual source must remain byte-identical. This is process-crash evidence, not a power-loss or hostile-filesystem guarantee.

The [reproducible measurement script](../scripts/recovery_timing.py) retains each sample capsule outside the live recovery directory, source/restored project files, per-host logs, series journals and full report. Report checks reject missing repetitions/cases/transports, reduced capacity, incomplete restore data, missing preservation evidence, invalid identities, summary mismatch and any over-five-second sample. Private endpoint descriptors and live lease directories are excluded from published evidence.

The paused pass requires exact live state/history preservation. An advancing-simulation pass additionally distinguishes ordinary physical time progression from prohibited graph/history/run replacement or clock rewind. Neither pass establishes all storage devices, every topology, maximum recording storage, arbitrary CPU load, other hardware/platforms or complete native input/accessibility acceptance. Evidence and recorded machine/build identities belong with each measurement, rather than generalizing the result to every machine.

Native setup waits for observed canvas allocation to settle, then uses the ordinary zero-distance Pan command to retain the actual fitted view without following later allocation changes. Recovery controls can still alter canvas height by a few pixels and publish session-only viewport dimensions. Those publications are retained in before/after snapshots and may advance the command counter. The checker permits that counter difference only for an observed native user publication with changed valid viewport dimensions and fixed-follow state; the document, fitted view, graph, run identity and history must still agree. Headless counters remain exact. A matching capsule may contain a later command revision, but must contain the exact expected document.

Run from the repository after building the app and agent:

```sh
python3 -B scripts/recovery_timing.py \
  --native-binary /absolute/path/to/ELab.app/Contents/MacOS/ELab \
  --simulation-mode paused --repetitions 12 --artifacts build/recovery-paused
python3 -B scripts/recovery_timing.py \
  --native-binary /absolute/path/to/ELab.app/Contents/MacOS/ELab \
  --simulation-mode running --repetitions 12 --artifacts build/recovery-running
```

Each output directory must be new. Omitting the native binary runs headless only. Repetitions must be a multiple of four, at least four, so the membership sequence ends with its recording included.

Measured October 17, 2026 on Apple M4 Pro, 14 logical CPUs, 64 GiB, macOS 26.6.2, Go 1.27.2, using the unchanged private ELab 0.111.0 /126 candidate and matching agent. All **240 samples** pass the five-second bound:

| Simulation | Interface | Samples | p95 completion | Maximum |
| --- | --- | ---: | ---: | ---: |
| Paused | Headless socket | 60 | 2.512 s | 2.529 s |
| Paused | Native socket | 60 | 2.521 s | 2.544 s |
| Advancing | Headless socket | 60 | 2.518 s | 2.527 s |
| Advancing | Native socket | 60 | 2.518 s | 2.523 s |

Twenty actual SIGKILL cases restore exact expected documents, including **3,344 retained recorded rows**. The 144 component edits and 96 kept-record membership changes preserve every manual source. All 240 capsule hashes/documents and 480 before/after snapshots receive a separate artifact review; five native viewport-only publications remain explicit in that evidence. First-sample maximum is 2.035 s; warm-sample maximum is 2.544 s. These prepared-session categories do not imply a cold OS cache.

[Receipt, complete reports and indexed fixture archive](validation/recovery-timing-validation.json) retain source/binary hashes, failed prototype/matcher evidence, every successful sample and reproducible review scripts. All 683 preexisting inputs and qualified binaries remain unchanged; full lint/imports/Staticcheck, 37 Python tests and root/native race/vet pass, with unchanged Go tests cached. Prior electrical/smoke evidence is reused only after exact runtime-input/binary verification. The candidate remains unpromoted. General storage/hardware, maximum recording storage, automatic failure-display qualification and the separate release gates remain open.

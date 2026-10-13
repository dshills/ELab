# Bounded Verify and recorded results

**Implemented October 13, 2026.** Verify runs the captured supported circuit from startup, independently of the live editor. Its result is explicitly recorded. It never changes live time, charge, pause/input mode, graph, file state or undo history. Paused/stale live readings do not prevent verifying a supported current graph with a new startup scenario.

The right inspector exposes duration, optional sample step (0 follows the project), input mode and global/per-capacitor startup overrides. Use **Use project startup / sampling** to clear overrides. Per-capacitor voltage means pin 1 minus pin 2 (±12 V), with the same polarity as the schematic inspector. A global default is 0–12 V and is overridden by a capacitor's saved or scenario-specific initial condition. RC global default means output relative to ground; mixed global default means each capacitor's pin 1 minus pin 2. Mode is Auto, RC Charge/Discharge, or NPN/mixed High/Low, with the same selected-source rules as live simulation.

**Verify** returns an operation ID immediately. The selector shows retained operations and progress; **Cancel verification** requests cancellation. Completed records show final selected readings and their whole-run min/max, engine identity, duration and sample count. Source changes are labeled; a record stays valid for its captured input and never becomes a live result. DC verification records one operating point at time zero. It defaults to useful voltage/current channels where there is no live DC scope.

## Bounds and lifecycle

- At most one active verification, 10 simulated seconds, 1000 output steps and 4096 retained recorded frames. Shorten duration or increase Verify's sample step if admission exceeds the bound.
- Sample steps are 1 µs–1 s (or zero to follow project settings); pulse durations must still be at least one step. Offline computation does not apply the live playback-rate restriction.
- A 30-second wall budget bounds the operation. Existing kernel convergence/work limits remain in effect; there is no model substitution. Cancellation is checked before/after output steps and during cancellable DC solves; a mixed solve is bounded and observes cancellation after it returns.
- Status is `queued`, `running`, `canceling`, `completed`, `failed`, or `canceled`. Failure/cancellation publishes no partial record. Captured input metadata stays inspectable even when a run fails.
- Keep the latest eight operations per process. Starting a new one may evict the oldest terminal operation. IDs are never reused during the process. App close cancels active operations. Records are not project undo actions or saved live physical state.
- Progress updates use the shared revision, leaving command/electrical revisions and document dirty state untouched. Agents handle ordinary admission conflicts as usual. A retry key returns the original accepted ID; it does not start the computation twice.

Models and kernels are identical to live LED, RC, NPN DC/pulse and mixed simulation. RC uses the exact exponential response; pulse NPN uses the same phase solves; mixed uses the same adaptive backward-Euler kernel. The same existing generic/ideal-device limits apply. This is application numerical verification, not hardware certification or independent SPICE validation.

## Agent commands

| Command | Parameters / result |
|---|---|
| `verification.settings.set` | Full `{duration_s,sample_step_s,input_mode,capacitor_initials:[{id,voltage_v}],initial_voltage_v?}`; session-only defaults, no physical/document edit |
| `simulation.verify` | Optional duration/sample step/mode/initial overrides and `probe_ids`; captures defaults if omitted; DC explicit duration must be zero. Returns Snapshot with `last_operation` and queued summary |
| `operation.inspect` | `{id}`; structured status/progress/error plus captured graph, settings, model/engine identity and, when completed, final results/statistics |
| `operation.samples` | `{id,offset?,limit?}`; completed recorded rows with SI probe descriptors; default limit 100, max 200, `next_offset` / `total` |
| `operation.select` | `{id}`; selects the recorded native preview; empty ID clears it |
| `operation.cancel` | `{id}`; accepts queued/running cancellation; inspect until terminal. Finished operations refuse cancellation |
| `verification.export` | `{id,path,format:"json"|"csv",overwrite?,expected_document_generation?}`; completed record only, atomically published |

A sample row has `time_s` and `values` in metadata probe order. Preserve duplicate timestamps: both sides of ideal pulse edges are recorded, followed by the output sample. Full recorded statistics use samples, including edge samples, rather than time-weighted integrals. Metadata includes source revision/document generation, captured graph/startup/settings, input mode, engine/model versions and all final solver results. `source_unchanged` compares origin electrical revision and document generation; it is not a claim that the live clock or manual source events match the recorded experiment.

JSON export (`elab.verification`, version 1 in metadata) contains the complete captured input and recorded rows. CSV includes operation/input/engine/model identity, mode, duration, step, time and SI channel names. Exports use captured results after source edits, never the current live readings. Existing destinations require explicit overwrite; links and the current schematic path refuse. Output is bounded to 8 MiB. Native pickers preserve the selected operation ID and document-generation guard. Recorded files are readable offline; importing them into a dedicated replay viewer remains a separate follow-on item.

## Discovery and quality

Capability discovery now uses `schema_version:2`. `output_schema.$ref` values are local identifiers resolved through `capabilities.list.schemas`, e.g. `urn:elab:snapshot:1`; never fetch them from the network. This removes repeated snapshot definitions and keeps the 1 MiB response bound. Protocol version remains 1; existing command/part IDs and numerical data remain compatible.

Tests cover analytical RC startup, signed discharge, NPN operating point and pulse edge sides, mixed live/offline parity, immutable captured metadata/pages, interleaved edits, deduplicated requests, limits, cancellation/close, explicit numerical failure, native settings/readout parity and file export identity/refusals. Pinned Staticcheck/import formatting, Go vet and root/desktop race tests run through `scripts/check.sh`; native smoke invokes Verify in the actual window.

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke
python3 scripts/verification-demo.py --output docs/validation/headless-verification.json \
  --artifacts docs/validation/verification-artifacts
# Owned development session only: replaces its schematic.
python3 scripts/verification-demo.py --connect build/verify.endpoint.json \
  --output docs/validation/desktop-verification.json
```

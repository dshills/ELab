# Separately declared LED/RC follow-up

October 19, 2026. Frozen signed **ELab 0.156.0 /173**. The
[forty-ID declaration](../../../research/39-led-rc-followup-plan.md) was committed
as `13669d94` before any follow-up trial. Original results remain unchanged and
are never pooled with these observations.

| Block | Attempted | Available | Unavailable | Not attempted |
|---|---:|---:|---:|---:|
| New LED, `led2-01`–`led2-20` | 20 | 20 | 0 | 0 |
| New RC, `rc2-01`–`rc2-20` | 4 | 1 | 3 | 16 |

LED bounds, milliseconds, n=20. Nearest-rank p95 is the 19th ordered value:

| Marker → visible response | Minimum | Median | p95 | Maximum |
|---|---:|---:|---:|---:|
| Committed 2 kΩ parameter | 102.122 | 127.890 | 146.289 | 146.484 |
| Fresh 1.477 mA result | 118.903 | 146.387 | 162.495 | 186.781 |

The LED fresh-result bound supports 500 ms **for this fixture within the
software-composition method**. Acknowledgment above 100 ms remains inconclusive
because the before-input marker includes controller overhead. These are not
exact OS input/acceptance or physical display scanout times. Raw eligibility
flags stay false and raw gate outcomes null. This is not a general PRD pass.

## Native and numerical outcomes

Both blocks use the frozen original app/agent/recorder, geometry, fixtures,
three-second duration, thresholds and public canonical-baseline reset. The
controller refreshes private draft/Apply AX after the public before-state
observation. Each ID gets one CUA Apply, with no model/tool round trip between
the acknowledged marker and input. Sources, configs, original outcomes, complete
images and journals are retained. No failed ID is retried.

All twenty LED edits commit once. Native wire IDs remain exact; endpoint
semantics, graph/model and full electrical result match the packaged agent.
Shockley/KVL checks and the existing independent 70-digit reference agree.
Every one of **3,252 complete frames /3,098 original PNGs** classifies against
the original manually inspected baseline, committed/stale and fresh region
hashes. The new first/final originals are also inspected before continuing.
Eight analysis corruption classes are rejected per recording. The accepted
schematic parameter and fresh result are measured independently.

`rc2-01` passes matched-time native/agent and whole-period RC-law checks, with
all 166 complete frames/164 PNGs admitted. Its marker-to-committed/fresh bound is
153.609 ms. That one observation does not establish an RC percentile.

`rc2-02` and `rc2-03` commit the requested 2 µF and pass numerical checks, but
their actual recordings contain non-ELab windows. Strict frame admission rejects
26 and seven complete frames respectively. Those observations are unavailable;
unknown frames are not removed and regions are not narrowed to rescue timing.
Both original producer statuses remain `collected_pending_pixel_analysis`.

`rc2-04` has sixteen overlapping complete frames. Its completed CUA Apply call
does not commit private `2uF`: accepted 1 µF, electrical revision and run persist;
the later producer Pause is its only subsequent command revision. The crop first
contains another app and later shows ELab with the uncommitted draft. The cause
of input delivery is unproven under those visibility/focus conditions. **No
application defect or speculative repair is inferred.** The worker stops, and
IDs 05–20 remain untouched. There is no full RC percentile or replacement loop.

All other-app pixels remain only in ignored local archives and original private
capture directories. Git contains their hashes and unavailable observations,
without those screenshots. All issued input targets the owned ELab handle; no
other app is selected, closed or cleaned up. Input delivery under overlap remains
unproven.

The LED worker/session exit 0, with shared settings and all app artifacts
unchanged. RC closes normally through public shutdown, `open -W` exits 0 and its
owned process/endpoint disappear without forced cleanup. Its worker exits 1 on
the stop condition; the session wrapper also exits 1 because shared window-file
hashes changed during the run. The owned sandbox's read/write probes are denied,
and the source of the concurrent change is unproven. Settings are not restored.
Copied/review/canonical apps and all 650 Go source hashes remain unchanged.

## Local evidence storage and replay

Raw frame archives live under ignored
`build/validation-evidence/latency-followup-173/{led,rc}/{id}/frames.tar.xz`.
Every capture's tracked `frame-store.json` pins its relative path, SHA-256, size,
member count and raw-image bytes. Member manifests retain every original PNG's
hash, size and 0600 mode. The original private captures also remain untouched.
This keeps about 86 MiB of new raw archives out of Git while retaining their exact
bytes locally; metadata, decisions, annotations and replay code remain tracked.

The reader verifies size/hash before archive parsing, enforces the declared local
directory and original 300-image/64 MiB bounds, and never extracts or launches
anything. Eight in-memory refusal checks cover missing/changed archives,
escaped paths, changed member identity/count and over-budget data. A missing
archive fails clearly: restore the pinned file at its declared path, rather than
silently recreating a trial. Copy this local archive directory alongside the
repository when transferring the full evidence to another machine.

Read-only replay requires Pillow and those declared archives:

```sh
python3 -B docs/validation/latency-followup-173/verify.py
```

The launcher's root LED registration prelude is separate from per-trial baseline
and expected states. Both CUA postflights report no running ELab. The RC overlap
refusals and shared-settings assertion remain failures in the record, not passes.

Next: perform independent work while desktop isolation is unresolved. The
controller currently waits for a complete baseline frame without verifying that
it contains ELab before input; add a data-only baseline-content admission check
before marker/Apply, using known valid and overlapping originals. It must reject
unknown pixels without sending input or weakening all-frame admission. Do not
resume native recording while unrelated windows are opening over the crop, and
do not automatically declare another RC cohort to obtain a favorable percentile.
Owner walkthrough and broader acceptance remain open; no app rebuild/promotion.

# Current /156 wiring controls and packaged-agent parity

October 18, 2026. Validation-only requalification after the shared native ownership helper changed. No product source, SDK, version or existing package changes in this checkpoint.

The existing release-pinned runner admits clean-source review 0.139.0 /156 from `22a84d5`, checks current source against that commit, and verifies 68 native source hashes, 343 runtime inputs, SDK lock and all 135 review-manifest entries/hashes/modes. It pins the exact packaged agent, fresh test command, 120 producer-source hashes and transcript. Source/release/producer identity is checked again before and after comparison. [Report](report.json), [producer witness](producer.json).

Four parts (5 V source, 1 kΩ resistor, generic red LED and ground) are seeded through public placement commands. The tested native path is **wiring**, not new UI component-placement acceptance. Production selectors receive mounted Tab/arrow input and Connect receives routed mouse input. Four connections each commit once, reset endpoint placeholders and return focus to From. The assembled macOS adapter’s captured callbacks independently wire the same parts. They do not open or drive an OS menu. [Fresh race transcript](focused.txt).

All three captured cases—mounted completed circuit, mounted Undo and assembled-adapter completed circuit—match the exact packaged /156 headless agent on every component/wire/net field, selection, drawing draft, Undo label, current and staleness. Generic-model LED current is 0.0029167965774405296 A. Undo leaves three wires and a stale prior reading. Duplicate Connect and held-agent endpoint replacement preserve graph/history/focus; fresh Connect works. Native adapter checks reject retired agent/document/new-menu, invalid-index and duplicate completions without leaking context.

The focused run is uncached (`-count=1`) and passes under race in 2.793 s; the worker and controller exit 0. Existing full source and copied-runtime quality evidence is reused because code/SDK/package bytes are unchanged. This does not establish new physical Cocoa menus, UI placement, IME/VoiceOver, visible p95, every runtime binding, another environment or owner acceptance. P0 remains 17 implemented /10 partial. The owner walkthrough remains “Not tried yet.” The historical /149 and /151 artifacts stay intact.

To run a future matched review, select its receipt and a new output directory:

```sh
python3 -B scripts/wiring_replay.py \
  --receipt research/ui-audit/first-version/recording-owner-review-checkpoint.json \
  --artifacts build/wiring-qualified-next
```

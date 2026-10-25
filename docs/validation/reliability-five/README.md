# Ordered reliability and examples qualification

October 25, 2026. All five [ordered items](../../../research/116-reliability-release-and-examples-plan.md)
are complete. Final unsigned release: **ELab 0.195.3 /264**, clean binary source
`d9af4fc49f4bb6f6f86e4ae9b3a823afd7c815fc`. No new components or electrical
model families; 53 kinds and 62 existing examples remain.

## Findings and completed work

1. **Canvas visibility:** saved native schematic/scope pixels are intact and
   byte-identical in frames that seemed empty in the large preview. Pan changes
   the expected pixels; resize, inspector and theme changes preserve the circuit.
   No speculative renderer change. [Pixel diagnosis](canvas/pixel-diagnosis.json).
2. **Numeric help on scroll:** SDK elab5 cancels visible/pending help, refuses
   clipped controls and rejects queued stale hover callbacks without stealing a
   fresh same-widget hover. Values remain unchanged. Portable/ordinary/rendered
   framework checks and native PWM Duty scrolling pass. [Tooltip evidence](tooltip/).
3. **Keyboard and agents:** actual native LED construction/wiring, rejected
   value/correction, Undo/Redo, Save and focus, RC stepping/cursors and protected
   replacement Cancel pass. The matching agent reopens the exact saved 2 kΩ
   drawing at 1.476723549 mA. Four-adapter public checks pass; 45 external processes
   exit normally. [Workflow receipt](workflow/report.json).
4. **Preserved intermediate release:** [0.194.5 /260 archive](release-260/archive.json)
   includes all current electronics, matching agent and 62 examples. Its copied
   runtime and all extracted projects pass. ZIP SHA-256 remains
   `4132046779dd14442187e37baa426de55ccff4c9f5f7b587d691bc31c5ae857a`.
5. **Examples browser:** Start here, categories, global word search, actual
   schematics and short experiments use the same public commands as agents.
   Native all-theme/compact layout, category, keyboard, empty search/Clear,
   wheel, thumb drag, Escape and Save/Discard/Cancel checks pass. Held callbacks
   refuse agent interleavings. Final packaged agents return identical SVG/source
   hashes for all62 examples. [Native receipt](browser/browser-native-report.json),
   [agent comparison](browser/browser-agent-final-report.json),
   [initial failures and repairs](browser/failures.md).

## Final quality and package

| Complete race suite | Top-level tests | Including subtests |
| --- | ---: | ---: |
| Portable | 866 | 1,945 |
| Ordinary desktop | 336 | 836 |
| Rendered desktop | 459 | 1,414 |

Lint/imports, root and both desktop vet modes, Python, PRD and build pass.
[Quality summary](browser-quality/quality-complete.json) and
[source/compilation scope](browser-quality/scope.json) retain exact boundaries;
raw JSONL logs are compressed alongside them. Earlier item-4 source checks and
failed smoke assumptions remain under quality/ and release-260/; they are not
relabeled as final candidate results.

The [exact copied /264 runtime](release-264/runtime-report.json.gz) passes complete
native smoke and extended public electrical/project/inductor/op-amp/storage/
capacity suites across in-process, headless stdio/socket and native socket
adapters, with source/development-file and outbound-IP denial. All
[102 external processes](release-264/runtime-process-exits.json) exit normally.
The [archive receipt](release-264/archive.json) verifies 375 member bytes/modes,
macOS extraction and its ad-hoc integrity seal. The extracted matching agent
[reopens all62 bundled examples](release-264/offline-examples.json), paused and
clean, with exact graphs and accepted startup. The public registry has 187
commands; compact ordinary discovery stays within its unchanged 64 KiB bound.

Download: `build/releases/ELab-0.195.3-macos-arm64.zip` (38,746,820 bytes).
SHA-256: `31fa3386d57850e5d68ead285d65f2c8a7319ef1e4866ba3e6f070ea76c85fe1`.
The /260 and /205 releases remain unchanged. SDK upstream f9dee54,
compatibility source bc599ea5, exported elab5; earlier SDK evidence remains in
[the tooltip record](tooltip/). Project format and solver capacities/precision
are unchanged; ordinary schematic report SVG bytes are frozen in regression tests.

## Limits

This is local workstation qualification, with representative actual native
input plus deterministic mounted/agent checks. Flattened mounted browser-body
checks do not claim native centering; native captures qualify centering separately.
All62 thumbnail raster tests cover nonblank/fitting generated drawings, with
representative native previews observed. Owner walkthrough, participant/
screen-reader/IME acceptance, downloaded Gatekeeper/quarantine and separate-
account installation remain outside these receipts; existing owner/clean-account
deferrals remain in force. No Developer ID/notarization, public upload or push
is claimed. Owned test sessions close normally and owner work is preserved.

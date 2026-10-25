# Current Vellum SDK integration

## Current dependency · October 25, 2026

ELab consumes Vellum **v0.1.0-personal.1** from the owner's newly pushed
[`f9dee54`](https://github.com/dshills/vellum/commit/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40) on
`codex/mac-personal-app-stabilization`. This is newer source than both `main`
(`39250c6`) and the published `v0.1.0-alpha.3` assets. The alpha.3 download was
verified against its published SHA-256 during discovery; it is not the selected
runtime. The separate owner Vellum checkout was not modified.

The reproducible local compatibility patch preserves ELab's existing control
ownership/settlement, queued field and held-gesture callbacks, numeric drafts and
decimal steps, system cursors, event-position/wheel behavior, partial-scroll
accessibility, divider gestures, disclosure colors, window backgrounds and text
Undo boundaries. Upstream's newer controls, secure text and focus reveal remain.
ELab also subtracts the new managed focus-border allowance from explicit action
padding, preserving constrained toolbar and tab geometry.
Six compatibility dependency stages follow upstream's 106 stages, numbered
107–112; preparation verifies the complete ordered recipe and each input/output
hash. The private derived source is `bc599ea5f39b31e45c41bbe003e167612ed43275`.

The active archive is `vellum-f9dee54-elab5.tar.gz`; all four native module
replacements resolve into `build/.vellum-sdk-f9dee54-elab5`. The lock pins source, complete
archive/manifest, compatibility patch and all bootstrap scripts before execution.
The SDK is a **private compatibility build**, not an upstream release or a public
redistribution approval. The SDK compatibility changes add no electronics models,
user/agent commands or project-format changes. ELab’s widget integrations are
qualified separately below. Previous SDKs and qualified app packages remain preserved. See [update checks](validation/vellum-f9dee54-update/README.md) for evidence
and its exact native/package scope.

## Compact stepper compatibility · October 25, 2026

The widget migration uses the same upstream source with a small additional
compatibility fix: compact steppers share ELab's existing decimal addition with
inline number fields. Compact bounds still clamp; inline bounds still wrap.
Repeated decimal steps, off-grid values and typed precision retain their original
contract. No solver/model limits change. The complete new patch is
[here](../third_party/vellum/patches/elab-f9dee54-stepper-compat.patch).
The preceding elab2 archive, patch, manifest and qualified /254 package remain
preserved. See the [widget migration qualification](validation/vellum-widget-adoption/README.md) for exact source/test scope.

## Inspector tooltip retirement · October 25, 2026

The elab4 SDK adds a native popup lifetime seam and calls it before retiring a
managed subtree. Only tooltip stages originating in that subtree are closed;
menus and retained controls’ help remain. The seam runs on the originating
render owner and invokes no input/value-change handlers. Existing numeric
bounds, typing, stepping, public ELab commands and simulation limits are unchanged.
The exact-input/output stage 111 is part of the [complete compatibility patch](../third_party/vellum/patches/elab-f9dee54-tooltip-compat.patch).
The preceding elab3 SDK, archive and /255 package remain preserved. Qualification
and bounded native verification are recorded in the [ELab tooltip checks](validation/inspector-tooltip-retirement/README.md).

## Scroll hover lifetime · October 25, 2026

The elab5 SDK cancels visible and pending hover help before wheel scrolling
moves a field. Queued native callbacks keep their originating hover epoch and
cannot publish for a new hover on the same widget. A delayed hover also checks
that its control is actually visible before opening. Numeric values, field
callbacks, menus, existing public ELab commands and solver limits are unchanged.
Exact-input/output stage 112 records the native change and queued-timer test.
See [ordered-work qualification](validation/reliability-five/README.md). The preceding SDK/app and
qualified unsigned release remain preserved.

## Historical integration records

## Public repository migration · October 21, 2026

The active source dependency now comes from [dshills/vellum](https://github.com/dshills/vellum) at public commit `e58d40581fe21d1de53fd54cab1bcf6acf11c163`. Native preparation and the four explicit replacements remain necessary under the upstream consumer contract. ELab's previously required control ownership/numeric settlement APIs are absent from that snapshot; the [explicit compatibility patch](../third_party/vellum/patches/elab-public-e58d405-compat.patch) carries those and the existing held-input, legacy-handoff and scroll-publication fixes forward. Local derived source is `96e22bf25809148b9a28169baf1b24cba72af475`; it is not an upstream commit.

The active archive is `vellum-e58d405-elab1.tar.gz`, with 8,408 hashed members and a separate manifest. Schema 2 pins the public repository/base, derived source, compatibility patch, archive/manifest and six bootstrap files, now including `native_linkers.py`. All four module replacements use `build/.vellum-sdk-e58d405-elab1`. [Validation](validation/vellum-public-source-update/README.md) records source/build checks and their limits. No app was launched or promoted; /189 and /193 retain their original binaries and dependency qualification. Native acceptance with this dependency is still required for a future replacement package.

## Previous integration: 677548d

Private SDK `677548d0ea93d7d6a689998eb8cf3d6f3ff9f59f` preserves the action from an ordinary left-button press through release and queued delivery. Existing ELab revision/history guards reject old Close/Undo actions after an agent edit, even when repaint retains the same logical owner. Keyboard/programmatic current-callback and placement behavior remain unchanged.

Only `drivers/cogent/controls.go` and the new `drivers/cogent/button_context_test.go` differ from `ade66eb`. Dependencies/notices and all five bootstrap sources are identical. The archive has 7,454 hashed files, 18,627,256 bytes, SHA-256 `9233f875cf11829e68ccc62772f9d841c4e01c0da9b2364c0f6f6a7e0d871fe9`, manifest SHA-256 `f5e9dcab46eb4c6805587f0301f96745a4802abf2f2f50614bb42ebf3479cc89`. All four native replacements use `build/.vellum-sdk-677548d`. [Evidence](validation/button-press-context/README.md) records the before/after mounted checks and framework qualification. Review /146 remains immutable and contains the held-button defect; the qualified replacement is recorded below.

The original `b560f6d` /147 candidate and its failed native smoke are preserved in the [candidate checkpoint](../research/ui-audit/first-version/button-press-candidate-checkpoint.json). [Causal diagnosis](validation/native-focus-admission/README.md) captures an admitted focus action retiring at native generation 4→5; complete probes take approximately 14 ms. SDK semantics are correct and unchanged. Clean `77f9319` corrects only the smoke completion protocol and qualifies a separate [same-version /147 replacement](../research/ui-audit/first-version/focus-completion-review-checkpoint.json): full quality, exact copied native smoke/four-adapter conformance and reference/live/save acceptance pass. The new native run exercises two focus admissions before observed completion. No SDK upgrade, deadline extension or broader acceptance claim.

## Previous pinned integration: ade66eb

Private SDK `ade66eb3586c6d881647cea52f19048087e598c5` adds optional `ControlStateKey` for twelve simple managed control kinds. ELab scopes it to document generation and, in the component inspector, object ID. New owners cannot inherit private numeric text or pressed controls; public IDs stay stable. The earlier `TextStateKey` API remains compatible. Caller-owned editor/grid/tree/placement bindings retain their established contracts and refuse this key.

Exactly four members differ from `178945c`: `ui/node.go`, `ui/control_state_key_test.go`, `drivers/cogent/reconcile.go`, `drivers/cogent/control_state_key_test.go`. The archive has 7,453 hashed files, 18,625,527 bytes, SHA-256 `53b824706108f481ec1dcca842a72fee8ee7797ad2127e00489fda4e645ecf04`, manifest SHA-256 `9e30d55182850a11c54cb754782c3a7a66af2c3487cb0ccb85f874c8e31beb9e`. Prepared dependencies/bootstrap are unchanged; all four native replacements point to `build/.vellum-sdk-ade66eb`.

Framework root/driver race/vet and focused offscreen checks pass. Consumer regressions cover wrong-owner numeric drafts, held Rotate/Restart, normal/fresh input and one-click Apply. The Apply repair uses only consecutive accepted value-field responses to continue its rendered context; intervening actor/field/document changes still refuse. [Evidence](validation/control-owner/README.md) records source qualification. Clean [review /146](../research/ui-audit/first-version/control-owner-review-checkpoint.json) includes the repair and passes exact copied native smoke/four-adapter conformance, reference and pulse/save acceptance. Review /145 remains immutable and affected.

## Previous pinned integration: 178945c

Private SDK `178945c802f144f9313b43fa4c42c8cca4eaff05` adds optional `ui.Node.TextStateKey` for managed TextFields. ELab scopes it to document generation and, inside the component inspector, selected component ID. Old private text cannot survive into another logical owner simply because its public ID and controlled text match. Ordinary redraws and own accepted edits preserve the binding. Public commands/IDs, models and project format are unchanged; the SDK API addition is limited to this explicit text-state identity.

Exactly four SDK source members differ from `88dc650`: `ui/node.go`, `ui/text_state_key_test.go`, `drivers/cogent/reconcile.go` and `drivers/cogent/state_key_test.go`. Prepared upstream/typesetting and bootstrap sources are byte-identical. The archive has 7,451 hashed files, 18,622,757 bytes, SHA-256 `2bfc4dc6438251485cc4b52704f9b57d87ae1456ef9d88d22f66beebb47fc00c`, manifest SHA-256 `fff345ab03cc8f26a4c612fd7e55dfdc0471f3f5aca8bf14c4110f7f9020cfcb`. All four native replacements point to `build/.vellum-sdk-178945c`. Prior source/cache/archive and sibling checkout are preserved.

Framework root/driver race and vet, focused offscreen routing/native-field/lifetime tests and tagged vet pass. The consumer before/after uses real retained widgets and routed keyboard input; both cross-owner cases fail on the prior pin, while the repair preserves ordinary pending text and accepts fresh edits. [Qualification evidence](validation/text-field-state/README.md) distinguishes source validation from a replacement review package. Clean [review /145](../research/ui-audit/first-version/text-state-review-checkpoint.json) includes the repair and passes exact copied native smoke/four-adapter conformance, reference and pulse/save acceptance. Review /144 remains immutable and affected.

## Previous pinned integration: 88dc650

ELab now pins private source `88dc650884f97629a6370e8408fd9e9242f1a7e0`, a narrow slider-context repair on `bc6b456`. A held potentiometer drag could borrow a newer callback after repaint and overwrite an intervening agent edit or replacement document. The repair retains the callback from gesture start through release and queued delivery, allowing ELab's existing context guards to refuse the old action. Discrete controls retain their established delivery semantics; no public API, model or project format changes.

The archive has 7,449 hashed files, 18,621,213 bytes, SHA-256 `c5be13c5e74e555ebd3e13f82626d6c182bba34d08f034259538aea053a1c13a`. Its manifest SHA-256 is `cc39252c78acf160eb95b6b1a603e1b22aa4c716b0b2d0c1ab2a58c42b202d8c`. Exactly two source members differ: `drivers/cogent/controls.go` and its new `control_context_test.go`. Prepared upstream/typesetting and all five bootstrap sources are byte-identical. The lock and four native replacements point to `build/.vellum-sdk-88dc650`; previous archives/caches and the adjacent Vellum project are preserved.

Framework driver race tests/vet and focused offscreen control/lifetime/accessibility tests/vet pass. The consumer's real retained-widget reproduction fails against the previous pin and passes against the repair. [Evidence and consumer qualification](validation/held-slider-context/README.md) distinguish source validation from a replacement packaged app. Clean [review /144](../research/ui-audit/first-version/slider-review-checkpoint.json) now includes this repair and passes exact copied native smoke/four-adapter conformance plus reference/pulse/save acceptance. Review /143 remains immutable and contains the defect.

## Previous pinned integration: bc6b456

The first-version UI pass reproduced stale native accessibility bounds after child-only scrolling. ELab now pins private source `bc6b456f0c71b0327d0c02629b109eabbbc8994c`, based on `47b7153`, with a narrow scroll-publication repair. The adjacent Vellum checkout is unchanged. This is the release-freeze exception for a demonstrated defect, not a broad framework upgrade.

The archive has 7,448 hashed files, 18,617,992 bytes, SHA-256 `f202a535b4cd931767e589e981c6e9265309ec262b48f28e9d5a57bbdf903fef`. Its manifest SHA-256 is `b3cc6914bb0557a297a2b34b75c2a076883c78ac65725348b83508e14c50b9a2`. Exactly four SDK source members differ from the previous archive: accessibility host publication, Scroll mounting, the new partial-scroll host and its regression test. All prepared upstream/typesetting files and verifier/bootstrap sources are unchanged. The lock and four native replacements point to `build/.vellum-sdk-bc6b456`; the previous archive/cache remain intact.

The [portable source patch](validation/vellum-ax-scroll.patch) and [first-version UI evidence](../research/ui-audit/first-version/README.md#scrolled-accessibility-repair) record the reproduction and validation. Normal driver race tests and vet, plus targeted offscreen accessibility/native lifetime/grid/modal regressions, pass. The same original native scrolled cursor edit succeeds without a theme repaint. Consumer qualification for the new pin is recorded at that UI checkpoint; the historical integration results below do not by themselves qualify this new pin.

## Previous pinned integration: 47b7153

ELab now consumes Vellum's clean local revision `762dcbf79480da2ebc4381d9a3ec3f4b7bcb6b98`, with the existing ELab mouse-position compatibility patch carried into private source commit `47b7153109735006305e3610520e489a953fcd97`. The adjacent Vellum checkout is unchanged. The new dependency is pinned by archive, manifest, exact source/file inventory and verifier hashes; its alpha version alone does not identify it.

Compared with the previous private SDK, the current framework adds intrinsic grid sizing, adaptive tile binning, exact-location glyph placement/coverage, owned accessibility snapshot copying and the key-event publication race repair. It also provides private Mac GPU overlay packaging. This ELab integration uses the standard private SDK and existing native module versions.

The local mouse fix was absent from the current framework source. An isolated source copy carries the same patch as stage 78 after the 77 framework stages. Preparation verifies every stage's declared input/output hashes. The mouse hook, tests and three affected prepared desktop files are byte-identical to the prior SDK; the patch bytes are unchanged. Source identity, zero-context reconstruction patch (`git apply --unidiff-zero`), raw private commit and old/new SDK member hash differences are retained in [the evidence](validation/vellum-refresh-validation.json).

The new archive contains 7,446 hashed files and is 18,615,050 bytes, SHA-256 `931d3a7217dc2f63618323fc523100fc6cee57000b49ad55df2bd4b91a3cbf68`. `third_party/vellum/sdk.lock.json` and all four native replacements point to `build/.vellum-sdk-47b7153`. The previous archive/cache remain intact. New verifier imports require five explicitly pinned bootstrap sources, including `native_gpu.py` and `native_archive.py`; missing/extra sets refuse before executing a subprocess. No optional native GPU binary overlay is present in this archive.

Framework preparation tests, mouse/key race tests, driver fallback/field tests and all 29 invisible Cocoa coordinate/refusal checks pass. ELab's full lint/imports/Staticcheck, 38 Python tests, root/native race/vet, packaged smoke and four-adapter electronics checks qualify this consumer. The same checkpoint repairs [batch selection normalization and documents history](edit-history-contracts.md). Recorded/live RC references, exact captures, saved files and fresh reopening are independently checked.

The separate 0.121 /138 app is unpromoted. Canonical/preserved/previous app bundles retain their exact identities. This checkpoint does not certify a public framework/app release, signing/notarization, native component distribution rights, physical multi-display input, full accessibility, general hardware, visible latency or participant acceptance.

## Native legacy/Cocoa text handoff · October 18

The source pin moves from `677548d` to isolated clean `274f5cc9b75873b2ad01fb86249bbda800bd568f` for a reproduced native typing blocker. A fallback key now queues a validated empty acknowledgment after that key in the originating toolkit FIFO. Cocoa stays in fallback routing until the owner acknowledges the key and publishes matching text/selection. Capacity refusal preserves the pending handoff and retries during publication. This prevents partial legacy text and later Cocoa insertions from using incompatible baselines; native expected-text, document/run and command/value guards are preserved.

Three real Cocoa no-window fixtures fail on the previous source and pass after repair (initial handoff, Select All and capacity retry). SDK driver race/vet and focused field-ordering/malformed-marker checks pass. A development native replay accepts all six previously tested cursor entries and a complete interval time, matching the exact packaged agent while preserving circuit/physics/document/history and saved bytes. [Reconstruction, test scopes and consumer gate](validation/native-legacy-handoff/README.md). This is a source repair, not replacement-app qualification; preserved /157 contains the old SDK. A clean app/agent package, exact copied-runtime checks and owner/broader acceptance remain separate. No broad framework upgrade, new command, device/model or format change is included.

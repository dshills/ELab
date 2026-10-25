# Pinned Vellum SDK from public source

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
ELab adjusts explicit action padding for the new reserved keyboard-focus border.
Four compatibility dependency stages follow upstream's 106 stages, numbered
107–110; preparation verifies the complete ordered recipe and each input/output
hash. The private derived source is `218c0dec8cfb52de4d63b91e0701f340396b7d84`.

The active archive is `vellum-f9dee54-elab2.tar.gz`; all four native module
replacements resolve into `build/.vellum-sdk-f9dee54-elab2`. The lock pins source, complete
archive/manifest, compatibility patch and all bootstrap scripts before execution.
The SDK is a **private compatibility build**, not an upstream release or a public
redistribution approval. No new widget adoption, electronics behavior, user/agent
command or project-format change is included. Previous SDKs and qualified app
packages remain preserved. See [update checks](../../docs/validation/vellum-f9dee54-update/README.md) for evidence
and its exact native/package scope.

## Historical integration records

ELab now consumes [dshills/vellum](https://github.com/dshills/vellum), pinned to public commit `e58d40581fe21d1de53fd54cab1bcf6acf11c163` (October 21, 2026). There are no upstream release tags at the time of this update. The public framework still documents prepared Cogent/typesetting sources and explicit local module replacements as its supported native consumer setup; replacing only the module URL is insufficient.

The public snapshot omits APIs and input fixes already used by ELab. The checked-in [compatibility patch](patches/elab-public-e58d405-compat.patch) preserves control/text ownership keys, numeric settlement, held button/slider context, legacy text acknowledgments and partial-scroll accessibility publication. It includes their existing regression tests. These changes are applied to public source in an isolated local commit, `96e22bf25809148b9a28169baf1b24cba72af475`; they are not claimed to be upstream. The one merge overlap retains both public numeric pointer-focus behavior and ELab's field-settlement epoch. No electronics feature or application command changes.

`vellum-e58d405-elab1.tar.gz` is the resulting prepared SDK. Lock schema 2 records the public repository/base commit, compatibility patch checksum, derived source commit, complete archive/manifest digests and six verifier/import scripts. `python3 scripts/prepare-vellum.py` checks provenance and archive identity before executing bootstrap code, then verifies every SDK member. All four native replacements resolve into `build/.vellum-sdk-e58d405-elab1`. Unknown or edited caches are refused and preserved.

The driver requirement names public pseudo-version `v0.0.0-20261021143700-e58d40581fe2`. The root module's `v0.0.0` requirement is retained by upstream's driver requirement and Go module selection; both modules are replaced by the same exact locked SDK. The local replacements carry the compatibility patch; a module version alone does not describe those derived bytes.

## macOS pointer repair (ELab 0.180.1)

That repair's compatibility source is `6ebc97bff24817b63586c920d42700c0761db304`.
The [complete compatibility patch](patches/elab-public-e58d405-pointer-compat.patch)
adds standard macOS arrow, I-beam, pointing-hand, crosshair and horizontal/vertical
resize cursors. Their artwork and accessibility sizing come from macOS instead
of Vellum's 32dp raster artwork. Unsupported/custom shapes retain the existing
raster path. Per-window native ownership, request retirement and repeated-hover
fast paths remain enforced. Regression tests cover cache bounds, allocation
failure, teardown, switching back to cached raster artwork and repeated hover.

`vellum-e58d405-elab2.tar.gz` and `build/.vellum-sdk-e58d405-elab2` preserve that
repair's archive/cache. The lock records the active derived patch and inventory.
The preceding `elab1` archive, cache and compatibility patch remain preserved;
the preceding paragraphs describe that historical pin. ELab separately restores
the default system title bar and removes the unified-header inset so normal
window dragging is owned by macOS. No electronics or agent command changes.

## macOS wheel and scrollbar input repair (ELab 0.181.1)

That repair used source `ebb20812644fb179af280fe2a1683490eef294e5`, with the
[complete compatibility patch](patches/elab-public-e58d405-event-position-compat.patch),
`vellum-e58d405-elab4.tar.gz` and `build/.vellum-sdk-e58d405-elab4`.
Button and wheel callbacks use the matching current AppKit event's window-local
coordinates instead of polling a later hardware cursor position. The native
helper requires the main thread, the exact owning window and an eligible event
type. Logical coordinates pass through the existing finite/range/DPR admission
once. GLFW disabled-mode virtual coordinates and the unavailable-event fallback
remain intact. The prepared dependency stage records input/output hashes and
regression tests. A second stage removes a redundant wheel inversion from the
scrollbar handler; regression tests compare frame and scrollbar movement in
both directions and at the bounds. Prior SDK archives and caches remain unchanged.

## Coalesced divider drag repair (ELab 0.181.2)

That preliminary repair used source `71605ec3c9bbf774686395d181809a02ec2d667b`, with the
[complete compatibility patch](patches/elab-public-e58d405-ui-input-compat.patch),
`vellum-e58d405-elab5.tar.gz` and `build/.vellum-sdk-e58d405-elab5`.
The controlled split-pane handle now accepts SlideStart and SlideStop positions
as well as SlideMove. A fast/coalesced drag with one movement no longer loses
its first or final position. Secondary buttons do not resize the split. The
existing numeric bounds, owner admission and retained pane/draft behavior remain.
The regression uses actual manual-window mouse input with one movement and a
distinct release position, in both orientations. This builds on both preceding
macOS event-position and scrollbar wheel fixes; old archives remain preserved.

## Fast release repair (ELab 0.181.3)

That repair's source is `899b2ba3bc01c9708385a2b10d4f8154a5f257e2`, with the
[complete compatibility patch](patches/elab-public-e58d405-ui-release-compat.patch),
`vellum-e58d405-elab6.tar.gz` and `build/.vellum-sdk-e58d405-elab6`.
Native testing of /203 found that a fast gesture can release before any SlideStart.
The controlled split now also retains primary MouseDown/MouseUp input and applies
the final position when native slide classification never starts. Classified
slides retain their start/move/stop path. Both paths retain bounds and owner
admission. A new manual-window test performs movement and release without advancing
the drag timer, for both orientations; the existing classified-slide tests remain.
All preceding compatibility fixes are included; historical archives are preserved.

## Session-colored disclosure headings

That repair's source is `211e3fe22aa2069721e77c607d0005f78fc0d9f0`, with the
[complete compatibility patch](patches/elab-public-e58d405-disclosure-theme-compat.patch),
`vellum-e58d405-elab7.tar.gz` and `build/.vellum-sdk-e58d405-elab7`.
Disclosure headings use the native action-button style to inherit their managed
foreground. The previous text-button style replaced it with the process-wide
primary color, ignoring a consumer's dark session palette. A manual-window
regression paints two independently colored disclosures and swaps their styles
after mounting; existing ownership, accessibility and keyboard tests remain.
Only the disclosure implementation, its test and the generated SDK provenance
note differ from elab6. Dependency bytes, notices and bootstrap sources are
unchanged. ELab separately supplies each disclosure's session surface color.

This is an ELab compatibility build from the same public base, not a new upstream
release. The /204 and /205 packages retain their qualified elab6 runtime. The
elab7 source/offscreen checks do not qualify a newly launched native app or
replace those preserved downloads. [Repair and evidence](../../research/85-disclosure-theme.md).

## Per-window backgrounds and stable divider grips

That repair's source is `a4ae0319ec1f1ed1ec5f85aee825af893a2a4f8a`, with the
[complete compatibility patch](patches/elab-public-e58d405-window-background-and-divider-compat.patch),
`vellum-e58d405-elab8.tar.gz` and `build/.vellum-sdk-e58d405-elab8`.
Each host scene follows its managed root's opaque background without changing
its existing content inset. Clear/translucent roots fall back to the configured
theme or native surface. Per-window snapshots avoid global palette changes.

The split handle retains the accepted fraction and physical press position.
A focus click submits no resize; a drag applies its movement relative to the
grip. Existing fast-release and classified-slide paths remain. Manual-window
regressions cover both orientations, exact pane movement, independent window
palettes, palette replacement, fallback and unchanged content bounds. A gallery
metric is updated for elab7's shadow-free disclosure action, retaining the
exact geometry and enablement assertions. All dependencies, notices and the six
bootstrap sources are unchanged. Previous archives/caches remain preserved.

These are local compatibility repairs from the same public base. Source and
offscreen checks do not replace fresh native acceptance or the qualified /205
download. [Repair and evidence](../../research/86-window-background-and-divider-grip.md).

## Text-field Undo boundary

That repair's source is `8b8eb73bff531e911448236a576c704c5da97db2`, with the
[complete compatibility patch](patches/elab-public-e58d405-text-undo-boundary-compat.patch),
`vellum-e58d405-elab9.tar.gz` and `build/.vellum-sdk-e58d405-elab9`.
Compatibility stage 0109 preserves initial text and selection when the field's
Undo history is empty or exhausted. Normal Undo/Redo and a new edit after Undo
are covered, including Unicode text. Existing text/IME/native-edit tests pass.

Two preparation tests now locate historical stages by identifier while retaining
their checksum assertions and requiring all 109 stages in order. Nine exported
members change, including five additions; every other member and all six
bootstrap sources match elab8. The export comparison refused generated golden
images from a supplemental upstream test run before the lock was changed.
The final archive uses dependencies re-prepared from the pinned recipe.

The framework's root/driver race and vet checks, normal and offscreen, pass.
A supplemental full upstream Core race suite reports existing render/menu
races with both original and changed text-field sources; that separate suite
is not claimed green. Previous archives/caches and the qualified /205 download
are preserved. Fresh native acceptance remains separate.
[Repair and evidence](../../research/93-text-undo-boundary.md).

## Number-field wheel input

That repair used source `2a0f783acb7f2ef20a6003c3277309ef6490be3c`, with the
[complete compatibility patch](patches/elab-public-e58d405-number-field-wheel-compat.patch),
`vellum-e58d405-elab10.tar.gz` and `build/.vellum-sdk-e58d405-elab10`.
Managed number fields leave wheel input to their enclosing scroll pane, even
when focused. Standalone fields ignore the wheel. Typed drafts and accepted
values are preserved; ordinary typing, step buttons and arrow keys remain.
The change removes the inherited spinner wheel handlers without changing
upstream Core or the public UI/agent API.

The old native /206 session reproduces a focused playback-rate edit from 0.1
to 0.13000001, while the unfocused field scrolls normally. Current-source
retained-widget cases reproduce the same issue for live initial voltage and
recorded playback. Driver checks preserve a private unfinished draft, move the
containing viewport and keep explicit Up/Down increments available.

Exactly three exported members change: controls.go, its new regression and the
provenance note. The 8,434-member manifest matches every file. Core/typesetting,
notices and all six bootstrap sources match elab9; historical archives/caches
remain preserved. Root/driver normal/offscreen race and vet gates pass, with
local Unix socket access for the workbench example tests. Source/offscreen
checks do not qualify a newly launched native app or replace the /205 download.
[Repair and evidence](../../research/94-number-field-wheel.md).

The exporter still marks SDK artifacts as private builds because public SDK/distribution qualification remains incomplete. **Public source availability is distinct from a qualified public SDK release.** MIT licensing, upstream notices, `SDK_MANIFEST.json` and `PRIVATE_BUILD.txt` remain included. Existing archives/caches, app bundles and their historical validation are unchanged. This dependency update does not qualify or replace the /189 owner app or /193 candidate.

Run SDK Python utilities with `python3 -B` to preserve cache integrity. Normal build/lint/check scripts prepare the SDK automatically. Portable headless Go commands remain independent of Vellum. [Update validation](../../docs/validation/vellum-public-source-update/README.md) records the compile incompatibility, preserved fixes and final checks.

## Previous pin: b22478c

ELab source builds consume `vellum-b22478c.tar.gz`, exported from clean isolated commit `b22478cf8cec8dda1ccdcd93d6ff6b62e2cc11de`, based on the preserved `274f5cc`. The numeric-field settlement barrier commits a complete coordinate draft before an action that does not move pointer focus. It retains originating-window FIFO ordering, numeric validation, record/window lifetime and application command/document guards. Four source/test members change; prepared dependencies, notices and all five bootstrap sources are unchanged. The preceding Cocoa/legacy handoff repair remains included. The sibling project, prior archives and prepared caches are preserved. [Source repair and qualification](../../docs/validation/number-field-placement/README.md), [integration history](../../docs/vellum-refresh.md).

The previous lock pinned the complete archive, manifest, source identity and five verifier/import scripts. Its verified cache remains under `build/.vellum-sdk-b22478c`; active module replacements now use the public-source pin above. Never edit a verified build cache to develop framework changes.

The archive carries `LICENSE`, `THIRD_PARTY_NOTICES/`, its inventory, `SDK_MANIFEST.json` and `PRIVATE_BUILD.txt`. Preserve those files. The exporter labels this a private SDK; it is not a public framework release or evidence of platform/distribution acceptance. Its internal version `v0.1.0-alpha.2` alone does not identify the snapshot; use the lock's commit and SHA-256.

Use `python3 -B` when running SDK Python utilities directly so they do not create bytecode files inside the verified cache. Normal ELab build/check scripts handle preparation. Portable headless Go commands remain independent of the SDK.

To update, select a fixed public commit, review compatibility patches against it, prepare dependencies and export a checksum-pinned SDK with the upstream release script. Update the archive, provenance lock, bootstrap set and four `desktop/go.mod` replacements together. Run source/offscreen consumer checks; native acceptance remains separate when no desktop session is available. The [historical mouse patch and reconstruction hashes](../../docs/validation/native-mouse-sdk-sources.json) target their recorded base and must not be applied blindly to a different revision.

The separate [packaged application baseline](../../docs/vellum-baseline-current.json) records the last qualified app bundle. This source pin does not change that bundle or its release status.


## Focus visibility and numeric drafts during layout

That repair used source `b8ffabbe3f6050415247760863e66e1f59c33d87`,
with the [complete patch](patches/elab-public-e58d405-focus-scroll-compat.patch),
`vellum-e58d405-elab12.tar.gz` and `build/.vellum-sdk-e58d405-elab12`.
After layout inserts content above a focused control, reveal that control once
if its ownership and scroll position still match. Deliberate scrolling away,
a replaced/disabled editor or a different focus owner cancels the old reveal.
The managed number field also retains private text across parent layout changes;
its accepted model value is still set explicitly by managed synchronization.

The pinned Spinner updater layout is checked before removing its unconditional
value-to-text refresh. Intentional typing, buttons and keys retain their existing
paths. This changes four existing driver files and adds the helper and behavioral
regressions. All 8,437 exported files match their manifest. Upstream dependency
bytes, notices and all bootstrap scripts are unchanged; earlier archives/caches
and the qualified /205 download are preserved. Framework normal/offscreen race
and vet checks pass. Native acceptance of the rebuilt app is separate.


## Decimal number stepping

That arithmetic repair used source `35d9d8d67b053adce223b714c8b62164119d3373`,
with the [complete patch](patches/elab-public-e58d405-decimal-step-compat.patch),
`vellum-e58d405-elab13.tar.gz` and `build/.vellum-sdk-e58d405-elab13`.
Managed numeric buttons and keys step the displayed decimal value and increment
using exact rational arithmetic, then round once to the existing native float32.
Thus repeated 0.001 steps from 0.03 reach 0.031 and 0.032, without binary-addition
artifacts. The text remains a round-trip representation of the stored native
value; display precision is not reduced. Existing typed and agent-supplied
values are not normalized or migrated.

Native wrap behavior, tight bounds, off-grid values, private drafts and wheel
scrolling remain covered. The seven driver cases, focused ownership/precision
checks and both live/recorded production-control regressions pass. Full framework
preparation and normal/offscreen race/vet pass. All 8,439 exported files match;
only the factory line, helper, new regression and generated provenance differ
from elab12. Dependency/notices/bootstrap bytes, earlier archives/caches and
/205 remain unchanged. Full application and changed-candidate native checks
are separate from those framework results.

## Queued field callback ownership

The active compatibility source is `23063545f871d8de0724077744d53f8f90d32c69`,
with the [complete patch](patches/elab-public-e58d405-field-origin-compat.patch),
`vellum-e58d405-elab14.tar.gz` and `build/.vellum-sdk-e58d405-elab14`.
Queued text and number commits retain the callback present when their native
Change event was admitted. A later descriptor repaint cannot lend that old edit
a newer form's request context. Future independent edits use the fresh callback;
settlement epochs, lifetime checks and other discrete-control behavior remain.

Both queue regressions fail before the repair and pass afterward. Full framework
preparation, normal/offscreen race and vet checks pass. Every one of 8,440 SDK
members matches its manifest; only controls.go, its new regression and generated
provenance differ from elab13. Dependencies, notices, release metadata and bootstrap
scripts are unchanged. ELab application checks and changed-candidate native
qualification remain separate. The existing /223 app and /205 download are
preserved. [Repair status](../../research/110-queued-field-origin.md).

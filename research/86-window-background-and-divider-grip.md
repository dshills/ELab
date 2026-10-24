# Window background and divider grip

October 23, 2026. Work in progress after the clean held /208 candidate. Current
download /205 and all prior candidates remain preserved. No fresh graphical
launch or permission request is used.

## Native baseline

The already-running owned /206 session switches from Light to Dark through its
native theme menu. The inspector scrollbar drags correctly and its track fits
the native dark content; the partial offscreen inspector's white track is not
sufficient evidence for a scrollbar patch. A separate white 8-pixel host inset
is clearly visible around the dark managed content. The test session is restored
to Light, paused, with the inspector back at its first controls. Owner sessions
and projects are untouched. Captures are in `research/ui-audit/window-background-209`.
They establish the baseline only, not changed /208 or /209 native behavior.

## Backdrop repair under validation

The Core host scene paints a default surface outside the managed root. The
private Vellum source now binds a per-window immutable background snapshot and
updates it after accepted root reconciliation. An opaque resolved root controls
the host background; translucent/unset roots fall back to the configured theme
or native default, avoiding a second application of alpha. The existing scene
inset remains unchanged. Native and manual backends use the same helper.

The source regression first captures white corners instead of the dark root.
The repair passes two simultaneous windows, palette swaps, configured/native
fallback, transparency and exact unchanged content bounds `(8,8)-(192,112)`.
ELab's consumer paint probe independently fails in both themes against elab7.
It uses the actual application root style with its schematic children omitted;
its images are paint probes, not full application screenshots.

## Divider bug found by broader checks

The first full offscreen framework run exposed a stale gallery metric: the
elab7 inherited-color disclosure action no longer reserves a shadow margin.
The same failure is reproduced against the unchanged verified elab7 cache.
The gallery now expects heading/field/button shrink of 0/2/6 pixels, retaining
its exact geometry, disablement and restoration assertions.

Once past that assertion, the gallery exposes an actual divider issue. A plain
focus click can submit a new ratio without moving. The independent regression
records 0.35 becoming 0.3791209 horizontally and 0.4019608 vertically. Absolute
pointer mapping loses the grip offset and includes native handle geometry.

The controlled split now retains its press position, accepted fraction and
span, then applies primary-axis movement relative to that grip. An unmoved
click submits nothing; pane/orientation replacement invalidates the grip.
Existing clamping, owner callbacks, keyboard and fast-release paths remain.
The new test also compares actual pane movement with physical pointer movement.
An older pointer fixture now targets deltas from its measured grip while
retaining the expected 0.75 ratio, pane geometry and private-draft assertions.

The already-running /206 native test window independently reproduces the
divider issue: a physical click at its visible handle, with no drag, changes
the accepted fraction from 0.65 to 0.66611844. Its accessibility value is restored
to 0.65 immediately afterward. This is native baseline evidence from the old
binary, not verification of the repaired binary. No application launch is used.

Focused split/background/disclosure race tests pass (3.323 s), and the full
structure gallery workflow passes at both tested DPIs (5.012 s). Earlier failing
runs remain in `build/window-background-209-tools/`. No new component, model,
public command or file format is added.

## Validated dependency and application source

The private framework source is committed cleanly at
`a4ae0319ec1f1ed1ec5f85aee825af893a2a4f8a`. ELab's active elab8 pin includes the
complete patch against the same public e58d405 base. All 8,428 SDK members are
verified; the nine changed members contain only the reviewed implementation,
regressions, gallery correction and generated provenance note. Dependencies,
notices and all six bootstrap sources are byte-identical. Previous archives
and caches remain preserved and verified.

The [full source gate passes](../docs/validation/window-background-209/README.md):
lint/staticcheck, 67 Python tests, complete root/desktop race and vet, required
mounted UI race and offscreen vet. Desktop race takes 181.456 s and mounted
race 156.936 s. The full framework normal/offscreen race suites and offscreen
vet pass as well. ELab's Light/Dark root probes pass after failing with elab7;
accepted service state and the exact existing content inset are preserved.
The native command ledger remains byte-identical with 166 public commands.

Fresh native acceptance of the accumulated /207 guidance/file editing, /208
disclosure and /209 background/grip changes remains separate. The existing
owned /206 session remains available; no unattended fresh-launch prompt is
queued and no further owner permission or review is requested. Preserve the
qualified /205 download until those changed native paths pass. No component,
model, command or format is added by this work.

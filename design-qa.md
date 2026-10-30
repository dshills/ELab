# Experimental Native UI design QA

final result: passed

Scope: the native adaptation documented in
[implementation-notes](exp-design/implementation-notes.md), on `exp-ui` only.
This result does **not** claim all fourteen acceptance criteria in the browser
Draft 1 specification, or four-pixel identity with its artboards.

## Visual truth and normalization

Sources: the four `exp-design/Native · …@1x.png` images, each **1440×900**, plus
`exp-design/exp-design-spec.md`. The supplied files were preserved verbatim.
Implementation: native ELab; no CSS/browser runtime. Primary comparison viewport
is **1440×900 content**, captured as **1440×932** including the actual 32px macOS
title bar. Only that OS title bar is cropped; no scaling, redrawing, state
fabrication or image replacement is used. The source's illustrated traffic lights
remain a known platform difference. Exported CUA JPEG bytes are decoded into real
PNG artifacts for the comparisons.

Source and implementation are joined in each **2880×900** comparison image,
reference left / implementation right:

- [Simulate + Spectrum](docs/validation/experimental-native-ui/simulate-spectrum-comparison.png).
  Paused amplifier, real selected probes; native data covers 0–8 ms, rather than
  the mock's later run. Engine/sample semantics remain unchanged.
- [Wire + Connect by name](docs/validation/experimental-native-ui/wire-connect-comparison.png).
  Wire tool and popup open. Native Inspect selects the speaker; the mock selects
  a dangling wire and proposes a different wiring operation. Compare applicable
  chrome and popup affordances, not the different object editor or graph geometry.
- [Document + Examples](docs/validation/experimental-native-ui/document-examples-comparison.png).
  Document selected, actual AppKit Examples menu open, Scope visible. Local files
  replace unsupported cloud/metadata controls; no fake storage status is shown.
- [Sweeps](docs/validation/experimental-native-ui/sweeps-comparison.png).
  Parts closed, VIN selected, popup open and a real 12-point AC result visible.
  The mock depicts a running transient-settled current sweep. Native AC results
  and source settings are preserved; running-state/pending-point parity is not claimed.

Focused comparisons: [toolbar](docs/validation/experimental-native-ui/toolbar-detail.png)
and [inspector](docs/validation/experimental-native-ui/inspector-detail-comparison.png).
Both contain actual source and implementation crops side by side. Native compact
Dark [1053×731 content](docs/validation/experimental-native-ui/dark-compact-drawers.png)
was checked with both drawers open, all three tabs and the retained sweep. The
existing native minimum wins over the draft's 1024px browser assumption.

## Findings and iteration history

No actionable P0/P1/P2 findings remain **within the adapted scope**. Earlier
findings and fixes are retained here rather than treating the first capture as a pass.

| Severity / earlier finding | Fix and post-fix evidence |
| --- | --- |
| P1: primary Resume had a square clipped right edge | Fixed outer action slot; rounded-end paint regression passes; final toolbar crop shows both ends |
| P1: floating controls could place a part underneath | Native reproduction followed by bridge + fallback exclusion; native retest preserved 17 parts / 16 wires / clean state |
| P1: spectrum editors disappeared into overflow | Bounded editor rows retain original callbacks; mounted field/focus/peak/agent tests pass |
| P1: popup selectors opened nothing | Retained popup-scene selectors; native terminal list opens and accepts a choice |
| P1: Sweeps was visually blank despite AX controls | Explicit scroll allocation; mounted paint allocation passes and final sweep comparison shows working fields/result |
| P2: selected controls lost contrast, Fit became faint, Scope/Sweep could both look selected | Central selected tokens, neutral Fit and one instrument selection; theme paint tests plus final sweep comparison |
| P2: live availability moved buttons and readouts | Stable slots, fixed readout allocation; mounted live/paused/FFT update tests pass |
| P2: compact layout caused horizontal overflow and clipped actions | Reduced title allocation, named icon actions, bounded center and vertical-only inspector scrolling; final /302 compact capture |
| P2: old allocation stranded floating controls after resize | Layout-time custom units use current parent allocation; resize/density tests and full↔compact native retests |
| P2: long inspector disclosure captions clipped | Short visible captions with full accessibility descriptions; final inspector and compact captures |

Raw negative/intermediate evidence remains under ignored `build/native-design`.
The [validation record](docs/validation/experimental-native-ui/README.md) separates
software callback tests, mounted routing and actual native input observations.

## Required fidelity surfaces

- **Fonts/typography:** Helvetica Neue native controls at 13px, 11px secondary
  text and a 17px selected-object title preserve the draft's hierarchy. Bundled
  antialiased Go outline glyphs replace bitmap sheet text. They are a documented
  canvas fallback, not an assertion of SF Pro identity. Caption wrapping is
  bounded before shaping; compact title truncation intentionally yields to commands.
- **Spacing/layout:** full-height side rails, unified 52px toolbar, paper sheet,
  floating pills and lower instrument dock follow the composition. Drawer widths
  and the 70% split remain adjustable. Native focus allowances and dividers make
  some controls/region boundaries larger than the browser reference; exact 4px
  boundary parity is not asserted. Persistent actions fit the compact viewport.
- **Colors/tokens:** central Go tokens implement neutral Light/Dark surfaces,
  blue primary/selection and distinct trace colors. System/Apollo remain usable
  through the same theme path. Actual pill foreground contrast is tested at ≥4.5:1.
  Electrical coloring preferences remain authoritative.
- **Image/assets:** original electrical symbols/meshes remain real schematic
  content. Bundled standard Material symbols supply UI icons and paper thumbnails;
  no decorative image/logo or fake waveform was substituted. Combined full views
  are downscaled only by the chat viewer; saved files preserve 1:1 comparison pixels.
- **Copy/content:** real analysis names, source advice, local Save/Save As and
  actual trace values remain. No non-working cloud, solver-quality, metadata or
  automatic-retention controls are painted to imitate the mock. Existing advanced
  operations remain reachable via disclosures, tool chooser and native menus.

## Interactions and evidence limits

Checked: native tool/zoom click routing, popup Cancel/Escape, terminal selection,
Examples menu, drawer close/reopen, compact resizing, Light/Dark, real AC sweep,
Scope/Spectrum/Sweep selection, plus mounted theme/focus/editing/live-update and
user/agent parity regressions. The app is native: browser-console checks do not
apply. Local runtime logs were inspected; ordinary linker duplicate-library
warnings are toolchain output, not rendering failures.

The draft's new product/solver/storage features are explicitly out of scope in
the implementation notes. Existing native keyboard/IME/hardware-rate acceptance
limits remain open; this report does not turn source or mounted evidence into
OS-queue qualification.

## Follow-up polish

P3: at the narrowest size with both drawers open, the document title can be very
abbreviated. Full accessible names remain; closing either drawer restores space.
Further optical matching of SF typography and popup-arrow/shadow treatment can be
considered after this experiment is reviewed.

## Implementation checklist

- [x] Preserve supplied references and isolate changes on `exp-ui`.
- [x] Implement the native composition and retain shared commands/functionality.
- [x] Reproduce and fix interaction, painting and responsive regressions.
- [x] Compare all four full views plus focused controls and compact Dark.
- [x] Record deliberate adaptations and test/platform limits.
- [x] Internal code review; no external provider, merge or push.

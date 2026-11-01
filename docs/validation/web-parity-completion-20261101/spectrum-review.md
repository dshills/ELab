# Spectrum review resolution

Prism reviewed the staged nine-file milestone with the configured Wafer/GLM-5.3
defaults and the owner-requested larger output budget. The receipt records full
coverage, no excluded files, no skipped chunks and no truncated input. There were
zero high, five medium and eight low findings. This is a completed review, not a
claim that the reviewer returned no findings.

## Accepted findings

- Right-edge cursor labels could clip: clamp measured label width and position
  inside the plot; the canvas regression checks a B marker at the last bin.
- Canvas and peak actions could reenter while another command was pending: guard
  the shared action path and disable editable controls while pending.
- Improve finite-coordinate test failure messages to identify the canvas method.

Internal inspection also aligned the initial live probe with the native automatic
selection and restored the Scope owner/header immediately when switching tabs
while paused. Browser verification confirms RECORDING, visible Scope controls,
and no stray aria-selected attribute on the analysis container after returning.

These small fixes followed the attached Prism receipt. They were internally
reviewed and checked with the 17 frontend tests, syntax checks and diff checks;
the receipt describes the pre-fix staged snapshot rather than asserting a second
external review of these fixes.

## Medium findings resolved by inspection

- Controller initialization: boot constructs the Spectrum controller before
  binding instrument handlers. No inline handler can invoke it earlier.
- Missing replacement submit handler: spectrumWorkbench installs its own delegated
  submit listener. preventDefault in the legacy listener does not stop propagation.
  Recorded analysis and live application were exercised in the browser.
- CSS-pixel coordinate assumption: prepareCanvas returns bounding-rectangle
  dimensions and applies the device-pixel-ratio context transform. Pointer and
  plot coordinates therefore use the same CSS-pixel domain on Retina displays.
- Two-command clear ownership: native cursor edits do not change document
  generation or source identity. A real intervening source change deliberately
  refuses the second edit instead of clearing another capture's cursors.
- Scope/analysis CSS overlap: setInstrument enforces mutually exclusive panel
  visibility; collapse handling preserves this invariant.

## Remaining low findings resolved by inspection

- Instrument selectors are intentionally scoped to the tabs; there are no other
  intended data-instrument action controls. The analysis container also carries
  that attribute solely as an ownership marker.
- Scope tab switching has no same-tab early return; Scope repaints on switching.
- Cursor markers outside the plotted data interval remain clipped intentionally;
  native cursor availability/reasons are shown separately.
- Empty live probe and zero fundamental are accepted native defaults (automatic
  selected probe and no distortion estimate). They are not mandatory-field errors.
- The settings element is a direct child of the wrapping flex toolbar, so its
  flex-basis applies. The broader descendant selector is not evidence otherwise.

## Supporting validation

The race-enabled web suite passed, including native FFT/THD, exact peak/cursor
bins, CSV rows and live/recorded independence. The complete shared race suite
passed before the final presentation-only refinements (workbench 330 seconds).
Python quality-runner tests passed (104), as did requirement and native-command
inventory checks. No remote CI run or physical printer output is claimed.

The Prism change is installed locally: initial output allowance 32,768, final
unsplittable retry 65,536, Wafer HTTP request timeout 900 seconds. Review and
provider race tests and vet passed; a 24,000-token regression completes without
splitting. Its four source changes remain uncommitted in the separate Prism
repository; they are not part of this ELab commit.

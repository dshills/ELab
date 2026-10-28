# Internal code review

Reviewed locally; no Prism/external provider. Scope: all changed Go source,
new tests, command inventory, and the user/agent documentation.

## Findings resolved before commit

1. A frequency commit on focus loss advanced command revision before an already
   rendered Clear action. Mounted reproduction failed with expected revision 8,
   current 9. Measurement actions now bridge only consecutive accepted spectrum
   field responses belonging to their exact source/document. A later agent edit
   breaks that chain. Mounted one-click Clear and adversarial retained-action
   tests pass. This does not substitute the latest agent revision.
2. An uncommitted frequency survived a voltage-to-current source switch.
   Mounted reproduction retained `100 Hz` in the new current editor. Controls
   now use stable source/run/settings ownership, excluding ordinary acquisition
   and Hold. The old draft retires while rolling frame updates preserve a draft.
   The initial fixture accidentally awaited a new held frame; corrected to
   resume derivation before the actual draft failure was reproduced.
3. A nullable measurement view could be dereferenced when publishing a plot
   context. Plot contexts now choose A safely if the view is absent.
4. Marker visibility discovery initially inherited the cursor action contract.
   Separate contracts now describe only each command's accepted parameters;
   a regression test prevents visibility from claiming frequency is required.
5. Fixed measurement readouts were initially wider than the existing summary.
   Final readouts retain the existing 800-unit width. All new controls fit at
   900×600 and remain stationary in every theme, including warm/running updates.

## Reviewed invariants

- Peak candidates exclude DC and display-floor bins; local plateaus choose the
  first bin, amplitude ties lower frequency, top five within -60 dB relative.
  Known Fourier-series tests establish 10/20/30 Hz with 1/.1/.05 peak amplitude.
  Markers are not interpolated frequency estimates.
- Cursors use nearest actual bins, exact midpoint ties lower frequency. A/B
  amplitude is peak V/A, not RMS. Deltas are signed B−A. Floor-limited dB ratios
  are omitted rather than treating the display clamp as an exact reading.
- Service mutex owns stored anchors. Public snapshots and clones detach nested
  cursors/readings/peak slices/delta pointers. Bounded derivation performs no
  FFT and never advances acquisition or simulation time.
- Live anchors bind to document/electrical/run/actuation/settings/probe/window
  key, not rolling result ID. Recorded anchors bind to exact captured result ID.
  Hold preserves valid measurements. Source changes and document replacement
  cannot borrow retired positions; immutable captured exports stay unchanged.
- Plot callbacks capture command/document context before owner dispatch, retire
  on remount, bridge only their own accepted drag writes, reject invalid/right
  input and guard hidden/modal panes. Recorded clicks pin the result; live
  clicks use the latest valid frame. Failed preparation clears its input frame.
  Actual custom plot events are not qualified by the manual driver; direct
  gesture tests and mesh preparation are narrower evidence.
- Both commands use ordinary service admission/retry/event paths; native UI
  fields/actions call these commands. Real stdio/socket tests establish equality
  and exactly-once retry behavior. Compact discovery remains 65,457 bytes,
  under the unchanged 65,536-byte budget, with 226 commands. Shared schema
  references keep detailed definitions available without widening limits.
- No component/model/integration/solver/sample-history limit changes. FFT and
  analysis function bodies are byte-identical to the baseline (identity report).
  No owner projects/settings/sessions, verified SDK caches or older qualified
  packages are changed. Native OS input/rate gates remain separate/open.

No unresolved source finding identified for this change. This is an internal
review and software-test conclusion, not a native OS presentation verdict.

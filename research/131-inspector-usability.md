# Competitive-improvement implementation order

The owner authorized starting the recommendations in
[the current comparison](130-current-competitive-comparison.md), with usability
and inspector layout first, shared user/agent commands, tests, internal review
and commits. This note tracks that work without treating research recommendations
or internal tests as completed user acceptance.

## 1. Effortless workflow and public entry — in progress

The first inspector batch is complete: basic value/Apply and Plot actions precede
advanced settings; op-amp live readings remain adjacent to their Plot controls.
Clipped checkbox captions and overlong accessible names were repaired for model,
loss, leakage, motor, transformer and current/load controls. Detailed descriptions
remain in Help. Conditional transistor explanations now agree with enabled model
settings. Plot no longer incorrectly promises to evict the sixth trace.

An observed field-to-Plot stale-action refusal is repaired through the existing
shared revision/field-settlement mechanism. Finishing the draft does not apply
it to the circuit. Tests cover direct agent parity, unchanged physics, held
input across newer agent work/document replacement, and fresh input afterward.
All 226 commands remain the same. No new component or solver-limit change.

[Evidence and native checks](../docs/validation/inspector-usability-20261029/README.md)
and [internal review](../docs/validation/inspector-usability-20261029/internal-review.md)
record passing final suites and precise limits. Previous release/SDK bytes are
preserved. A development test bundle is not a newly qualified release package.

Next within this priority: finish the novice-facing workflow/discovery checks
and prepare a current versioned ready-to-run distribution with accurate unsigned
macOS status. Keep the existing first-use protocol/results honest: internal
checks can proceed without owner review, but they cannot manufacture participant
outcomes. Historical native checks 3/6/9/11 remain separate and open.

## Remaining order

2. **Straightforward agent setup:** inspect the existing clients/adapters first,
   then provide the simplest supported connection and reproducible build/measure/
   compare workflow over the guarded service. Keep bounded reads and stale-edit
   refusals; measure task completion rather than command count.
3. **Useful circuit envelope:** qualify four-discrete-MOSFET and six-capacitor
   representative circuits and additional amplifier stages before expanding
   support. Preserve correctness, cancellation, saved editable refusals and
   numerical work guards.
4. **Real-device/model path:** define a bounded supported subset, pin mapping,
   source provenance and reference tests before adding model compatibility or
   simulator handoff. No promise of arbitrary manufacturer SPICE import.
5. **Hardware continuation:** define and qualify bounded editable KiCad handoff,
   explicit missing footprints/models, parts lists and experiment exports. MCU/
   HDL execution remains a later product decision.

Items 2–5 are authorized direction, not implemented by this inspector commit.
Work remains in order; owner review is not an engineering gate.

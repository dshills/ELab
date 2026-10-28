# Temperature/self-heating evidence

Implementation item 2, following committed transparent block support. Generic
resistor/diode/NPN/PNP temperature laws and optional independently budgeted thermal
RC; original disabled laws, capacities and work limits remain exact. Shared saved,
undoable thermal.configure control, ambient DC sweep, live separate-domain thermal
readings and captured-state validation. Two examples: self-heating resistor and
class-AB mismatch under imposed temperature. No manufacturer/package fidelity.

Independent tests cover analytical heat-RC response/refinement, coupled DC
quadratic, resistor coefficient, diode fixed-current forward drop, NPN/PNP
transport at three temperatures, full cross-domain stamp Jacobians, original
capacity boundaries and out-of-range hot-point refusal. Save/capture/reopen,
thermal-field forgery refusal, snapshots, undo, disabled equivalence, all-theme
inspector and stale-control/user-agent parity controls pass.

Prism initial complete 4/4 Anthropic Claude Opus 5.5 chunks, 24 files, no skips or
truncation: one high, one medium, six low. Dispositions:

- 6a164cc19c067193: premise disproven. Graph.Clone clones Thermal explicitly;
  buildSweepPlan owns a clone and every worker point owns another clone. New
  independent ambient sweep test asserts source temperature remains 27 °C.
- 8227c4a3b9c4265b: validateThermal already refuses all other kinds both at
  configuration and graph/file validation. New explicit unsupported-kind tests
  cover LED/Zener/Schottky/MOS/JFET/capacitor/op-amp.
- 10b7776578957e8d: Newton iterates and accepts using *all* equation rows, including
  scaled thermal rows, plus full-system voltage correction. Only the public
  electrical KCL diagnostic excludes thermal units. Separate thermal balance is
  reported and validated; analytical tests assert its actual residual.
- 730f13d84319b727: intentional existing device-configuration contract; complete
  valid fields required even when disabling, documented in schema/guide and UI
  submits them. Matches transistor/passive behavior; no silent invalid config.
- b62ab576a3f20732: now explicitly refuses missing private storage/current slot.
- 51f773790a6aad8d / eb4bf28beb64b93a / 107eff483c0f2a08: component constructors
  checked, derivative callback unshadowed, failed-test services cleaned up.

First unrestricted source run was attempted inside the ordinary sandbox. Local
Unix sockets/CUPS were denied, accounting for transport/conformance/spool failures.
It also exposed stale source inventory, catalog/batch counts and the missing
thermal semantic discovery contract; these are corrected before the next gate.
An initial test compile used Seconds rather than StepParams.Duration; corrected
fixture compilation, not a product failure. Earlier focused electrical and
ordinary desktop checks passed. Final commands/results are retained separately.
Native OS queue/hardware checks 3/6/9/11 remain open; no desktop acceptance claim.

Follow-up Prism is complete, 3/3 chunks over 9 unstaged repair/inventory files,
no skips/truncation. d0e1584af63b6e64: removed redundant successful-loop Close;
registered cleanup owns it. 8c906ddad7711f6b: disabled thermal controls return
immediately after the enable checkbox, before any value/self-heating fields are
constructed; no disabled-model value handler is reachable. Existing revision guards
refuse a retained enabled handler after an intervening disable. 36d4cfe9c6fcd52a:
removed redundant model-ID assignment. Final cleanup-only changes are directly
inspectable; no product model/behavior change after reviewed implementation.

The next root run passes socket/conformance/spooler/inventory checks and exposed
a second stale batch-count assertion: the old 33-entry expectation predates the
four block commands. Current reviewed allowlist is 38 including thermal.configure;
fixed-count and explicit advertised-command controls are updated. Additional final
capture validation compares thermal resistor/diode electrical branch readings to
the same reconstructed state, including unselected readings.

Final Prism completes 3/3 chunks, 10 files, no omissions, zero medium/high and four
low observations. 7b3619a29b8b5b98: schema comment clarified. 6f132915b892b975:
capacitorIndex(k) is nodes+sources+nonnegative range index, so negative is impossible.
ef9d90f37da0b113: validateRecordedBipolars independently verifies all terminal
currents/power/junction readings and thermal conductive heat after restoration.
920b946cd282dbd5: direct unsupported-kind test renamed to match its actual scope;
default configs are valid numeric/version configs, so kind gating causes refusal.

Final focused thermal/capture race passes 1.477 s; corrected contract/catalog race
passes 7.647 s. Ordinary/offscreen thermal/block/sweep controls pass 1.198/2.308 s,
lint passes. Full root control's non-workbench packages pass; its sole remaining
workbench failure was the explicitly corrected old batch-count assertion. A final
whole-root run is reserved for the completed five-item batch. All 30 protected
package/SDK hashes remain exact. No qualified package bytes replaced.

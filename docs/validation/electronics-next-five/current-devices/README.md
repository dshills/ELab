# Current sources and electronic loads — item 3

56 kinds /206 commands /73 examples. Three examples exercise pulsed LED current,
CC supply sag, and CP negative incremental resistance. Shared saved/batchable
configuration, history, full parameter/model metadata, sampling advice, probes,
recordings/projects and both desktop drivers are covered. Three-source, part,
node, storage, residual/error, attempt and wall limits remain unchanged.

Independent Norton/resistor and compliance checks, CC/CP supply roots, RC ODE,
finite-difference derivatives, passivity, low-voltage continuation, current limit,
current-driven capacitor, no-storage pulse classification, combined source budget,
initial terminal-voltage trace, model capture/reopen and forged-current rejection
pass. UI/agent configuration and step parity, stale input, all four theme trees
pass in ordinary and offscreen drivers. Discovery remains below 64 KiB. Inventory
is source evidence; native OS queue/rate acceptance 3/6/9/11 remains open.

Prism: only Anthropic claude-opus-5-5, fallback same provider/model; complete
8/8 chunks,47 files, no excluded/skipped/truncated bytes. All observations:

- 229dc92c3dd92251, c870343042628beb: strict constructors/import reject missing
  configs; nonetheless presentation now refuses/labels unavailable config rather
  than panicking, with malformed presentation controls.
- ca8ad0b2ebd257ed, 3c51e61a620a96df: added schema property guards.
- 3108f2173af3fd76: prefer a voltage source when kinds coexist; regression control
  puts current first and verifies V1 selection. Current level lives in Device,
  never Component.Value/voltage units. Current-only supply net is intentionally its
  positive terminal; initial trace uses solved voltage, not a fictitious setpoint.
- f495af733b2bafd4: waveformComponents already expands configured current/load
  pulses before nextSourceCorner/maximumWaveformStep. Actual corner and plateau
  tests cover this. Their primary-mode bypass is explicit and remains independent.
- 3a4dfd19815a894e: Go1.25.6 module has per-iteration range variables; no old Go
  compatibility is claimed. No change needed.
- cfb044a92caf57f7: model description moved to Model/Limitations.
- 0c51be4fb81ea6bd: retained documented piecewise load law. No reachable convergence
  failure shown; independent derivatives on each smooth branch and RC crossing
  the foldback knee pass. No speculative smoothing/model change.
- d7a3053b171a4a08: initial-readout test now requires a real pulsed transient and
  nonnil final result, not a vacuous DC condition.
- 20bd83d81804db06: consolidated waveform expansion through withDeviceWaveform.
- dbaabbd46f8ab6ac: atomic contract explicitly asserts both new commands.
- d9a1153376b8afff: removed redundant overwritten metadata assignment.
- a6043df1a52b402a: not an ideal current source. Declared 1 GΩ bleed/compliance
  gives a conductive terminal path; source-source/inductor cutsets do not share
  the proposed ideal-current assumption. No hidden gmin or limit increase added.
- c9e4b89548a05e2a: only cc/cp modes are admitted; CR/CV are not advertised.

Initial fixture/type-name/connection mistakes and failed setup commands remain
in build/electronics-next-five, excluded from passing evidence. No numerical
refusal is counted as a successful electrical test. Review-triggered changes
have fresh core/desktop race and lint receipts. Broad package qualification
follows all five items. Protected packages/SDK and owner session are preserved.

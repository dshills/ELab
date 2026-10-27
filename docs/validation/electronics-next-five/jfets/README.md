# Configurable N/P-channel JFETs — item 4

58 kinds /207 commands /76 examples. Generic depletion channel, symmetric
reversed drain/source, both polarities, explicit gate-source/gate-drain PN laws
plus declared bleeds, analytic derivatives and signed terminal KCL/heat. Shared
jfet.configure is saved, undoable and batchable. Models/configs, DC/transient/AC,
probes, drawings/printing and captures/projects are integrated. Existing combined
two-FET budget now explicitly includes MOSFETs/JFETs; no solver limit increased.

Independent square-law points, full finite-difference Jacobian, passivity,
polarity/reversal, gate forward current, self-bias quadratic, transconductance/
external-capacitor AC pole and explicit nonlinear scalar transient ODE pass.
Preamp plus complementary current-sink/source examples capture and reopen;
forged readings/models, missing/invalid configuration and capacity refuse. Both
desktop drivers pass shared inspector/agent configuration, electrical step/DC
parity, stale-control rejection and theme trees. Ordinary discovery stays below
64 KiB by sharing existing save/load/step definitions with the same full fields.

Prism used only Anthropic claude-opus-5-5, same fallback; all 7 chunks/46 files
complete, no skipped/excluded/truncated bytes. Findings:

- 21c27407343e0806 (medium): test clones the source graph under the service mutex
  for later component/export inspection. Fresh race controls cover the change.
- 439d600be124c82a: renderer now chooses the correct symbol generator directly,
  in both canvas and SVG; no discarded current-source symbol calculation.
- 456619bd4691b1d3: item 3's prose incorrectly said four sources; the unchanged
  engine constant is three, before and after every item in this batch. Corrected
  prose and a new exact MaxNetworkSources+1 boundary control document that fact.
- b20efa82902d682b: guide explicitly names version as the fourth required field.
- 23c530f8b0d131a1: raw local lint receipt retains its actual SDK path as evidence.
  It is not a runtime path dependency or secret; nothing is uploaded/published.
- 11271f0acdf9808e: JFET compile admission uses its explicit predicate, not hasMOS.
- c77245ec88dbbc88: MakeComponent supplies required JFET configuration; Compile
  and strict import validate it before numerical routines. New missing-config
  control verifies invalid-parameters diagnostic. Internal law requires validated
  compiled input, like existing model laws; no silent fallback model introduced.
- d96287fbd71ca1f2: source probe is explicitly positive *out* of source, like the
  existing MOSFET source probe. wire-current injection reverses it. Full terminal
  KCL tests verify gate+drain=source. Changing sign would break that declared law.
- 5193a33cf88f9546: compile inventory renamed to fets. It supplies only shared
  capacity, topology/classification and drain-output selection; JFETs have their
  own conducting gate paths, stamps, startup cuts and result validation. No MOS
  body diode or insulated-gate assumption is applied.

The initial DC capture fixture requested nonzero duration and was corrected to
an operating-point capture; preamp retains its positive-duration transient test.
Discovery initially exceeded64 KiB by97 bytes; pooling definitions fixes that
without changing the bound. Raw failures remain in build/electronics-next-five.
Native inventory is source/routed evidence; OS acceptance3/6/9/11 remains open.
Prior packages, verified SDK and owner session remain preserved. Full source and
matching-package checks follow all five items.

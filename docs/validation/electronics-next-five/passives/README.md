# Optional passive losses — item 2

Adds versioned capacitor series ESR / terminal-parallel leakage and standalone
inductor winding resistance. Defaults and disabling preserve ideal models.
Two new examples; 54 kinds, 204 commands, 70 examples. Shared passive.configure
is saved, undoable and batchable. Public probes report terminal quantities;
structured and inspector readings also show stored voltage, charging current
and separate heat losses. No extra storage slots or solver-limit changes.

Independent Thevenin RC/refinement, RL exponential, two-state RK4 RLC, AC complex
impedance, DC leakage, collocation Jacobian, energy/KCL, initial constraints,
internal-node voltage range, model capture/reopen and forged heat rejection pass.
Ordinary/offscreen desktop controls match public agent commands, reject stale
input, cover themes and preserve exact decimal SI endpoint values. Discovery
keeps the original 64 KiB limit through shared existing schemas. Native source
inventory is reviewed; native OS acceptance 3/6/9/11 stays open.

Prism review uses only Anthropic claude-opus-5-5 including fallback, complete
7/7 chunks, 41 files, no skipped/truncated bytes. Four medium and nine low
observations were considered:

- c71f7477e841c6f5: no change. capacitorResult's third argument is *charging*
  current, not terminal current; it adds leakage itself. mnaEquations stamps
  leakage from MixedGraph independently of n.caps. Independent DC reference
  verifies its voltage/current and power residual. Passing leakage as suggested
  would double-count it.
- 1beed2b9767ce1eb: fixed with explicit hasPassiveLosses general-network admission.
- 65dc57a68f4087c1: no change. The sibling branch is private BJT junction bias
  initialization, constraining current to zero, not voltage. Authored ESR caps
  take the voltage/ESR constraint; direct-source ESR startup test covers it.
- 269e2dc0edfe34d9: explicit finite checks added before reconstructed-state use.
- 89433f00bdbac00c: added narrow supported-kind guard to inspector.
- 6bb9224847303002: parsed input no longer shadows field value.
- f89054cc4e81106f, 005154fae9859003: strengthened test lookup/nil diagnostics.
- 091bed344dd17baa: initial unstaged empty receipt was mistakenly in the review
  index; replaced by this actual complete receipt, never claimed as review proof.
- a5bf5c0d72b5e650: parenthesized/documented ESR constraint skip.
- d6be389450796754: residual roundoff magnitude includes ESR term; residual/error
  constants unchanged.
- 666bfb9ad37fc547: renamed local loop variable.
- 0596860863a3881a: completeness guidance includes passive dissipation.

Review-triggered source changes have fresh focused core and both desktop race
receipts. Final full application/package checks follow all five items. Earlier
failed numerical/guard/discovery trials remain in build/electronics-next-five;
coarse RC tolerance is backed by an explicit independent fine-step convergence
check. Protected packages and owner session are not altered.

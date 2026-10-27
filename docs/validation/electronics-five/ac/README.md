# Semiconductor AC analysis

Extends the existing normalized-source/DC-bias Bode path to NPN/PNP,
N/P MOSFET, silicon/Schottky/Zener diodes and LEDs. No device-model,
accuracy, integration-work or source-limit change. Existing class-AB example
now defaults to its useful VIN/load frequency sweep. AC bias opens authored
capacitors; only that algebraic context may use the larger-BJT source-step
fallback. Live reactive startup remains excluded.

Independent complex references cover both follower polarities, both common-source
MOS polarities and external capacitors. Independent diode tangent references
cover four forward types and reverse Zener. Full-amplifier low-frequency gain
agrees with central DC perturbations; negative DC bias, recording validation,
project reopen, native-handler/agent parity and unchanged live state pass.
Focused portable and native-handler race runs pass; logs are retained here.
An initial MOS test accidentally grounded its drain and was corrected. One
runner invocation placed Go's -C flag after -race and did not run tests; the
corrected command passed. Neither setup error is counted as product evidence.

All three Prism reviews used the user-selected **Anthropic claude-opus-5-5**.
The initial medium finding (Fatal on a possible UI goroutine) and all initial
low findings were repaired. The second review's three low test-maintainability
findings were also repaired. Final complete review has no medium/high findings.
Remaining lows are evaluated in findings.json rather than suppressed or baselined.
The configured OpenAI review was blocked before process creation, and the owner
then explicitly chose Anthropic; no code review diff was sent to OpenAI.

This is source/routed software evidence; original native acceptance 3/6/9/11
remains separate. The complete combined release gates follow all five additions.

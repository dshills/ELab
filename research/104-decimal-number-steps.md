# Decimal number steps

October 24, 2026. [Evidence](../docs/validation/number-step-223/README.md).

The native /222 observation is an arithmetic defect, not merely text formatting:
0.03 plus 0.001 twice lands on the neighboring float32, 0.031999998. Hiding those
digits would disagree with the stored native value. The compatibility repair
steps the displayed decimal values exactly, then rounds once to float32. It
retains full round-trip display precision, off-grid entries, wrap limits, tight
bounds, private drafts and wheel behavior. Existing typed/agent/saved values are
not normalized; the public float64 command boundary is unchanged.

Two framework failures and both real production playback failures are repaired.
Seven stepping scenarios, ownership/draft/wheel/precision checks and full
framework/application quality pass. No production ELab Go code or electronics
model changes. The previous SDK cache, /222 app and /205 download remain intact.
Fresh candidate native checks are separate. Continue the remaining native
state-contract checklist after qualifying this repair, without another owner
launch or desktop permission question.

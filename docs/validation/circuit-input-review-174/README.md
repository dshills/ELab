# Selected circuit input guard review

October 19, 2026, following the native /174 slider qualification. The current
72-file command-ledger source hashes still match. Review covers selected-part
values/presets, waveform fields, capacitor/inductor startup, potentiometer and
switch controls, plus their common document/field ownership plumbing. No
actionable defect was found in this bounded pass. No production source, SDK,
model or package changed.

| Input | Existing protection and reviewed evidence |
|---|---|
| Value/label and presets | Selected-object control state, guarded private-draft settlement and same-action continuation; `value_apply_test.go`, `value_presets_test.go`, and retained-widget owner tests cover intervening actor/document work. |
| Sine and pulse fields | Rendered field revision and selected-object/document identity; full waveform payload keeps other accepted fields. `field_context_test.go`, waveform quantity tests and source/shared pulse timing tests cover exact values, invalid input and stale callbacks. |
| Capacitor/inductor startup | Selected-object state and an epoch for Reset, guarded numeric/text settlement and captured continuation; deferred owner, document, error and viewport schedules are covered in the existing startup tests. |
| Wiper and switch | Shared guarded commands; requested wiper is distinct from applied readings, and unchanged displayed wiper preserves an exact stored fraction. Existing stale-field, retained numeric owner and generic held-button tests keep their source scopes. The separate native slider proof adds actual pointer evidence. |

A suspected capacitor startup precision loss was tested before considering a
repair. One temporary Go overlay adds an offscreen retained-widget probe to the
unchanged production package and existing mounted startup helper. Starting from
a saved, focused capacitor field with exact startup **0.12345678912345 V**,
untouched Return retains the value and command revision 10. Untouched Restart
retains the exact value and advances only its command revision 10→11. Both cases
assert unchanged complete components, document and undo entry. Exit 0, 0.690 s;
no OS window, owner file/settings access or capture is involved. This is not new
physical keyboard evidence or a race-instrumented run.

The SDK settlement code compares parsed numeric input with the controlled
float32 value before publishing Change; its existing spinner synchronization
retains enough digits to round-trip that float32. This explains why the suspected
passive rounding did not occur in these cases. The probe is retained as text
evidence rather than adding another production test for a disproved concern.
No unchanged suite was rerun merely to inflate validation totals.

`report.json` pins the reviewed sources and original evidence hashes.
`startup-precision-probe.go.txt` is the exact temporary probe;
`startup-precision-probe.txt` is its passing output. The initial overlay-path
refusal is preserved separately: the producer incorrectly resolved its root
from `desktop/`; the corrected overlay uses the explicit repository root. This
setup error ran no test and changed no app source. The initial Python traceback
is in the original tool output, not reconstructed as raw log data here.
The first evidence-report producer also guessed a nonexistent separate presets
source file; the mapping was corrected to `desktop/value_apply.go` before the
report was written. That traceback is retained in the tool history.

The existing EDIT-03/SIM-02 and broader agent/accessibility/environment gates
remain partial. A source mapping and two no-edit probe cases do not establish
all widget schedules, physical blur/replacement, reader/IME or owner acceptance.

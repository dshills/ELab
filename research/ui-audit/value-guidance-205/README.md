# Native component-value guidance /205

October 23, 2026. CUA observations from an isolated ELab Value Guidance app,
1053×802 pixels, Light. The test app used the same compiled source as
0.181.4 /205, with a distinct bundle identifier and signature. Unsigned native
executables compare byte-for-byte; see
[native equivalence](../../../docs/validation/value-guidance-205/native-equivalence.json).
No owner document or session was changed. Recovery/appearance were disabled for
the test session; shared Cogent geometry access was denied by its sandbox.

1. [Rejected resistor value](01-invalid-value.jpg): draft 0 remains editable;
   accepted R1 is 1 kΩ, current stays 2.917 mA, summary names resistance.
2. [Inspector guidance](02-readable-guidance.jpg): wheel scrolling reveals the
   complete wrapped range, syntax, previous value and details disclosure.
3. [First-Apply correction](03-first-apply-correction.jpg): typing 2k, Tab and
   one Space on Apply yields 2 kΩ and 1.477 mA and clears the warning. One Undo
   subsequently restored 1 kΩ / 2.917 mA, observed in the CUA tree.

Native menu item selection required selection/confirmation sequencing in this
tool. The initial Home/Return attempt did not load an example; no success was
claimed for it. These images prove visible guidance and correction, not complete
keyboard or screen-reader acceptance. The separate native-socket check then
exercised all eight numeric component kinds and closed only the owned test app.
Its private endpoint disappeared and CUA's running-app inventory no longer
contained the test bundle. The normal owner ELab session was preserved.

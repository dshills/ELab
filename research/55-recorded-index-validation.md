# Recorded sample-index refusal and correction

October 20, 2026. **RECORD-INDEX-01 (P2)**: the /185 callback converts the numeric sample index to `int` before shared validation. Both a direct native callback and actual mounted numeric-key entry reproduce `1.5` selecting row 1 without error. The public agent command already refuses fractional indices. This is an input/parity defect, not a numerical solver defect.

The UI now sends the original number to `recording.seek`. The shared integer decoder refuses `.5`, `1.5` and `19.75` without changing the full accepted snapshot. Refusal messages match the public command and remain available to the user. Fresh whole-number input works: direct indices 4/20/0 and mounted correction to row 3 at 3 ms, with one accepted command and live physics preserved. Existing picker/agent and single-row DC controls still pass.

The four final regression leaves fail against original production source and pass with the repair. Full default formatting/imports/lint, 65 Python tests, root/native race/vet and configured mounted UI checks pass in **228.307 seconds** on **661 application Go files plus three existing tool files**. Only `desktop/recording.go` changes in production; two files add regressions. All command IDs/routes/aliases and ledger totals remain unchanged. No SDK, model, solver, tolerance or file-format change.

Original diagnostic compile, negative-test-helper and ledger-collector mistakes are distinguished from the product reproduction and retained in the [source proof](../docs/validation/recorded-index-validation/README.md). Its read-only replay verifies the current source and archived bytes; it opens no app or new test. This is mounted software input, not physical OS typing or reader/IME acceptance.

A **clean replacement app/agent and copied-runtime qualification remain next**; /185 keeps its original affected callback, immutable timing and other receipts. Canonical /120 and prior reviews/candidates remain preserved. Owner walkthrough remains Not tried yet, P0 17 implemented /10 partial, and broader bindings/accessibility/environment/study/MVP gates remain open. No promotion.

# Recorded sample-index validation · source repair

October 20, 2026. **RECORD-INDEX-01 (P2)** is reproduced and repaired in source. On the original /185 implementation, a `1.5` entry in the mounted numeric widget selects sample 1 with no error and advances command revision. The external command's integer decoder already rejects that same entry. The UI had truncated the value before calling the shared command.

The callback now passes the original numeric entry to **`recording.seek`**. Shared validation refuses fractional indices without changing the viewer, live circuit, captured data, history or document; the same structured refusal feedback reaches the UI. Ordinary integer input and a fresh correction remain available. No new command, decoder, numerical model, solver, tolerance, file format or SDK change.

Four final regression leaves fail against the original production `recording.go`: fractional `.5`, `1.5`, `19.75` callback cases and a mounted private `1.5`/Enter case. They pass with the repair. Shared public-request and UI errors agree exactly. Valid indices 4/20/0 select the exact stored rows; the mounted field refuses its private fraction, then accepts a freshly focused `3` once at 3 ms, preserving live physics. Existing recording-picker/agent parity and single-row LED/NPN DC presentation controls pass too. This is source and mounted widget evidence, not new physical macOS typing or a new packaged app.

Full default `scripts/check.sh` passes in **228.307 seconds**: lint/imports/Staticcheck, 65 Python tests, root/native race tests and vet, and the configured mounted UI checks. All **661 application Go files plus three existing tool files** stay exact during the gate. The two added Go files are regressions; only `desktop/recording.go` changes in production. Canonical /120, SDK and the /185 release receipt remain unchanged.

The binding ledger changes one source hash, two callback/ingress expressions and subsequent line positions. Its 72 production sources, 400 sites, 164 public commands, 132 routed IDs, aliases and coverage categories remain exact. Presence in the ledger is not full runtime binding acceptance.

Diagnostic mistakes remain separate: the first temporary test did not compile because it used `key.CodePeriod`; the corrected observed key is `CodeFullStop`. An initial fixed negative test incorrectly used a helper that requires success; the mounted refusal/correction already passed. The final test observes the expected refusal directly. An initial ledger collector expected an unqualified function name instead of `*screen.recordingPanel` and refused before writing the ledger. Original failure logs, original production source, corrected/final tests and the reviewed ledger records are retained. No product defect is inferred from those collector mistakes.

[Report](report.json), [raw original/final regressions](raw/stage/original-regression-final-output.txt.gz), [quality](raw/stage/full-quality.json.gz), [terminal outcomes](raw/stage/terminal-results.json.gz), [byte manifest](manifest.json). Source copies use `.go.txt` suffixes to avoid creating accidental Go packages.

Read-only replay checks original/final source identities, all current source hashes, quality and regression logs, ledger changes, prior artifact pins and evidence bytes:

```sh
python3 -B docs/validation/recorded-index-validation/verify.py
```

It opens no app and runs no test or capture. A clean replacement app/agent and exact copied-runtime checks are **still required**. /185 retains its original affected callback; its historical timing and other receipts remain immutable. Owner walkthrough, P0 17 implemented /10 partial, complete binding/reader/IME/environment/study and overall MVP acceptance remain open. No promotion.

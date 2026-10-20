# Timing fields and probe ownership

Source repair after /192; the owner's /189 alpha and separate /192 candidate are unchanged. No native app is launched.

An invalid threshold/interval could leave a fresh correction blocked when paging without focus loss, or suppress a correction back to the accepted value before choosing an arrival direction. Accepting another field could instead erase the first refusal and silently measure with its old accepted value. Four additional cases show private text typed for one captured probe committing into an agent-selected second probe whose accepted settings happen to match.

Timing refusals now retain each field's text independently, owned by document, capture, probe and accepted field value. Fields settle before the action checks remaining refusals. A successful unrelated field cannot accept another field's bad input. Native control state is scoped to both capture and probe so switching channels discards old private edits while fresh input remains usable. Existing accepted-response, request-help identity, viewport, menu serial, revision and document guards remain in place. No public commands, backend, solver, models, SDK or formats change.

All 42 final regressions replay against original production bindings with 18 failures and 24 passing controls. After repair all 42 pass under race detection: 36 correction cases, two other-field refusal cases and four probe-switch/fresh-input cases. Refusals preserve whole service snapshots; accepted actions preserve live physics, document and circuit history. These are software-mounted inputs and assembled selector callbacks, not actual OS typing or native menu acceptance.

The initial fixture incorrectly used a helper that aborts on expected validation errors; eight extra failures remain preserved with its original test source. The final original-source replay uses the corrected helper. Separate pre-repair probe-owner records show the four wrong-target commits explicitly.

Full `scripts/check.sh` passes in 316.235 s with 670 Go inputs unchanged during the gate. It includes prior public timing parity plus interrupted/deferred/menu/viewport/agent/document/recording guards. The reviewed native inventory retains 73 sources, 402 sites, 164 public commands and 132 routes; route and binding expressions are unchanged.

[Report and raw hashes](report.json). Packaging and actual native input acceptance remain open. The owner window stays reserved; no broader qualification gate is closed.

# Web follow-up milestones — November 4, 2026

Owner order: installation/signing (item 1), agents in the browser workspace
(item 3), whole-library backup/restore (item 5). Internally review and commit
each implementation before the next. Preserve the published alpha and owner data.

## Item 1: installation tooling

Implemented Developer ID identity preflight, hardened/timestamped executable
signing, Apple notarization submission/Accepted-status validation, ticket
stapling, signature/team/debug-entitlement checks, Gatekeeper and available
`syspolicy_check` distribution gates. Final archive hashes are recorded after
stapling. Notarized builds require clean shipping source and cannot use test
library overrides. Credentials remain in Keychain, outside commands/receipts.

The maintained exact-archive installation check pins SHA-256, refuses unsafe or
oversized archives, checks signatures/tickets/system policy before execution,
then runs a copied host with a fresh library and no Go/Node in PATH.
See [setup and acceptance scope](../../macos-distribution.md).

Internal review covered real versus requested notarization status, signer/team
identity, hardened runtime/timestamps, post-stapling seal/hash ordering, output
preservation, archive traversal/link/size bounds, private credential handling
and failure paths that must not execute or claim acceptance.

All 71 Python tests pass, including the new signing/notary refusal controls.
The [development-tree ad-hoc smoke](installation-adhoc-smoke.json) passes real
package/extraction/file-mode/signature/copy/restart checks. The preserved alpha
ZIP was independently refused by the notarized-installation gate as expected;
it was not modified or run as a notarized artifact. No Go application code changed.

This Mac currently has only an Apple Development signing identity. The owner is
configuring Developer ID/notary credentials and explicitly instructed continuing
to the next item. Live Apple acceptance and clean-account/Finder qualification
remain pending; no notarized distribution or new release is claimed here.

## Item 3: agents in the visible workspace

An explicit Connect agent control enables a private, revocable descriptor for
one browser workspace. The installed web backend doubles as the stdio MCP bridge;
no extra user build is required. Requests pin the instance/workspace and share
the engine, revision/generation guards, actor history and checkpoint path.
Revocation is checked inside the mutation lock. Missing/restarted/revoked hosts
never fall back to another session. File, printing and host-lifecycle commands
are excluded from attached discovery and dispatch. Browser Undo explicitly
acknowledges another actor and targets the observed history entry.

Internal review covered descriptor permissions/symlinks/redirects/proxies, bounded
transport, workspace isolation, concurrent revocation, missing guards, retained
retry keys, applied-but-uncheckpointed edits and subsequent repair, bridge closure
without host closure, and dependency licenses in packaged apps.

Focused root, webagent and MCP race suites pass, including a real subprocess
stdio MCP edit of the same browser workspace. Lint, all-package vet, 71 Python
tests and 28 frontend checks pass. Six journeys pass in each of Chromium, Firefox
and WebKit. WebKit exposed missing focus on clicked dialog launchers; explicitly
focusing those launchers fixes return focus. Running all engines under one host
exceeds the intentional 16-workspace limit, so engine runs use fresh hosts as CI
does. The development-tree package/copy/restart smoke passes with all 23 packaged
files and linked MCP licenses. These are development checks, not a signed release.

Local logs: `build/web-agent-race-final.log`, `build/web-agent-frontend-final.log`,
`build/web-agent-python-final.log`, `build/web-agent-lint-final.log`,
`build/web-agent-vet.log`, `build/web-agent-browsers-verified.log`,
`build/web-agent-webkit.log`, and `build/web-agent-package-final/report.json`.

## Item 5: whole-library backup and restore

Projects now offers a portable backup download and a validated restore preview.
The snapshot checkpoints active workspaces and includes named projects, Trash,
automatic/restored history, inactive/lost-cookie workspaces, retained native
recordings/sweeps and embedded models. Dormant legacy per-workspace stores are
included; older differing versions are preserved. Workspace source IDs are anonymized so they do not export browser-cookie values.
No credentials, host controls, logs, browser state or scratch files are exported. See [scope/format](../../library-backups.md).

Restore validates the exact bounded file list, strict manifest, per-document
SHA-256 and native project/results/models before staging. A single directory
rename publishes the complete batch. New identities preserve all existing data
and the open circuit. Unsaved workspace copies remain discoverable in Recovery
history; restored history is protected from automatic newest-50 rotation.
Only one transfer is admitted at a time to bound aggregate validation/ZIP memory.

Internal review covered lock order and snapshot consistency, dormant storage,
exact retained sample/AC-result bytes, project overwrite/trash after import,
restart indexing, stale guards, revocation/credential exclusion, corrupted source
refusal, malformed/duplicate/path-substituted/link/oversized ZIPs, cancellation,
failed publication and cleanup. Browser hosts are now independent per engine so
combined runs do not exceed the production workspace cap.

All 21 final browser journeys pass (seven each in Chromium, Firefox and WebKit).
The final root/agent race suite and additional retained-AC/corrupt-source/transfer
admission tests pass. Logs are `build/library-final-browsers.log`,
`build/library-reviewed-race.log`, `build/library-final-transfer-tests.log`, and
`build/library-final-lint.log`. This milestone does not publish a new alpha.

## Final validation

The [frozen-source macOS quality run](quality.json) passes all six maintained gates:
lint, 71 Python tests, 28 frontend checks, the complete Go race suite, all-package
vet and actual package/extraction/copy/restart checks. `inputs_unchanged` is true;
the [input hashes](quality-inputs.json) bind the checked working tree to the
subsequent implementation commit. The runner's source commit is the preceding
agent milestone because this was a development-tree check before the requested
backup milestone commit. No clean-release/notarization claim is made.

The [copied-package smoke](package-smoke.json) also verifies library backup,
checksum/preview, restore as a separate copy and discovery after restart with
Go/Node absent from PATH. [All 21 browser journeys](browser-results.txt) pass on the
frozen executable across Chromium, Firefox and WebKit; [scope and binary hash](browsers.json).
Test libraries and hosts are disposable; owner projects and the published alpha
remain preserved. Live signing and clean-account acceptance remain deferred
to the owner's credential setup. No remote CI or new publication was requested.

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

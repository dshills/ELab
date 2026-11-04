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

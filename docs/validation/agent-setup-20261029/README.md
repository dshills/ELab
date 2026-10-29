# Standard local agent setup

After completing the usability/release checks in unsigned 0.200.5 /283, add
`elab-agent --mcp` and `--mcp-config [--connect descriptor]`. All 226 existing
commands, parameters, model/solver bounds, revisions, document guards and retry
identities pass through the shared service. The raw JSON modes remain available.
[Setup guide](../../agent-setup.md), [internal review](internal-review.md).

The official pinned Go SDK supplies the protocol; transport/remote.go factors
existing private forwarding for reuse without an independent command engine.
Missing/failed/retired endpoints cannot select a new circuit. Explicit empty
paths and ambiguous modes refuse before session creation. Generated configuration
uses absolute arguments, includes no credential contents and edits no client file.

Tests exercise reference clients in both protocol generations, real executable
setup outside the repo, headless/socket LED construction and analytical RC
capture comparison, bounded reads, exact refusals/retries, cancellation, concurrent
admission and process ownership. New dependency notices are pinned/verified and
included in newly assembled candidates. Final quality/package receipts follow;
the preserved /283 archive and older releases remain unchanged.

No new component, numerical law/limit or Vellum runtime change. Existing complete
usability/rendered qualification remains separately scoped. Original native
checks 3/6/9/11, human participants, clean-account installation and public signing
are still independent. No push or upload and no external-provider code review.

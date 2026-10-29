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
included in the newly qualified package. The preserved /283 archive and older
releases remain unchanged.

No new component, numerical law/limit or Vellum runtime change. Existing complete
usability/rendered qualification remains separately scoped. Original native
checks 3/6/9/11, human participants, clean-account installation and public signing
are still independent. No push or upload and no external-provider code review.

## Final qualification

Unsigned **0.201.0 /285**, clean source `70e113d5f2767e12963fe47337f35a7200f10a9e`.
Archive `46,309,442` bytes; SHA-256 `f61ef990202f395955d92aef9489e8e9ad49ce968574d0fc76a136f3c4bcbc65`.
[Current artifact paths/setup](../../current-release.md).

- Complete initial portable race run passed all packages except two workbench
  relay elapsed-time safety checks. Both passed three targeted repetitions;
  the subsequent full workbench race run passes in isolation. No limit or test
  assertion was relaxed. Retain the initial failed receipt explicitly.
- All Go/module inputs are exact between those runs; only three packaging Python
  files changed. [Coverage record](final-coverage.json), initial/final input maps
  and compressed logs bind carried passed packages to the final clean source.
  Final tooling: **104 Python tests**, vet and lint pass.
- Complete ordinary desktop tests, targeted mounted UI/agent-interleaving race
  tests and desktop vet pass. Full prior desktop/rendered qualification carries
  only after checking every desktop source/test/module byte and used dependency
  version against /283; shared transport is separately retested.
- Exact copied native smoke and four existing adapter suites pass under
  development-file/outbound-IP denial. All **102** conformance external processes
  exit0 normally. All **87** curated projects reopen paused/clean and match current
  built-in graphs; **61 kinds /226 commands** remain unchanged.
- Generated MCP configuration executes from another cwd in both fresh headless
  and owned private-socket modes. Both build/measure an LED, capture two RC
  responses, compare their analytical values and read a bounded five-row page.
  Exact packaged native MCP placement/value editing matches the raw adapter;
  bridge EOF leaves the host alive, then explicit close exits0. These are software
  routes, with zero automatic MCP mutation retries, not native OS input trials.
- Archive membership/bytes/modes, OS extraction and strict ad-hoc seals pass.
  **47** preserved package/SDK identities (including the new archive/app/agent)
  verify. All owned fixtures close normally; no owner session/project is used.

The failed /284 assembly remains unqualified: module `@version` names were refused
by the pinned packager. The corrected generated notice spelling is tested against
that actual validator, while original license paths/bytes remain supplied intact.
No verified SDK, old archive or security setting was changed.

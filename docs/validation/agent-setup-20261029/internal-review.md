# Internal MCP/setup review

Reviewed protocol versus service ownership, raw-envelope preservation, input and
response bounds, explicit guards/retry identities, endpoint lifetime/cancellation,
CLI mode/config generation, process cleanup, copied-runtime boundaries and
pinned dependency notices. No Prism or external code-review provider was used.

## Decisions and corrections

- Use official Go MCP SDK v1.8.0 for protocol negotiation, framing and lifecycle.
  Keep it isolated in internal/agentmcp and the CLI; desktop controls, numerical
  models, work limits, public catalog and underlying service remain unchanged.
- Three MCP tools cover complete discovery, inspection and exact command
  dispatch. The low-level SDK handler preserves raw arguments so duplicate/
  unknown fields cannot disappear into a decoded map. Readonly wrappers strictly
  validate their own shapes. Mutations receive no invented key, revision or retry.
- Eight concurrent dispatches, 64 KiB MCP input framing, existing 32 KiB ELab
  envelopes and 8 MiB service responses remain explicit. SDK protocol cancellation
  is forwarded, and the remote socket read now interrupts on caller cancellation.
- Pin one explicitly chosen host credential. Failure cannot create a fallback
  headless circuit or silently adopt a replacement host. Review found that an
  explicitly empty --connect could select headless mode; reject it (and empty
  --listen) before creating a service. Ambiguous modes likewise refuse clearly.
- Configuration prints absolute executable/descriptor paths without reading a
  token or editing client settings. Tests execute the result from another cwd
  and a binary path with spaces. Disconnect closes only the bridge; app.close
  remains explicit. A headless MCP bridge delivers the service's close result
  and then follows MCP EOF lifecycle, rather than dropping that response.
- Transport errors remain tool-visible and explicitly uncertain. Exactly-once
  retries return cached responses without reapplying an older edit. Tests cover
  intervening command/document guards and fresh host selection.
- The demo initially read operation.inspect.status instead of its documented
  operation.status member; corrected this controller error, not the product.
  It now completes LED construction, two independent RC captures, comparison and
  a bounded five-row read through generated headless and socket configurations.
- An initial go mod tidy scanned old ignored build fixtures and tried to resolve
  unrelated UI modules. Stopped only that owned process. Normalized a maintained
  source-only projection; no such dependency entered go.mod/go.sum. Go version,
  Gonum and the pinned Vellum/runtime sources remain unchanged. All modules verify.
- Include verbatim SDK/runtime license/notice/author/patent files pinned to module
  versions/sums. Test changed/missing/extra/link data and version/checksum drift.
  Refresh only a new package's output copy, preserving supplied/old inputs. The
  SDK's complete Apache/MIT transition text is retained without narrowing it.

## Evidence and limits

Focused SDK-client tests pass under the race detector for legacy 2025-11-25 and
current 2026-07-28 protocol, local/private socket service calls, all 226 catalog
commands, explicit refusals/retries, bounded/error responses, concurrent admission
and normal disconnect. CLI tests exercise the real executable, raw JSON backward
compatibility, configuration, missing endpoints, oversize rejection and EOF exit.
All 103 Python tooling tests pass before clean final gate/package qualification.

This is local protocol/electrical evidence, not a benchmark of an AI model's
independent task completion or every third-party client's settings UI. No owner
session/project, cloud provider, credential installation, network listener,
security weakening, new component or solver-limit change is part of this work.
Complete qualification receipts are added only after actual final runs pass.

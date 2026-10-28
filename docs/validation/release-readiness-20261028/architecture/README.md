# Item 5: compilation and analysis-job ownership

Pure compiler code is extracted unchanged; compiler-extraction.json records the
exact source comparison. Service graph consumers use a private single-entry
content-keyed cache, returning detached graph/model/map/net data. Summary uses
the same compilation on live restart. Graph.Compile remains available for
independent worker trial graphs. Analysis job state/admission/retention is grouped
under analysisJobs without a second mutex, command handler or revision authority.
The shared command surface remains 222 commands and 61 component kinds.

Focused race controls compare fresh/hit results over all 87 examples and damage
mutable references on both sides of the boundary. Metadata/model changes,
invalid/capacity graphs, signed zero, nonfinite encoding bypass and concurrent
user edits/agent queries pass. Store controls cover kept/active refusal, direct
retention at capacity, evicted selection/cursor retirement and store isolation.
Affected analysis, blocks, persistence/capture and snapshot race tests pass.

Complete initial/follow-up Prism reviews use Anthropic Claude Opus 5.5. Initial
findings and dispositions:

- Cache encoding completeness: added a reflected graph/model leaf coverage test;
  unexported/omitted fields or custom encoders require explicit coverage.
- Possible pre-revision stale cache: the cache is keyed on full current graph
  content, not a revision counter. Mutation/invalidation controls pass.
- Possible worker aliasing: every return deep-clones pointers/maps/nets; all
  example ownership-damage controls prove separation. The step worker owns its
  Compiled result rather than a reference to the entry.
- Retention capacity: the extracted store now refuses direct full-store retain;
  callers preserve fail-fast checks and handle the store error.
- Miss-copy overhead: intentionally retained ownership. Fresh Graph.Compile
  shortcut components can alias source model pointers, so returning its raw
  value would violate the new boundary. Microbenchmarks measure the actual cost.

Follow-up has zero medium/high findings. Its recursion observation concerns
future recursive graph field types; current graph/model types are acyclic. The internal review added an explicit
recursion guard so new recursive types fail clearly rather than exhausting the stack. Early service admission
checks deliberately preserve no-allocation/no-ID-change refusal, while the same
helper inside retention protects the store boundary. Both run under one service
lock and use the same policy, so no differing admission contract was introduced.

Local M4 Pro/Go benchmark, 200 ms samples: LED fresh 4424 ns/op, cached/owned 3663;
discrete class AB 140133 vs 42954; RC block 6098 vs 5493. Class AB allocation count
falls from 646 to 100 and bytes from 107336 to 41984. LED bytes rise from 2784 to
3650 while allocations fall from 45 to 13. These are compiler microbenchmarks,
not end-to-end UI/simulation latency measurements. No numerical/work limit or
component/public command change. Full final gate results are recorded separately.

The first complete final gate reproduced two additional instrumentation wall
refusals: buck capture and the independent NPN/coil flyback reference. Both pass
in isolated race and ordinary runs; no electrical/capture assertion was changed.
Only the buck leaf (other extended cases remain instrumented) and the NPN
reference now use the existing ordinary-child harness. Child test patterns anchor
each nested path segment. The focused guarded controls pass; the full failed
receipt stays retained. The additional external review had started before the owner changed the review
policy; its completed receipt has no findings. No further
Prism run is authorized by the old task requirement.

On the owner's October 28 instruction, code review switched to internal review
for all remaining work; AGENTS.md records the standing preference. Internal review
checked current-field JSON identity, detached map/model/label ownership, mutation
before revision increments, direct store capacity rejection, cancellation on
refused admission, import publication guards, child deadline/short-mode/nested
selection and bounded test-pipe cleanup. Existing early service capacity checks
preserve refusal without allocating metadata, contexts or serial IDs. The final
focused controls pass, with no unresolved actionable production finding from
that pass. This is an internal review, not an independent reviewer claim.

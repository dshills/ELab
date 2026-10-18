# Wiring replay pins the release and observation producer

October 18, 2026. This is a validation-only follow-up; ELab 0.134.0 /151 and every prior review remain unchanged.

The [historical /149 replay](../named-wiring-bindings/replay.py) correctly compared its recorded wiring outcomes, but its source check read the **current checkout's** ledger. Running it after /151 therefore still passed while reporting 66 source files against an agent release built from 64. [Before report](historical-replay-before.json). This mixes release identities; it does not demonstrate a wiring failure or invalidate the original /149 observations at their recorded checkpoint. The historical script, transcript, report and receipt are preserved verbatim.

The supported [release-pinned runner](../../../scripts/wiring_replay.py) first checks the review's clean source identity, exact source-ledger bytes from that commit, all production/runtime source inputs, SDK lock, complete release file set/hashes/modes, native/agent digests and recorded agent build identity. A mismatching checkout is refused **before** SDK preparation, test/agent execution or output creation. [Actual stale /149 refusal](stale-review-refusal.txt). No live endpoint argument is accepted; the agent is a newly owned stdio process.

On admission, the runner verifies the pinned SDK, runs the actual retained-widget and macOS-adapter wiring tests with `-count=1`, records the focused command/exit code, all desktop test-source hashes plus runner/client hashes, and the transcript digest. It revalidates source/release/producer identity before comparison and again before publishing success. The worker checks this sidecar and complete, uncached passing observations before launching the exact **packaged** agent. The pinned process-group runner bounds both test and worker execution and cleans its owned group on failure/interruption. The shared Python client's default behavior is unchanged; an optional exact agent path avoids changing its module-global root.

The [new /151 report](report.json), [producer witness](producer.json) and [fresh focused transcript](focused.txt) bind 66 native source files to clean source `5766dd194e124fbbd2e5782129109fbcb813c1e7`, verify all 129 review-manifest files, and compare the completed and undone circuit fields against the matching /151 agent. Components, wires, nets, selected object, drawing draft, Undo label and current/staleness match. Current is 0.0029167965774405296 A; Undo leaves three wires and stale closed-loop readings.

Regression checks reject stale ledgers, runtime-source changes or additions, SDK/agent/native/build/manifest mismatches, links, reused output directories, altered producer/transcript metadata, failed/cached/incomplete observations and duplicate cases. They also preserve the existing client's default executable selection.

Run from the repository root:

```sh
python3 -B scripts/wiring_replay.py \
  --receipt research/ui-audit/first-version/output-page-review-checkpoint.json \
  --artifacts build/wiring-qualified-new
```

The destination must be new. This is source/observation provenance and functional wiring parity, not a security boundary, reproducible-binary certification, physical Cocoa-menu interaction, IME/VoiceOver, visible p95 or owner acceptance. Existing OS walkthrough observations retain their original scope. No product rebuild/version increment, promotion or physical printing occurs. AGENT-01 remains partial.

The [quality log](full-check.txt) records 51 passing Python tests, lint/audit success and passing root/native/mounted race suites (Go results cached). Its terminal exit receipt was not retained across context compaction; the final silent tagged vet was independently rerun and returned zero ([receipt](final-vet.txt)). The fresh focused wiring run above uses `-count=1`. Historical evidence hashes were rechecked unchanged.

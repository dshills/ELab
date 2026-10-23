# First engineering release: ELab 0.179.0 /196

**Current successor:** [ELab 0.181.3 /204](79-first-version-release-closeout.md).
This document preserves the original /196 release designation and its evidence.

**Latest delivery:** [the owner-selected unsigned macOS release is ready](72-unsigned-release.md). It uses the same /196 runtime bytes, adds distribution instructions and completes exact extracted-native checks. The private engineering-alpha checkpoint below remains preserved.


October 22, 2026. **The private macOS hobbyist engineering alpha is packaged and ready to use.** The five-item queue is complete within its declared scope. Owner review is deferred by instruction; no new features were added. This is the first engineering release designation, not full PRD MVP or public-distribution acceptance.

- [Open ELab](../build/ELab-0.179.0-196-alpha/ELab.app)
- [Matching agent](../build/ELab-0.179.0-196-alpha/elab-agent)
- [Release ZIP](../build/ELab-0.179.0-196-alpha.zip) and [SHA-256](../build/ELab-0.179.0-196-alpha.zip.sha256)
- [Bundled instructions](../build/ELab-0.179.0-196-alpha/README.md), [agent protocol](../build/ELab-0.179.0-196-alpha/AGENT_PROTOCOL.md), [limitations](../build/ELab-0.179.0-196-alpha/RELEASE_NOTES.md) and [rollback](../build/ELab-0.179.0-196-alpha/ROLLBACK.md)
- [Engineering evidence](../docs/validation/engineering-release-196/README.md) and [machine-readable receipt](../docs/validation/engineering-release-196/report.json)

## What changed

The sole product repair since /195 permits saving during independent Verify or sweep work. A save captures only completed kept results. Later analysis completion remains unsaved, including completion that races file publication. Existing file-job exclusivity, overwrite/conflict guards, retry identity and cancellation behavior remain. No device, model, simulation budget, project format, command identity or dependency version changed.

The signed app and agent were built from clean `885af1ad3c2a130092e532e8ffe966a5c6789889` and copied unchanged from /196 candidate into the release. The repository is now on `main`; release-evidence documentation is a later commit. Vellum remains pinned to public `e58d4058` with the recorded `96e22bf2` compatibility source and bundled notices.

## Completed queue

| Item | Result |
|---|---|
| 1. Work during demanding simulation | Reproduced the old Save refusal, repaired it and retained the failing evidence. Seven real in-flight backend interactions pass. Native Save succeeds while the same flasher Verify advances from 50% to 57%; completion subsequently marks recordings dirty. Earlier valid captures and manual files survive. |
| 2. User/agent behavior | Dynamic target and unit refusal, exact-state preservation, stale revisions, idempotent retry, conflicting keys and asynchronous Save job identity pass. Exact release agent contracts pass. Existing binding tests run in the full gate; the static inventory alone is not treated as coverage. |
| 3. Core keyboard workflow | Blank LED placement and four wires, value correction, native Save/Open, RC voltage/current probes and 1/10 ms cursor entry pass. LED reads 2.917 mA at 1 kΩ and 1.477 mA at 2 kΩ; RC cursor B reads 3.161 V and Δt is 9 ms. Valid times commit on focus loss. Scope limits below remain explicit. |
| 4. Schematic output | Complete fitted Letter/A4 PDF/PNG, SVG identities, 12 mm outer margins, 100% explicit-scale tiles and 5 mm overlap pass. All 12 tile previews/direct PNGs agree byte-for-byte. Independent PDF parsers and Poppler renders confirm geometry and were visually inspected. No printer job submitted; earlier owner-confirmed LED paper output remains separate. |
| 5. Release | Full source quality matches all 672 Go files. Exact native/agent reference LED, RC and flasher results agree. Three adapters pass project contracts and independent electrical references. All 127 ZIP members preserve bytes/modes on macOS extraction, extracted signature verifies and all three examples reopen paused and clean in fresh agents. Final native and agent sessions complete joined shutdown. |

The changed Save admission was reviewed against mutex-owned capture, membership hashing and file publication. Regressions cover both sides of the publication race and pending sweeps without weakening existing electrical tolerances or storage guards. The original restricted full-check socket denial and the permitted successful run are both retained; the passing suite was not rerun merely for documentation or branch naming.

## Limits and operating notes

This is tested on macOS 26.6.2 arm64 with an ad-hoc signature. Clean-account/default-startup testing remains explicitly deferred for this private alpha. No notarization, quarantine/install, wider platform, hobbyist study, exhaustive accessibility/IME, command/event coverage or broad visible-latency acceptance is inferred. The embedded app manifest retains build-time audit statuses; the external release receipt supplies subsequent private engineering results.

Invalid cursor Return gives generic refusal and resets that rejected draft; refocusing and entering a valid time works. Native file panels can append a second `.elab` suffix when an extension is typed in the selected basename. These are disclosed limitations, not silently passed draft-retention or polished-wording claims. Sparse tiles and split labels are expected at explicit scale; choose Fit to page for a single complete page or use overlap for assembly.

The first /196 native session was interrupted and has no orderly-shutdown receipt. Its saved projects/evidence remain; its processes were verified absent and its isolated runtime was decommissioned. A fresh final owned /196 native session separately supplies normal joined-shutdown and endpoint-removal evidence. No old failure or unavailable attempt was replaced by a pass.

Canonical /120, the owner /189 alpha, /195 ZIP, SDK archives and prior candidate trees retain their recorded hashes/modes. The owner app/settings/projects were not replaced. No Git remote is configured; nothing was published or pushed. Further feature work is outside this release task.

# Next five engineering items

October 21, 2026. The owner directed: **“Continue the electronics work without my review.”** Owner walkthrough results remain unobserved, but are no longer a prerequisite for continuing engineering or preparing the first engineering release. This does not manufacture a usability-study pass, waive numerical/data-integrity requirements or authorize public distribution.

The existing /195 private alpha remains usable while this queue closes remaining MVP gaps. These are improvements to shipped behavior, not additional gates that retroactively invalidate its completed reference checks. Preserve the app/agent/ZIP and prior evidence. Keep the existing feature freeze: no new devices, analysis modes or broad framework update.

| Order | Work | Completion condition |
|---|---|---|
| 1 | Keep circuit work usable during demanding simulations | Use an owned coupled RC/NPN case to check editing, navigation, Pause/cancel, Undo and Save during actual in-flight work. Distinguish cancellation from a job that already completed. Verify current/stale/failed states and preserve the prior valid results and saved project. Repair a reproduced blocker with a focused regression. |
| 2 | Close remaining user/agent command gaps | Follow the uncovered or ambiguous bindings for existing editing, probes, measurements and job controls into the public contract. Cover dynamic targets, units, rejection, stale revisions, retries and interrupted jobs. Each selected gap gets a behavior comparison or explicit remaining limitation; static command counts alone do not establish coverage. |
| 3 | Finish the core keyboard editing workflow | Complete blank LED placement/wiring, value editing, probe/cursor entry and save/reopen using the keyboard. Focus on fast typing, focus loss and correcting errors without lost input or the wrong target. Reuse existing successful native cases; this is not a general visual redesign or a claim of full screen-reader/IME acceptance. |
| 4 | Qualify schematic output | Check existing SVG/PDF/PNG output for complete symbols/text, Letter/A4 bounds, margins, explicit scale and tiling. Compare output settings and errors through UI/public commands. Use local files and previews; no additional physical printer job is needed. Preserve the already owner-confirmed readable LED print. |
| 5 | Finalize the first engineering release | Review fixes from items 1–4, run applicable independent electrical regressions and required quality checks on changed source, and package one matching app/agent with examples, locked dependency notices, limitations and rollback. Rebuild only for a concrete fix. Record owner review as deferred by instruction, preserve /189 and the canonical app, and keep clean-account/public-distribution limitations explicit. |

## First item: bounded starting point

The repository is clean at `a19cae74`. /195 was built from clean `8d7d6463`; all 671 Go source hashes still match its full quality gate. Its actual native LED/RC/flasher walkthrough and exact-agent comparisons already pass. Those checks will not be repeated without an invalidating change.

Initial source inspection of `internal/conformance/performance.go` confirms that its sequential edit/Verify measurements are backend timing evidence. They do not alone prove that a person can edit, navigate, save or cancel during a difficult solve. Start by comparing the existing job/conformance coverage with those interaction schedules, then run only the uncovered cases. Do not start another visible-latency recording cohort or relabel backend timings as visible acknowledgment.

Each item gets one bounded evidence/reproduction pass. Preserve failures, fix actual defects, and proceed when its declared scope is satisfied. A missing external environment is a disclosed limitation, not a reason to invent more feature work or repeat completed tests.

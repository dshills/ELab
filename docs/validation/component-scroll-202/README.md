# All components scrolling /202

October 23, 2026. ELab 0.181.1. The [receipt](report.json) identifies the exact
native app, matching agent, pinned SDK and extracted archive. [Native pointer
results](native-ui.json) and [capture captions](../../../research/ui-audit/component-scroll-202/README.md)
distinguish /201 failures, the first preview and final packaged-app checks.

macOS button/wheel callbacks now use the matching event location. Disabled-mode
virtual coordinates, finite/range validation and one DPR conversion are retained.
A redundant scrollbar wheel inversion was removed. The complete derived patch
and both reconstruction stages are checksum-pinned in the SDK lock. Regression
tests cover event-vs-hardware selection/fallback, relative mode, invalid event
positions and frame/scrollbar direction and bounds on both axes.

The full source suite passed with Go packages serialized. The first sandboxed
attempt could not start local agent sockets; its electrical reference also hit
the pre-existing integration work-time budget. Both attempt logs are retained.
The final authorized run passed without solver-limit changes or skipped tests.
Dependency race tests and consumer native/offscreen checks passed. The packaged
agent checks the unchanged catalog, categories, aliases, purity, retries and
placement. Archive extraction preserves files/modes in a Unicode/spaces path;
the extracted app passes strict ad-hoc signature verification. A final metadata
review corrected the download notices folder's copied SDK lock before handoff;
the app binary and its native pointer qualification stayed unchanged.

No component models, shared commands or document formats changed. Owner documents
and the preceding release were preserved. Developer ID/notarization, clean-account
installation and full accessibility/PRD qualification remain separate.

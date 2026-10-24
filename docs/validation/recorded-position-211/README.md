# Recorded-position request context /211

October 23, 2026. [Receipt](report.json), [source/check inputs](source-inputs.json),
[repair notes](../../../research/88-recorded-position-context.md).

The exact /210 packaged agent reproduces an old 20 ms seek request receiving
the replacement 10 ms capture's bounds. State remains unchanged, but the help
describes the wrong context. UI callback regressions separately reproduce old
malformed text appearing after agent cursor, playback-speed or live-model edits.

One central request-context check now protects all specialized diagnostics and
target references. Strict position intent retains useful whole-index/time and
DC guidance. Malformed/ill-typed requests do not invent position intent, and
stale UI parser failures cannot install text or a continuation in the new editor.
Original error codes/details, parsing precedence, accepted values and schemas
are preserved. Fresh corrections retain the agent's other edits.

Focused backend checks pass in 1.574 s. Callback and existing mounted playback/
private-buffer checks pass in 5.931 s. The complete gate passes lint/staticcheck,
67 Python tests, root/desktop race and vet, mounted UI race and offscreen vet.
Desktop race takes 170.080 s; mounted race takes 153.997 s. All 876
captured source/check inputs match. Every semantic native site, public command,
alias and route set is unchanged; only one source hash and line locations move.
Inventory remains 166 commands, 134 routes, 404 sites and 77 native sources.

[Raw records](raw.tar.gz) and [member hashes](raw-manifest.json) include original
source/callback/packaged failures, current and baseline sources, final checks
and producers. An added test initially expected raw JSON `1e309` to be a strict
seek intent. The common decoder rejects that unrepresentable number; it now
correctly remains in generic-decoder cases. Quantity text `"1e309 s"` still gets
current-range guidance. No validation was relaxed for the test.

No graphical application is launched for this checkpoint. The SDK and /205
download remain unchanged, and previous candidates are preserved. Fresh native
acceptance of this UI guard and the accumulated /207–/210 changes remains
separate from source/offscreen and headless evidence. No permission or owner
review is requested and no launch prompt is left waiting.

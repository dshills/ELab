# Native UI audit and repairs — October 14, 2026

The original empty-workspace capture showed clipped Inspector copy, dark inputs with dark text, and an increasingly crowded header. Smaller-window testing exposed a second problem: an unnecessary growing spacer reduced the toolbar allocation, and its horizontal scrollbar cut into the buttons. The user also identified the custom popup menus as boxy and unlike macOS menus.

The repaired app uses consistent content/padding widths, bounded controls, readable Light/Dark palettes, grouped actions, explicit toolbar height, and slim themed native scrollbar handles. File, Examples and Theme now open actual AppKit menus through Vellum’s native menu service. Native RC menu selection and keyboard activation were observed, with shared command tests preserving unsaved-change checks. The compact transient layout keeps the scope, cursors, metrics and footer visible; Fit reveals the complete circuit.

## Capture ledger

These are exact native screenshots, not mockups. Pixel dimensions describe the capture, not device-independent window units. The initial labels “final” in several filenames refer to intermediate candidates and do not certify the final release.

| Step | Capture | Observed health |
|---|---|---|
| 1 | [Original empty workspace](01-before-empty.png), 1419 × 1076 | Problem: clipped Inspector text, dark fields, crowded header |
| 2 | [First attempted Inspector scroll](02-before-inspector-bottom.png) | Inconclusive: viewport unchanged; filename does not prove bottom access |
| 3 | [Second attempted scroll](03-before-inspector-scrolled.png) | Inconclusive: viewport unchanged; no claim of successful scrolling |
| 4 | [First width/theme repair](04-after-empty.png) | Partial: copy/contrast improved; new toolbars collapsed because native overflow needed a minimum height |
| 5 | [Inspector bottom reached](05-after-inspector-bottom.png) | Improved: actual scroll-area navigation reaches recovery/verification controls; toolbar still intermediate |
| 6 | [Wide light candidate](06-final-light-wide.png), 1750 × 1207 | Improved: grouped controls visible; smaller size still required checking |
| 7 | [Resized light candidate](07-final-light-resize.png), 1185 × 855 | Problem: horizontal header scrollbar and missing Theme action; user correctly rejected this candidate |
| 8 | [System dark, wide](08-chrome-system-wide.png), 2389 × 1295 | Improved: themed slim scroll handles; no heavy black tracks |
| 9 | [System dark, compact](09-chrome-system-small.png) | Improved: full header/theme controls; no horizontal header scrollbar |
| 10 | [Light, compact](10-chrome-light-small.png) | Improved: same header and rails in Light |
| 11 | [LED, compact](11-light-led-small.png) | Improved: full LED and live metrics; short palette copy still showed a wrapping artifact |
| 12 | [Earlier RC, compact](12-light-rc-small.png) | Partial: scope controls visible, but global vertical overflow hid metrics/footer |
| 13 | [First native-menu presentation attempt](13-native-examples-menu.png) | Inconclusive: menu bar installed, popup absent; fixed native window selection/focus |
| 14 | [Actual AppKit Examples popup](14-native-menu-final.png) | Improved: native rounded/material popup, accessible native menu items, correct anchor |
| 15 | [Native RC selection](15-native-rc-selected.png) | Improved: RC scope, cursors, metrics and footer visible; circuit needed Fit |
| 16 | [Paused RC after Fit](16-native-rc-fit.png) | Improved: complete circuit and controls; low zoom makes labels small |
| 17 | [Native Theme popup over dark RC](17-native-theme-menu-dark-rc.png) | Improved: actual AppKit theme choices; themed schematic, axes, traces, controls and scroll handles |

## Repairs and their evidence

- **High — unreadable inputs:** screenshot 1 combined a light product UI with inherited dark native field backgrounds. App-local foreground/background defaults and resolved descriptor palettes make inputs readable; screenshots 10 and 17 show both themes.
- **High — clipped header actions:** screenshot 7 showed the scrollbar intersecting buttons. Removed the extra growing header spacer and allowed sufficient toolbar height. Screenshots 9, 10, 14 and 17 show full controls at the same 1185 × 855 capture size.
- **Medium — clipped Inspector paragraphs:** the rails constrained some text/control content independently of their padding. Applied consistent descendant constraints, zero label padding and a wider Inspector content area. Screenshots 5, 10 and 17 show readable text and reachable lower controls.
- **Medium — heavy scrollbars:** native dark tracks conflicted with Light. The local mounted-scene adapter styles native tracks/handles and uses 8-unit width while preserving native input/layout/accessibility. Screenshots 9, 10 and 17 show the result.
- **Medium — custom popup appearance:** Vellum MenuButton painted its own popup. The macOS implementation now presents the AppKit menu installed by Vellum’s service; screenshots 14 and 17 are direct evidence. Native menus follow macOS appearance, so they can be dark while the circuit UI is Light.
- **High — compact transient overflow:** screenshot 12 hid the bottom metrics/footer. The schematic canvas can now shrink below its backend default minimum while scope toolbar heights remain explicit. Screenshots 15–17 show the complete lower layout. Fit remains a shared command.

The final package also shortens the palette instruction to “Pick a part to place.” to avoid the visible wrapping artifact in earlier captures. Those earlier screenshots are retained honestly; they were not edited to resemble the final copy.

## Verification and remaining limits

[Appearance/observation native receipt](../../docs/validation/desktop-appearance-events.json) and [headless receipt](../../docs/validation/headless-appearance-events.json) prove shared theme choices preserve paused RC charge, time, history and document state. Native menu/unsaved-edit tests, root/native race tests, lint, vet, actual packaged GPU smoke and four-adapter reference conformance are recorded in [the release receipt](../../docs/validation/ui-repair-validation.json).

The full GPU pixel fixture runs at 1260 × 1100 device-independent units so its two deliberately distant pinned reading cards remain legible. Its old hard-coded Fit dimensions and old background-color expectation were corrected to the actual viewport and selected palette. Compact-window screenshots provide separate layout evidence; no small-zoom reading-card legibility qualification is claimed.

Remaining work includes persistent/global theme preferences, automatic System-theme changes while idle, current-theme checkmarks in native menus, native selector popups, compact scope collapse/resizing, labels/cards at very small zoom, and full keyboard/IME/screen-reader testing. This audit is visual and command-level evidence, not a WCAG or accessibility-compliance certification. The original MVP acceptance gates remain open.

## Subsequent native dropdown and scope checkpoint

Version 0.41.0 / 56 extends the initial menu repair to every macOS selector. [The native dropdown ledger](native-choices/README.md) shows direct AppKit File/Examples/Theme and object popups, selected-choice checkmarks, complete compact header actions and observed mouse/keyboard selection. Original Vellum callbacks remain shared command paths, with command-admission guards against interleaved agent edits. Vellum installs the native application menu bar independently of popup presentation.

[The scope ledger](scope-collapse/README.md) retains the rejected header-control candidate and shows corrected compact expansion, collapse and restoration at unchanged center/zoom. Version 0.40 already added independent persistent appearance profiles. These checkpoints improve the open items recorded above without claiming full keyboard/IME/screen-reader, distribution, performance or participant acceptance.

October 18: [invalid-value recovery reproduction and repair](first-version/native-value-recovery/README.md) records a new /153 native correction blocker, the bounded source fix and a passing native development harness/agent comparison. Clean replacement packaging remains pending; existing packages and owner walkthrough status are preserved.


### Invalid-value correction replacement · 0.137.0 /154

Clean `2293ff8` packages the bounded repair with the matching agent and unchanged SDK/examples/notices. Exact copied native smoke and four public adapters pass under development-file/outbound-IP denial, followed by LED/RC/flasher parity, invalid-value/correction native-headless parity and live pulse/Pause/Resume/save/reopen through 3.436 s. [Receipt](first-version/value-recovery-review-checkpoint.json). The 151-entry manifest, fourteen earlier qualified reviews, original /147 candidate and canonical /120 are verified. Owner walkthrough remains Not tried yet. Prior source-native typing/Apply proof keeps its recorded development-binary scope; physical OS input, reader/IME, visible p95, broader environment and study gates remain open. Packaging is not promotion; no owner window or printer was operated.

October 18: [recorded-cursor handoff reproduction and source repair](../../docs/validation/recorded-cursor-handoff/README.md) closes one-click Clear/Select through the shared field-settlement path with recording identity and existing command/document guards. Source/mounted validation is separate from clean replacement packaging and OS/reader input qualification.


### Recorded-cursor replacement · 0.138.0 /155

Clean `7062594` packages recorded-cursor settlement with the matching agent and unchanged SDK/examples/notices. Exact copied native smoke and four public adapters pass under development-file/outbound-IP denial, followed by reference circuits, invalid-value correction, saved-capture Select/8 ms interval/Clear/startup placement and data-preserving fresh reopening. Pulse/Pause/Resume/save/reopen reaches 3.414 s. [Receipt](first-version/recorded-cursor-review-checkpoint.json). The 134-entry manifest, fifteen earlier reviews, original /147 candidate and canonical /120 are verified. Owner walkthrough remains Not tried yet. Pinned source mounted input is distinct from copied public commands and physical OS/IME/reader evidence. No owner window or printer was operated; packaging is not promotion.

October 18: [recorded-control ownership reproduction and source repair](../../docs/validation/recording-control-owner/README.md) verifies private cursor/seek/window text cannot carry to another selected capture, while same-capture repaint and fresh new-capture input work. Source/mounted evidence remains separate from clean replacement and OS/IME/reader acceptance.


### Recorded-control ownership replacement · 0.139.0 /156

Clean `22a84d5` packages capture-owned controls with the matching agent and unchanged SDK/examples/notices. Exact copied native smoke and four public adapters pass under development-file/outbound-IP denial, followed by reference circuits, invalid-value correction, recorded measurement/reopening and two distinct saved-capture target/refusal/data-preservation checks. Pulse/Pause/Resume/save/reopen reaches 3.416 s. [Receipt](first-version/recording-owner-review-checkpoint.json). All 135 manifest entries, sixteen earlier reviews, original /147 candidate and canonical /120 are verified. Pinned-source mounted cursor/seek/window draft reset remains distinct from copied public commands and physical OS/IME/reader evidence. Owner walkthrough remains Not tried yet; packaging is not promotion. No owner window or printer was operated.

October 18: [current /156 named-wiring parity](../../docs/validation/wiring-current-156/README.md) binds fresh mounted/assembled-adapter observations to the exact packaged agent and producer identity. Completed LED wiring and Undo match; source/SDK/app bytes remain unchanged. Publicly seeded parts, callback/menu scope and remaining physical/reader/IME/latency gates are explicit.

October 18: [native /156 Save As/Open and agent parity](first-version/native-files-156/README.md) uses actual macOS panels to save the LED baseline and restore it after a separately saved agent change. Fresh graph/results match; the independent packaged agent produces byte-identical saved JSON. Original host exits normally. Initial unintended filename evidence and remaining file/typing/reader/timing gates are explicit. Source/SDK/release bytes are unchanged; owner walkthrough remains Not tried yet.

October 18: [native /156 unsaved choices](first-version/native-decisions-156/README.md) verifies Cancel, picker Cancel, Discard and Save and continue on saved LED projects, including dirty-state preservation, unchanged original manual save, outgoing edited file and fresh packaged-agent reopening. Two observer schema failures are retained separately from the passing final host, which closes normally. Source/SDK/release bytes, version and owner walkthrough status remain unchanged.

October 18: [native /156 untitled Save-dialog cancellation and continuation](first-version/native-untitled-156/README.md) returns to the same decision with the circuit intact, then saves the untitled graph before opening the requested target. Native/agent graph, scope, results and final history agree; a blank-start agent reopens both files. An earlier observer reference failure and excluded blank tool instance are explicit. Final host exits normally; source/SDK/release bytes and owner status are unchanged.

October 18: [native /156 live RC cursor entry/timing](first-version/native-rc-cursors-156/README.md) checks actual ASCII unit entry, one-click focus-loss selection, Enter and Clear against the exact packaged agent and independent RC laws. Visible timing is 12 ms /83.33 Hz reciprocal; all 21 samples, circuit, history and saved document remain unchanged. The original host closes normally. Seeded graph/trace and broader input/reader/timing limits are explicit; source/SDK/release bytes and owner status are unchanged.

October 18: [native /156 selected-window RC CSV Cancel/Save](first-version/native-rc-csv-156/README.md) exports 13 selected samples with matching packaged-agent bytes, SI/provenance and independent RC checks. Cancellation and export preserve the circuit, full 21-sample trace, scope/history and saved project. Original host closes normally; source/window seeding and remaining output/input/reader/timing gates are explicit. Source/SDK/release bytes and owner status are unchanged.

October 18: [live cursor run-ownership source repair](../../docs/validation/live-cursor-owner/README.md) reproduces A/B private drafts committing into an agent-restarted run, then clears old-run state while preserving same-run repaint/sampling and fresh input. Full source quality and development native smoke pass. Clean replacement qualification is pending; /156 and canonical /120 remain unchanged and do not contain this new repair.

October 18: [live interval run-ownership source repair](../../docs/validation/live-window-owner/README.md) reproduces both boundary drafts changing an agent-restarted run and then restores correct ownership, preserving same-run input and matching fresh edits to the full scope of an in-process public service. Final combined source quality and development native smoke pass. The bounded sibling check is complete; one clean replacement with both cursor and interval repairs is next. Preserved /156 contains neither fix; owner status is unchanged.

### Live measurement ownership replacement · 0.140.0 /157

October 18: [clean replacement qualification](first-version/live-measurement-review-checkpoint.json) incorporates both live cursor and interval run-ownership repairs. Source/mounted quality and exact copied smoke/four-adapter/reference/measurement/live/save checks pass. Native/headless restart, stale-revision refusal and fresh scope edits agree; private input reset remains pinned-source mounted evidence. All 158 files, seventeen prior reviews, the original /147 candidate and canonical /120 verify. Manifest-digest metadata reporting is corrected without changing older sealed receipts or their per-file verification. SDK/models/format/commands are unchanged; owner Not tried yet and broader MVP gates remain open.

October 18: [native /157 live cursor/interval run ownership](first-version/native-live-owner-157/README.md) confirms private A/start drafts reset after an external agent restart, with replacement-preserving Enter and observed fresh input matching the packaged agent. Initial rapid cursor refusal and visibly truncated interval text remain explicit, corrected retries; cause is unestablished and rapid typing is not qualified. Original host closes normally. Source/SDK/release bytes and owner status are unchanged.

October 18: [native /157 six-attempt typing reproduction](first-version/native-typing-157/README.md) fails requested-input qualification: four intended values, one refusal retaining the prior cursor, and one requested draft visibly absent before Enter. A separate-focus control works, but cause remains unestablished. All domain/saved state survives and original host exits normally. One bounded causal session is next; no product/SDK/release change or broader typing pass is claimed.

October 18: [native input causal diagnostic](first-version/native-typing-trace/README.md) captures legacy characters followed by Cocoa inserts using an old baseline, plus an exact truncated-time validation refusal. The trace narrows the repair boundary; no fix or successful typing qualification is claimed. Original host exits normally and reusable test bundle returns to exact /157. Source/SDK/review bytes remain unchanged. One native-boundary regression/source-repair pass is next.

October 18: [native legacy/Cocoa handoff source repair](../../docs/validation/native-legacy-handoff/README.md) passes three old-source failing Cocoa cases, bounded acknowledgment/FIFO/Enter and malformed-marker guards, complete SDK/consumer quality and development native smoke. All six actual native cursor entries and a complete interval match the packaged agent without corrections while preserving work. SDK `274f5cc` changes eight members, with dependencies/bootstrap and prior reviews preserved. Clean replacement is next; owner and broader input/reader/IME/latency gates remain open.

### Native text handoff replacement · 0.141.0 /158

October 18: [clean replacement qualification](first-version/native-text-review-checkpoint.json) incorporates SDK `274f5cc` from clean source `622d31b`. Exact copied native smoke/four-adapter/reference/measurement/recorded/live/save checks pass. Source Cocoa/FIFO and development native six-entry/interval behavior retain distinct scopes. All 196 files, eighteen prior qualified reviews, original /147 candidate and canonical /120 verify. Commands/models/format and native ledger remain unchanged; owner Not tried yet and broader typing/IME/reader/latency/environment/distribution/study gates remain open.

October 18: [native /158 value/Apply and keyboard close](first-version/native-close-158/README.md) verifies complete ASCII entry, one Apply, Cmd+W/Cancel preserving edits/manual save, and fresh Cmd+W/Save-close with exact user save metadata and agent reopening parity. Clean semantic close precedes normal original-host exit; queued shutdown flags retain their scope. Source/SDK/release bytes and owner status are unchanged.

### Numeric placement replacement · 0.142.0 /159

October 18: [clean replacement qualification](first-version/numeric-placement-review-checkpoint.json) incorporates SDK `b22478c` from clean source `37046dc`. Exact copied native API smoke/four-adapter/reference/measurement/live/save checks pass. Source-mounted and development native accessibility/pointer commitment retain their separate scope. All 257 files, nineteen prior qualified reviews, original /147 candidate and canonical /120 verify. Commands/models/format remain unchanged; owner Not tried yet and broader input/reader/latency/environment/distribution/study gates remain open.


### Native /159 LED construction from blank

[Four actual placements, native named-wire menus, Undo/Redo and Fit](first-version/native-draw-159/README.md) now match a separate packaged agent, with fresh 2.917 mA, truthful stale readings after Undo, and public save/fresh-reopen parity. No native graph seeding, owner work or sealed package change. This closes the stopped construction portion, while current-build value editing/Delete/reconnect and other acceptance scopes remain separate.

October 18: [native /159 LED editing](first-version/native-edit-159/README.md) verifies 2 kΩ Apply, Undo/Redo, Delete/reconnect with truthful stale state, invalid-value refusal and pointer correction to fresh 1.345 mA against the exact packaged agent. Public saves and fresh-agent reopening match; original PID54987 exits normally and the disposable bundle is restored. **A new AX recovery blocker is open:** invalid Apply leaves inspector controls visible but absent from repeated accessibility trees until successful pointer correction. Domain assertions passing does not qualify AX recovery. No source/SDK/version change, promotion or owner acceptance.

October 18: [grouped-inspector source repair](../../docs/validation/inspector-accessibility/README.md) preserves the external AX value/Apply targets through invalid-value refusal and actual CUA correction without coordinate fallback, matching packaged-agent graph/history/results/scope and fresh saved-file reopening. In-process Cocoa controls remain published even in the flat diagnostic; the evidence does not establish a framework root cause. The complete consumer gate (189.732 s), full native smoke (39.112 s), final exact-binary rebuild, normal original-host exit and preservation of all 20 reviews/canonical pass. No SDK/command/model/format change. Clean replacement qualification is next; /159 remains affected, owner remains Not tried yet, and broader acceptance stays open.

### Inspector recovery replacement /160 qualified · October 18

[Clean replacement qualification](first-version/inspector-recovery-review-checkpoint.json) packages source `4403097` with unchanged SDK `b22478c`. Exact copied native smoke (38.840 s), four public adapters (48.287 s), LED/RC/flasher parity, value/measurement/recording checks, pulse Pause/Resume through 3.442 s and save/reopen pass. Actual CUA AX-identified refusal/correction on the exact /160 executable preserves protected state, keeps targets discoverable, applies once and matches the exact agent at 1.345 mA; saved files/fresh reopening agree. All 343 entries, 20 earlier qualified reviews, original candidate and canonical verify. Original test PID65563 exits normally and its stopped custom bundle is restored. Source and exact-native scopes remain explicit; this is not whole-bundle/general reader/IME/latency or owner acceptance. Owner remains Not tried yet; P0 stays 17 implemented /10 partial. No promotion or feature expansion.

### RC input-edge replacement /161 qualified · October 18

[Clean replacement qualification](first-version/rc-input-frame-review-checkpoint.json) packages `35371ee` with unchanged SDK `b22478c`. Exact copied native smoke (39.164 s), four adapters (48.825 s), reference/measurement/recorded/live/save checks pass. Independent native/headless RC equations validate both edge samples, all 23 scope/CSV rows, cursors/extrema and correct −.3160602794 mA at 10 ms. Native closes normally; all 381 entries, 21 earlier reviews, original candidate and canonical verify. No new physical input, reader, latency or owner acceptance is inferred. Owner Not tried yet, P0 17 implemented /10 partial, no promotion or scope expansion.

### Native RC from blank · /161

[Construction and core-control proof](first-version/native-rc-draw-161/README.md) verifies four placements, 10k Apply, named terminal wiring, charge/discharge samples, right-hand cursors and actual Resume/Pause against the packaged agent. All 43 final scalar/probe rows obey independent RC laws. Public saves/fresh startup reopening match; original host closes normally and stopped custom bundle is restored. Producer failures retain their scopes. Minor incomplete-network scope caption is deferred; no numerical failure. Source/SDK/version and 381 sealed entries unchanged, owner Not tried yet and P0 17 implemented /10 partial. Remaining paused post-trace value edit and flasher checks are separate.

### Native paused RC resistance edits · /161

[Stale/reset/history proof](first-version/native-rc-paused-edit-161/README.md) verifies actual value Apply, retained old trace ownership, visible stale cues/disabled Step, explicit Restart, Undo/Redo and direct Resume. Ninety fresh scalar/probe rows match independent laws; native state matches the packaged reference, and a separate actual agent Resume verifies zero-time reset. Both saved configurations reopen clean, original hosts close normally and baseline files/bundle are preserved. The evidence-routing error and missing first baseline image are disclosed. Source/SDK/version unchanged, owner Not tried yet, P0 17 implemented /10 partial; native flasher is next.

October 19: [Verify form handoff repair](../../docs/validation/verification-handoff/README.md) retains eleven original-production mounted failures and focused guarded field/agent/RC-equation tests. Full source quality passes in 217.577 s and development native smoke in 43.183 s. Failed fixture assumptions and their guarded repairs are retained. Preserved /166 remains affected until a clean replacement is qualified; no new physical OS input, latency or owner-acceptance claim.

### Verify replacement · 0.150.0 /167

Clean `d58c145` /SDK `b22478c` packages guarded Verify/reset field settlement and field-specific invalid-text refusal. Exact copied native smoke (41.856 s), four adapters (51.375 s), reference/value/preset/measurement/live/save checks and four Verify scenarios pass, including identical 424-row native/headless data and independent RC laws. [Receipt](first-version/verification-review-checkpoint.json), [evidence](../../docs/validation/verify-review-167/README.md). All 1,073 manifest entries, 26 earlier reviews, two candidates and canonical /120 verify unchanged. Source-mounted private input remains distinct from copied public commands and historical physical evidence. The ZIP preserves all 1,074 files/modes and extracted examples/fresh reopens pass. Owner Not tried yet; no promotion. The preceding source checkpoint’s pending replacement statement is superseded; /166 remains affected and preserved.

October 19: [saved-coil startup reset reproduction and repair](../../docs/validation/verification-startup-reset/README.md) retains original RL/RLC captures incorrectly starting at +30 mA after a reset of a project saved at −20 mA. Shared-default projection restores the intended capture while preserving duration, saved components, live state/history and intervening agent work. Source/mounted checks pass; full source quality passes in 213.058 s and development native smoke in 40.913 s. A clean replacement remains pending. /167 remains preserved and affected by this scenario; no new physical OS input or owner acceptance is claimed.

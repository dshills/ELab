# Shutdown error action cutoff

October 19, 2026. **SHUTDOWN-CUTOFF-01**: long actual cleanup errors can leave
only six vertical pixels of “Close window anyway” visible at 1040×720. The
source repair puts error details in their own scrolling region and keeps the
shared acknowledgement action outside it. No command, error, shutdown, solver,
model, saved-format or SDK semantics change. Replacement packaging is separate.

## Evidence and scope

The regression creates a real malformed appearance file within its 4,096-byte
limit and an owned nonempty directory at the private recovery lease path. The
resulting errors are about 796 and 3,516 bytes. It saves a synthetic LED project
before admitting shutdown. No owner files, preferences or applications are used.

The exact final test, with only the original `desktop/shutdown.go` supplied by a
Go overlay, fails in light and dark at 1040×720: button bounds are
`(32,706)–(208,712)`. The 1280×800 control passes. This is an initially clipped
action, not proof that every route to closing is inaccessible: the original
generic probe can activate its tiny target and the original outer layout has a
scrollbar. The issue is the unreadable initial action at the supported minimum.

With the repair, all three scenarios pass under race instrumentation. The
button is 49 pixels high, wholly inside each scene. In both minimum-size cases,
wheel input moves the error bounds while the footer bounds stay exact. One
pointer click invokes the existing `app.close.acknowledge` as user, advances the
command revision once, retains failed status and full original errors, and
preserves graph/result/document and all three protected files. The existing
button/window acknowledgement cases also pass. Full errors are retained rather
than shortened to make the layout fit.

These are real retained widgets and input routing in the pinned offscreen
driver, with 160 logical DPI and supplied embedded fonts. The final scenarios
use `desktopTheme()` kind styles and sizes, adding only a transparent managed
default background and Noto Sans family required by the manual adapter. They
are software layout evidence, not a new physical native input, OS font/display,
reader, installation or timing qualification.

The final light/dark minimum-size frames and larger control are in `final/`.
The test writes frames only when explicitly given an artifact directory;
ordinary checks do not write into the repository.

## Initial diagnostic limitations

The first producer used `Image` instead of the driver's `Capture` method and
failed to compile. Its corrected generic-theme probe then passed a weak check
that required only in-scene bounds and click success; image inspection exposed
the six-pixel target. The stronger readable-height check fails. These producer
errors and the misleading initial pass remain in `diagnosis/`.
The initial written decision condition was likewise too narrow: an in-scene,
clickable six-pixel target would satisfy it. Image inspection justifies replacing
that condition with the readable-height regression; the original condition and
pass are preserved rather than relabeled as an original failing test.

The generic control-test theme also forces a light default background, so it
cannot establish native dark-theme contrast. Direct `desktopTheme()` initially
refuses manual configuration because managed colors and a font family are
required. The final adapter control supplies those required values, preserving
the app's kind styles and sizes. Its original-layout overlay still reproduces
both minimum-size failures. No framework or general theme repair is inferred.

Only `desktop/shutdown.go` changes among production Go sources. Two existing
offscreen test files add the regression and a sized/theme-aware helper; its
existing 520×180 callers retain their exact original configuration. The reviewed
native ledger differs only in the shutdown source hash and the unchanged
acknowledgement site moving from line 15 to line 22. It remains 72 sources,
397 sites, 132 source-routed IDs and 164 public commands.

Full quality, source pins, preservation and the final checkpoint are recorded
separately. No package pass is inferred from the focused tests. Owner walkthrough,
broader accessibility/conformance/environment and actual hobbyist-study gates
remain open; canonical /120 and the /175 review are preserved.

## Final source gate

The final default `GOPROXY=off ./scripts/check.sh` exits 0 in 226.748 s:
SDK/import/format/Staticcheck, native ledger and PRD-audit consistency,
54 Python checks, both modules' race/vet, and selected mounted controls/vet pass.
The final 650-file Go pin set is checked before this checkpoint. The original
overlay remains an expected failure and its larger-window control remains a
pass. This is not a new solver, visible-latency or full-accessibility decision.

Read-only replay:

```sh
python3 -B docs/validation/shutdown-error-layout/verify.py
```

The source repair is qualified; one clean matching private app/agent replacement
and its copied-runtime qualification are next. The existing /175 app, agent,
ZIP, SDK and canonical /120 hash unchanged. No replacement release or promotion
is claimed by this source checkpoint.

The repository whitespace check flags literal context whitespace inside the archived unified patch. `source.diff.gz` now preserves its exact bytes without creating whitespace warnings in source review. The initial manifest/checkpoint and failed check remain in `diagnosis/`; no Go source or test outcome changes.

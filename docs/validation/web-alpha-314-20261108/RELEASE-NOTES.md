ELab **0.207.0-alpha.2 (build 314)** fixes an uncaught WebKit error when reloading the web workbench while a state update is in flight.

Download **ELab-Web-0.207.0-alpha.2-macos-arm64.zip**, extract it and open **ELab Web.app**. The launcher starts the local engine and opens your default browser. No development tools or account are needed.
[Start guide](https://github.com/dshills/ELab/releases/download/v0.207.0-alpha.2/START-HERE.md).

The workbench now explicitly cancels the old page's read before navigation, stops polling when a page is hidden for navigation, and resumes after a Back/Forward restore. Cancelling navigation keeps the current page updating. Real connection errors remain visible; editing commands and solver limits retain their existing safeguards.

This update retains guided LED, capacitor and four-MOSFET H-bridge activities, connected group editing, circuit diagnostics, library backups and private support reports. Existing web projects remain in `~/Library/Application Support/ELab/web`. Earlier releases and native-source archives are preserved.

**Qualification:** the fresh local macOS quality plan passed on release source `6c0d66e3cafcedb0228e5db978fb6c4f6089d5b8`: lint, 75 Python tests, 36 frontend checks, the complete Go race suite, vet and package checks. All **46 browser journeys passed**, with two planned non-Chromium load skips and no unexpected or flaky results.

The exact GitHub-downloaded shipping backend passed **three consecutive in-flight reloads each in Chromium and WebKit**, explicitly aborting the old reads and resuming polling without uncaught errors. Both engines also passed LED editing, group undo, four-MOSFET reversal and exact preservation of all **50 recorded H-bridge rows** through named save/reopen and browser export/import, plus private diagnostic and library-backup checks. Archive inventory/signature, recovery and normal restart/quit checks passed.

[Hosted CI](https://github.com/dshills/ELab/actions/runs/41888150836) passed all five jobs on `e5136106d1d49cb13e979f0ad9337bccd012f21a`. Its application, test and workflow input hashes exactly match this release commit; the intervening commit contains documentation only. Fresh local release gates and downloaded-binary checks use the release commit itself. Detailed scope is in `QUALIFICATION.json`.

This private alpha is **ad-hoc signed and not notarized**. Follow the start guide for Apple's per-app opening guidance if macOS blocks the first launch. The exact ZIP is tested on Apple Silicon macOS 27.2. macOS 13 is a configured build target, not a verified minimum. Intel, clean-account/Finder installation, native file-picker interaction, VoiceOver and whole-app accessibility conformance are not qualified by these automated checks.

The release includes `START-HERE.md`, `PACKAGE.json`, `QUALIFICATION.json`, `release.json` and `SHA256SUMS`. Verify all downloaded payloads with `shasum -a 256 -c SHA256SUMS`.

Built from clean `main`: `6c0d66e3cafcedb0228e5db978fb6c4f6089d5b8`.

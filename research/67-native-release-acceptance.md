# Native release acceptance and inspector repair

October 21, 2026.

The first native session on the public-Vellum candidate **0.177.0 /194** passed the targeted recording-input, coupled-threshold, probe-ownership, Verify/reset and DC-sweep correction paths. Exact numerical and saved-data comparisons passed through its matching agent. The [evidence and limitations](../docs/validation/native-acceptance-194/README.md) distinguish actual input, accessibility observation, public fixture setup and headless comparison.

The session also exposed a first-version blocker: after scrolling the long recording inspector and opening the LED example, the inspector could remain blank. The schematic and toolbar still worked, and the LED displayed 2.917 mA, but the new document inherited the old document's scroll viewport. This is not an accepted release state.

The mounted retained-widget regression fails before the repair, with an empty selection-control rectangle after replacing the recording with the LED document. The repair changes only the inspector container's ownership: a replacement document gets a fresh viewport; ordinary repaints and component edits preserve the current viewport. User and agent replacement paths share the same document generation, and public command identities are unchanged. The source inventory changes only its `desktop/view.go` checksum.

The exact /194 app does not contain this repair. [Full source quality](../docs/validation/inspector-scroll-ownership/README.md) now passes, including the new mounted regression, the required desktop suite, race/vet/static checks and all 67 Python tests. A fresh native candidate must still demonstrate the repaired document transition before review. No feature has been added, no owner package replaced and no release promoted. The four owner reference workflows, remaining native field branches and complete layout checks are still open.

## Repaired candidate result

**0.178.0 /195** is now a separate [review app](../build/ELab-0.178.0-195-candidate/ELab.app), built from clean `8d7d646344cf8a98a9638494fa0fe4d5661e2e3a`. The [native/package evidence](../docs/validation/inspector-candidate-195/README.md) confirms that the deeply scrolled recording-to-LED transition restores the inspector. Native LED value editing and Undo/Redo give 2.917 → 1.477 → 2.917 → 1.477 mA. Canceling Save and unsaved replacement preserves the edit. Native Save, Open and SVG export pass; the saved graph/results match the packaged agent.

LED, RC and flasher reconstructions match the native endpoint, packaged agent and historical numerical references. The three reference reconstructions use public commands; they do not constitute physical drawing/wiring. Both themes render the restored inspector at 1053×802. The full source gate passes on all 671 Go inputs. The 124-file package passes signature, ZIP/extraction and three extracted-agent example reopens. The tested native host is macOS 26.6.2 arm64. Both isolated native sessions closed normally, and all original app packages remain unchanged.

The next work remains the unchecked input branches, minimum/normal-size layout pass and complete physical four-workflow walkthrough, followed by owner first-use feedback and release designation. No feature expansion or further framework update is needed. /195 is a concrete owner-test candidate, not a promoted release. The [walkthrough](35-first-version-review.md) now names this exact candidate; the previous /189 guide is preserved separately.

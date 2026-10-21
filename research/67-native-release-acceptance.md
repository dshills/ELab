# Native release acceptance and inspector repair

October 21, 2026.

The first native session on the public-Vellum candidate **0.177.0 /194** passed the targeted recording-input, coupled-threshold, probe-ownership, Verify/reset and DC-sweep correction paths. Exact numerical and saved-data comparisons passed through its matching agent. The [evidence and limitations](../docs/validation/native-acceptance-194/README.md) distinguish actual input, accessibility observation, public fixture setup and headless comparison.

The session also exposed a first-version blocker: after scrolling the long recording inspector and opening the LED example, the inspector could remain blank. The schematic and toolbar still worked, and the LED displayed 2.917 mA, but the new document inherited the old document's scroll viewport. This is not an accepted release state.

The mounted retained-widget regression fails before the repair, with an empty selection-control rectangle after replacing the recording with the LED document. The repair changes only the inspector container's ownership: a replacement document gets a fresh viewport; ordinary repaints and component edits preserve the current viewport. User and agent replacement paths share the same document generation, and public command identities are unchanged. The source inventory changes only its `desktop/view.go` checksum.

The exact /194 app does not contain this repair. [Full source quality](../docs/validation/inspector-scroll-ownership/README.md) now passes, including the new mounted regression, the required desktop suite, race/vet/static checks and all 67 Python tests. A fresh native candidate must still demonstrate the repaired document transition before review. No feature has been added, no owner package replaced and no release promoted. The four owner reference workflows, remaining native field branches and complete layout checks are still open.

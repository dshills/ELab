# Native window lifetimes

Vellum's `2ced41a` window API identifies a particular opening with an opaque `app.WindowRef`. A logical name such as `main` may be reused after its window closes; resolving that name when delayed work finishes could target a different opening.

ELab's production startup captures the reference atomically through `OpenWindow` before installing the native pointer bridge. A screen retains its original reference. Fixture screens may capture it once at their first window operation; retirement never makes that old screen adopt a replacement. References are process-local implementation state, not project data or public command IDs.

File pickers and file dispatch, print-preview decode, and recovery-result decode use the originating window context and retain its reference before starting background work. Publication uses `PostWindow(ref, ...)`. A file picker that ignores cancellation cannot cause a later file dispatch after the window context is canceled. Already admitted commands retain their existing cancellation/publication rules; this is not arbitrary-code preemption.

Popup results, native pointer packets, queued animation and chrome setup also retain the original opening. The schematic, scope and sweep extensions capture the managed mount's scope. Their owner callbacks recheck its canceled context after acquiring the UI owner, preserving existing command/document/frame/gesture guards. Framework cleanup remains responsible for releasing retired resources. Protected Close still uses the same shared Save/Discard/Cancel commands and decisions.

Synthetic tests keep a second fixture window open, retire `main`, reopen that name, and prove that the old screen/callback refuses while a fresh screen can deliberately capture the replacement. A blocked picker observes window cancellation while the application stays live. Existing close and sampling regressions run with the new API. This does not add multiwindow editing as a product feature or establish full physical-input, screen-reader or platform-lifecycle acceptance.

The SDK checkout is reviewed read-only. An external update interrupted the prior release pipeline after its tests/workflows/timing passed; the SDK consistency assertion correctly prevented promotion. Current qualification must use the migrated sources and an unchanged source/manifest baseline. Earlier screenshots retain their original SDK and candidate attribution. See [sampling behavior](lc-sampling.md) and the [continuing implementation record](../research/10-continuation-progress.md).

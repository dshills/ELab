# Inspector viewport ownership

The /194 native recording-to-LED transition could leave the right inspector blank. The retained-widget regression reproduces the shorter document's selection control with empty bounds after a deeply scrolled NPN recording. The first attempted test fixture lacked explicit offscreen fonts/colors; that setup refusal is retained separately from the corrected, failing reproduction.

The repair gives the inspector scroll host an identity belonging to `Document.Generation`. Replacing a document mounts a fresh viewport. The mounted regression verifies that ordinary repaint and component-value changes keep the existing host and scroll position, then verifies the LED replacement exposes its selection control. Child command/control IDs and public commands are unchanged. This is an ELab change; the pinned Vellum SDK is untouched.

The complete `scripts/check.sh` passes: source inventory, formatting/static analysis, 67 Python tests, root and desktop race tests, vet and the required mounted suite including the new inspector case. All 671 Go input hashes remain unchanged during qualification. The initial full check correctly rejected the stale source-inventory checksum; review found only the `view.go` checksum changed and no source route changes, then the ledger was regenerated and the entire check rerun successfully. Desktop race suites completed in 168.171 s and 139.434 s; these are test durations, not product latency claims.

[Report](report.json), [raw manifest](raw-manifest.json), [raw evidence](raw.tar.gz). Source acceptance does not establish the repaired native package's UI, owner workflows, clean-account startup or release approval. A new native candidate must demonstrate the same document transition.

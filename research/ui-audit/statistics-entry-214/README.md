# Existing native measurement baseline

October 23, 2026. Reused the already-running owned `ELab Cursor Recovery` test
session, version 0.181.5 /206, source
`4010b891b66fc56d57ffd879a6a2ee6aa86ebfd4`. This is an original-behavior check,
not native acceptance of the /214 repair. No new graphical app was launched.

1. Selected C1 voltage in the real captured-measurement menu.
2. [Typed `-1 ms`](01-typed.png) into Measurement start.
3. [Return refused it and replaced it with `0 s`](02-refused.png), preserving
   the accepted 0–20 ms result.
4. [Re-entering `0 s` cleared the error](03-correction.png). A stuck correction
   is not demonstrated by this old native session.
5. Turned measurements off and [restored the inspector scroll](04-restored.png).

[Receipt and image hashes](report.json). Full controller snapshots, source
build info and exact runtime provenance are retained with the /214 validation
archive. The restored full snapshot differs only in `revision` and
`command_revision`; the owned app remains open for reuse.

The first paste tool call reported a possible clipboard conflict. A fresh AX
and screenshot inspection proved the intended `-1 ms` was present before Return;
no unknown text was committed. The tool warning is not classified as an ELab
input failure. Screenshots are app-window captures; pointer/automation overlays
are not evidence of a product cursor defect.

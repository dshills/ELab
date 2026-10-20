# Eight-route junction review · /179

The unchanged **0.162.0 /179** copied headless agent passes twenty bounded
wiring checks using ordinary shared commands. No application defect is found.

Nine synthetic routes initially remain nine separate nets. An explicit
`wire.junction.insert` joins exactly eight, retaining the ninth crossing as a
separate net. The original eight route IDs stay on their first legs; eight new
IDs identify second legs. Giving the junction the valid ID `W1` correctly skips
that reservation when allocating `W2` through `W9` for the new wires.

Fourteen malformed/domain/target requests preserve the **complete** public
snapshot. These include empty/over-eight/duplicate/unknown routes, missing/null
multi-route positions, incomplete/null coordinates, off-route/occupied/outside
positions and invalid/already-used IDs. The case named “near endpoint” snaps
onto an occupied helper center: it proves refusal, not independent coverage of
the later endpoint-clearance check. These cases do not test the 300-wire or
100-object capacity boundaries; those limits are checked against source and the
runtime snapshot here and retain their separate existing capacity evidence.

A valid eight-route insertion followed by a bad connection in `edit.batch`
rolls back the entire graph **and** its staged ID reservations. The subsequent
successful insertion uses those same `W1`–`W9` IDs. One Undo restores the original
graph, and Redo restores exact split identities. Deleting the junction removes
all sixteen incident legs and retains only the unrelated ninth wire; Undo
restores it. After undoing a successful insertion, explicit reuse of `W1`, `W2`
or `W9` correctly refuses without changing the snapshot or Redo history.

A fresh process opens the final graph and resaves **byte-identical JSON**.
An independent explicit-wire traversal agrees with all reported net memberships
before/after the join, history operations, deletion and fresh opening. These are
incomplete connectivity-helper drawings; this adds no numerical solver claim.

All **78 requests** and responses are retained. Eighteen expected refusals
(fourteen ordinary inputs, one late batch and three reserved IDs) preserve
complete state. Both owned processes finish with completed, joined,
recovery-finalized, exit-ready shutdowns and exit 0. No collection or cleanup
failure occurs. No UI window, user settings, owner project, privacy grant,
display recording, printer or network operation is involved.

The junction guide's obsolete 50-object/150-wire statement is corrected to the
current **100 objects /300 wires**, with a link separating drawing capacity from
the 64 simulated-part bound. Capacity documentation also distinguishes grounded
waveform sources from permitted floating DC sources. `workbench/limits.go`,
`network_compile.go`, the drawing handlers and runtime snapshot establish these
rules. These documentation corrections change no runtime limits or handlers.

All **654 Go inputs**, the sealed /179 review, SDK and canonical /120 remain
unchanged. No rebuild, version change, promotion, P0 status change, additional
native/latency/accessibility/owner or full-MVP acceptance follows. Existing full
Go quality results apply to unchanged sources; they are not repeated for this
data/documentation-only review.

`raw/` contains lossless compressed protocol, snapshots, project and process
outcomes. `producers/` preserves the original collector as nonexecuting text and
its output; private paths describe this historical run. Replay without launching
an application:

```sh
python3 -B docs/validation/junction-boundary-review-179/verify.py
```

Replay verifies protocol identity/guards, whole-state refusals, graph equations,
net partitions, history, ID allocation, persistence, lifecycle, file hashes and
source/artifact pins. Optimized Python refuses before evidence reads. Broader
physical drawing/input, exhaustive agent contracts and owner walkthrough remain
separate gates.

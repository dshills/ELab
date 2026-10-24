# Native interval baseline /210 work

October 23, 2026. These unmodified CUA captures come from the already-running
owned `ELab Cursor Recovery` /206 fixture. **They show old code, not the repair.**
No application launch, user-project access or device preference change is used.

- `01-native-before.jpg`: live interval end is 20 ms. Its full AX companion is
  captured before scrolling; the image shows the field after scrolling it into
  view. The underlying tree and field identity are unchanged by that scroll.
- `02-native-rejected.jpg`: typing 30 ms and Return discards that text, restores
  20 ms and shows “That request could not be applied.” The AX diff adds that
  refusal; its end-field value remains 20 ms.
- `03-native-corrected.jpg`: a temporary valid 18 ms entry clears the refusal and
  shows 19 interval samples, without changing the retained simulation history.
- `04-native-restored.jpg`: after entering 20 ms and choosing Follow live history,
  the original interval and 21-sample statistics are restored.
- `05-native-parked.jpg`: the inspector is returned to its original top position,
  with the fixture still paused at 0.02 s.

Fresh matching-agent reads preserve the complete before/after snapshots and
prove that only `revision` and `command_revision` differ. All numerical samples,
scope/cursors, recording, graph, document, history, viewport and other snapshot
fields match exactly. The ordinary owner session and projects are untouched.
CUA pointer glow/tooltips are capture effects, not attributed app defects.

[Repair and check scope](../../87-scope-interval-correction.md). Changed-binary
native acceptance remains pending and cannot be inferred from these captures.

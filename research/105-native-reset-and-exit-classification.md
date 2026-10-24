# Native interval reset and exit classification

October 24, 2026. [Evidence and current checklist](../docs/validation/native-interval-reset-223/README.md).

Native item 8 passes without changing product code. Both live and recorded views
reset on the first click after valid or invalid unfinished start/end entries.
Timed held-button trials overlap real agent edits; the old gesture preserves
the complete agent state exactly, and a fresh reset succeeds. All physical state,
1,001 recorded rows and 2,044 independent voltage references are preserved.
The test app is restored clean and paused; signatures, saved files and /205 stay
unchanged. Existing source gates still apply to all 897 unchanged inputs.

The previous unexplained exit is now traced to an AppKit Quit-menu action at
04:12:42.358, followed by the native terminate/close sequence. NSTerminateCancel
is expected in the pinned GLFW delegate while it forwards window-close requests.
The menu action's initiator remains unknown. There is no demonstrated spontaneous
crash or restoration defect, so no speculative fix is made. General lifecycle
acceptance remains separate from this classification and bounded surviving process.

Nine original native checks remain. Next: recorded-position refusal/correction
and queued pre-agent parser ownership, then captured-measurement defaults and
cached-result presentation. Preserve the private-alpha download and continue
without waiting for owner review or another launch/desktop permission reply.

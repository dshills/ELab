# Spectrum peak markers and cursors — 2026-10-28

User request: add peak markers and frequency/amplitude cursors, shared user/agent
commands, tests, internal review and commit. No external review provider used.

Implementation:

- Two public commands: `spectrum.markers.set` and `spectrum.cursor.set`.
- Deterministic top-five local peaks: exclude DC/floor, retain candidates within
  60 dB of strongest, plateau/ties use lower frequency. No interpolation claim.
- Independent live/recorded A/B frequency anchors, nearest actual bin readings
  in Hz, peak V/A and dB relative to one unit. Signed B−A deltas; floor-limited
  bins omit an exact dB difference. Optional exact-result guard for agents.
- Live frame updates preserve anchors and recompute readings, without changing
  the FFT engine. Source changes retire anchors and private UI drafts. Hold
  freezes the displayed measurements. Document replacement resets session state.
- Fixed controls/readouts, selectable A/B, click/drag, unit-text fields, strongest
  peak placement, peak visibility and clear actions. Small-current dB scale is
  usable below one unit. Existing captured exports and physics stay unchanged.
- Ordinary command discovery retains its existing 64 KiB budget by referencing
  shared input schemas; detailed discovery still expands complete contracts.

Internal review includes ownership of nested snapshot data, command admission,
source/result retirement, actual bin math, absence of physics/history/dirty writes,
plot gesture revision protection, retained field ownership and focus-loss actions.
The mounted tests reproduced a new focus-loss Clear conflict and a private
frequency draft crossing from voltage to current. Narrow accepted-field
continuation and source-specific control identity repair both. An initial
source-switch fixture kept Hold enabled while awaiting a new frame; that fixture
was corrected before the draft defect was reproduced.

Tests distinguish pure FFT/measurement math, public service commands, real
stdio/Unix sockets, direct plot gesture handlers and managed mounted controls.
The manual driver refuses custom plot extensions (`custom extension is not a
qualified workbench`); therefore no full managed plot-event/GPU or native OS
presentation acceptance is claimed. Mesh preparation and gesture logic are
exercised separately. Historical native gates 3/6/9/11 remain separate/open.

Detailed final verification evidence is retained in
`docs/validation/spectrum-markers-20261028/`.

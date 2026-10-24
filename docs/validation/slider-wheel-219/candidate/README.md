# /219 clean package and deliberate actuation checks

The clean 0.181.18 /219 app, matching agent and conformance executable embed
the recorded source revision with vcs.modified=false. The bundle's strict
ad-hoc seal remains valid and its full tree is unchanged after headless checks.

The exact agent changes a paused pot from 50% to 60% at 2 ms with exact charge
and time continuity. Two live voltage endpoints match independent piecewise RC
equations; four instantaneous current/power relations match the accepted voltage.
Graph Undo/Redo works, with stale/fresh states treated explicitly. A separate
21-row startup capture checks 84 independent voltage/current references. The
combined 90 numerical checks use declared tolerances, not a new device model.

A later 70% live edit leaves capture data/metadata exact. Dirty Open refuses
without mutation; explicit Open restores the saved 60% graph and capture. The
original fixture and saved project bytes remain intact. Owned workers join and
the headless process exits 0. These public-command tests do not exercise native
wheel input or qualify the accumulated native acceptance list.

The candidate remains held; the previous /218 package and qualified /205 download
are preserved. No graphical launch or unattended confirmation is queued.

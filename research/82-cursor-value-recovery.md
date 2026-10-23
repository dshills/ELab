# Cursor time refusal and correction

October 23, 2026. Second item in the overnight plan. The packaged /205
component-value repair and qualified /204 first-version baseline are preserved.

## Reproduction

Mounted live and recorded cursor fields lose rejected input on Return. Entering
`oops` resets the live field to empty and the recorded field to its old `0 s`
value, while leaving a generic warning. The accepted measurements correctly
remain unchanged. The failing regressions are retained in
`build/cursor-guidance-206-tools/before.txt`.

## Repair under validation

Rejected complete time text now remains in its originating native editor for
correction. It is private presentation state, like unfinished native typing;
no cursor position is committed by a refusal. First corrected Return uses the
existing `scope.cursor.set` or `recording.cursor.set` command and clears the
rejection. Select/Clear cannot silently bypass an already rejected field after
the native editor considers that text delivered. Those actions revalidate the
retained text through the same guarded command rather than restoring an old
error. A formerly future live time can therefore become valid after the history
grows, without requiring the user to retype it before selecting a cursor.

A rejection belongs to a document and live run or selected capture, and to the
accepted position of that specific cursor. A replacement accepted cursor, stale
transient, replacement document/run or different recording clears the old text.
An ordinary repaint preserves it; restarting live simulation does not clear an
independent recorded capture's input. Stale callbacks cannot retain another
owner's text. Command/document guards on deferred actions remain in force.

Shared user/agent help names cursor A/B and the actual available time interval,
explains unit syntax, and states that cursor positions and circuit state are
unchanged. Live bounds come from retained samples rather than assuming the run
still begins at zero. Missing fresh live samples receive a restart/step action.
Malformed requests, invalid cursor names, stale parser contexts and mismatched
recording guards do not acquire invented time-domain guidance. Error codes,
technical details, numeric parsing, boundary tolerance and measurements retain
their existing contract.

## Checks so far

- Root race regressions pass for live/recorded bad syntax, incorrect dimensions,
  negative/out-of-range numeric and unit times, actual rolled-history bounds,
  unchanged full state and technical errors, malformed and stale request context.
- Mounted cursor race checks pass, including retained error text, exact shared
  response parity, blocked selection until correction, first-Return recovery,
  owner replacement and independent recorded/live state.
- Existing native cursor, deferred action, time-field and recording-position
  help checks pass with the new stale-parser callback regressions.
- A follow-up mounted regression reproduced an unchanged-text retry becoming
  stuck after live history grew from 20 ms to 40 ms. The 30 ms draft now commits
  and selects the next cursor through exactly two normal commands, preserving
  the circuit and physics. Earlier failures and final focused results remain
  in `build/cursor-guidance-206-tools/`.
- Reviewed inventory has 166 public commands, 134 source routes, 404 binding
  sites and 76 source files. Site definitions are unchanged when source line
  numbers are ignored; the added source contains editor rejection handling.

The final full quality gate passed after the additional retry repair: lint,
staticcheck, 67 Python tests, root/desktop race and vet, and required mounted
control race checks and offscreen vet. Root results were reused from cache where
unchanged; desktop race took 175.481 seconds and mounted checks 148.989 seconds.
The initial full pass is retained separately. A clean candidate build,
packaged-agent checks and visible native correction/layout checks remain
required before promoting /206. These
focused tests do not establish full PRD, IME, screen-reader or participant-study
acceptance. No new feature, model, solver mode, public command or file format is
part of this repair.

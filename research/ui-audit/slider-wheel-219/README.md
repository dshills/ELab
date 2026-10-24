# Native scroll and slider baseline /219

The already-running owned /206 session supplies these observations; no new
app is launched. [Receipt](report.json). Current source's original managed
slider factory and entire upstream slider match the old native sources.

1. [RC baseline](01-rc-before.png).
2. [Wheel over the time field](02-text-wheel.png): inspector moves.
3. [Wheel over a probe selector](03-selector-wheel.png): pane moves, choice stays.
4. [Inspector thumb back at the top](04-inspector-thumb-home.png).
5. [Scope thumb at top](05-scope-thumb-top.png): playback controls are reachable.
6. [Scope viewport returned](06-scope-thumb-restored.png).
7. [Owned pot fixture loaded](07-pot-loaded.png), after saving the prior RC.
8. [Pane scrolled past the slider](08-pot-visible.png).
9. [50% slider fully visible](09-pot-control.png), without clicking it.
10. [Wheel changes it to 100%](10-pot-wheel.png), adds user history and dirty state.
11. [One native Undo restores 50%](11-pot-undone.png); the paused result is stale.
12. [Explicit Restart restores fresh readings](12-pot-restored.png), paused at zero.

The first six observations preserve the exact full public snapshot. Pixels show
viewport movement despite an unchanged accessibility tree. The previous RC
graph and its complete 21-row capture/metadata are saved separately and reopen
exactly in a fresh headless agent; its owned workers join and process exits 0.
The pot fixture stays unchanged on disk. The reused session remains on that
disposable pot at 50%, fresh and paused; Undo/restart advance revisions/run/history
and retain completed-work metadata. This is scoped restoration, not full snapshot
equality. Original owner sessions and files are untouched.

The current-source production-control regressions fail with and without focus.
The repaired-source driver/mounted checks are documented separately. These old
native captures do not qualify changed-candidate native behavior.

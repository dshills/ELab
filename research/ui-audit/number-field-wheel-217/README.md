# Numeric wheel baseline on the existing /206 test session

October 23, 2026. [Receipt](report.json), [repair notes](../../94-number-field-wheel.md).
No new app is launched; the ordinary owner app/projects are untouched.

1. `01-ready`: recorded rate 0.1 and the top of the scrollable scope.
2. `02-unfocused-scroll`: the wheel moves the scope; the entire service snapshot
   stays exact.
3. `03-focused`: the actual playback number field has focus.
4. `04-focused-scroll`: the same normal wheel gesture changes playback to
   0.13000001 instead of scrolling. The public snapshot changes only recorded
   speed and revision counters; live graph/results/measurements stay exact.
5. `05-restored`: effective rate 0.1, original viewport positions and focus moved
   to the unchanged Sample step field.

Before the trial the public snapshot stored the unset speed sentinel 0, while
the UI displayed its 0.1 fallback. The public setter requires .001–10, so the
restoration stores explicit 0.1. Last actor and revision counters also change;
all remaining snapshot fields are exact. This is not a byte-identical restore.
The raw snapshots and helpers are retained with the source validation.

The native number-field factory and complete spinner source match elab9 before
the repair. Current-source mounted failures independently reproduce both live
initial-voltage and recorded-speed edits. These old native images cannot prove
the new SDK fix, new AppKit behavior or the other held-candidate native checks.

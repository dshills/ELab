# Native cursor guidance and file-dialog lifecycle

October 24, 2026. [Evidence and current checklist](../docs/validation/native-guidance-221/README.md).
The running code-equivalent /221 test installation completes three more native
checks without changing production source or asking for launch/desktop permission.

- The Simulate inspector visibly explains the live cursor's actual 0–20 ms range,
  accepted units and state preservation. Invalid syntax and an out-of-history
  value retain their text. The first correction to 4 ms succeeds.
- Light and Dark disclosure headings are readable with native fonts. Direct
  pointer clicks collapse the heading, and Space expands it with actual focus.
- Native Save As creates a new disposable file. Its filename editor's Select All,
  Undo and Redo menu actions work without changing the circuit snapshot. Save
  and a repeated Cancel remove the temporary Edit menu; Tab returns to the
  application's Save button. Normal Quit removes the owned process/endpoint.
  Reopening the same installation shows no temporary menu, and the native-saved
  project reopens with all 1,001 recorded rows exactly preserved.

The original saved fixture, source app bundle and /205 download remain unchanged.
The 42 live and 2,002 recorded voltage references pass. The app is restored to
the original disposable document, clean, paused and Light.

Command-A in the native Save filename remains unqualified through CUA. A direct
pointer focus and a 200 ms held chord also fail to select it. Native menu editing
and Tab navigation inside the panel work. The dispatch cause and ordinary
hardware-keyboard behavior are not established; no speculative patch is made.

Cancel changes only viewport registration and revision counters when the previous
Save status message clears: height grows from 297 to 319 pixels. The circuit,
history, live result/time, captured data and document remain exact. Do not claim
full snapshot equality for that presentation transition.

Seven of eighteen accumulated native checks now have scoped evidence; eleven
remain. /221 stays held and /205 remains the preserved download. Broader MVP,
screen-reader, study, clean-account and latency requirements remain separate.

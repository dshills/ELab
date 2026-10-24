# Native acceptance progress /220

October 24, 2026. [Receipt](report.json), [verified checks](checks.json),
[raw evidence](raw.tar.gz), [member hashes](raw-manifest.json).

The existing owned ELab test installation was updated to 0.181.19 /220 and
opened normally through CUA. Its path, name, bundle identifier, launcher and
sandbox policy stayed unchanged. The previous /206 installation was quit
normally and retained as a rollback copy. No new user permission question,
security exception or alternative GUI launcher was used. Standing authorization
is recorded in the root AGENTS.md, including the owner's October 24 instruction.

This is native acceptance of a copied runtime with the qualified candidate's
nonempty Mach-O sections, rather than a launch of the original candidate bundle
bytes. The copied executable has its own ad-hoc signature after packaging under
the existing test identity. Strict bundle verification passes, and its executable
hash remains unchanged after the checks. The original /220 bundle and qualified
/205 download are preserved.

## Observed passes

- Wheel input over the potentiometer slider, before and after focusing its thumb,
  scrolls the inspector while preserving the entire public snapshot. A subsequent
  Right key changes 50% to 51%, proving the slider retained focus. A deliberate
  drag changes it to 73.07692%. Both changes carry the user actor and Adjust wiper
  history entry; native Undo restores 51%, then the saved 50%. The paused stale
  result remains explicitly labeled after Undo.
- Unfocused initial-voltage and recorded-speed fields forward wheel input to
  their enclosing panes. Focused fields retain unfinished `0.25` V and `0.125`
  playback-rate text while their accepted values stay unchanged. Seven pairs of
  complete public snapshots are exactly equal across wheel/focus/private-edit
  checks. Return commits the typed values. The voltage field's plus button and
  Up key still work; its float32 stepping displays approximately 0.252 V.
- Native Verify entries `1 ms` and `1 us` complete 1,000 intervals and publish
  1,001 captured rows. View recorded plot opens the recording. All 2,002 voltage
  values match independent ideal RC/constant-source calculations to an absolute
  error below 5e-16 V. Live transient state, graph and time are unchanged.
- A native `1.001 ms / 1 us` request refuses without replacing the operation,
  graph, settings, history, captured data or live physics. The diagnostic reduces
  canvas height and automatic Fit changes the viewport and its revision counters;
  these are the only differing public snapshot fields. Full snapshot equality
  is therefore not claimed for this refusal.
- The shared agent Save/Load path preserves all 1,001 captured rows exactly and
  reopens them visibly in the native app. This independently checks persistence;
  it does not qualify the native Save dialog's editing shortcuts.

## Open observations

Command-A did not select the native Save filename in two CUA keyboard attempts.
Typing appended to `Untitled`; no file was written. Choosing Edit > Select All
through the native menu did select the complete filename. A subsequent Right,
Command-A sequence again left the insertion point instead of a selection. Both
dialogs were canceled, their temporary Edit groups disappeared, and the app
returned to its project. The shortcut failure is reproducible through this input
route; ordinary physical keyboard behavior and the precise dispatch cause remain
unqualified. Do not treat the menu-object unit test as proof of this native path,
or change responder routing without reproducing the cause.

The 1053×802 captures also show a footer summary wrapping below its fixed-height
row. Record this as a layout defect to reproduce in the mounted layout before
repair. No new feature was added during these checks.

The copied-runtime acceptance list closes checks 16–18 for the controls/scenarios
above. The other 15 checks, including the unresolved Save shortcut, remain open.
The application is left on a clean, paused disposable saved RC document, with
the recording visible and no dialog open. The owner's other app and projects
were not used. Complete release qualification, clean-account startup, owner
study, broader accessibility and latency claims remain separate.

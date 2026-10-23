# Native cursor correction, /206

October 23, 2026. CUA captures of the isolated owned test session at 1053×802.
[Receipt and scope](../../../docs/validation/cursor-guidance-206/README.md).

- `01-live-ready`: paused RC fixture and live cursor fields.
- `02-live-invalid`: rejected `oops` remains available for correction.
- `03-live-corrected`: first Return accepts 4 ms and clears the warning.
- `04-live-history-retry`: a retained 30 ms draft accepts after history grows
  from 20 to 40 ms; Select B succeeds. Blue B and public snapshots establish
  selection; the unchanged text does not produce a new AX text diff.
- `05-recorded-invalid`: recorded cursor B retains rejected `10 Hz`.
- `06-recorded-corrected`: B accepts 16 ms; A at 4 ms yields Δt 12 ms.
- `07-scope-scrollbar-bottom`: after wheel scrolling to the scope toolbar,
  dragging its scrollbar reaches the lower cursor controls again.
- `08-inspector-wheel-bottom`: wheel scrolling reaches the final Simulate
  controls while the inspector section buttons remain visible.
- `09-inspector-scrollbar-top`: dragging the inspector thumb returns to Sample
  step and pulse controls. The separate scope viewport stays at its prior place.
- `10-document-wheel-bottom`: Document scrolls to its final help/shortcut rows,
  with section navigation still visible.
- `11-document-scrollbar-top`: dragging back restores Print preview and the
  complete SVG/PDF/PNG export buttons.

The detailed correction action is absent on Simulate. The short warning alone
is not sufficient; /206 promotion remains held. These images do not qualify
the subsequent inspector-layout or native editing-menu changes. Full AX
snapshots reset numeric element indices; always read new indices before acting.
The three follow-up scrolling checks reuse the already-running paused session;
they involve no new launch. Scroll movement changes pixels without changing the
AX tree; a no-change AX diff alone is not evidence of a scrolling failure.
The Document checks use the same existing session. A follow-up Command-Shift-S
opens the native Save project panel with its default Untitled name; Cancel
returns to the paused RC drawing. That shortcut is registered and works in
this session, so the earlier unsuccessful attempt is not a reproduced missing
shortcut. No file was written by this cancellation check.

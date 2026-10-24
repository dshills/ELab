# Host background and divider baseline

October 23, 2026. Captured through CUA from the already-running owned
`ELab Cursor Recovery` /206 fixture. The owner's ordinary ELab session and
projects are preserved. These are unmodified native captures of the **old**
binary; none shows the changed /209 binary.

- `01-native-dark-baseline.jpg`: the inspector scrollbar works in Dark, but a
  white inset remains around the managed root. Its `.txt` companion is the AX
  state. This evidence supports the host-background repair, not a scrollbar fix.
- `02-native-light-restored.jpg`: restored Light, paused, inspector at the top.
- `03-native-divider-before.jpg`: divider at 0.65 before the click.
- `04-native-divider-click.jpg`: a physical click at `(526,495)`, without a drag,
  changes the accepted fraction to 0.66611844. The companion AX diff records it.
- `05-native-divider-restored.jpg`: accessibility value restored to 0.65 after
  the reproduction. Simulation remains paused at 0.02 s.

CUA captures can show its pointer glow and transient tooltips. Those effects
are not attributed to ELab without separate evidence. The scope was already
scrolled; these captures do not establish a new scope clipping regression.

Source fixes, regression checks and their scope are described in
[the repair notes](../../86-window-background-and-divider-grip.md).

`06-final-dark-root-probe.png` and `07-final-light-root-probe.png` are separate
320×180 **offscreen paint probes** after the fix. They use ELab's actual root
style with its schematic children omitted. They verify all four scene corners
and unchanged root bounds; they are not full application screenshots or native
macOS acceptance evidence.

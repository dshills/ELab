# Manual transistor LED experiment

Choose **Examples → Manual transistor switch**. S1 is selected when the example opens; use **Close switch** / **Open switch** in the Inspector. Closing it supplies base current through RB, turning Q1 on and lighting D1. Opening it leaves RPD holding the base near ground. Select Q1 to inspect its signed base/collector readings, or RB to change the drive resistance. Undo restores a switch change.

The circuit has separate ideal 5 V supply and base-drive sources, a 10 kΩ base resistor, 100 kΩ base pull-down, 1 kΩ LED load, generic red LED, grounded-emitter NPN and SPST switch. The switch is initially open. The template is editable and follows the same unsaved-work protection, reset, history, SVG export and project save/reopen behavior as the other examples.

Users and agents use the same commands:

```json
{"command":"example.load","params":{"id":"switch-npn"}}
{"command":"switch.set","params":{"id":"S1","closed":true}}
{"command":"component.select","params":{"id":"Q1"}}
{"command":"measurement.inspect","params":{"target":{"kind":"object","id":"Q1"}}}
```

These are command fragments; public transport requests also need the normal version, request ID, mutation precondition and retry key. `example.switch-npn` is a named load alias. `examples.list` provides the question, expected behavior, experiment and limitations. Reset restores the open-switch template with S1 selected, keeps the current pause preference and detaches any current save path without overwriting the saved file.

This is a DC operating-point experiment. Actuation recalculates the circuit; it does not model transistor charge storage or contact bounce. The finite switch model has 1 Ω on resistance and 1 GΩ off resistance, so open is not mathematically zero leakage. Brightness is illustrative; there is no physical part identity, thermal or rating qualification.

The [headless workflow](validation/headless-manual-transistor.json), [packaged workflow](validation/desktop-manual-transistor.json) and [release receipt](validation/manual-transistor-validation.json) record the engineering evidence. The switched base network is independently reduced to a Thevenin equivalent and compared with the separate specialized transistor solver under the release voltage tolerance max(1 mV, 1%) and 1 nA current bound. Conservation, closed/open behavior, undo, native/agent equality, reset guards, strict save/reopen and kept recordings are tested. Very small cutoff leakage makes the LED-side voltage poorly conditioned; no extra voltage precision is claimed there.

Run `python3 scripts/manual-transistor-demo.py --output REPORT --artifacts NEW_DIRECTORY` after building. `--connect ENDPOINT` is for a designated owned development session and replaces its circuit. Participant comprehension and full keyboard/screen-reader acceptance remain separate gates.

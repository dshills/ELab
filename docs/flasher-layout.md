# Flasher example spacing

The bundled capacitor-timed LED flasher spreads its two 47 kΩ bias resistors away from the neighboring LED terminals and moves its supply left to leave their routed leads clear. Native drawing, SVG, PNG, PDF and printing use the same graph geometry. This is a template adjustment; general automatic label placement remains future work.

Newly loaded flasher examples use the revised spacing. Existing saved schematics retain their positions, connectivity and recordings. **Reset current example** offers the revised template through the existing unsaved-work protection and document guards; undo restores the previous drawing. As with other project replacements, reset clears session recordings; undo covers schematic history. The original saved project and its recording remain intact. Users and agents use `example.load` with `id: "flasher"` or the guarded `example.current.reset` command.

Regression checks cover bias-label clearance from neighboring terminals, white export label masks versus all routed wires, and the existing native route-versus-symbol-body check. A frozen graph/settings fixture came from a real 0.95.0 saved project. Comparing both layouts at startup and every millisecond through 600 ms requires identical voltages, currents, charge and trace values. Only the three component positions change.

`scripts/flasher-layout-demo.py` additionally reopens the previous-release project and its kept recording, saves them unchanged, tests protected reset/undo/redo, and compares a fresh 200 ms capture with every row in the previous-release recording. It exports SVG/PNG/PDF without changing circuit, physics, history or document state, then reopens the revised saved project. Its native mode uses the same public commands as its headless mode.

At the 0.96.0 checkpoint, native inspection at 49% Fit confirmed the separated bias labels but found C1/D1 caption crowding and capacitor export-mask crossings. The [0.97.0 capacitor-caption repair](capacitor-labels.md) resolves those specific issues; small fitted text remains difficult to read. This work does not qualify general automatic routing or typography, every possible circuit layout, physical printing, participant usability or complete accessibility.

To repeat the compatibility workflow in a fresh headless session after building:

```sh
python3 scripts/flasher-layout-demo.py \
  --legacy docs/validation/flasher-layout-legacy-095.elab \
  --output build/flasher-layout-report.json --artifacts build/flasher-layout-assets
```

Use new output/artifact paths. Add `--connect` only for a fresh owned native endpoint. Both the default and native modes close their own client; native mode also closes the supplied owned window unless `--keep-open` succeeds.

The [validation receipt](validation/flasher-layout-validation.json), [native/headless workflow](validation/flasher-layout-native.json), [visual observations](validation/flasher-layout-visual-review.json) and [four-adapter electrical evidence](validation/flasher-layout-conformance.json) qualify this 0.96.0 /111 checkpoint.

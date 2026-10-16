# Capacitor caption clearance

Capacitor IDs and values use one upright position in the native canvas and schematic output. Vertical capacitors put their caption beside the plates; horizontal capacitors put it above and to the left, clear of the plates and terminal routing stubs. Fit includes the caption bounds, including long IDs. PNG, PDF and print preview inherit the SVG layout.

The previous exported caption backgrounds touched the plate geometry at all four rotations. In the bundled flasher they also covered parts of the cross-coupling wires. Its native C1 caption sat beside the LED’s D1 caption. The shared position separates those captions and leaves the flasher’s bias and capacitor label backgrounds clear of every routed wire. These are presentation changes: no saved graph positions, connectivity, circuit values, numerical models, command or file versions change.

Regression geometry covers short and 32-character IDs, both native plates, terminal routing stubs, actual export background masks and long-label Fit. Native mesh checks verify actual glyphs at all four shared positions and separate the flasher capacitor captions from both LED captions.

The public reference `scripts/capacitor-labels-demo.py` builds and saves a four-rotation incomplete drawing through the same commands available to users and agents, reopens/resaves the retained previous-release flasher fixture and recording, then exports all eleven guided examples. SVG/PNG/PDF capture must preserve circuit, physics, history and document state. Use fresh output paths and an owned blank session:

```sh
python3 scripts/capacitor-labels-demo.py \
  --legacy docs/validation/flasher-layout-legacy-095.elab \
  --output build/capacitor-label-report.json --artifacts build/capacitor-label-assets
```

Optional `--connect` targets a fresh owned native endpoint; the reference closes that owned window unless `--keep-open` succeeds. In that case the caller must close the window after visual inspection.

This does not add general automatic neighboring-object avoidance or qualify all circuit layouts. Typography, small fitted text, complete keyboard/accessibility, participant usability and physical printing remain separate work.

Direct native review covers the four-rotation grid and complete flasher; PNG review also covers RC, rectifier and LED-fade pages. Native/headless SVG and PNG bytes match for all thirteen export cases. A separate rectifier diode-caption/wire crossing remains open. The [validation receipt](validation/capacitor-labels-validation.json) and [visual observations](validation/capacitor-labels-visual-review.json) record the 0.97.0 /112 checkpoint and its limits.

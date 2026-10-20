# Component values and discoverable limits

Select a numeric component to see its range, accepted units and any waveform constraint in the Inspector. Value text remains a draft until **Apply value**. The UI and agents commit through `component.set`; placement uses the same value parser and validation. Invalid input preserves the graph, physical state and undo history.

`parts.list` now reports `value_type` (`number`, `label`, `none`). Numeric entries have an owned `value_domain` with parameter, SI unit, inclusive bounds, readable range, syntax and any coupling rule. `capabilities.list` filtered by `component.place`, `component.set` or `capacitor.initial.set` adds structured domains, state requirements, known errors and examples. It also describes placement IDs, snapping and capacity.

| Part | Ordinary value edit |
| --- | --- |
| DC / ideal pulse source | 0–12 V |
| Sine source | Amplitude 0–12 V, with amplitude + absolute current offset ≤12 V |
| Finite pulse source | High level −12–12 V, at least the current low level |
| Resistor / potentiometer | 10–100000 Ω |
| Capacitor | 1 pF–1 F |
| Net label | Case-sensitive 1–32 ASCII letters/digits or `_+-./`; no spaces |
| Other curated symbols | Fixed model; no generic editable numeric value |

A newly placed sine source has factory offset zero; a finite pulse starts with low level zero. Consequently a negative finite-pulse high level cannot be placed directly over its factory low. Place it with a valid factory-compatible value, then use the existing full waveform command to set both signed levels. Ordinary `component.set` changes only amplitude/high level and preserves the other waveform fields. Signed DC sweep overrides have their own analysis domain; their bounds do not change the editable source's domain.

## Unit entry

Bare numeric text means SI in the selected component's units. Optional suffixes are V, F, Ω or `ohm`; prefixes are case-sensitive:

- Voltage: `m`, `u`, `µ`, `μ`, for example `500 mV`, `250 µV`, or `0.5 V`.
- Resistance: `k`, `K`, `M`, for example `2.2k`, `10K ohm`, or `0.1M Ω`.
- Capacitance: `p`, `n`, `u`, `µ`, `μ`, `m`, for example `100nF`, `1 µF`, or `0.1mF`.

Numeric text supports decimal/scientific notation. Previously accepted hexadecimal float text remains accepted by the Go numeric parser. Prefixes may also be used without the final unit where the field's unit is explicit. Wrong units, nonfinite values, oversized text and nonzero values that round to zero refuse. Numeric value text is limited to 128 UTF-8 bytes; labels retain their smaller bound. Explicit zero remains valid where the part permits it, including zero with a negative exponent. A representable tiny voltage is distinct from a nonzero value which underflows to zero; solver/model tolerances are separate from input representation.

Existing forms retain their original parse-then-scale floating-point rounding. This avoids silently changing already-supported component values while adding voltage prefixes. Saved graphs store numeric SI, so millivolt entry does not change the project format or introduce a new dependency.

`capacitor.initial.set` uses the voltage grammar with signed bounds −12–12 V, measured as pin 1 minus pin 2. Exact text `default` removes that component's override and uses the experiment startup setting. It does not resume prior physical charge.

## Placement and agent use

`component.place` accepts a known `kind`, finite `position.x/y` within ±10000, optional unused ID, and optional value. Coordinates snap to the nearest 10-unit grid; a snapped position cannot already be occupied. Omitted/empty ID allocates a designator. Explicit IDs are 1–64 ASCII letters/digits/underscore/hyphen beginning with a letter, and cannot be reused within that session even after deletion. Omitted/empty numeric value uses the factory value; a net label uses its new ID as the default name; fixed-model symbols require omitted/empty value.

The 100 drawing-object limit includes helpers. Simulation capacity and topology/model restrictions remain separately inspectable in the existing snapshot/library data. Placement or a valid value does not imply the unfinished circuit is complete or supported.

These three command details now expose normalized value domains, source relations and observation behavior. Other command-family semantics, full native interaction/accessibility and broader release gates remain separate. There is no expanded device/solver range, custom-model support or physical rating claim.

## Reapplying an unchanged value

The selected component's `snapshot.drawing.value` is also the native Inspector text. It now preserves the exact stored floating-point value. The formatter tries compact engineering-unit spellings, verifies each by parsing it, and falls back to a shortest round-trip SI decimal. Shorter exact spellings win; ties prefer a normalized engineering magnitude. Examples include `1 µF` and `1 kΩ`. A high-precision value keeps enough digits rather than silently rounding on the next Apply. Net-label text retains its existing case-sensitive semantics.

## Numerical method discovery

Detailed `component.place` and `component.set` discovery includes
`x-network-numerics.time_integration`. It describes the current mixed RC/NPN,
general capacitor and waveform engines as two-stage SDIRK2, with their current
engine IDs. The single-RC exact exponential, algebraic NPN pulse phases and
inductor Radau IIA paths remain separate. Capacitor-free networks retain their
algebraic equations within the corresponding engine family.

Historical backward-Euler recordings remain readable with their original engine
and rows; importing one does not switch the current live solver or relabel that
capture. See [capacitor integration](capacitor-integration.md). The source
correction replaces an obsolete backward-Euler description in discovery only;
no method selector, device law, tolerance, budget, file format or ordinary
catalog change is introduced. The [0.163.0 /180 replacement](validation/integration-discovery-review-180/README.md)
qualifies this correction through exact copied native/headless discovery and six
live families. Preserved /179 retains the old text.

Applying that untouched text preserves the graph, run/electrical identity, simulated time, dirty state and undo/redo entries. It is still an admitted command, so command/event revisions and actor information may advance. Saving/reopening retains numeric SI and reconstructs an exact editable draft. Plot/schematic captions and measurement readouts may still use display precision; they are not used as editable source values.

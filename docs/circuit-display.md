# Voltage color and illustrative current cues

**Circuit display** in the general Inspector controls voltage colors, their symmetric range, current arrows, moving dots, reduced motion, and conventional/electron visualization. Defaults keep colors/cues/motion off, use a ±5 V range and conventional current. Settings are saved and undoable presentation edits; they preserve physical time, capacitor state, run/electrical revision, models and numerical readings. `document.preferences_dirty` covers both display settings and pinned readings.

Voltage color uses an accepted wire endpoint voltage relative to ground: blue is negative, neutral is zero, orange is positive. Magnitudes saturate at the configured 0.01–30 V range. Unknown/stale voltages use a distinct gray; they never become a valid zero. Inspection and pinned readings retain exact signed values and named references, so color is optional. A selected wire also shows its signed conventional current or an explicit reason why it is unavailable.

## Honest wire direction

Wire currents are derived from accepted device terminal currents and Kirchhoff's current law. Conventional current is positive from the wire's declared `from` terminal to its `to` terminal. The derivation includes two-terminal devices, source delivery, NPN base/collector input and emitter output, and all three potentiometer terminal currents. Ground/junction/label helpers inject no device current. Equal net labels supply virtual edges for connectivity, not fabricated physical routes.

A linear-time bridge traversal finds uniquely determined cut currents. An ideal-wire cycle does not have a uniquely determined current split; its cycle edges receive no guessed current, arrow or particle. Unrelated bridge currents remain available. Missing injections, stale frames or wire-group current imbalance above 10 nA also refuse current cues. This calculation uses the existing solver/model accuracy and does not solve a new circuit, choose an arbitrary split, or weaken numerical limits. Zero current remains an accepted numeric zero when determined, while currents at/below 1 pA receive no illustrative cue.

Conventional mode follows positive charge flow. Electron mode reverses visual arrow/dot direction only. Signed amperes and all numerical/CSV measurement conventions remain conventional. Directions therefore remain inspectable numerically even when cues are hidden.

## Motion and export

Static arrows are available when current cues are enabled. Moving dots additionally require requested motion, a fresh nonzero unique current, a running simulation and reduced motion off. Pause, solve failure, missing/stale data or reduced motion suppress moving dots. Reduced motion retains static arrows. A failed live integration preserves the last accepted readings as stale and stops cues; it does not publish fake progress.

Dots follow an illustrative one-second wall-time cycle with two dots per eligible wire; their speed and density are not calibrated physical transport or proportional to amperes. LED brightness is also illustrative. Native motion uses an approximately 8 Hz drawing-only repaint through the existing mounted canvas; it does not rebuild the value-editing form or admit editor commands. Physics/command revisions stay unchanged for DC motion. This evidence does not qualify every slider/focus/navigation scenario or automatic OS reduced-motion preference integration; the app preference is explicit.

SVG preserves the current palette, static direction arrows, pin cards, preference settings and accepted measurement provenance. It freezes the current sample, omitting moving particles. Stored `.elab` documents optionally include strict version-4 `display_preferences`; they contain flags/range/direction, never trusted measurements, inferred currents or animation phase. Default preferences omit the extension. Older readers lacking it refuse the field instead of dropping it. New/example/legacy-file opening restores defaults; current files and recovery restore their settings, while simulation starts fresh.

## Shared user/agent interface

```json
{
  "voltage_color": true,
  "voltage_range_v": 5,
  "current_cues": true,
  "current_motion": true,
  "reduced_motion": false,
  "direction": "conventional"
}
```

`display.preferences.set` takes the complete object above with normal revision/retry guards. All six fields are required. Motion requires current cues; the native Motion button enables both in one command, and disabling cues disables requested motion. Reduced motion suppresses effective motion while preserving the requested setting. Unknown fields/directions, missing/null fields and out-of-range values refuse before any state change.

`snapshot.display` exposes preferences, accepted run/electrical revision/time, effective `animate`, and `wires`. Each wire has ID, `from`/`to`, optional signed `voltage_v` and `current_a`, live/stale/unavailable state, optional voltage/current reason and the shared color hex. Current absence is distinct from zero. `display.inspect {"phase":0.35}` is read-only and returns this same display plus deterministic arrow/dot geometry (`wire_id`, kind, world position and unit direction). Phase is 0–1, with 1 equivalent to 0; it is illustrative animation, not simulated time. Native drawing, this agent query and static SVG use the same portable frame builder.

## Evidence

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke
python3 scripts/display-demo.py --output docs/validation/headless-display.json \
  --artifacts docs/validation/display-artifacts
# Owned development session only: replaces its circuit.
python3 scripts/display-demo.py --connect build/display.endpoint.json \
  --output docs/validation/desktop-display.json \
  --artifacts docs/validation/desktop-display-artifacts
```

Independent LED/RC references validate +0.183939721 mA charging at 10 ms and −0.285983167 mA after 1 ms discharge. Tests cover ideal-wire cycle ambiguity, conservative net-label aliases, signed/saturating palettes, read-only phases, electron reversal without numeric sign changes, paused/reduced/disabled/failure behavior, strict inputs, dirty/undo/save/fresh reopen and preference-only recovery. Native tests cover controls, the exact float32 minimum-range boundary and changing/static meshes. Actual built/packaged GPU smoke confirms voltage/current colors, automatic mounted animation without forced rebuild or physics/editor revision changes, then stable reduced-motion pixels. Complete accessibility, visible-latency, hardware/participant and distribution qualification remain separate gates.

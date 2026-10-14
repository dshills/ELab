# PRD audit repairs

The [requirement audit](../research/11-prd-implementation-audit.md) covers every P0 functional ID, preserves the original scope/targets and distinguishes implementation from unperformed performance/accessibility/participant gates. `scripts/prd-audit.py` checks exact ID coverage; a green coverage report does not establish product acceptance.

## Search and model assumptions

The tray now recognizes common terms such as battery, power supply, variable resistor, rheostat, square wave, BJT, AC source and rectifier. Searches are case-insensitive, ignore repeated whitespace and require every query word to match the part's name/aliases/kind/pins/unit/default/model ID. Search changes session presentation only.

`parts.list` and `snapshot.matching_parts` expose each entry's `aliases` and `limitations`. Every selected component displays its model identity and assumptions before its controls/readings, including ideal resistors/capacitors, voltage sources and ground/reference helpers. Model definitions and saved dependency hashes remain unchanged.

## Scope recovery and finite measurement

Explicitly selected and visible default probes keep their source association when the drawing becomes incomplete. Surviving traces remain marked stale. Deleted sources appear in `scope.unavailable` with ID, unit and explanation; the scope legend and Inspector identify them as unavailable. They never become apparent zero-valued current traces.

`scope.probe.remove {"id":"v:C1"}` removes one selected request, including a deleted-source request, without discarding other missing probes or changing graph/physics. Existing `scope.probes.set` supports explicit reassignment. Undo restores a source and its still-requested probe association. Unavailable requests are session state; saving stores current valid selected probes and preserves kept recorded results independently.

Extremely close cursor times can have a finite delta whose reciprocal overflows. Such a delta and its valid cursors remain inspectable, while `interval_hz` is absent. Snapshots remain JSON-serializable. The exact RC update uses `expm1` to retain representable small voltage changes instead of losing them to subtraction roundoff; its physical model/engine identity is unchanged.

## Duplication and validation

Negative-level finite pulse sources now duplicate through the shared command and native Duplicate control. The factory seed is validated first, then the complete waveform, current high level and orientation are copied independently. Wires are not copied. One undo removes the duplicate.

Root/native race tests, import formatting, Staticcheck and vet cover these changes. Native smoke exercises alias search, selected assumptions and deleted-default-source removal. `scripts/hardening-demo.py` verifies the public stdio/private-socket workflow, including finite snapshots, scope recovery and negative pulse duplication.

Run it after building with `--output docs/validation/headless-hardening.json`; `--connect` replaces only the designated owned development session. The local release remains private/unsigned. Full state-contract conformance, physical capacity, responsiveness, screen-reader and clean-installation qualification remain open as recorded in the audit.

Use `--artifacts PATH` to keep the final clean RC project instead of a temporary fixture.

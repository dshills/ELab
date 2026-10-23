# Mouse and scrolling maintenance /204

ELab 0.181.3, October 23, 2026. The [receipt](report.json) identifies the clean
source, exact native executable, matching agent, SDK and extracted download.
[Native checks](native-ui.json) distinguish the final package from the
[first-pass and preview evidence](../../../research/ui-audit/ui-input-203/README.md).
The [research notes](../../../research/78-ui-input-checks.md) describe all findings.

Fast scope/split drags now accept raw release events when the native drag timer
never starts. Classified start/move/stop paths remain supported. The sweep bridge
accepts its own cursor repaint while its retained command/document/sweep guard
rejects intervening actor changes. Secondary clicks cannot edit the plots or
schematic; completed placement releases cannot emit false stale-input errors.
The existing shared commands, models and formats are unchanged.

The preliminary /203 native fast-drag failure was retained, diagnosed and repaired
before promotion. Regression tests exercise raw release, coalesced slide events,
actual split manual-window input and the real native owner delivery after redraw.
The complete source/race/vet/mounted suite, exact pinned SDK split regressions,
packaged-agent protocol check, extraction/mode/Unicode-path verification and strict
ad-hoc signature verification passed. This remains the owner-selected unsigned
channel without Developer ID or notarization. No download was publicly uploaded.

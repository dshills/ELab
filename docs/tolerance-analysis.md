# Seeded component tolerance analysis

Open **RC filter tolerance spread**, then Component tolerances in Simulate.
Choose AC gain, V1 excitation, C1 voltage, 1 kHz, 16 trials, seed 1,
±5% resistance and ±10% capacitance. Run again with the same seed to reproduce
all draws and results. Raise a tolerance to compare the spread. The live circuit,
simulation time, undo history and dirty state remain unchanged.

Users and agents share `tolerance.settings.set`, `tolerance.run`,
`tolerance.cancel` and `tolerance.export`; structured state is in
`snapshot.tolerance`. A nonempty settings/run object is a complete replacement;
empty parameters use current settings. For example:

```json
{"mode":"ac-gain","source_id":"V1","probe_id":"v:C1","frequency_hz":1000,"trials":16,"seed":1,"resistance_fraction":0.05,"capacitance_fraction":0.1,"beta_fraction":0.2}
```

Mode `ac-gain` reports dB voltage gain at the declared frequency about an
independently solved configured DC bias. Mode `dc` reports the selected existing
voltage/current probe; it opens capacitors, so capacitor tolerance has no DC
effect; DC may omit frequency. History-bearing circuits are refused rather than silently freezing
latches or controls. The existing AC whitelist and model limits still apply.

Independent uniform fractional draws vary authored resistors, authored
capacitors and forward beta of explicitly configured transistor profiles.
Generic-v1 transistor parameters remain fixed. The algorithm is SplitMix64,
using the top 53 bits for a uniform draw in captured component order. Trials
are 2–64, seed is an exact integer from 0 through 9007199254740991, and each
fraction is 0–0.5. Component limits remain in force: a draw outside its allowed
range becomes a failed trial, not a clamped value.

Each trial exposes its actual varied parameters, result or failure. Statistics
include successes only, show both counts, and use population standard deviation.
No successful trials means no meaningful range/mean; the UI suppresses those
numbers. These sampled ranges are not guaranteed worst cases, manufacturing
statistics or safety ratings.

Runs are asynchronous with a cooperative 30-second deadline and one worker at
a time. Cancel uses the exact current ID and the worker retains its slot until
retirement. Close/document replacement cancels owned work. Later electrical
edits mark completed results as captured earlier circuits without changing them.
Settings/results are session-only; export JSON for the full captured model-bearing
circuit, settings, source SHA-256, trial parameters, outcomes and statistics, or
CSV for all trials. Text cells with spreadsheet formula prefixes are apostrophe-prefixed in CSV; JSON retains the original text. Export requires the exact current completed result ID and
uses the ordinary atomic file/overwrite/generation guards.

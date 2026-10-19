# Copied /161 RC edge qualification

The exact clean copied native socket and headless stdio paths pass the independent 23-row RC equations, source/passive voltage/current probes, cursor tie behavior, extrema and public CSV. Both retain correct negative current at the manual 10 ms falling edge. Complete transient/scope states match except independent run identities. The native host saves its owned test circuit, closes normally with exit 0, and is reaped. [Report](report.json), [native result](native/report.json), [headless result](headless/report.json), [producer](edge_acceptance.py), [equation check](edge_reproduce.py).

The runtime wrappers enforce the denial profile established by the copied-runtime qualification. The controller may read source. This is public-command integration, not physical pointer/typing, reader, latency or owner acceptance. The source regression and historical native observations retain their own scopes.

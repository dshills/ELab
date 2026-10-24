# Represented Verify output grid /218

October 23, 2026. [Receipt](report.json), [source inputs](source-inputs.json),
[raw evidence](raw.tar.gz), [member hashes](raw-manifest.json),
[repair notes](../../../research/96-verification-output-grid.md).

The unchanged /217 public agent falsely refuses both numeric and unit-based
1 ms / 1 µs requests, while its lower neighbor succeeds. The source regression
fails at the same exact 1,000-step endpoint. Division and worker multiplication
disagree by rounding. Shared admission/import now count actual represented
output endpoints, keeping 1,000 steps, distinct final times and completeness
checks. No simulation kernel, manual step, SDK or desktop source changes.

The new successful capture regressions match analytic RC voltage and survive
JSON import and embedded project validation. The upper neighbor still refuses
with an exact unchanged snapshot. Existing tiny-final-grid and mixed-solver
minimum-step failure regressions remain unchanged and pass. Focused race checks
take 2.371 s. The full gate passes lint/staticcheck, 67 Python tests, root/desktop
race and vet, mounted race and offscreen vet: desktop 174.489 s, mounted
159.428 s. All 888 frozen source/check inputs match. Native UI acceptance is
separate; /205 remains the download. Clean packaging follows this source receipt.

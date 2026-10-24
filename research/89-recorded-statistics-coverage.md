# Recorded interval coverage /212

October 23, 2026. Correctness repair to existing captured measurements; no new
commands, components, simulation models or file formats.

## Reproduced numerical failure

The exact /211 packaged agent exported a 20 ms RC capture of the constant 5 V
source. Changing only its final saved timestamp from 0.02 s to
0.0199999999995 s remains within the importer's existing grid-roundoff allowance.
The imported rows are still all 5 V; metadata still declares 0.02 s.

Requesting statistics from 0.019999999999 s to 0.02 s returned a mean of 2.5 V
and RMS of 3.5355339059327378 V, despite minimum and maximum both being 5 V.
The integrator covered half of that tiny interval, then divided by the full
requested duration. That silently assigned zero signal to the unsupported tail.
The query itself was pure and the disposable agent shut down normally.

## Repair

Both the read-only statistics command and the command that caches UI statistics
now require the entire requested interval to lie inside actual saved timestamps.
An unsupported interval returns `unavailable` with the available bounds. Its
previous cached result and the complete circuit/document/history state remain
unchanged. Existing invalid-parameter and missing-probe precedence is preserved.

The desktop's initial measurement ends at the final saved timestamp. Agents can
read that selected-capture endpoint directly, or discover the final row of any
retained capture with `operation.inspect` and `operation.samples`. Existing
command annotations describe that bound. Timestamps and imported files are
preserved; no extrapolation, zero filling, snapping or tolerance-based extension
is introduced.

## Verification and limits

The frozen regression fixture is an actual packaged export with only its final
timestamp changed. Backend regressions cover partial and entirely unsupported
tails, a one-floating-point-step overrun, exact cached-state preservation,
ordinary covered intervals and a covered interval only one floating-point step
wide. Covered constant-source mean and RMS remain 5 V. A desktop callback test
selects the real measurement control and verifies its default and preservation
of simulation/document state. A public sample-read test discovers the endpoint
without depending on a private representation.

Focused backend race checks pass in 1.755 s; focused desktop race checks pass in
1.913 s. The [full source gate passes](../docs/validation/statistics-coverage-212/README.md):
lint/staticcheck, 67 Python tests, root/desktop race and vet, mounted race and
offscreen vet. Desktop race takes 177.986 s; mounted race 159.929 s. All
879 captured source/check inputs remain exact.

Native acceptance of the changed desktop default remains separate from source,
callback, offscreen and headless-agent evidence. The standing authorization to
run ELab and use the desktop is saved in `AGENTS.md`; no additional permission
or owner review is needed. No new graphical launch prompt is queued. The /205
download and previous candidates are preserved.

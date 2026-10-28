# Spectrum peak markers and cursors

Implemented `spectrum.markers.set` and `spectrum.cursor.set` for users and agents.
The live/recorded analyser has ranked P1–P5 markers, selectable A/B frequency
anchors, actual-bin peak-amplitude/dB crosshairs and signed measurement deltas.
The [user/API contract](../../spectrum-analysis.md#peak-markers-and-frequencyamplitude-cursors)
and [research note](../../../research/129-spectrum-markers-and-cursors.md)
describe precision, floor behavior and source retirement.

## Verification

Complete ordinary portable, desktop and rendered suites pass on final production
source. Focused core, real stdio/Unix-socket and rendered-control race suites
pass. Final focused rendered assertions additionally cover every new control
through warm-up, accepted stepping and actual Resume updates, across all themes.
New measurement controls also fit at 900×600. Lint/imports, both vets, native
command inventory, PRD audit and desktop executable build pass. See
verification.json and byte-exact compressed logs (hashes in logs.json).

Tests cover a known Fourier series, DC/plateau/tie/floor/relative-threshold peaks,
nearest-bin rounding, invalid inputs, optional exact-frame guards, live/held
source freshness, recorded separation, detached nested snapshot ownership,
source/document retirement, user/agent parity and exactly-once transport retry.
Direct plot gesture tests cover quick release, own-write continuation, retired
agent/document/tab/modal input, clamping, right-button/nonfinite refusal and
cleared context after failed preparation. Mesh generation covers all themes and
small-current dB scaling. No numerical solver or integration limit changed.

The first full root run correctly rejected a stale native inventory. Its routes
were reviewed, the ledger regenerated, and the final complete run passes. Mounted
initial failures and repairs are documented in [internal-review.md](internal-review.md).
The manual driver refuses custom plot extensions (`custom extension is not a
qualified workbench`); that diagnostic is retained in mounted-complete.txt.gz
and excluded from product-failure/acceptance claims. Mounted controls use a
controls-only plot placeholder. No native OS input, hardware-rate, GPU display,
or historical native check 3/6/9/11 acceptance is inferred from these tests.

The linker reports the existing duplicate-objc warning. The development build
also reports a read-only Go stat-cache write warning but exits 0. Neither is a
source-test failure; no cache permissions or platform security were changed.

[Engine identity](engine-identity.json) proves the FFT/analysis bodies remain
byte-identical to a771f207. [Preservation](preservation.json) verifies 41 previous
package/SDK/executable identities, including /282. Source-inputs.json pins changed
Go/test files and module/SDK lock inputs. Earlier candidates and owner sessions
remain untouched. This commit adds source functionality; it does not promote a
new qualified release package. No external review provider, push or upload.

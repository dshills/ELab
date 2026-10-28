# Scope / Spectrum and live FFT

User request: add Scope/Spectrum tabs using the existing analyser, bounded live
updates, stationary controls, the same user/agent commands, tests, internal review
and a commit. No external review. Implementation and regression tests are complete.

- Shared scope.view.set and spectrum.live.set; 224 discovered commands.
- Live selects existing scope voltage/current probes, uses Hann by default and
  supports Hold without stopping simulation. Recorded analysis remains separate.
- One coalescing 250 ms worker, at most 512 accepted source rows / 256 FFT samples;
  existing recorded FFT, history, physical and transport limits unchanged.
- Fixed control/readout allocations across four themes; mounted click routing,
  background updates and warm-up transitions verified.
- Full portable, desktop and rendered tests plus focused race, lint/vet, discovery
  and inventory/PRD checks pass. Detailed failures/corrections are retained.

[Evidence and internal review](../docs/validation/scope-spectrum-20261028/README.md).
[User/agent guide](../docs/spectrum-analysis.md). Matching package checks follow
from clean committed source; prior /281 and earlier packages remain preserved.

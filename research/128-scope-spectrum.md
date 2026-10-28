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
[User/agent guide](../docs/spectrum-analysis.md). Committed as e591347f. Clean matching 0.200.4 /282 passes all 87 examples,
standard copied runtime/four-adapter checks, exact-package three-interface live
checks and strict archive/extraction qualification. All owned clients/hosts close
normally and all 38 earlier identities are preserved.

App: build/ELab-0.200.4-282-Spectrum/ELab.app.
ZIP: build/releases/ELab-0.200.4-macos-arm64.zip.
No source changes follow the implementation commit; final closeout is documentation.
No external review, push or upload.

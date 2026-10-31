# Four-MOSFET H-bridge alpha qualification

ELab 0.205.0, build 311, ships the first alpha with the discrete four-MOSFET H-bridge on Apple Silicon. The app, agent and conformance controller identify the same clean source commit on main, `0e282e6dabf5cd8f9a4d3ade54a1c1ff03a57d8e`. [Release assets](https://github.com/dshills/ELab/releases/tag/v0.205.0-alpha.1) · [Qualification record](qualification.json).

## The downloaded example

The ZIP, guide and checksum file downloaded from the GitHub draft matched the upload byte for byte. A fresh extraction preserved all 473 files, their modes and the app’s strict ad-hoc signature. The packaged agent connected to the extracted native app over MCP. The app opened `Examples/discrete-hbridge.elab` paused and clean.

| 100 Ω load | At restart | After a 21 ms step |
| --- | ---: | ---: |
| Voltage | −4.725332 V | +4.725332 V |
| Current | −47.253319 mA | +47.253319 mA |

The native run exercised two PMOS and two NMOS devices. A 45 ms recording captured the direction input, differential load voltage and load current in 50 rows. Both polarities and the return matched an independent transistor load-line calculation. Saving, clearing and reopening the project restored the graph and every recorded row exactly. The app closed normally.

## Release checks

All 88 packaged projects reopened and matched their built-in graphs. The copied native app passed smoke, and the public conformance suites passed on the in-process, headless stdio, headless socket and native socket adapters. Both packaged MCP transports passed project save/reopen, SVG and PDF export. The archive, extracted app and matching agent were checked against the package manifest and ZIP checksum.

The full local portable, desktop and offscreen race suites and all three vets exercised the H-bridge source. The CI and packaging corrections touched no Go, module or SDK inputs. SDK preparation, 104 Python tests, PRD and command-inventory verification, and lint ran again on the final release source. The machine-readable record identifies the source of each gate.

The native download rehearsal used an existing macOS 27.2 account. The command-line download had no browser quarantine attribute. Gatekeeper rejected the ad-hoc app during explicit assessment. A separate clean-account first launch, first-time hobbyist study and measured slow/fast physical divider gestures remain open.

## Hosted checks

The maintained CI checkout now includes the historical objects used to verify archived receipts. The Linux MCP walkthrough verifies SVG output and the documented refusal for its unavailable PDF renderer. Its macOS counterpart verifies both SVG and PDF. The original CI failures in the Python gates were resolved on both runners.

The Linux race suite reported 26 failing tests. Their names match all 26 failures from the pre-H-bridge hosted baseline. They include simulation wall-time budgets and numerical reference validation. On a separate macOS x86-64 run, the published 0.204.0 source and the new release source both rejected the same noise-sweep and capacity fixtures. The new H-bridge reversal, saved recording and four-FET budget tests passed there.

GitHub’s macOS 15 job also stopped in the portable race suite, with 17 failing tests. Sixteen had failed in the baseline. The additional failure came from the MCP stale-document test, which checks both the conflict response and equality of the complete before/after snapshot. The hosted receipt does not distinguish which assertion condition failed. The exact protocol-version and transport subtest passed 20 consecutive runs under the race detector on the target Mac. The runner did not reach the desktop or rendered suites.

[Release-source hosted workflow](https://github.com/dshills/ELab/actions/runs/41212276815) · [Pre-H-bridge hosted workflow](https://github.com/dshills/ELab/actions/runs/40859825622).

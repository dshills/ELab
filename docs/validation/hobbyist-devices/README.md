# Hobbyist device qualification

This addition implements four follow-ups to the H-bridge: powered PWM with a
switched buck example; optional op-amp bandwidth/slew limiting; an eight-bit
serial shift/parallel latch register; and passive piezo/active-buzzer models with
explicit audio previews. Users and agents share the same commands. The catalog
has 53 parts and 182 commands. See the [model and command guide](../../hobbyist-devices.md).

The six templates are `buck`, `opamp-slew`, `opamp-bandwidth`, `shift-register`,
`555-tone`, and `light-alarm`. Old op-amps keep their original equations unless
optional dynamics are enabled. The buck is open-loop; audio is illustrative,
not a calibrated sound-pressure or mechanical-resonance model. No autoplay.

## Electrical and command checks

Checks cover PWM clock/stamp derivatives and passivity; buck startup, ripple,
duty and load response with KCL/power balance; op-amp small-signal AC/transient
bandwidth, slew bounds, compensation energy, output-stage derivatives and
supply-droop clamping; shift/latch/OE/clear semantics, serial cascade and
simultaneous pre-edge sampling; explicit audio rendering/WAV encoding,
play/replay/stop/stale guards and failure ownership. They also cover automatic
placement, duplicates, atomic edits/undo, model provenance, strict capture and
project reopening, source sampling, public diagnostics and native/agent parity.

The default buck measured about 4.69 V with 67 mV ripple in the steady test
interval. This is a measured generic switched circuit, not a hardware regulator
rating or guaranteed conversion ratio. Internal register data uses binary
memories with explicit loading; public outputs retain finite 50 ohm drive and
1 Gohm off paths. Hidden implementation parts count toward existing bounds.

## Repository checks

| Suite | Top-level tests | Including subtests | Result |
| --- | ---: | ---: | --- |
| Complete root race | 861 | 1940 | Passed |
| Complete desktop race | 326 | 755 | Passed |
| Complete rendered desktop race | 444 | 1316 | Passed |
| Focused new electronics race | 15 | 27 | Passed |
| Focused new rendered controls | 4 | 22 | Passed |

- `preflight-lint.txt`: passed (11.49 s).
- `root-vet.txt`: passed (6.95 s).
- `desktop-vet.txt`: passed (1.77 s).
- `mounted-vet.txt`: passed (2.88 s).
- `lint.txt`: passed (10.56 s).
- `python.txt`: passed (1.74 s).
- `prd.txt`: passed (0.03 s).

Raw JSON test logs are gzip-compressed beside this file; `suites.json` records
packages, elapsed times and skips. `quality-complete.json` retains exact commands.
Test/build processes use GOMAXPROCS=2; the application launcher does not override
its runtime environment. Simulation accuracy, circuit capacity, the 512-attempt
and 100 ms per-output guards are unchanged. The test runner uses a 30-minute
suite timeout. Source-route inventory is separate from runtime coverage.

The 64 KiB ordinary command-discovery bound is preserved using the existing
schema registry. The browser inventory assertion now expects 53 parts. Rendered
label tests check text containment in full-width label frames; short labels are
not required to occupy 100 pixels. These checks do not weaken simulation limits.

## Package and native check

ELab **0.192.0 /253** is built from clean source
`28c13cc365c9687fb03c5c096412febf1ae3e2b3` at `/Users/dshills/Development/projects/ELab/build/ELab-0.192.0-253-Hobbyist/ELab.app`, with its matching agent alongside it.
Both build-info receipts report that revision and `vcs.modified=false`.
Strict ad-hoc signature verification passed. This development candidate does
not replace the qualified unsigned alpha.

The packaged stdio replay captures, saves/fresh-reopens and exports SVG/PNG/PDF
for all six examples. Both sound examples render and export matching WAV bytes
without autoplay. `packaged-replay.json` and `artifact-manifest.json` retain
results and digests. Selected exported figures are kept under `exports/`.

The separately profiled native fixture uses the same packaged executable and
commands. Saved UI/state evidence and `native-notes.txt` describe the exercised
controls and remaining observation limits. It was closed normally after testing;
its owned endpoint is gone. Owner applications/projects were preserved.

The alpha archive remains unchanged, SHA-256
`d13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b`. No public upload/push, Developer ID signing,
notarization or new physical printer check is claimed.

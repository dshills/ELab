# Timing at circuit capacity

`elab-conformance --capacity-performance-suite` measures three declared boundary fixtures through the ordinary user/agent API. Construction, value edits, live results, probe selection, Verify and lifecycle use existing commands. The flag belongs to the qualification CLI; it adds no private product command or solver mode.

| Fixture | Size including ground | Timed edit |
|---|---|---|
| Fifty potentiometers and pulse source | 51 simulated parts /100 nodes | P1 total track resistance, 10/20 kΩ |
| Pulse source and resistor chain | 64 parts /64 nodes | R1, 1/2 kΩ |
| Four-stage RC/NMOS LED | 64 parts /60 nodes | LED series load RL, 1/2 kΩ |

Each has four live/captured probes and 100 ms high/low pulse phases. Verify spans 200 ms at 1 ms outputs, retaining 205 rows including source-edge observations. The RC ladder itself stays at 14/14/14/13 kΩ and 100 nF per stage; its insulated gate makes the voltage reference independent of the edited drain load. The scalar load-line reference uses the actual series resistance.

The suite records twenty fresh and twenty warm measurements per fixture and adapter. Fresh sessions are constructed before the timer starts. Warm sessions perform one unmeasured edit/Verify before the alternating edits. Process launch is excluded and OS caches are not purged. Successful records remain subject to the ordinary bounded unkept-record store. No repeated edit may be a no-op: captured graphs and increasing warm run/electrical identities must demonstrate the declared alternation.

Command roundtrip includes transport/admission/JSON work; it is not input-to-visible acknowledgment. Edit-to-fresh includes the command and any wait for a correctly associated result. Verify completion includes admission and status polling, ending before sample paging and independent report checks. Full rows, model parameters, graphs, source settings, metadata and final readings remain in every timing sample. Targets retain p95 edit-to-fresh ≤500 ms and maximum Verify ≤2 s.

The report validator reconstructs all six exact graph variants, verifies four selected probes, engine/model/settings/capture associations and size, recomputes all 820 row references per sample, checks final node/device results and sample statistics, and recomputes timing distributions and target flags. Missing coverage, altered identity/model/data, nonfinite time or mismatched summaries refuse. The retained ordinary 120-sample fixture exercises thirty corruption categories. These are specified consistency/numerical checks, not authentication of arbitrary solver history.

The full-size functional regression runs the same commands against a newly built ordinary agent from an instrumented controller. This keeps the real solver wall guard enabled without treating race instrumentation as workstation latency. Core controlled-clock numerical tests and the report checker remain in the race suite; no full-size fixture or capture grid is skipped or reduced.

Run from a clean build into a new artifact directory:

```sh
GOPROXY=off ./scripts/build.sh
build/elab-conformance --capacity-suite --capacity-performance-suite \
  --native-binary /absolute/path/to/private/ELab.app/Contents/MacOS/ELab \
  --artifacts /absolute/path/to/new-results
```

Omit the native binary for the three portable adapters. Invalid repetition counts refuse before creating artifacts. All sessions are newly created and owned; existing projects/apps/endpoints are untouched. See [capacity limits](circuit-capacity.md), [capacitor method](capacitor-integration.md) and [research protocol](../research/18-capacity-performance.md).

This covers three named fixtures at the individual published boundaries. Simultaneous maxima, every allowed topology/device combination, other hardware, visible-frame latency, complete native input/accessibility, clean installation/distribution, physical printing and participant acceptance remain separate gates. The canonical app is not promoted by a passing engineering timing run.

[Recorded qualification](validation/capacity-performance-validation.json) retains all 480 measurements and 98,400 rows. The highest case p95 edit-to-fresh is 23.375084 ms; the slowest Verify is 470.2145 ms on the recorded workstation. [Raw evidence](validation/capacity-performance/README.md) distinguishes these named-fixture results from the remaining gates.

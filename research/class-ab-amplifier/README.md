> Historical /274 artifact: this op-amp-assisted circuit predates the transistor
> capacity expansion. The new built-in [all-discrete amplifier](../../docs/discrete-class-ab.md)
> uses eight BJTs within a combined limit of sixteen. Original data below is preserved.

# Class-AB amplifier example

Open **[class-ab-amplifier.elab](class-ab-amplifier.elab)** in ELab using File →
Open. The project starts paused and clean with six probes restored. Resume to
watch it run, or Step. Two two-cycle captures are kept in the project: the baseline and the finer-step
check. Select a recording to inspect its complete waveform immediately; choose
Live when you want to watch Resume/Step update the traces.

[Schematic SVG](class-ab-amplifier.svg) · [schematic PNG](class-ab-amplifier.png) ·
[measured response](response.png) · [CSV measurements](class-ab-response.csv).

## Circuit

This is a small **op-amp-driven complementary class-AB amplifier**, made with
15 existing simulated parts. It uses ±5 V rails, a 0.5 V-peak 10 Hz input,
one generic op-amp, one NPN/PNP output pair, two silicon bias diodes, 22 Ω emitter
resistors and a 100 Ω resistive test load. RF and RG are both 10 kΩ, setting the
closed-loop voltage gain to approximately `1 + RF/RG = 2`. Feedback is taken
from the load, after the output transistors and emitter resistors.

The bias network keeps both output devices conducting a small current near zero
input; Q1 sources the positive half-cycle and Q2 sinks the negative half-cycle.
Diode bias and emitter resistors follow the class-AB principles described by
[Analog Devices](https://wiki.analog.com/university/courses/engineering_discovery/lab_14).
The component values and numerical results here are this project's own design
and ELab measurements, not results from that reference.

## Measured results

| Check | Result |
|---|---:|
| Input amplitude | 0.5 V peak |
| Output positive/negative peaks | +0.999972 V / −0.999972 V |
| Zero-input collector bias, each output transistor | 0.390196 mA magnitude |
| Independent scalar bias reference | 0.390196 mA |
| Conducting transistor at its output peak | 9.95022 mA magnitude |
| Opposite transistor at that peak | 4.394 nA magnitude |
| Maximum output error relative to `2 × VIN` | 28.343 µV |
| Coarse/fine samples | 401 /801 across two cycles |
| Maximum common-time output difference after halving sample step | <2×10⁻¹⁵ V |

These checks establish bias, complementary handoff, loaded low-frequency gain
and numerical consistency **in these generic models**. PNP terminal currents
are negative under the simulator's into-device convention; the response chart
labels the plotted PNP-current magnitude explicitly.

[Checks](checks.json), [independent reproduction](reproduction-checks.json),
[fresh reopen](reopen-checks.json), [native check](native-checks.json).

The native check uses the exact copied qualified /274 runtime through public
agent commands. It restores the exact graph/probes, runs live, pauses, steps
through more than 400 samples and checks gain. The owned fixture closes normally
with exit 0 and endpoint removal. No owner app/session command is sent. This is
native service/runtime evidence, not physical mouse/keyboard or hardware testing.

## Current boundaries

ELab's published capacity is 64 simulated parts and 100 electrical nodes, with
separate per-device limits. In particular, it currently allows only **two total
NPN/PNP transistors** and three sources. This example fits those limits. A larger
all-discrete amplifier with input, voltage-gain, driver and output transistor
stages requires an explicit capacity expansion and qualification first.

The [generic BJT models](../../docs/bipolar-transistors.md) omit internal charge,
bandwidth, Early effect, breakdown, heating and device ratings. The generic
op-amp also omits internal bandwidth, slew rate and physical stability limits.
Consequently these results do not establish audio bandwidth, physical distortion,
thermal stability, speaker-power ratings or a build-ready hardware design. The
slow input makes live handoff visible; a 100 Ω resistor is the test load.

No model, solver limit, component, public command, application code or built-in
example catalog is changed. There are still 53 kinds, 194 commands and 62 built-in
examples. This is a separate saved research circuit for the existing /274 app.

## Reproduce

Using the matching /274 agent, generate a new exclusive output folder:

```sh
python3 research/class-ab-amplifier/reproduce.py --output build/my-class-ab-check
```

`--agent` can select another matching agent executable. The script constructs the
entire circuit from [existing shared commands](build-commands.json), checks its
bias against an independent scalar load line, verifies both polarities, halves
the sampling interval, records a live run, saves through the documented file workflow and exports
the schematic/capture. The separate fresh-reopen receipt checks the saved file. It uses a fresh owned headless
process and closes it normally. It never connects to an owner GUI session.

`plot_response.py` creates the vector chart from the measured rows using
ReportLab's line-chart renderer. The checked-in PNG is an export of that SVG.
The schematic PNG/SVG come from ELab's own export commands. Both have been
visually inspected; the second schematic layout uses named feedback nets to
avoid an ambiguous crossing through the bias network.

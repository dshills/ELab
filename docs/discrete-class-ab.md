# All-discrete class-AB amplifier

Choose **Examples → Amplifiers & filters → All-discrete class-AB amplifier**,
or search for “class AB”. Agents open the identical circuit with
`example.load {"id":"discrete-class-ab"}`. The combined bipolar budget is now
**16 NPN/PNP devices**, including embedded optocoupler phototransistors. All
other drawing, component, node, source, storage and numerical work limits remain.

This example has **8 BJTs, 27 simulated parts and 20 electrical nodes**. It uses
±9 V supplies, a 100 mV peak / 10 Hz sine and a 100 Ω load. No op-amp component
or hidden amplifier model is used.

| Stage | Parts | Role |
|---|---|---|
| Differential input | Q1/Q2, RDI1/RDI2 | Compare feedback with input; 100 Ω emitter degeneration |
| Active load | Q3/Q4 | PNP current mirror combines the differential collector currents |
| Tail sink | Q8, RT, RREF, D3/D4 | Approximately 0.9853 mA bias from a two-diode reference and 680 Ω emitter resistance |
| Voltage gain | Q5, RV, RPULL | PNP common-emitter stage drives the output bases |
| Bias and output | D1/D2, Q6/Q7, RE1/RE2 | Diode-biased complementary followers with 22 Ω emitter resistors |
| Feedback | RF/RG | 1 kΩ / 100 Ω gives nominal non-inverting gain `1 + RF/RG = 11` |
| External compensation | C1 | 1 nF from Q5 base to collector; explicit initial voltage 7.55 V |

The long-tailed pair and mirror are standard analog building blocks, described
in [Analog Devices' differential amplifier chapter](https://wiki.analog.com/university/courses/electronics/text/chapter-12).
The topology and values here are an ELab educational fixture. Generic devices
use ELab's existing transport-form Ebers–Moll equations; all voltages and
currents come from solving the complete connected network.

Pause, Restart and Step to inspect exact samples. The default six traces show
input, load voltage, Q6/Q7 collector currents, Q8 tail current and C1 voltage.
Q6 sources positive output current; Q7's negative signed collector current
indicates sinking. Both conduct near zero input, demonstrating class-AB bias.
Default output samples are 500 µs apart, with playback at 0.02 simulated seconds
per wall second.

## Inspect the default circuit

1. Open the example and leave it running to see the input/output traces and
   alternating output-device currents. The nominal input period is 100 ms;
   slowed playback makes one period take about five wall-clock seconds.
2. Pause, then Restart to return to the explicitly biased startup condition.
   Restart reruns the current graph; **Reset current example** restores any
   edited component values as well.
3. Use Step or the existing Verify panel to inspect 200 ms at a 500 µs sample
   interval. A captured Verify run leaves the live circuit and time unchanged.
4. Place scope cursors near the positive and negative output peaks. Q6 has
   positive collector current on the positive swing; Q7 has negative collector
   current on the negative swing. Near a zero crossing, both conduct.
5. Change RF from 1 kΩ to 2 kΩ and Restart to observe increased gain. Undo
   restores the original value. Save As keeps an editable copy of the circuit;
   retained Verify captures can be saved in that same project.

| Default probe ID | Measurement |
|---|---|
| `v:VIN.positive` | Input relative to ground |
| `v:RL` | Load pin 1 minus pin 2, the output relative to ground |
| `i:Q6.collector` | Upper NPN collector current, positive into the device |
| `i:Q7.collector` | Lower PNP collector current, negative while sinking the output |
| `i:Q8.collector` | Tail-sink collector current, approximately 0.9853 mA |
| `v:C1` | Compensation capacitor pin 1 minus pin 2 |

The primary input is a sine source. Edit VIN's amplitude, offset, frequency and
phase in its source inspector or with `source.sine.set`. Auto follows that
waveform. The existing primary-input High/Low modes hold offset plus/minus
amplitude (here +100 mV / −100 mV); they do not change its frequency.

## Measured behavior and model scope

Independent cold DC solves with C1 removed give approximately **−21.93 mV**
zero-input output offset and **10.9063 V/V** gain between ±100 mV inputs.
The two-period transient capture gives approximately **+1.06872 / −1.11254 V**
output peaks. The explicit capacitor initial condition makes the first frame
differ from the settled DC point; it does not erase the offset. Tests compare
scalar tail-current/load-line references, mirror base-current loading, KCL and
power, both output half-cycles, half-step refinement and recording round trips.

Try changing RF to 2 kΩ, comparing both output currents, or increasing C1 to
10 nF. Keep source sampling adequate for any frequency change. Smaller loads,
larger signals or arbitrary added stages can encounter existing convergence,
voltage and work guards; exceeding a budget produces an explicit diagnostic.

These generic devices omit heating, breakdown, Early effect, safe operating
area, internal junction capacitance and charge storage. The two output devices
are not specified physical power transistors. The default is a low-frequency,
100 Ω educational fixture: it does not qualify a speaker power amplifier,
physical bandwidth/stability or zero-energy power-on behavior.

## Frequency response

The existing AC sweep panel now supports this circuit. Its default is VIN,
1 Hz–100 kHz, 201 logarithmic points and `v:RL`. Run the sweep to compare gain,
phase and external compensation while the live scope continues independently.
Agents use the same `sweep.start` and `sweep.settings.set` commands. Capacitors
open for the solved DC bias; their startup voltages and the current live frame
are ignored. See [AC analysis and semiconductor model limits](ac-frequency-sweeps.md).

## Shared commands and files

Read `capabilities.list` and `project.inspect` before automating. Command/params
snippets in this guide omit the transport envelope: mutations still require
version/request ID, a retry key, and the observed command revision and document
generation described in the [agent protocol](agent-api.md). `simulation.wait`
observes asynchronous initialization before dependent simulation commands.

Existing placement, `component.set`, `source.sine.set`, `capacitor.initial.set`,
`wire.connect`, batch/proposal, history, scope/cursor, simulation, project and
export commands remain shared by users and agents. There are **53 component
kinds, 194 commands and 63 examples**. Saved projects remain format version 4;
no device model or parameter changed. Older versions with a two-BJT capacity
cannot simulate this larger circuit.

`scripts/discrete-class-ab-demo.py` verifies the example, exports two captures
and a schematic, reconstructs the graph from blank with existing editing
commands, and reopens the recorded project. It requires an exclusive artifact
directory and refuses to drive an existing owner document. Use a fresh owned
agent, or an explicitly disposable blank native endpoint:

```sh
python3 scripts/discrete-class-ab-demo.py \
  --agent build/elab-agent --output /tmp/my-discrete-amplifier
```

The [qualification record](validation/discrete-class-ab/README.md) links exact
package identities, electrical references, UI/agent parity, captured data and
all 63 reopened examples. Earlier research's
[op-amp-assisted amplifier](../research/class-ab-amplifier/README.md) remains a
historical /274 fixture; it is separate from this fully discrete template.

Cold algebraic networks with more than two BJTs have a narrowly scoped startup
fallback after ordinary Newton starts fail: eight private source-ramp stages,
then a strict solve with the full authored sources. Intermediate stages are
never published or saved. Reactive circuits retain their authored initial
conditions. The 100-iteration Newton bound, damping, voltage/residual guards
and integration budgets remain unchanged. Convergence is not universal.

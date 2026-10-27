package workbench

import (
	"fmt"
	"slices"
	"sync"
)

type ExampleInfo struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Question   string `json:"question"`
	Expected   string `json:"expected"`
	Try        string `json:"try"`
	Limitation string `json:"limitation"`
}
type ExampleContext struct {
	Info     ExampleInfo  `json:"info"`
	Modified bool         `json:"modified"`
	Changes  []EditChange `json:"changes"`
}
type ExampleLoadParams struct {
	ID                 string  `json:"id"`
	DiscardChanges     bool    `json:"discard_changes,omitempty"`
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
}
type ExampleResetParams struct {
	ExpectedID         string  `json:"expected_example_id,omitempty"`
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
	DiscardChanges     bool    `json:"discard_changes,omitempty"`
}

// legacyExampleID keeps named loaders and their restart semantics in sync.
func legacyExampleID(command string) string {
	switch command {
	case "example.reset":
		return "led"
	case "example.rc":
		return "rc"
	case "example.npn":
		return "npn"
	case "example.npn-pulse":
		return "blink"
	case "example.flasher":
		return "flasher"
	case "example.switch-npn":
		return "switch-npn"
	case "example.rectifier":
		return "rectifier"
	default:
		return ""
	}
}

var immutableExampleCatalog = sync.OnceValue(buildExampleCatalog)

func ExampleCatalog() []ExampleInfo { return slices.Clone(immutableExampleCatalog()) }

func buildExampleCatalog() []ExampleInfo {
	catalog := []ExampleInfo{
		{ID: "buck", Title: "PWM buck converter", Question: "PWM buck converter", Expected: "Duty changes the switched LC output; close SLOAD for a load step. This is an open-loop converter, not a regulated feedback supply.", Try: "Select PWM and change Duty, then compare C1 voltage and inductor current. Close SLOAD for a load step.", Limitation: ExtendedDeviceModel().PWM},
		{ID: "opamp-slew", Title: "Op-amp slew-limited follower", Question: "Op-amp slew-limited follower", Expected: "The output takes finite time to follow each input edge. Disable dynamics to compare the original model.", Try: "Select U1 and change slew rate. Compare input and output edges, then disable dynamics to compare the original model.", Limitation: ExtendedDeviceModel().Dynamics},
		{ID: "opamp-bandwidth", Title: "Op-amp bandwidth limit", Question: "Op-amp bandwidth limit", Expected: "A 1 kHz, 10 mV input is attenuated near the configured 1 kHz gain-bandwidth. Change input frequency or use an AC sweep.", Try: "Change VIN frequency or U1 gain-bandwidth and compare the traces. Run an AC sweep to see the frequency response.", Limitation: ExtendedDeviceModel().Dynamics},
		{ID: "shift-register", Title: "Serial shift and parallel latch", Question: "Serial shift and parallel latch", Expected: "SDATA sets data, SLATCH captures the shifted byte, SCLEAR clears only the shift chain, and SOE disables the parallel outputs. D7 shows bit 7.", Try: "Toggle SDATA, step a clock edge, then close SLATCH. Compare the displayed bits; try SCLEAR and SOE.", Limitation: ExtendedDeviceModel().Shift},
		{ID: "555-tone", Title: "555 piezo tone", Question: "555 piezo tone", Expected: "The real 555 drives a capacitive piezo through RS. Select BZ1, render a preview, then explicitly Play or export WAV.", Try: "Select BZ1, render a short preview, then Play. Change RA, RB or C1 and render again to compare the pitch.", Limitation: ExtendedDeviceModel().Sound},
		{ID: "light-alarm", Title: "Light-sensitive buzzer alarm", Question: "Light-sensitive buzzer alarm", Expected: "Lower the LDR light level to activate the comparator and buzzer. Audio preview is explicit; simulation never starts playback.", Try: "Lower the light level on LDR1, then render and Play a BZ1 preview. Raise the light level and compare.", Limitation: ExtendedDeviceModel().Sound},

		{ID: "hbridge-motor", Title: "Reversible PWM motor driver", Question: "How do direction, PWM, braking and coasting change motor current and speed?", Expected: "S1 closed / S2 open drives forward; reverse those switches for reverse. Both equal brake; open SEN to coast. Flyback clamps carry winding current after disable.", Try: "Select MTR1 for RPM and current. Change PWM high/low times, then compare brake and coast from the same speed. Mode changes preserve shaft speed and winding current at the switching instant.", Limitation: HBridgeDeviceModel().Limitations},
		{ID: "led", Title: "LED current limiter", Question: "How does resistance change LED current?", Expected: "Increasing the series resistance lowers current and the illustrative LED brightness.", Try: "Change R1 from 1 kΩ to 2 kΩ and compare signed current.", Limitation: "Generic red LED; brightness is illustrative and there is no thermal behavior or physical part rating."},
		{ID: "split-supply", Title: "Split-rail DC supply", Question: "How do two positive batteries create a negative rail?", Expected: "The midpoint reference makes the rails +3 V and −3 V. Each source remains +3 V across its own terminals; R1 sees 6 V and carries 6 mA.", Try: "Inspect V2.negative relative to ground, then V2 across its terminals. Change either source to 5 V or move ground to the bottom rail and compare.", Limitation: "Ideal DC sources have no output resistance, current limit, battery discharge or physical ratings. The network still needs one common reference; ideal voltage-source/capacitor constraint loops refuse."},
		{ID: "rc", Title: "RC charge and discharge", Question: "How quickly does a capacitor charge and discharge?", Expected: "Capacitor voltage changes smoothly; switching from charge to discharge reverses current while preserving voltage at the switching instant.", Try: "Pause, step 10 ms, then choose Discharge. Increase R1 or C1 to slow the response.", Limitation: "Ideal capacitor/resistor with zero ESR and leakage; this example uses an ideal source edge."},
		{ID: "rc-high-pass", Title: "RC high-pass edge detector", Question: "Why does a capacitor pass changes but block a steady voltage?", Expected: "The voltage across R1 jumps positive at a rising input edge and negative at a falling edge, then decays toward zero. C1 voltage stays continuous at each edge.", Try: "Pause and step 50 ms to the falling edge. Compare input, R1 output and C1 voltage; increase R1 or C1 to lengthen the decay (10 ms in the template).", Limitation: "Ideal resistor/capacitor and instantaneous 0–5 V square source. ESR, leakage, component ratings and additional output loading are omitted."},
		{ID: "rl", Title: "Coil current buildup and decay", Question: "Why does coil current keep flowing when the voltage changes?", Expected: "Current approaches 50 mA while input is high. At 50 ms, coil voltage reverses while current stays continuous and decays through R1; the time constant is 10 ms.", Try: "Plot coil current and voltage across the falling edge. Double L1 to 2 H, or set a signed startup current such as -20 mA and Restart.", Limitation: "Ideal 1 H coil and explicit 100 Ω resistance; no winding resistance beyond R1, saturation, parasitic capacitance, heating or ratings. Source low supplies an ideal return path."},
		{ID: "rlc", Title: "RLC ringing and damping", Question: "How do electric and magnetic stored energy produce ringing?", Expected: "Capacitor voltage overshoots 5 V and rings while current reverses; R1 dissipates energy and the response settles.", Try: "Compare 50 Ω underdamping with 200 Ω critical damping and 500 Ω overdamping. Inspect signed L1 current and C1 voltage; startup edits reset the experiment.", Limitation: "Ideal L/C with explicit series resistance. No winding saturation, ESR, parasitic capacitance or ratings. Internal integration and output sampling have separate work/resolution limits."},
		{ID: "flyback", Title: "Coil switching and flyback diode", Question: "Where does coil current go when the MOSFET turns off?", Expected: "After the falling gate edge, D1 provides a return path for the coil current. Drain voltage rises above the 5 V supply while stored magnetic energy drains through RL and D1.", Try: "Pause and step across 50 ms. Compare L1 current, drain voltage and diode current; increase L1 to lengthen decay. A missing diode path can exceed the model's node-voltage envelope.", Limitation: "Generic MOSFET/diode and ideal coil with explicit 100 Ω load resistance. No real coil saturation, switch avalanche, diode charge storage, heating or ratings."},
		{ID: "pnp", Title: "PNP high-side LED switch", Question: "How does a PNP transistor switch the supply side?", Expected: "Low base drive lights D1; high drive turns it off. PNP base and collector currents are negative under the shared into-device convention.", Try: "Choose Low / High under the scope. Plot Q1 VBE and collector current; increase RB and compare.", Limitation: "Generic complementary Ebers–Moll model; no device identity, breakdown, heating, internal capacitance or switching delay. LED brightness is illustrative."},
		{ID: "emitter-follower", Title: "NPN emitter follower", Question: "Why does the emitter follow the base with a voltage drop?", Expected: "The emitter follows the 10 Hz input below the base voltage; RL carries much more current than the base resistor.", Try: "Compare VIN, Q1 base and emitter voltages and base/emitter currents. Change RB or RL to see loading.", Limitation: "Generic transistor with instantaneous response; no high-frequency bandwidth, Early effect, heating or physical ratings."},
		{ID: "bipolar-push-pull", Title: "Complementary transistor buffer", Question: "How can an NPN/PNP pair source and sink load current?", Expected: "Q1 supplies the positive half-cycle, Q2 the negative half-cycle. The unbiased stage has a visible crossover dead band near zero input.", Try: "Plot VIN, the shared emitters and both collector currents. Pause and compare positive, negative and near-zero input samples.", Limitation: "Unbiased educational class-B stage with illustrative complementary parameters; no thermal behavior, internal charge, real matched pair or physical power ratings."},
		{ID: "discrete-class-ab", Title: "All-discrete class-AB amplifier", Question: "How do a differential pair, current mirror and voltage-gain stage drive a class-AB output without an op-amp?", Expected: "The eight-transistor circuit amplifies the 10 Hz input by about eleven. Q6 sources positive load current and Q7 sinks negative current; both conduct near zero input. The tail current is about 1 mA and the feedback loop has a small DC offset.", Try: "Compare VIN and RL voltage, Q6/Q7 collector currents and the Q8 tail current. Change RF from 1k to 2k to increase gain, or C1 from 1nF to 10nF to explore the external compensation state.", Limitation: "Generic matched Ebers–Moll devices, not named power transistors. No heating, breakdown, safe operating area or internal junction charge/bandwidth. C1 starts at an explicit 7.55 V biased initial condition; this is not a zero-energy power-on or physical stability qualification."},
		{ID: "npn", Title: "Transistor LED switch", Question: "How does a small base current control the LED?", Expected: "Low base drive turns the LED off; higher drive increases collector current until the transistor approaches saturation.", Try: "Choose base 0 V, 0.8 V and 5 V, then change RB and compare base/collector currents.", Limitation: "Educational Ebers–Moll NPN; no physical transistor identity, heating, breakdown or internal charge storage."},
		{ID: "switch-npn", Title: "Manual transistor LED switch", Question: "How can a switch control an LED through a transistor?", Expected: "Closing S1 supplies base current and lights the LED. Opening S1 lets RPD hold the base near ground; the LED turns off.", Try: "Select S1 and choose Close switch / Open switch. Inspect Q1 base and collector currents, then increase RB.", Limitation: "Finite 1 Ω on / 1 GΩ off switch; educational NPN and illustrative LED brightness. No contact bounce, heating or physical ratings."},
		{ID: "pmos", Title: "High-side MOSFET LED switch", Question: "How does a P-channel MOSFET switch the supply side?", Expected: "Gate high at the 5 V source turns M1 off. Pulling the gate low turns the LED on; signed drain current is negative because it leaves the drain.", Try: "Compare M1 gate-source voltage and drain current across a pulse edge. Change RG to 10k and add an external 1uF gate-source capacitor for delayed switching.", Limitation: PMOSDeviceModel().Limitations},
		{ID: "nmos", Title: "MOSFET LED switch", Question: "How does gate voltage switch an LED without base current?", Expected: "A high gate voltage makes M1 conduct in its low-resistance triode region; a low gate turns the LED off. Gate current is zero in this model.", Try: "Pause and step across a pulse edge. Compare M1 gate-source voltage, drain-source voltage and drain current; change RG or add an external gate capacitor.", Limitation: NMOSDeviceModel().Limitations},
		{ID: "cmos", Title: "Complementary MOSFET inverter", Question: "How do N-channel and P-channel MOSFETs invert a signal?", Expected: "High input turns MN on and drives output near ground. Low input turns MP on and drives output near 5 V. PMOS drain current is negative; NMOS drain current is positive.", Try: "Pause and step 200 ms to the falling input edge. Compare input/output and both drain currents. Change VIN high to 2.5 V and Restart to see both channels conduct.", Limitation: "Generic educational MOSFETs, not a physical matched pair or rated logic gate. The explicit 10 kΩ output load defines an otherwise undriven output. Zero intrinsic gate charge, capacitance, delay, heating and breakdown; ideal pulse edges have no modeled transition loss."},
		{ID: "opamp-follower", Title: "Op-amp voltage follower", Question: "How does negative feedback make an output follow its input?", Expected: "The 10 Hz, ±1 V sine and output closely overlap. The explicit ±5 V supplies provide the delivered load current; input current is much smaller.", Try: "Pause and step 25 ms from Restart to inspect the positive peak. Reduce RL from 10 kΩ to 10 Ω and Restart to see the generic output-current limit distort the signal.", Limitation: OpAmpDeviceModel().Limitations},
		{ID: "opamp-inverting", Title: "Inverting op-amp amplifier", Question: "How do two resistors set an amplifier's sign and gain?", Expected: "With RI 10 kΩ and RF 20 kΩ, the ±0.5 V sine produces approximately ∓1 V at the output. The input difference stays small; signed output current reverses with the waveform.", Try: "Pause and step 25 ms from Restart. Change RF from 20 kΩ to 10 kΩ, then Restart: the output amplitude halves and remains inverted.", Limitation: OpAmpDeviceModel().Limitations},
		{ID: "blink", Title: "Pulse-controlled LED", Question: "What happens when transistor base drive alternates?", Expected: "The LED follows the high/low base-drive phases; the scope separates signed base and collector behavior.", Try: "Change pulse high/low durations, then pause and measure matching edges.", Limitation: "Instantaneous junction response; no physical switching delay or stored transistor charge."},
		{ID: "flasher", Title: "Capacitor-timed LED flasher", Question: "How can two RC stages make alternating light?", Expected: "The LEDs alternate as capacitor voltages drive the two transistor bases; larger timing capacitors generally lengthen the period.", Try: "Plot both LED currents and collector voltages; double both capacitors and compare measured periods.", Limitation: "Generic grounded-emitter NPN/LED models, ideal capacitors and an explicit asymmetric startup; hardware tolerances are not modeled."},
		{ID: "led-fade", Title: "Capacitor-buffered LED fade", Question: "How does stored capacitor charge change an LED pulse?", Expected: "C1 charges across D1 while the input is high; after the input falls, stored charge keeps LED current flowing briefly as voltage decays.", Try: "Compare input voltage, C1 voltage and D1 current. Change C1 from 100 µF to 220 µF and compare the falling-edge tail.", Limitation: "Ideal capacitor and generic red LED; source low discharges through R1 as well as D1. Brightness is illustrative; no physical LED rating, ESR or heating."},
		{ID: "coupled-windings", Title: "Transformer ratio and winding polarity", Question: "How do turns, coupling and dots change the secondary signal?", Expected: "The loaded secondary follows the primary with winding losses and leakage; reversing its dot reverses the signal.", Try: "Select T1 for ratio, coupling, winding L/R and dot polarity. Inspect both winding voltages/currents and stored magnetic energy.", Limitation: PowerDeviceModel().Transformer},
		{ID: "transformer-supply", Title: "AC transformer to regulated 3.3 V", Question: "How does a low-voltage AC source become a DC supply?", Expected: "Coupled windings feed a full bridge, C1 smooths rectified pulses, and U1 regulates 3.3 V after startup.", Try: "Compare T1 secondary, C1 voltage and U1 output. Inspect winding loss; change turns or load to observe headroom/dropout. RSN/CSN explicitly snub commutation; RREF defines secondary common mode when diodes turn off.", Limitation: PowerDeviceModel().Transformer},
		{ID: "opto-switch", Title: "Optically isolated LED switch", Question: "Can an LED control a separate electrical circuit?", Expected: "Closing S1 lights the internal input LED; its photocurrent enables D1 on the separate output supply. CTR and output headroom limit transferred current.", Try: "Close S1; select OC1 to compare input/output currents and effective CTR. Change CTR and Restart. The two supplies use separate wiring and common zero-volt measurement references.", Limitation: PowerDeviceModel().Optocoupler},
		{ID: "ac-rc", Title: "RC filter Bode plot", Question: "Where does a low-pass filter begin to attenuate?", Expected: "A 1 kΩ /1 µF low-pass crosses about 159 Hz at −3 dB and −45°. Gain rolls off above it.", Try: "Open Sweeps and run the prepared AC range. Select gain or phase, click a point and compare half-power estimates.", Limitation: "Normalized small-signal response around DC bias; captured sweep leaves the live circuit unchanged."},
		{ID: "ac-rlc", Title: "RLC resonance Bode plot", Question: "Where does inductance boost capacitor voltage?", Expected: "Series RLC shows a sampled resonant peak and phase rotation.", Try: "Run the prepared AC range, compare gain/phase, then change R1 and rerun to see damping.", Limitation: "Ideal L/C and explicit series resistance; sampled peak and interpolated half-power estimates."},
		{ID: "ac-opamp", Title: "Op-amp low-pass Bode plot", Question: "How does feedback capacitance reduce high-frequency gain?", Expected: "The inverting filter starts near 6 dB gain and −180° phase; its external RC feedback sets the rolloff.", Try: "Open Sweeps and run the prepared frequency range; compare CF and RF changes.", Limitation: OpAmpDeviceModel().Limitations},
		{ID: "motor-dc", Title: "DC motor startup and load", Question: "Why does motor current fall as speed rises?", Expected: "Current starts at zero through armature inductance, rises during startup, then back EMF reduces it as the rotor accelerates.", Try: "Select MTR1 to vary mechanical load live. Lock the rotor in startup settings and Restart to measure stall current.", Limitation: PowerDeviceModel().Motor},
		{ID: "motor-pwm", Title: "555 PWM motor controller", Question: "How does a pulse train control motor speed?", Expected: "The 555 drives an N-channel MOSFET; armature inductance, inertia and flyback conduction smooth the motion between pulses.", Try: "Select P1 to change timer timing/duty; select MTR1 to change load and watch RPM/current. Compare motor voltage/current, gate pulses and D1 flyback current.", Limitation: PowerDeviceModel().Motor},
		{ID: "battery-led", Title: "Battery sag and LED fade", Question: "Why does a weak battery dim an LED?", Expected: "Internal resistance sags under load; stored charge and open-circuit voltage fall as current is delivered.", Try: "Select B1 for live charge and voltage. Close S1 to add load. Change startup charge, capacity or ESR and Restart.", Limitation: PowerDeviceModel().Battery},
		{ID: "battery-regulator", Title: "Battery discharge and regulator dropout", Question: "When does a regulator lose its output as a battery weakens?", Expected: "The 5 V regulator enters dropout as battery terminal voltage loses headroom.", Try: "Compare B1 terminal voltage, U1 output and battery current; change startup charge or capacity and Restart.", Limitation: PowerDeviceModel().Battery},
		{ID: "night-light", Title: "Automatic night-light", Question: "How can light switch an LED automatically?", Expected: "LDR resistance rises as light falls. U1 switches the LED on in darkness, retaining its state across a narrow threshold band.", Try: "Select LDR1 and move Light level or enter lux. P1 adjusts the light threshold. Compare sensor resistance, divider voltage and LED current.", Limitation: ResistiveSensorModel().Limitations},
		{ID: "thermostat", Title: "Thermistor relay thermostat", Question: "How does a thermostat avoid relay chatter?", Expected: "A cold NTC divider makes U1 energize the relay through Q1; warming releases it. Hysteresis separates switching temperatures, and D2 carries coil flyback current.", Try: "Select RT1; move temperature between 20 and 30 °C. P1 adjusts the setpoint. Plot sensor voltage, comparator output, coil current and heater-indicator current.", Limitation: ResistiveSensorModel().Limitations},
		{ID: "regulator-supply", Title: "Regulated supply and load", Question: "How does load change a regulated output?", Expected: "A 9 V input charges COUT toward 5 V. Closing S1 increases load and output droop; a 10 Ω RLOAD reaches the 100 mA current limit.", Try: "Plot output and regulator current, close S1, then change RLOAD from 100 Ω to 10 Ω. Inspect regulator power loss.", Limitation: VoltageRegulatorModel().Limitations},
		{ID: "regulator-dropout", Title: "Linear versus LDO dropout", Question: "How much input headroom does a regulator need?", Expected: "Both target 5 V. As VIN falls, the standard regulator loses regulation below 7 V; the LDO below 5.3 V, with finite load droop.", Try: "Plot VIN, U1 and U2 outputs. Pause and step through a cycle; change the output setpoints or loads.", Limitation: VoltageRegulatorModel().Limitations},
		{ID: "relay-switch", Title: "Pushbutton transistor relay", Question: "How can a transistor switch an isolated load?", Expected: "S1 energizes the 5 V coil through Q1; normally-open contacts switch a separate 9 V LED load. D2 carries flyback current after release.", Try: "Click S1, compare coil current and LED current, then change K1 inductance to change pickup/decay timing.", Limitation: ControlledDeviceModel().Limitations},
		{ID: "relay-flasher", Title: "Relay coil and flyback protection", Question: "Where does the coil current go when its driver switches off?", Expected: "The coil current ramps up, pulls in the contacts, then decays through D2 after the transistor turns off. Contacts release below the dropout current.", Try: "Plot K1 current/voltage, Q1 collector and D2 current. Removing D2 can exceed the simulator voltage range; failed advances preserve committed state.", Limitation: ControlledDeviceModel().Limitations},
		{ID: "comparator-threshold", Title: "Comparator threshold detector", Question: "How does hysteresis prevent repeated switching near a threshold?", Expected: "With 2.5 V reference and 200 mV hysteresis, output rises at 2.6 V and falls at 2.4 V.", Try: "Compare VIN, VREF and output. Edit hysteresis width or the reference and watch both switching thresholds.", Limitation: ControlledDeviceModel().Limitations},
		{ID: "comparator-counter", Title: "Analog signal to digit counter", Question: "Can a slow analog signal clock a digital counter cleanly?", Expected: "The comparator turns each sine cycle into one rising clock edge; the decimal display counts these crossings.", Try: "Change VIN frequency, reference or hysteresis; click SRESET to clear the count.", Limitation: ControlledDeviceModel().Limitations},
		{ID: "button-counter", Title: "Pushbutton decimal counter", Question: "How can a button clock a counter?", Expected: "Click SCOUNT to advance one digit; click SRESET to clear it.", Try: "Press and hold: a clean button counts once. Release before pressing again.", Limitation: ControlledDeviceModel().Limitations},
		{ID: "button-debounce", Title: "Contact bounce and RC debounce", Question: "Why can one physical press create several counts?", Expected: "Raw U2 counts five edges per complete bouncing click; RC-filtered U4 counts once. D1/D2 show each least significant bit.", Try: "Click SCOUNT and compare clocks/counter bits. Wait at least 250 ms of simulation time before another click; CDEB needs time to discharge.", Limitation: ControlledDeviceModel().Limitations},
		{ID: "display-counter", Title: "555-driven digit counter", Question: "How does a timing capacitor clock a decimal digit?", Expected: "The real 555 output clocks U2 through 0–9; U3 drives seven individually current-limited LED segments.", Try: "Plot C1 and clock; change RB or C1 to change counting speed. Inspect each segment current.", Limitation: DigitalDisplayModel().Limitations},
		{ID: "d-flipflop", Title: "D flip-flop sampler", Question: "Which data changes are remembered?", Expected: "Q takes D only at a rising clock edge; /Q is complementary. Closing SRESET clears Q immediately.", Try: "Compare DATA, CLK, Q and /Q. Change data between clock edges; Q holds until the next rising edge.", Limitation: SequentialDeviceModel().Limitations},
		{ID: "jk-flipflop", Title: "JK flip-flop divider", Question: "What happens when J and K are both high?", Expected: "Q toggles on each rising clock edge, dividing clock frequency by two. Falling edges hold state.", Try: "Compare clock, Q and /Q. Close SRESET to clear Q; release reset while the clock is high and check no extra count appears.", Limitation: SequentialDeviceModel().Limitations},
		{ID: "t-flipflop", Title: "T flip-flop toggle", Question: "How does a toggle flip-flop divide a clock?", Expected: "With T high Q toggles at rising edges. Setting T low holds the bit.", Try: "Compare clock and Q. Rewire T to ground to hold, or close SRESET to clear asynchronously.", Limitation: SequentialDeviceModel().Limitations},
		{ID: "binary-counter", Title: "4-bit binary counter", Question: "How do four bits count clock edges?", Expected: "Q0 is the least significant bit; the LEDs count 0 through 15, then wrap to 0.", Try: "Open SENABLE to hold the count while clock continues; close SRESET to clear. Plot clock and all four Q bits.", Limitation: SequentialDeviceModel().Limitations},
		{ID: "decade-counter", Title: "Decade counter", Question: "How does a BCD counter stop at nine?", Expected: "The four LEDs count 0 through 9, then wrap to 0; values 10 through 15 are skipped.", Try: "Compare this sequence with the binary counter. Use SENABLE to pause counting or SRESET to clear.", Limitation: SequentialDeviceModel().Limitations},
		{ID: "logic-inverter", Title: "Logic inverter LED", Question: "Why does an inverter turn the LED on when its input is low?", Expected: "D1 lights for a low A input and turns off for a high input. The finite input edges reveal intermediate analog levels.", Try: "Compare A, U1 output and D1 current. Pause and step across an edge.", Limitation: LogicDeviceModel().Limitations},
		{ID: "logic-switches", Title: "AND gate with switches", Question: "Which switch combinations light the LED?", Expected: "D1 lights only when both SA and SB are closed. Pull-down resistors keep open inputs low.", Try: "Select SA and SB and try all four combinations; inspect A, B and output.", Limitation: LogicDeviceModel().Limitations},
		{ID: "logic-half-adder", Title: "Logic half adder", Question: "How can two gates add two binary inputs?", Expected: "U1 XOR gives the sum bit; U2 AND gives the carry bit. D1 and D2 show 00, 01, 10 and 11 input cases.", Try: "Compare A, B, sum and carry traces. Pause to inspect each combination.", Limitation: LogicDeviceModel().Limitations},
		{ID: "555-flasher", Title: "555 LED flasher", Question: "How does a timing capacitor make a timer switch repeatedly?", Expected: "C1 charges toward 5 V until about 3.33 V, then discharges toward ground until about 1.67 V. The latch, output and LED switch at those levels. The first charge from zero is longer than later cycles.", Try: "Compare C1 voltage, U1 output and D1 current. Double C1 or RB to slow the flashing; pause and step across a threshold.", Limitation: Timer555DeviceModel().Limitations},
		{ID: "555-oscillator", Title: "Adjustable 555 oscillator", Question: "Can a knob change oscillator timing while charge stays continuous?", Expected: "P1 varies the discharge/charge resistance and output frequency. C1 retains its voltage when you move the wiper; the next threshold changes the latch.", Try: "Select P1 and move its wiper while running, or pause and move it before stepping. Position 1 is the fastest; position 0 is the slowest. Compare output periods using scope cursors.", Limitation: Timer555DeviceModel().Limitations},
		{ID: "555-one-shot", Title: "555 one-shot timer", Question: "Why does a short trigger make a longer output pulse?", Expected: "A brief low trigger starts about a 0.516 s output pulse while C1 charges. At the upper threshold the output drops and the discharge pin empties C1. The trigger repeats every 2 s; holding trigger low keeps the output high.", Try: "Compare trigger, C1 and U1 output. Change R1 or C1 to change pulse length; a low Reset overrides the trigger and holds output low.", Limitation: Timer555DeviceModel().Limitations},
		{ID: "zener", Title: "Zener voltage regulator", Question: "How does reverse breakdown hold a voltage steady?", Expected: "An 8–12 V input produces about 5.2 V across RL while the reverse-biased Zener sinks the remaining current. Diode voltage and current are negative; absorbed power is positive.", Try: "Compare input, output and D1 current. Change D1 to 3.3 V, or increase R1 until regulation is lost.", Limitation: ZenerDeviceModel().Limitations},
		{ID: "schottky", Title: "Schottky low-voltage rectifier", Question: "How much of a small AC signal survives rectification?", Expected: "The Schottky branch charges C1 higher than the silicon branch charges C2 from the same 1 V peak input. Both capacitors discharge through identical loads between peaks.", Try: "Compare C1 and C2 voltage and signed diode currents. Increase the capacitors equally and compare ripple; reverse leakage is larger in the Schottky model.", Limitation: SchottkyDeviceModel().Limitations},
		{ID: "rectifier", Title: "Diode rectifier and smoothing", Question: "How does a diode and capacitor turn AC into a rippling DC voltage?", Expected: "D1 charges C1 during positive peaks. Between peaks the diode blocks and RL drains C1; a larger capacitor reduces ripple.", Try: "Compare input and C1 voltage on the scope. Change C1 from 10 µF to 47 µF, then compare settled ripple after 2 simulated seconds.", Limitation: "A 5 V peak, 10 Hz educational source; no mains, transformer or physical power-supply ratings. Ideal capacitor and generic diode omit ESR, breakdown, reverse recovery and heating."},
	}
	return catalog
}

type exampleDefinition struct {
	info     ExampleInfo
	graph    Graph
	settings TransientSettings
	view     View
	selected string
	probes   []string
}

func findExample(id string) (exampleDefinition, bool) {
	d := exampleDefinition{settings: DefaultTransientSettings(), view: View{Zoom: 1}}
	found := false
	for _, info := range ExampleCatalog() {
		if info.ID == id {
			d.info = info
			found = true
			break
		}
	}
	if !found {
		return d, false
	}
	switch id {
	case "buck":
		d.graph = BuckExampleGraph()
		d.selected = "PWM"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .00002, .005
		d.probes = []string{"v:C1", "i:L1", "v:PWM"}
		d.view = View{Zoom: .55}
	case "opamp-slew":
		d.graph = OpAmpDynamicsExampleGraph(true)
		d.selected = "U1"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .00002, .005
		d.probes = []string{"v:VIN", "v:U1"}
		d.view = View{Zoom: .55}
	case "opamp-bandwidth":
		d.graph = OpAmpDynamicsExampleGraph(false)
		d.selected = "U1"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .00001, .005
		d.probes = []string{"v:VIN", "v:U1"}
		d.view = View{Zoom: .55}
	case "shift-register":
		d.graph = ShiftRegisterExampleGraph()
		d.selected = "U1"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .0025, .2
		d.probes = []string{"v:U1.clk", "v:U1.q0", "v:U1.serial"}
		d.view = View{Zoom: .55}
	case "555-tone":
		d.graph = Piezo555ExampleGraph()
		d.selected = "BZ1"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .00001, .001
		d.probes = []string{"v:BZ1", "i:BZ1", "v:C1"}
		d.view = View{Zoom: .55}
	case "light-alarm":
		d.graph = BuzzerAlarmExampleGraph()
		d.selected = "BZ1"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .0001, .01
		d.probes = []string{"v:BZ1", "i:BZ1"}
		d.view = View{Zoom: .55}

	case "hbridge-motor":
		d.graph = HBridgeExampleGraph()
		d.selected = "U1"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .00005, .005
		d.probes = []string{"v:U1", "i:MTR1", "v:U1.enable", "i:U1.vm"}
		d.view = View{Zoom: .65}
	case "coupled-windings", "transformer-supply":
		d.graph = TransformerExampleGraph(id == "transformer-supply")
		d.selected = "T1"
		d.settings.StepSeconds = .0001
		d.settings.PlaybackSpeed = .01
		d.probes = []string{"v:VIN", "v:T1.secondary", "i:T1.primary1", "i:T1.secondary1"}
		d.view = View{Zoom: .6}
		if id == "transformer-supply" {
			d.probes = []string{"v:T1.secondary", "v:C1", "v:U1", "i:T1.primary1", "i:T1.secondary1"}
			d.view = View{Center: Point{X: 350, Y: 50}, Zoom: .5}
		}
	case "opto-switch":
		d.graph = OptocouplerExampleGraph()
		d.selected = "OC1"
		d.probes = []string{"v:OC1", "v:OC1.output", "i:OC1.anode", "i:OC1.collector"}
		d.view = View{Zoom: .7}
	case "ac-rc", "ac-rlc", "ac-opamp":
		d.graph = ACRCExampleGraph()
		d.selected = "C1"
		d.probes = []string{"v:C1"}
		d.view = View{Zoom: .7}
		if id == "ac-rlc" {
			d.graph = RLCExampleGraph()
		}
		if id == "ac-opamp" {
			d.graph = ACOpAmpExampleGraph()
			d.selected = "U1"
			d.probes = []string{"v:U1"}
		}
	case "motor-dc", "motor-pwm":
		d.graph = MotorExampleGraph(id == "motor-pwm")
		d.selected = "MTR1"
		d.settings.StepSeconds = .0001
		d.settings.PlaybackSpeed = .01
		d.probes = []string{"v:MTR1", "i:MTR1"}
		if id == "motor-pwm" {
			d.settings.StepSeconds, d.settings.PlaybackSpeed = .00005, .005
			d.probes = []string{"v:U1", "v:MTR1", "i:MTR1", "i:D1"}
		}
		d.view = View{Zoom: .6}
	case "battery-led", "battery-regulator":
		d.graph = BatteryExampleGraph(id == "battery-regulator")
		d.selected = "B1"
		d.settings.StepSeconds = .005
		d.settings.PlaybackSpeed = 1
		d.probes = []string{"v:B1", "i:B1", "i:D1"}
		if id == "battery-regulator" {
			d.probes = []string{"v:B1", "v:U1", "i:B1"}
		}
		d.view = View{Zoom: .8}
	case "night-light", "thermostat":
		d.graph = SensorExampleGraph(id == "thermostat")
		d.selected = "LDR1"
		d.settings.StepSeconds = .0005
		d.settings.PlaybackSpeed = .2
		d.probes = []string{"v:LDR1.2", "v:U1", "i:D1", "i:LDR1"}
		if id == "thermostat" {
			d.selected = "RT1"
			d.settings.StepSeconds = .0002
			d.settings.PlaybackSpeed = .02
			d.probes = []string{"v:RT1.2", "v:U1", "i:K1", "i:D1", "i:D2"}
		}
		d.view = View{Center: Point{X: 0, Y: 100}, Zoom: .6}
	case "regulator-supply", "regulator-dropout":
		d.graph = RegulatorSupplyExampleGraph()
		d.settings.StepSeconds = .0005
		d.settings.PlaybackSpeed = .05
		d.probes = []string{"v:U1", "i:U1", "v:COUT"}
		if id == "regulator-dropout" {
			d.graph = RegulatorDropoutExampleGraph()
			d.settings.StepSeconds = .005
			d.settings.PlaybackSpeed = .5
			d.probes = []string{"v:VIN.positive", "v:U1", "v:U2", "i:U1", "i:U2"}
		}
		d.selected = "U1"
		d.view = View{Center: Point{X: 100, Y: 100}, Zoom: .7}
	case "relay-switch", "relay-flasher":
		d.graph = RelayExampleGraph(id == "relay-switch")
		d.selected = "K1"
		d.settings.StepSeconds = .0002
		d.settings.PlaybackSpeed = .02
		d.probes = []string{"i:K1", "v:K1", "v:Q1.collector", "i:D2", "i:D1"}
		d.view = View{Center: Point{X: 0, Y: 100}, Zoom: .6}
	case "comparator-threshold", "comparator-counter":
		d.graph = ComparatorThresholdExampleGraph()
		if id == "comparator-counter" {
			d.graph = ComparatorCounterExampleGraph()
		}
		d.selected = "U1"
		d.settings.StepSeconds = .002
		d.settings.PlaybackSpeed = .5
		d.probes = []string{"v:U1.noninverting", "v:U1.inverting", "v:U1", "i:U1"}
		if id == "comparator-counter" {
			d.probes = append(d.probes, "v:U2.q0")
		}
		d.view = View{Center: Point{X: 0, Y: 100}, Zoom: .5}
	case "button-counter", "button-debounce":
		d.graph = ButtonCounterExampleGraph(id == "button-debounce")
		d.selected = "SCOUNT"
		d.settings.StepSeconds = .0005
		d.settings.PlaybackSpeed = .2
		d.probes = []string{"v:U2.clk", "v:U2.q0", "i:SCOUNT"}
		if id == "button-debounce" {
			d.probes = []string{"v:U2.clk", "v:U4.clk", "v:U2.q0", "v:U4.q0"}
		}
		d.view = View{Center: Point{X: 200, Y: 500}, Zoom: .35}
	case "display-counter":
		d.graph = DisplayCounterExampleGraph()
		d.selected = "DS1"
		d.settings.StepSeconds = .005
		d.settings.PlaybackSpeed = .5
		d.probes = []string{"v:C1.1", "v:U1", "v:U2.q0", "v:U2.q3", "i:DS1.a", "i:DS1.g"}
		d.view = View{Center: Point{X: 100, Y: 0}, Zoom: .4}
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
		d.graph = sequentialExample(id)
		d.selected = "U1"
		d.settings.StepSeconds = .002
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"v:U1.clk", "v:U1.q", "v:U1.nq"}
		if id == "d-flipflop" {
			d.probes = []string{"v:U1.clk", "v:U1.d", "v:U1.q", "v:U1.nq"}
		}
		if IsCounterKind(id) {
			d.probes = []string{"v:U1.clk", "v:U1.q0", "v:U1.q1", "v:U1.q2", "v:U1.q3"}
		}
		d.view = View{Center: Point{X: -30, Y: 130}, Zoom: .55}
	case "logic-inverter", "logic-switches", "logic-half-adder":
		d.graph = logicExample(id)
		d.selected = "U1"
		d.settings.StepSeconds = .002
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"v:A.positive", "v:U1", "i:D1"}
		if id == "logic-switches" {
			d.probes = []string{"v:U1.a", "v:U1.b", "v:U1", "i:D1"}
		}
		if id == "logic-half-adder" {
			d.probes = []string{"v:A.positive", "v:B.positive", "v:U1", "v:U2"}
		}
		d.view = View{Center: Point{X: -80, Y: 120}, Zoom: .65}
	case "555-flasher", "555-oscillator", "555-one-shot":
		d.graph = Timer555FlasherExampleGraph()
		if id == "555-oscillator" {
			d.graph = Timer555OscillatorExampleGraph()
		}
		if id == "555-one-shot" {
			d.graph = Timer555OneShotExampleGraph()
		}
		d.selected = "U1"
		d.settings.PlaybackSpeed = .5
		d.settings.StepSeconds = .005
		d.probes = []string{"v:C1", "v:U1", "i:D1"}
		if id == "555-one-shot" {
			d.settings.StepSeconds = .001
			d.probes = []string{"v:VIN.positive", "v:C1", "v:U1", "i:D1"}
		}
		d.view = View{Center: Point{X: -20, Y: 40}, Zoom: .6}
	case "zener":
		d.graph = ZenerExampleGraph()
		d.selected = "D1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"v:V1.positive", "v:RL", "i:D1"}
		d.view = View{Center: Point{X: 10, Y: 20}, Zoom: .8}
	case "schottky":
		d.graph = SchottkyExampleGraph()
		d.selected = "D1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"v:V1.positive", "v:C1", "v:C2", "i:D1", "i:D2"}
		d.view = View{Center: Point{X: 10, Y: 120}, Zoom: .6}
	case "opamp-follower", "opamp-inverting":
		d.graph = OpAmpFollowerExampleGraph()
		if id == "opamp-inverting" {
			d.graph = OpAmpInvertingExampleGraph()
		}
		d.selected = "U1"
		d.settings.PlaybackSpeed = .05
		d.probes = []string{"v:VIN.positive", "v:U1", "i:U1", "v:U1.input-difference"}
		d.view = View{Center: Point{X: 20, Y: -40}, Zoom: .7}
	case "led":
		d.graph = ExampleGraph()
	case "split-supply":
		d.graph = SplitSupplyExampleGraph()
		d.selected = "R1"
		d.view = View{Zoom: .9}
	case "rc":
		d.graph = RCExampleGraph()
		d.view = View{Center: Point{X: -130, Y: 20}, Zoom: .7}
	case "rc-high-pass":
		d.graph = RCHighPassExampleGraph()
		d.selected = "R1"
		d.probes = []string{"v:V1.positive", "v:R1", "v:C1"}
		d.view = View{Center: Point{X: -70}, Zoom: .7}
	case "pnp":
		d.graph = PNPExampleGraph()
		d.selected = "Q1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"v:VIN.positive", "i:D1", "v:Q1.base-emitter", "i:Q1.collector"}
	case "emitter-follower", "bipolar-push-pull":
		d.graph = EmitterFollowerExampleGraph()
		d.selected = "Q1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"v:VIN.positive", "v:Q1.base", "v:Q1.emitter", "i:Q1.base", "i:Q1.emitter"}
		if id == "bipolar-push-pull" {
			d.graph = ComplementaryBipolarExampleGraph()
			d.probes = []string{"v:VIN.positive", "v:Q1.emitter", "i:Q1.collector", "i:Q2.collector"}
		}
	case "discrete-class-ab":
		d.graph = DiscreteClassABExampleGraph()
		d.selected = "Q6"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .0005, .02
		d.probes = []string{"v:VIN.positive", "v:RL", "i:Q6.collector", "i:Q7.collector", "i:Q8.collector", "v:C1"}
	case "npn":
		d.graph = NPNExampleGraph()
	case "rl":
		d.graph = RLExampleGraph()
		d.selected = "L1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"i:L1", "v:L1", "v:V1.positive"}
	case "rlc":
		d.graph = RLCExampleGraph()
		d.selected = "L1"
		d.settings.StepSeconds, d.settings.PlaybackSpeed = .0001, .01
		d.probes = []string{"v:C1", "i:L1", "v:L1", "v:V1.positive"}
	case "flyback":
		d.graph = FlybackExampleGraph()
		d.selected = "L1"
		d.settings.PlaybackSpeed = .1
		d.probes = []string{"i:L1", "v:M1.drain", "i:D1", "v:VIN.positive"}
	case "switch-npn":
		d.graph = SwitchedNPNExampleGraph()
		d.view = View{Center: Point{X: -40, Y: 20}, Zoom: .7}
		d.selected = "S1"
	case "pmos":
		d.graph = PMOSExampleGraph()
		d.settings.HighSeconds, d.settings.LowSeconds = .2, .3
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{X: 20, Y: 20}, Zoom: .5}
		d.selected = "M1"
	case "nmos":
		d.graph = NMOSExampleGraph()
		d.settings.HighSeconds, d.settings.LowSeconds = .2, .3
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{Y: 20}, Zoom: .65}
		d.selected = "M1"
	case "cmos":
		d.graph = CMOSExampleGraph()
		d.settings.HighSeconds, d.settings.LowSeconds = .2, .3
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{Y: 30}, Zoom: .5}
		d.selected = "MN"
		d.probes = []string{"v:VIN.positive", "v:MN", "i:MP.drain", "i:MN.drain"}
	case "blink":
		d.graph = NPNPulseExampleGraph()
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{Y: -30}, Zoom: .5}
	case "flasher":
		d.graph = FlasherExampleGraph()
		d.settings.StepSeconds = .005
		d.settings.PlaybackSpeed = .5
		d.view = View{Center: Point{Y: -10}, Zoom: .55}
	case "led-fade":
		d.graph = LEDFadeExampleGraph()
		d.settings.HighSeconds = .2
		d.settings.LowSeconds = .3
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{X: 40}, Zoom: .65}
		d.selected = "C1"
	case "rectifier":
		d.graph = RectifierExampleGraph()
		d.settings.StepSeconds = .001
		d.settings.PlaybackSpeed = .1
		d.view = View{Center: Point{X: -180, Y: 30}, Zoom: .55}
		d.selected = "C1"
	}
	return d, true
}
func (s *Service) exampleContextLocked() *ExampleContext {
	d, ok := findExample(s.exampleID)
	if !ok {
		return nil
	}
	source := contentHash(s.graph, s.transientSettings)
	if s.exampleDiffID != s.exampleID || s.exampleDiffSource != source {
		s.exampleDiff = schematicChanges(d.graph, s.graph, d.settings, s.transientSettings)
		s.exampleDiffID, s.exampleDiffSource = s.exampleID, source
	}
	return &ExampleContext{Info: d.info, Modified: source != contentHash(d.graph, d.settings), Changes: cloneEditChanges(s.exampleDiff)}
}
func cloneExampleContext(in *ExampleContext) *ExampleContext {
	if in == nil {
		return nil
	}
	out := *in
	out.Changes = cloneEditChanges(in.Changes)
	return &out
}
func (s *Service) loadExampleLocked(id string, discard bool) error {
	d, ok := findExample(id)
	if !ok {
		return &actionError{"not-found", "Choose an example ID from examples.list"}
	}
	if err := s.requireCleanLocked(discard); err != nil {
		return err
	}
	s.detachDocumentLocked()
	s.exampleID = id
	s.graph = d.graph
	s.transientSettings = d.settings
	s.view = d.view
	s.paused = false
	// Model state belongs to the old document. Leave no old transient clock
	// interpretable as this template's startup while asynchronous work begins.
	s.transient = nil
	s.npn = nil
	s.result = nil
	s.mixed = nil
	s.mixedState = nil
	for _, c := range s.graph.Components {
		s.usedIDs[c.ID] = true
	}
	for _, w := range s.graph.Wires {
		s.usedIDs[w.ID] = true
	}
	s.drawing = DefaultDrawing()
	s.selected = d.selected
	s.viewport.Following = true
	s.fitViewportLocked()
	// A shipped template is the clean starting point for this editable copy.
	// It has no file association, but loading it is not an unsaved user edit.
	s.savedGraph = contentHash(s.graph, s.transientSettings)
	s.savedPreferences = preferencesHash(s.pins, s.displayPreferences, s.exampleID)
	return nil
}
func (s *Service) resetCurrentExampleLocked(p ExampleResetParams) error {
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	if p.ExpectedID != "" && p.ExpectedID != s.exampleID {
		return &actionError{"conflict", "Current example changed while reset was pending; choose Reset again"}
	}
	if s.exampleID == "" {
		return &actionError{"unavailable", "Load an example before resetting it"}
	}
	if _, ok := findExample(s.exampleID); !ok {
		return fmt.Errorf("current example is not recognized")
	}
	paused := s.paused
	if err := s.loadExampleLocked(s.exampleID, p.DiscardChanges); err != nil {
		return err
	}
	s.paused = paused
	s.restorePaused = paused
	return nil
}

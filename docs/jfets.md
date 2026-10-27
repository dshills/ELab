# Junction field-effect transistors

Add an N-channel or P-channel JFET from Semiconductors, or open `jfet-preamp`,
`jfet-current` or `pjfet-current`. Select J1 to edit IDSS, cutoff magnitude and
channel modulation. `jfet.configure` is the same saved, undoable, batchable
command used by the inspector and every agent adapter. All four configuration
fields—version, idss_a, cutoff_v and lambda_per_v—are required. The signed `cutoff_v` is negative for N, positive for P.
IDSS is 10 µA–100 mA; |cutoff| 0.1–12 V; lambda 0–0.1 /V.

`elab.generic-jfet.v1` models a normally conducting depletion channel. In normalized
N polarity, let u=VGS−cutoff and beta=2*IDSS/cutoff². For u≤0 channel current is
zero. For 0≤VDS<u it is beta*(u*VDS−VDS²/2); otherwise beta*u²/2. Multiply by
1+lambda*VDS. Reversed drain/source exchange roles; P-channel reverses voltages
and currents. There is no MOS body diode. Both gate-to-channel ends have the
existing generic silicon junction law (27 °C, IS 1 nA, n 1.7, Rs 1 Ω), plus
an explicit 1 GΩ bleed each. Reverse gate leakage and forward gate current are
part of terminal KCL and power. IDSS describes the channel at zero VGS/lambda;
terminal current can include gate leakage. Source current is positive out of
source; gate/drain positive into device.

The small-signal AC stamp uses the analytic Jacobian at solved bias. External
capacitors use the existing transient/AC engines. No intrinsic charge, noise,
breakdown, heating, ratings or fitted silicon identity is claimed. MOSFETs and
JFETs share the existing combined two-FET budget; the budget is not enlarged.
Source/part/node/storage/work/accuracy limits remain unchanged.

In the preamp, RD/RS establish bias and COUT removes DC before RL. Changing
IDSS/cutoff moves the bias and gain. In the current sink/source, RSET reverse-biases
the gate and reduces current below IDSS. Insufficient drain headroom enters the
triode region; current is not magically constant. Probe terminal voltages/currents
and inspect signed heat/state, then Verify, save/reopen or export the schematic.
Model parameters and validated readings accompany captures; stale readings stay
marked stale after edits.

Background references: [TI, Understanding Basic Analog—Active Devices, JFET
section](https://www.ti.com/lit/an/sloa026a/sloa026a.pdf), and
[Analog Devices, System Applications Guide, section 1](https://www.analog.com/media/en/training-seminars/design-handbooks/system-applications-guide/Section1.pdf).
The explicit equations above define ELab's educational model; these references
do not validate a particular physical device or its parameter choices.

# Current sources and electronic loads

`current-led`, `load-step` and `constant-power-load` are editable examples.
Select I1/LOAD1 in Inspect to configure levels, pulse timing, compliance or load
limits. Users and agents call the same saved, undoable, batchable commands:
`current-source.configure` and `load.configure`. Read discovery's filtered
capabilities and referenced schemas for every required field and domain.

`elab.current-devices.v1` is an educational algebraic model, not a named
instrument or a protected physical supply/load. Both devices include an explicit
1 GΩ parallel bleed. A source programs 0–1 A delivered out of its positive pin.
For terminal voltage v and compliance Vc, it delivers
`Iset * (1-smoothstep(v,Vc-1,Vc)) - v/1e9`.
Thus the final volt of headroom reduces current smoothly to zero, and a light
load approaches compliance. This is a finite Norton law, not an ideal voltage
constraint. Reverse current/sink operation is not a programmable feature.

The one-quadrant load draws current into positive. At positive voltage above
Vmin, CC requests Iset, CP requests Pset/v. Below Vmin these become Iset*v/Vmin
and Pset*v/Vmin² respectively. The controlled current is capped at Imax; the
bleed adds v/1e9. At negative voltage only the passive bleed conducts. This
explicit foldback avoids a singularity at zero. CP has negative differential
conductance above Vmin; inadequate supplies can collapse or refuse to converge.
A refusal is visible, not silently replaced by an ideal stable power point.

DC uses the high setting. Optional pulses interpolate low→high during rise,
hold through the remainder of high phase, interpolate high→low during fall,
and hold during low phase. Phase lengths include ramps. Each is 1 µs–10 s,
period ≥1 ms, rise≤high, fall≤low. Timing remains required/validated in DC mode.
Pulses are independent of the primary *voltage* input mode and participate in
existing corner, adaptive-step and scope-sampling advice. Current-only circuits
have no misleading voltage-input high/low buttons. DC/AC bias samples current
and load levels at t=0 (low for pulsed devices); voltage-source DC selection is
unchanged. AC can linearize these laws, but excitation remains the selected
voltage source; a current-only circuit has no voltage-excitation Bode target.

Scope probes show terminal voltage, source delivered current or absorbed load
current. Readouts, verification recordings, projects, undo/redo, schematic
metadata and print descriptions retain the model and full settings. Opening is
silent. Source count remains at the existing combined three-source budget;
normal part/node/storage/current/range/work/accuracy guards remain. No dynamics,
thermal protection, rated SOA, control-loop bandwidth or manufacturer fidelity
is claimed. These are model experiments, not physical hardware qualification.

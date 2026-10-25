# Transformers and coupled inductors

**Transformer / coupled inductors** represents two linear windings. Pins are
primary1/primary2 and secondary1/secondary2; voltage is pin1 minus pin2 and
current is positive into pin1. The primary dot stays at primary1. **Reverse
secondary dot** moves the secondary dot to secondary2 and reverses mutual sign.
The inspector shows both winding voltages/currents, flux linkages, shared
magnetic energy, winding losses, and primary/secondary powers.

Native controls and agents share `transformer.configure`:

```
{"id":"T1","parameters":{"primary_inductance_h":0.1,"secondary_primary_turns_ratio":0.5,"coupling":0.98,"primary_resistance_ohm":1,"secondary_resistance_ohm":1,"reversed_secondary":false}}
```

Primary L is 1 µH–100 H, secondary/primary turns ratio 0.1–10, coupling 0–0.9999,
and each winding resistance 0.01–1000 Ω. Derived secondary L = Lp × ratio² must
also be 1 µH–100 H. Every parameter, including the false polarity flag, is
required. These are saved, undoable startup edits. Running circuits restart;
paused results are stale until Restart/Resume. Each new run starts at zero
winding current. Both windings need a determined voltage reference; magnetic
coupling never creates a conductive path between them. Open/shorted windings
are supported when the remaining circuit establishes the required references.

The equations are:

- Vp = Rp Ip + Lp dIp/dt + M dIs/dt.
- Vs = Rs Is + M dIp/dt + Ls dIs/dt.
- M = ±k sqrt(Lp Ls); flux linkages are Lp Ip + M Is and M Ip + Ls Is.
- Shared energy = (Lp Ip² + 2 M Ip Is + Ls Is²)/2.
- Winding loss = Rp Ip² + Rs Is².

A completed-square energy expression avoids cancellation near strong coupling.
Radau storage rows integrate mutual flux; startup cut constraints use the full
inverse inductance matrix. The LC step guard includes the smallest coupled
inductance eigenvalue. Existing error estimators, node bounds, 512 attempts and
100 ms per-output integration budgets remain unchanged. Two private windings
consume two inductor slots and one extra simulated primitive part; they do not
appear as authored objects or independent energy results. Exact k=1 is singular
in this winding formulation and is refused rather than silently approximated.
No core saturation, hysteresis, frequency-dependent core loss, interwinding
capacitance, thermal dynamics or physical isolation rating is represented.

`mixed.transformers` exposes typed readings. `v:T1` and `v:T1.secondary` show
both winding voltages. `i:T1.primary1` / `i:T1.secondary1` show signed winding
currents; corresponding pin2 currents have the opposite sign. Native Plot
buttons dispatch the same scope commands. AC sweeps include ±jωM cross terms,
so agents can compare turns ratio, winding loss and phase against loaded complex
impedance calculations. Save/reopen, recovery, duplicates, batch edits, captures
and schematic exports preserve the configured winding parameters and dots.

**Transformer ratio and winding polarity** uses a 50 Hz, 12 V peak source,
0.1 H primary, 0.5 turns ratio, k=0.98, 1 Ω windings and a 100 Ω load.
The light-load induced voltage approaches k × ratio × primary inductive voltage;
finite coupling/resistance/load prevent an exact ideal-transformer ratio.
Reverse the secondary dot and Restart to compare phase.

**AC transformer to regulated 3.3 V** uses a 1 H /0.75-ratio transformer with
10 Ω windings, a full bridge, C1 = 1 mF, a 3.3 V LDO and 1 kΩ output load.
RSN/CSN = 100 Ω /10 µF explicitly damp winding commutation in a model without
intrinsic capacitance. RREF = 100 kΩ defines secondary common mode when the
bridge diodes turn off. These are authored, removable circuit parts, and their
current/energy/loss is included. The example uses a 100 µs output step and slow
playback to show startup, ripple and regulation. Compare secondary AC, C1 and
U1 output; lower turns or increase load to observe lost headroom.

Tests compare open-winding startup and transient decay with analytic RL/induced
voltage laws, polarity reversal, shorted-secondary initial slopes, positive
magnetic storage, loaded AC complex impedance, full bridge smoothing/regulation,
conservation, failed-work atomicity, strict capture integrity and saved roundtrip.
Native parameter controls are compared with the public agent commands.

The [Analog Devices transformer simulation reference](https://www.analog.com/en/resources/technical-articles/ltspice-basic-steps-for-simulating-transformers.html)
provides turns/inductance, mutual coupling and dot-polarity context. ELab uses
its own bounded winding model above.

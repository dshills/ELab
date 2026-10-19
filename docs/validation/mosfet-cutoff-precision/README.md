# MOSFET cutoff voltage convergence

The source repair closes the reproduced **3.217580 mV PMOS cutoff voltage discrepancy**. A small current residual does not bound node voltage error near a weakly conducting load. MOSFET networks now use the existing Newton voltage-correction convergence check already required by inductor and powered-op-amp networks. The existing precise LED evaluator also applies in these networks so leakage-scale current does not prevent convergence. Device equations/models, public commands, native bindings, saved format and SDK remain unchanged.

This is **development-source proof**, based on `561a33c` with modified production source. A clean replacement and copied native/public qualification are next. Preserved 0.144.0 /161 still has the discrepancy. This stricter MOSFET check is additional numerical qualification; it is not an invented claim that the original PRD set a universal 1 mV MOSFET specification.

## Reproduction and result

The [initial focused regression](before-test.go.txt) fails on the original source for PMOS and passes for NMOS. [Failure](before-regression.txt) reproduces the earlier native trace exactly: NMOS error 0.764745 mV; PMOS error 3.217580 mV. A preceding [cache sandbox refusal](before.txt) occurred before tests and is not numerical evidence.

The [final regression](../../../workbench/mosfet_cutoff_precision_test.go) checks cold/warm cutoff samples, both sides of pulse edges, live and bounded Verify traces, signed leakage and unchanged live physics. [Focused race results](focused-final.txt) pass existing MOSFET/CMOS and recorded-model validation/round-trip cases as well. [Patch](source-repair.patch.gz).

The [public producer](reproduce.py) uses fresh owned sessions and unchanged public example/probe/Step/Verify/sample/save/load commands. Each polarity retains **505 six-probe rows** from 0–500 ms, with both sides of the 200 ms and 500 ms transitions. Both live and independently calculated Verify rows are checked. Separate fresh-agent reopens preserve graph, paused state and capture metadata/rows.

| Maximum cutoff drain-voltage error | Preserved /161 | Repaired development agent |
|---|---:|---:|
| NMOS, live and Verify | 0.764745 mV | 4.45×10⁻¹⁶ V |
| PMOS, live and Verify | 3.217580 mV | 8.31×10⁻¹⁴ V |

[Original public report](old-public/report.json.gz) remains `passed:false`; [repaired public report](fixed-public/report.json.gz) passes. For cutoff, the declared square-law channel contributes zero. The body diode supplies approximately 1 nA through the 5 V /1 kΩ/red-LED loop. The independent inverse LED law gives |VDS| **3.689762762966 V**. The omitted reverse-body exponential is below 4×10⁻⁴⁶ A; its voltage effect is negligible relative to the unchanged strict 1 mV comparison. This calculation never invokes the application solver. It does not validate physical-device leakage or switching delay.

The [producer correction](producer-correction.txt) retains the first public attempt's unnecessary exact byte equality between separately solved live and Verify traces. That assertion failed before a complete report. Both curves now receive the same independent checks; their exact equality is reported as a diagnostic. No numerical tolerance was widened, no sample was dropped and the old PMOS result remains failed. Original producer and error output are retained.

## Quality and preservation

The complete [quality gate](quality.json) passes in **313.085 s**: formatting/imports/Staticcheck, fresh PRD audit, 51 Python tests, root/native race and vet, mounted control regressions and tagged vet. [Raw output](quality.txt) retains the existing duplicate Objective-C library linker warnings. Root conformance takes 117.644 s, workbench 82.780 s, native 144.554 s and mounted native checks 19.344 s.

[Development identity](development-identity.json) pins the modified source and agent; [build metadata](development-build.txt.gz) explicitly records `vcs.modified=true`. A cache-stat write warning accompanied the successful build; it did not prevent generating or running that exact binary. [Preservation](preservation.json) verifies all 22 sealed review manifests, canonical /120, unchanged SDK pin and native command ledger. Previous captures and affected review packages are not rewritten.

No fresh physical UI input, whole-bundle/copy-runtime, latency, clean installation, reader/IME or owner acceptance is inferred. The historical NMOS Pause observation remains unexplained; this solver repair does not address it. Owner remains **Not tried yet**. No app is promoted.

Verify retained evidence without launching an application:

```sh
python3 -B docs/validation/mosfet-cutoff-precision/verify.py
```

Run the public producer against a chosen agent (the artifact directory must be new):

```sh
python3 -B docs/validation/mosfet-cutoff-precision/reproduce.py \
  --agent-binary build/mosfet-cutoff-precision/elab-agent \
  --artifacts build/mosfet-cutoff-precision/new-public-run
```

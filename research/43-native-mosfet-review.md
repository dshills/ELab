# Native MOSFET controls and signed readings

October 19, 2026 · exact **0.166.0 /183** app/agent. Ten accepted native electrical states match the public agent and independent load-line/current/power calculations. No application source or version change is needed. One documentation overclaim is corrected.

| Experiment | Accepted native observations |
|---|---|
| NMOS LED switch | Restart at zero, Low at 1 ms and High at 2 ms. On-state drain/source current is about +2.9086 mA; gate current is zero. |
| PMOS high-side LED | High is off; Low turns it on. On-state VGS/VDS and drain/source current are negative, with ID about −2.9086 mA. LED current and VDS×ID dissipation are positive. |
| CMOS endpoints | Native Low/High reverse the output: about 4.9986 V versus nearly zero. The two drain-current signs remain consistent. |
| CMOS midpoint | Type VIN high = 2.5 V, Apply once, Restart. Both devices conduct: roughly −52.619 mA PMOS /+52.381 mA NMOS; output is 2.380952 V. Private typing preserves the entire accepted snapshot; Apply truthfully marks old data stale before Restart. |

The three examples are loaded with native menus; native source/step controls and the midpoint value field are exercised through CUA. Three separate synthetic projects are saved through public commands and reopen clean in fresh packaged agents. This is not native file-picker evidence.

Independent scalar bisections use the pinned generic square-law, body-diode and LED models. Explicit wires and signed currents independently establish node balances; V×I establishes power balance. Maximum voltage/current reference differences are 8.305e-14 V /4.272e-16 A, with KCL/power-balance residuals below 2.981e-15 A /5.703e-15 W. These finite-fixture results do not certify hardware or broader transistor behavior.

Two named Pause attempts do not establish Pause; coordinate Pause works. A mixed AX item-click/Return selector attempt unexpectedly leaves the app running and remains unqualified. A separate pointer-opened menu with ordinary keyboard navigation/Return changes selection while preserving Pause, exact time and complete physics/history. Cause of the first outcome is unestablished; it is not relabeled a product defect or a successful paused-selection case. Window enlargement aids inspection; narrow/dense Inspector text remains a layout limitation. These results do not close general input/IME/reader or minimum-size readability gates.

The PMOS guide incorrectly said native Inspector/probes displayed MOSFET power. The UI summary shows VGS, VDS, ID, IG and region; power exists in structured `mixed.mosfets[].power_w`. The guide now states that boundary. No new UI feature is claimed.

[Full evidence and read-only replay](../docs/validation/native-mosfet-input-183/README.md) retain original CUA images/AX, all snapshots/protocols, independent references, the initial reference-key and image-format packaging mistakes, and both menu outcomes. The owned native session closes with completed joined/ready cleanup; `open -W` exits 0, PID/endpoint disappear, and the native host's exact exit status remains unavailable. All 658 application Go inputs, copied/sealed review, SDK, canonical app and shared window settings remain unchanged. No privacy grant, recording, printing, owner-project edit or promotion occurs. Owner walkthrough and broader MVP gates remain open; P0 stays **17 implemented /10 partial**.

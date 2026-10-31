# Start with ELab

ELab is a local schematic editor with live educational circuit simulation.
This alpha runs on Apple Silicon. Check the minimum macOS version in the
README.md beside the app before opening it. The folder includes ELab.app, a matching
elab-agent, editable examples, dependency notices and these guides.

1. Extract the downloaded ZIP and keep the whole folder. Move it somewhere
   durable, such as your Applications folder. Open ELab.app.
2. This is an unsigned, ad-hoc-signed alpha, without Developer ID/notarization.
   macOS may block opening it. Check the published SHA-256 and origin first.
   If you trust this build, use macOS Privacy & Security's per-app Open Anyway
   route when offered. Do not disable Gatekeeper or remove quarantine globally.
   On a managed Mac, your administrator may prohibit unsigned apps. See
   [Apple’s app-opening guidance](https://support.apple.com/102445).
3. Open Examples and try **RC charge/discharge** or **Capacitor-timed LED flasher**.
   Complete supported circuits simulate as you draw. Pause freezes accepted
   readings; Restart restores startup; Step advances a paused transient circuit.
4. Use the sidebar button at the top of **Parts** to close it. The left end of
   the toolbar reopens Parts. The rightmost sidebar button toggles **Inspector**.
   Drag either divider to resize. In **Inspect**, select a part to edit its value
   or open Details for model settings.
5. **Simulate** chooses one tool. Basic inputs and its Record/Run action come
   first; Advanced retains the rest. The instrument dock's **Scope / Spectrum**
   tabs share live probes. Recorded runs and their spectrum are separate captures.
6. **Measure** chooses a component, wire, terminal, net or probe. Inspect shows
   readings for that target. Plot adds a trace. Pin adds a reading to the drawing.
   Full readings and the original network overview remain under More measurements.
7. Save your own copy before modifying something you want to keep. Document has
   SVG/PDF/PNG export and print preview. Start with Fit to page for one sheet.

To try the four-MOSFET bridge, open **Examples → Power & relays → Four-MOSFET H-bridge**
or open `Examples/discrete-hbridge.elab`. Pause, restart and step across 20 ms.
The 100 Ω load reverses from about −4.725 V to +4.725 V. See [the bridge guide](hbridge.md).

For a structured first try, open `first-use-test-kit.md`. It includes help and a
blank feedback form; the separate maintainer key contains expected values.
For agents, run `./elab-agent --mcp-config` from its final location and follow
`agent-setup.md`. No provider account or key is needed by ELab itself. Generated
configuration does not edit client settings. User and agent commands use the same
validation and revision guards.

Models are generic educational approximations. Inspect Details identifies model
settings/limits; do not treat simulated operation as hardware certification.
This alpha has separate open participant, native queue/hardware, clean-account
installation and signing checks. See the repository's current-release evidence.

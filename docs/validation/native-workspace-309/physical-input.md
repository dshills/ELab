# Physical input on build 309 source

The project owner operated a disposable ELab RC window on an Apple Silicon Mac
running macOS 27.2. A test-only observer logged button-down, drag and button-up
coordinates delivered to this ELab window. It did not capture keyboard events.
The matching agent sampled the accepted divider fraction and schematic height.
A hash of the paused circuit, document, electrical revision, transient state
and scope was sampled alongside the geometry.

**Check 3, Save filename editing, passed on the physical keyboard.** File →
Save As opened with **Untitled**. After Right Arrow, the owner pressed ⌘A and
typed **KeyboardCheck**. The native filename showed exactly **KeyboardCheck**.
The owner confirmed that ⌘Z undid the change, and the field was observed back
at **Untitled**. Cancel returned to the same RC schematic without writing a file.
This distinguishes ordinary keyboard behavior from the earlier [automation-route
failure](../native-workspace-306/native-input.md).

**Check 6 has an owner-reported successful drag.** During the measured window
the agent continued to report a schematic fraction of 0.7 and height of 774 px.
The observer logged one title-bar click with zero displacement.
The fixture did not capture the pair of slow and fast physical divider gestures.
The earlier [native automation drag and return](../native-workspace-306/native-input.md)
did change the fraction from 0.7 to 0.5736462 and back. The controlled
physical-rate comparison remains open.

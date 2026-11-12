# Browser workbench screenshots

Captured November 12, 2026 from the local development working tree, using a
separately built Go host, disposable project storage and headless Chromium.
These are unmodified 1600 × 1050 browser viewport screenshots, not mockups or
schematic exports. No user project or existing browser session was used.
The source tree had concurrent development changes; these images document the
development UI, not a qualified release binary.

- `amplifier-apollo.png`: **Amplifier driving a loudspeaker**, Apollo appearance.
  Used **Fit sample interval**, restarted, and advanced to 5 ms. The scope shows
  65 samples of input voltage, speaker voltage and speaker current.
- `hbridge-dark.png`: **Four-MOSFET H-bridge**, Dark appearance, paused at 62 ms.
  The scope shows control voltage, load voltage and load current across reversals.
- `learn-capacitor.png`: **Start capacitor activity**, Light appearance, paused
  at 20 ms with the first guided step visible and C1 selected.

All configuration used the application's controls. Captures were visually
reviewed for readable circuits, populated plots and unobstructed controls.
The owned browser and host were closed after capture. File hashes and dimensions
are recorded in `screenshots.json`. Older native screenshots in the parent
folder retain their original historical provenance.

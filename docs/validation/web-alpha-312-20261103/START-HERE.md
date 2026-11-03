# ELab Web 0.206.0 alpha 1 — build 312

Download `ELab-Web-0.206.0-alpha.1-macos-arm64.zip` and extract it. Move
**ELab Web.app** to Applications, then open it. The small launcher starts ELab's
local engine and opens your default browser. Go, Node, Python, an account and an
internet connection are not required to use the app.

This download is for Apple Silicon Macs. It is ad-hoc signed and not notarized;
macOS may block first launch. The build target is macOS 13, but runtime
qualification of this ZIP covers the current macOS 27.2 workstation. Hosted
macOS 15 CI builds and checks a separate package from the same source; it does
not test this exact downloaded ZIP. macOS 13/14 and Intel Macs are not qualified
by this release.

## Keep your work

Edits autosave on this Mac. **Save project** creates a named project. **Projects**
contains saved projects, recovery history and reversible Trash. **Document →
Download .elab copy** exports a portable copy, including retained recordings.

Existing web projects remain in `~/Library/Application Support/ELab/web`.
Keep that folder when upgrading. Native `.elab` files can be imported with
**Open**, then saved as named web projects. The former native application and
earlier alpha releases remain available separately.

Closing a browser tab leaves the local engine running. Use **Quit ELab** in the
launcher, or Command-Q with the launcher focused, to save and stop it. Reopening
restores your work. If an older command-line host owns the same library, close
that host normally before starting the app; keep its data folder.

## Try the four-MOSFET bridge

1. Choose **Explore examples → Four-MOSFET H-bridge**.
2. Pause and choose **Restart simulation**. The 100 Ω load starts near −4.725 V.
3. In **Simulate**, set **Step duration** to `21 ms` and choose **Advance simulation**.
   The load reverses to about +4.725 V.
4. Choose **Record**, enter `45 ms` with a `1 ms` sample interval, and start.
   Keep the recording, save the project and reopen it to inspect exact samples.

All 88 examples and 61 component kinds are built in. The browser workbench uses
the shared Go solver, device models, measurements and project format. Models
are educational approximations; this bridge omits gate charge, switching loss
and dead time.

## Verify the download

Put the five release assets in one folder, then run:

```sh
shasum -a 256 -c SHA256SUMS.txt
```

`PACKAGE.json` records source, toolchains, notices and sealed bundle hashes.
`QUALIFICATION.json` records the exact tested source and package, CI results,
browser journeys, archive/runtime checks and launcher-check scope. Automated
accessibility checks do not establish VoiceOver or whole-app conformance.

Source: `b90d7e95018be93afea94d45e246e4296d9978b4` on `main`.

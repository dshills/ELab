# ELab Web one-click launcher — 2026-11-01

The macOS app contains a small Cocoa launcher and the standalone Go web host,
including all frontend assets. Users launch the app directly; Go, Node, Python
and a terminal are development tools rather than runtime requirements.

## Behavior

- Launch starts the host and opens its actual address in the default browser.
  The launcher provides running/stopped/error status, Open ELab, projects folder,
  log access, retry and Quit. Window close and Command-Q use the same quit path.
- A busy preferred port falls back to an available loopback port. A second launch
  for the same project library authenticates and reuses the existing compatible
  host. It does not guess a port or terminate another program.
- Older CLI builds have no host handshake: close that old server once before
  opening the app. The launcher explains this conflict rather than opening a
  second library or replacing its contents.
- Normal Quit drains API requests, checkpoints all workspaces and rejects new
  mutations. A failed checkpoint refuses Quit and leaves the editor available.
  Native workers stop before the final capture; the library lock is released only
  after shutdown and instance-owned host metadata cleanup.
- Control credentials are private local files, never browser URLs or public
  catalog fields. Control requests validate loopback identity and do not use
  environment proxies or follow redirects. Launch reply files refuse replacement
  of an existing file/link.

## Checks and evidence

- Complete web race suite: passed (12.134 s); host checks cover actual listeners,
  occupied-port fallback, reuse, normal stop/restart, 22 kΩ recovery, unauthorized
  stop refusal, storage-failure refusal, private launch files and redirects.
- All 23 frontend tests/syntax checks and all 107 Python tests passed; web vet and
  diff checks passed. Swift compiles for the explicit macOS build target.
- The native fixture was launched via computer use before the owner's video
  restriction. Its actual status showed `localhost:62262` while another session
  used 8788. Open-browser startup succeeded; the copied browser workbench accepted
  an unsaved 2.2 µF edit. Clicking the native Quit button removed the listener and
  host record and preserved that edit in its checkpoint. Reopening reached a new
  running host; the follow-up interactive restoration check was deferred.
- Independently, a copied backend ran outside the source directory, without Go or
  Node in PATH. It served the embedded modules, accepted a 22 kΩ edit, quit
  normally and restored that exact edit after restart. This checks backend
  recovery without opening a browser or capturing the desktop.
- Package verification checks strict ad-hoc signatures, system-only runtime
  dependencies, linked Go dependency notices and executable identities. The
  embedded receipt records the backend hash; the external receipt records the
  final launcher hash after the bundle signature, avoiding a self-hash cycle.

Screenshots [launcher](launcher.png) and [workbench](packaged-workbench.png) were
saved before the request to stop desktop testing. Subsequent testing/assembly
uses only offscreen processes. The source was internally reviewed according to
AGENTS.md; no new external Prism submission was made for this milestone.

## Build and qualification

```sh
python3 -B scripts/package_web.py \
  --output 'build/web-alpha/ELab Web.app' \
  --archive build/web-alpha/ELab-Web-0.1.0-alpha.1-macos-arm64.zip
python3 -B scripts/qualify_web_package.py \
  --app 'build/web-alpha/ELab Web.app' \
  --artifacts /private/tmp/elab-web-alpha-qualification
```

Output/receipt/archive paths must be new. Test bundles require an explicit
`--test-data-dir`, carry a separate identifier, and cannot be archived for
shipping. Packaging leaves the qualified native app intact. User projects remain
in the existing `~/Library/Application Support/ELab/web` library when upgrading.

This is a private alpha. Ad-hoc signing is the default; an explicit Developer ID
uses hardened-runtime/timestamp options, but Developer ID/notarization has not
been qualified. The arm64 build and current workstation were exercised. The
minimum build target is macOS 13; clean-account and other-OS-version acceptance
remain separate gates. The fixture GUI evidence is distinguished from copied
backend evidence; neither establishes a clean-machine GUI installation result.

## Internal review closeout

Reviewed listener ownership, root-lock/host-identity checks, control-token
lifetime, request draining, storage-refusal behavior, worker shutdown ordering,
launch reply replacement, GUI retry/quit interleavings, version/fixture metadata,
notices, runtime linkage and signature/hash publication ordering. Fixes included
correcting post-sign launcher hashes, locating the Homebrew Go license, separating
storage errors from an occupied library, queuing Quit during an attached-host
check and bounding status probes. Swift sources participate in quality input
signatures. Native releases and existing output paths are preserved.

The race/Go, frontend and Python logs are retained in [checks](checks/).
Copied-backend receipts are explicit about prototype dirty-source scope; a final
clean-source package has its own generated package and copied-runtime receipt.
No screen capture, application/browser launch or GUI interaction was performed
after the owner requested desktop testing stop during video playback.

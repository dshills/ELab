# ELab application icon

October 26, 2026. Owner requested an application icon using the latest Vellum
release. Verified public main `4cf01763c1fd5501648ea19b4bb406fe62446319` contains
the new icon packager, introduced at `3fadc8f`; its release metadata is v1.0.0.
GitHub's published release list still has only the older alpha.3 prerelease.

The feature is a packaging option, so select and pin the latest unmodified
packager separately instead of changing the qualified UI runtime for an icon.
Keep runtime SDK elab9, existing public commands, electrical models and /272
drawing behavior. Preserve the owner's running /272 session and all previous
qualified packages. No push, upload, new components or security changes.

Original editable SVG/1024×1024 RGBA artwork shows a circuit E, resistor and
amber LED on a blue tile. The artwork is inspected at 16/32/64/128/256 pixels.
Vellum will generate the ICNS representations and embed them before signing.

The checked-in packaging dependency consists of 12 byte-identical upstream
files, including the MIT license, release metadata, helpers and seven icon
tests. Each is SHA-256 pinned and verified before execution. Source is read
from the exact public commit in the sibling checkout without changing it;
the public GitHub commit identity was checked first. Verified SDK caches and
the existing SDK lock remain untouched.

77 ELab Python tests and seven upstream icon tests pass. New bundle/native
checks and their precise scopes follow in the validation record; metadata or
resource decoding alone must not be called a verified Finder/Dock screenshot.

## Complete

Runnable **0.196.0 /273** has the icon embedded and ad-hoc sealed before
publication. Ten decoded representations and macOS file-icon lookup show the
expected artwork. The unsealed executable and matching agent are identical to
/272. Fresh copied native smoke and four public adapters pass; all 102 external
processes exit normally. All 126 example files and thirteen prior release/SDK
archive identities stay exact. No owner session command or project edit was
sent. No UI runtime upgrade, new component, solver-limit change, push or upload.
[Detailed evidence and boundaries](../docs/validation/application-icon/README.md).

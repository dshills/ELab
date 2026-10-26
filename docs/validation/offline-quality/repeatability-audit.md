# /265 release repeatability audit

The existing build, SDK packager, package_review.members/verify_archive and
runtime-portability controller remain the packaging mechanisms. The current
candidate uses clean source 1d4ff4602585a33a6a548fac4c4949466ad71cb6. Each binary advertises
vcs.modified=false; the framework source/archive/manifest remain pinned in sdk.lock.

The exact preparation scripts are retained in this evidence directory as text,
including strict clean-source checks, exclusive output paths, post-sign app pin,
current matching agent, all62 examples, notices and rollback. The final archive
step requires a passing full quality receipt and passing copied-runtime receipt;
both executable hashes must match, prior /264 /260 /205 hashes must match, and
codesign --verify --deep --strict must accept the bundle. Every archive member
then matches the macOS extraction bytes and mode; extracted examples reopen
under the same source/IP denial policy. This is a local reproducibility record,
not a new cross-platform build/bootstrap or signing service.

The final quality summary now passes. The archive/extraction step follows
that gate and the completed copied-runtime qualification. Preserve /264 unchanged.

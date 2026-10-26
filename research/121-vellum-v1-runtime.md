# Vellum v1 runtime refresh

October 26, 2026. Owner explicitly requested the latest UI runtime in addition
to the already-updated icon packager, with existing behavior and shared user/AI
commands preserved. No new electronics components, models or solver limits.

Verified public main is `4cf01763c1fd5501648ea19b4bb406fe62446319`; its metadata
identifies v1.0.0. GitHub's published release list remains the older alpha.3.
The selected source includes 43 commits after ELab's old f9dee54 upstream base.

Use an owned checkout in build/vellum-runtime-4cf0176. The sibling Vellum checkout,
old compatibility checkout, all verified SDK caches and earlier /271–/273 and
qualified packages remain preserved. Record protected hashes before any changes.
No push/upload or heartbeat restart. Existing owner app/projects are untouched;
all UI tests use owned fixtures and standing authorization, without new permission
requests. Original native checks 3/6/9/11 remain separate acceptance gates.

## Ordered checks

1. Carry ELab's cumulative compatibility patch onto latest upstream. Preserve
   admission/held-input origins, control/text ownership, numeric settlement,
   decimal steps, scrolling, cursors, tooltips, dividers and window styling.
2. Verify all ordered native input/output hashes, full framework Python,
   portable/ordinary/rendered race and vet checks, and stable v1 API compatibility.
3. Export a clean, private, fully hashed SDK; pin source/archive/manifest/bootstrap
   and all four module replacements. Preserve all old archives/cache bytes.
4. Run complete ELab source, ordinary/rendered UI, electrical and agent checks;
   reproduce and repair concrete regressions without changing solver limits.
5. Package a new icon-equipped candidate from clean source, verify exact copied
   native smoke/public adapters and all existing examples, close owned fixtures,
   commit reviewed evidence and hand off a clean repository.

## Merge progress

Only three source files conflict. Button press-origin capture is combined with
upstream NativeContent mounting; disposal keeps both new help-tag retirement and
ELab tooltip retirement. Native dependency/test metadata retains upstream's
prose stage 107 and shifts the six existing compatibility stages to 108–113.
No patch input overlaps the prose stage. Native preparation and its Python tests
pass, with exact recorded input/output hashes.

Two setup mistakes initially count stage records rather than patch ordinals
and assume every stage has a patch; asset-only stages invalidate that assumption.
Those are migration-script setup errors, not product defects. The corrected
recipe selects explicit patch ordinals and verifies the contiguous 1–113 list.
The earlier conflicted-manifest preparation is retained as setup evidence only.

## SDK qualification complete

Clean compatibility source `19c9ebc37ed9f06daa824e024b331caf9700f4b0` passes full framework
portable/ordinary/rendered race suites, all vets, 90 SDK Python tests and native
preparation tests. The stable v1 API check reports no incompatible changes.
The first API invocation used the parent module instead of the nested tool
module; corrected execution and the tool's own race/vet tests pass. That setup
failure is retained separately. No API baseline or SDK test is weakened.

The private archive contains 8862 fully verified files; source, manifest,
archive, patch and bootstrap hashes are pinned. All four native replacements
resolve into `build/.vellum-sdk-4cf0176-elab1`. ELab's complete fresh checks are now running.

## ELab source qualification complete

Complete fresh race suites pass 880/347/484 top-level (1,963/847/1,518 including
subtests), all vets, lint/imports, 77 Python tests, PRD and build. No app-source,
model, command or solver-limit change is needed after the compatibility merge.
Existing assertions remain intact. Packaging/native qualification follows the
clean source checkpoint; previous apps and qualified releases stay preserved.

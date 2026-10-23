# ELab project instructions

## Standing user authorization

The project owner has explicitly and repeatedly authorized running ELab and
using the desktop to develop, test, and verify this application. This permission
persists across turns, context resets, and release/test builds.

The owner's explicit instruction is: “Stop asking for permission to run ELab,
you have it!” Treat this as standing authorization, not a one-launch exception.

- Do not ask the user again for permission to launch or run ELab, attach to an
  existing ELab session, or use the desktop for ELab testing.
- Use this existing authorization when a tool needs the action's justification;
  do not turn a routine ELab test or launch into another user permission request.
- This includes rebuilding and running updated ELab test/release candidates and
  checking mouse, keyboard, scrolling, and other UI behavior. Do not ask whether
  now is a convenient time unless the owner has since restricted desktop use.
- If a platform approval or sandbox review still blocks an authorized action,
  use an available authorized route or report that specific technical block.
  Do not ask the owner to authorize ELab again or imply authorization is missing.
- Prefer reusing an already-running owned test session. Check its actual state
  after a tool timeout before launching another copy.
- The owner explicitly reported that launch-tool approval prompts still appeared
  despite this authorization. Do not assume that changing a test app's name or
  bundle identifier avoids those prompts. Reuse the running owned ELab session
  for desktop checks. Do not leave unattended work waiting on another launch
  approval; continue available source, agent and offscreen checks if launch
  tooling cannot honor the standing authorization. Do not weaken platform
  security or approval settings as a workaround.
- Preserve the user's projects and existing application sessions. Use disposable
  fixtures for testing and close owned test sessions normally when finished.

## Current work scope

The first usable unsigned private-alpha release is qualified and must remain
preserved. The owner subsequently authorized continuing remaining items or new
features while signed off, until they return. This supersedes the preceding
release feature freeze for ongoing development.

Prioritize known defects and outstanding requirements before optional features.
Keep code, repository, release evidence, and package quality in scope. Owner
review is deferred by instruction and must not block engineering progress.

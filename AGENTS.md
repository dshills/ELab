# ELab project instructions

## Standing user authorization

The project owner has explicitly and repeatedly authorized running ELab and
using the desktop to develop, test, and verify this application. This permission
persists across turns, context resets, and release/test builds.

- Do not ask the user again for permission to launch or run ELab, attach to an
  existing ELab session, or use the desktop for ELab testing.
- Use this existing authorization when a tool needs the action's justification;
  do not turn a routine ELab test or launch into another user permission request.
- Prefer reusing an already-running owned test session. Check its actual state
  after a tool timeout before launching another copy.
- Preserve the user's projects and existing application sessions. Use disposable
  fixtures for testing and close owned test sessions normally when finished.

## Current release scope

Finish the first version against the release checklist. Do not add new features.
Keep code, repository, release evidence, and package quality in scope. Owner
review is deferred by instruction and must not block engineering completion.

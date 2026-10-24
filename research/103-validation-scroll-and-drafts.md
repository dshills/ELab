# Validation layout and unfinished numeric values

October 24, 2026. [Evidence](../docs/validation/validation-scroll-222/README.md).

Native testing reproduced focus moving outside the scrolled Simulate inspector
when interval validation adds guidance. The same production inspector fails in
live and recorded mounted checks. The narrow Vellum repair reveals the current
eligible focus once after layout, preserving explicit scrolling and ownership.
A separate framework regression found that native Spinner updates discard an
unfinished number after a parent structure change. Managed synchronization now
owns accepted-value refresh without erasing the private draft.

The four production interval cases and focused framework regressions pass,
along with full framework/application quality. The public command surface,
electronics models and production application Go sources are unchanged.
Native candidate verification remains separate. Default native divider drags
passed; controlled slow/fast input is still unqualified. The standing launch
and desktop authorization remains saved in AGENTS.md; no new question is needed.

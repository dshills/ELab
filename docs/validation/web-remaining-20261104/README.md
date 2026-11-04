# Remaining web priorities — November 4, 2026

Source baseline: `f1fc39eb`, after items 1/3/5 merged to main. The original list
and current status are [recorded here](../../web-followups.md). Each milestone
receives internal review and a separate commit before subsequent implementation.
The owner has no testers available, is a novice, and requested best judgment
and online knowledge. That context is now in the project instructions.

## Item 2 — beginner activities and engineering rehearsal

Added three guided activities beside the drawing, accessible from Help. Removed
stale native-versus-web comparison instructions. Explanations cover units,
signed readings, changing one variable, pause/restart, captures and reopening.
Replacement uses existing save/recovery confirmation; cancellation preserves
both the circuit and the current activity. No telemetry or learning scores.

Internal review checked lesson values against the actual examples, exact UI
labels, safe text rendering, focus, replacement cancellation, no automatic
completion claims, and a current electronics-novice context in AGENTS.md.
References and human-research limits are in [activity guidance](../../learning-activities.md).
All frontend checks and three real-engine browser journeys pass, including a
resistor edit that reduces solved LED current and a save/reload of that edit.
Logs: `build/web-remaining-learning-frontend.log`, `build/learning-browser.log`.
No human pilot or human acceptance is claimed; the owner explicitly directed
engineering work without depending on recruitment or their circuit expertise.

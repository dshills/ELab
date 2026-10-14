# Background initialization

General networks and NPN pulse phase caches now initialize outside the shared service mutex. The immediate command response reports Updating; previous readings/trace remain explicitly stale until the captured calculation completes. `simulation.wait` observes readiness. Native paused restart shows Initializing · paused and publishes a fresh initial sample without starting its clock.

Live initialization and Verify share a private `simulationKernel` and compact `kernelPlan`. That plan contains the compiled circuit, settings, mode, run and electrical revision; it does not depend on recording metadata. Models, initial conditions, seed order and numerical tolerances remain unchanged. Network Newton and seed iteration check cancellation. Startup has a five-second wall budget; an expired current calculation reports a diagnostic rather than remaining Updating.

Publication requires the same run/electrical revision and an uncanceled context. Navigation, save and presentation edits remain available and survive valid publication. Undo, source/configuration edits, project replacement, Pause and Close prevent old results or failures from replacing current work. Pause cancels pending startup; Restart explicitly computes fresh results while respecting the paused preference. Repeated Resume on an already running/updating session keeps its current worker.

Restart now also refreshes paused LED and NPN DC circuits. Previously those paths marked their old result stale and returned without starting a new calculation. Paused electrical edits still retain stale readings until explicit Restart/Resume; the change does not silently calculate every paused edit.

## Client behavior

Use the shared commands normally, then inspect `status` or call `simulation.wait` before requiring fresh simulation data. Input controls and immediate `simulation.step.start` refuse while no fresh result exists. The waiting `simulation.step` compatibility command can await startup before admission; command preconditions remain necessary and avoid publication-only conflicts. Cached legacy Step replies bypass unrelated initialization, and malformed requests do not wait behind it.

Immediate mutation responses and their retry snapshots retain the original Updating admission. Query current state separately. Save stores committed graph/startup preferences, independent of a pending calculation; reopening computes fresh state from the saved configuration. Initialization is not a saved operation/physical history and never overwrites the graph, current actor or viewport on completion.

## Evidence and limits

Controlled blocked-startup race regressions cover flasher and NPN pulse admission, navigation/save/Pause, undo/source/close supersession, valid publication across geometry edits, repeated Resume, explicit timeout error, canceled Newton, malformed/cached legacy requests and fresh paused DC Restart. Native tests prove immediate Updating admission followed by fresh publication. Numerical fixtures and native/smoke helpers explicitly await startup only for final-result inspection; actual user callbacks stay immediate.

`scripts/initialization-demo.py` uses public commands for NPN DC, NPN pulse and the capacitor flasher. It checks immediate admission, readiness, stale paused edits, fresh paused Restart, legacy stepping and paused fresh-process save/open. Run after building with `--output REPORT --artifacts DIRECTORY`; `--connect` is only for an owned development session and replaces that project.

Input/wiper algebraic work and some file/recovery work still hold the service lock. Physical limits remain below the original 50-simulated-component / 100-node target. These changes do not establish complete visible latency, accessibility, clean-installation, participant or MVP acceptance. The [PRD audit](../research/11-prd-implementation-audit.md) retains those gaps.

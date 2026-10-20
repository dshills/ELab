# Structured malformed-file diagnostic

October 20, 2026. The unchanged current source passes a finite diagnostic over six valid fixtures: all 18 library parts, LED DC/RC/NPN pulse recordings, LED sweep, and a project with both asset types. **20,028 structured mutations** exercise every nested JSON path with twelve replacement/deletion modes. Of these, 19,142 reject; 886 accepted inputs remain stable across decode/encode/decode. Valid mutations are allowed rather than misclassified as failures.

Eighteen rejected nested inputs pass through the public in-process Open/recording-import/sweep-import commands. Full dirty project/physics/history state survives, allowing only the documented failed-job bookkeeping; the rejected source bytes and original manual save stay exact. This does not claim new physical UI or external-agent-adapter execution.

A separate **30-second coverage-guided Go fuzz pass completes 24,200 executions**, retaining 475 new synthetic corpus inputs. The first standalone-binary run explicitly lacks coverage guidance: its 175,612 passing executions remain unguided random evidence, unpooled. Original warning/log and a pre-run guided supplement survive.

The first full race pass reaches its 120-second diagnostic timeout during active nested JSON decoding, with no data-race report observed. One separately declared run of the **same full test and unchanged harness** with a 300-second test deadline passes in **195.582 seconds**. No reduced cases or product-limit change. Vet and formatting pass. The original timeout remains a failure in the record.

No product defect is confirmed or repaired. The harness uses an isolated Go overlay and is archived as `.go.txt.gz`; no test source is added to application package discovery. All 659 application Go hashes, current release receipt and canonical app stay exact; no app, SDK, model, command, format, numerical limit or release change. No repeat of completed LED/RC/NPN timing blocks.

[Evidence and frozen harness](../docs/validation/decoder-corruption-185/README.md), [results](../docs/validation/decoder-corruption-185/structured-result.json.gz), [terminal outcomes](../docs/validation/decoder-corruption-185/terminal-results.json.gz). The metadata/source/corpus replay does not execute tests or rerun fuzzing. Missing local tooling can be restored; originals must not be overwritten by a reproduction.

The [delivery plan](34-first-version-plan.md) now puts current status and next actions first. Its dated checkpoints move verbatim into a separate [engineering history](34-engineering-checkpoint-history.md), preserving their original scopes. Owner walkthrough, actual hobbyist study, complete bindings/accessibility/environment, broader corruption/power-loss and overall PRD acceptance remain open. P0 stays 17 implemented /10 partial; no promotion or MVP-complete claim.

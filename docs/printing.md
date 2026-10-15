# Printing a captured schematic

Open **Print preview**, then **Refresh printers** in the left rail. Choose a printer and 1–10 copies, then **Print all N pages**. Printing uses the preview's captured source, measurements and page settings even when later edits make the preview out of date. Refresh the preview first to print those edits. A4/Letter, orientation, margin, fit/explicit scale and color/grayscale use the existing renderer. Incomplete drawings can print. The source circuit, live time/charge, history and dirty state stay unchanged.

The macOS cgo backend connects to the local CUPS service using system libcups. It requests one-sided PDF printing, selected media/orientation/copies, no banner, and no additional spooler scaling. Device imageable margins, firmware scaling and actual paper output still require device qualification. There is no remote scheduler configuration or interactive authentication prompt; printer/service rejection is observable. Configure printers and credentials in macOS first. Remembered destinations can appear in the list; only those reporting that they accept jobs can be submitted.

Submission acknowledgment is **submitted**, with a system `spool_id`, PDF byte count/digest, actor and captured page/run/document identity. **Refresh job** reads the system state: pending, held, printing, stopped, canceled, aborted or completed. A spooler completion report is distinct from checking physical paper. If the system has expired a job, ELab reports `spool-job-unavailable` and retains its last observed state. It never assumes a missing job printed successfully.

**Cancel print job** sends cancellation for a job retained by this ELab session, records an acknowledged request and reads its state. It cannot retract pages already printed. **Cancel preparation** cancels the current discovery/render/control work. Cancellation before spooling publishes no job. If a submission acknowledgment arrives during cancellation or app Close, the job ID remains recorded as submitted; local cancellation cannot prove that the printer stopped. Missing acknowledgment produces **unknown** / `submission-unknown`. Check the macOS print queue before submitting again; ELab never automatically retries that submission.

## Shared commands

| Command | Input and result |
| --- | --- |
| `print.printers.refresh` | `{}`; asynchronously updates destination IDs, names, default flag, accepting flag and state. Never submits. Failure/cancellation preserves the prior inventory. A removed selection clears. |
| `print.destination.set` | Required `{printer_id,copies}`; session controls only. Empty printer ID clears selection. Does not submit. |
| `print.submit` | Required `{preview_id,printer_id,copies}`; exact visible completed preview and listed accepting printer. Admits one captured submission. |
| `print.job.refresh` | Required `{id}` naming an ELab-retained acknowledged job. Starts a system state read. |
| `print.job.cancel` | Required `{id}` naming a retained nonterminal acknowledged job. Starts a cancellation request and state read. Arbitrary system job numbers are not accepted. |
| `print.work.cancel` | Required exact current work `{id}`; cooperative cancellation. Observe until terminal. |

Inspect `project.inspect` / `events.poll` → `printing.work` and `printing.jobs`. All six commands support detailed `capabilities.list {command:...}` contracts, common revision/document guards and ordinary retry retention. Retrying an accepted request returns the original admission; inspect current state to observe its outcome. `simulation.wait` does not await printing. Publication uses the `printing` / `jobs` topics without advancing command revision.

One print discovery/preparation/control worker runs at a time. Its context has a cooperative 30-second deadline; libcups connection and I/O calls use 2-second connection / 5-second I/O timeouts and observe cancellation at call boundaries. These are not preemptive total-time guarantees. A prepared PDF is limited to 8 MiB in a private temporary directory/file and removed after the spool call. At most eight print jobs are retained; terminal jobs may be evicted for a new one. Unresolved/unknown jobs are protected from automatic eviction. Print jobs are session records, not saved project assets. New/source edits do not alter an already captured submission. Close cancels and joins local work; already submitted system jobs continue unless explicitly canceled.

Tests cover pinned preview source despite later edits, private PDF cleanup, exact render bytes/options/provenance, retry and snapshot ownership, live-state preservation, cancellation before/during spooling, unknown acknowledgment, failure/refusal purity and Close joining. An isolated Unix-socket IPP fixture exercises **actual libcups** discovery/submission/status/cancellation and exact PDF delivery without changing system queues or using a physical printer. Native tests exercise shared copy controls, disabled destination behavior and fractional-value refusal. Physical printer, media/margin/scale and end-to-end device cancellation acceptance remain open.

Implementation references: [CUPS programming manual](https://www.cups.org/doc/cupspm.html) for destination attributes and system job APIs, [CUPS job options](https://www.cups.org/doc/man-lp.html) for media/orientation/copies/sides, and [IPP model and semantics](https://www.rfc-editor.org/rfc/rfc8011.html) for job states. These references describe the platform protocol; they do not qualify physical output from ELab.

Tiled previews print the entire captured grid in one PDF job. `print.submit` never restricts submission to the currently shown page; the UI says **Print all N pages**, and each retained job includes `page_count`. Selected preview index is cleared for PDF rendering. Existing local CUPS options request one-sided output and `print-scaling:none`; printer/driver and paper acceptance still require physical device testing. [Tiled pages](tiled-output.md).

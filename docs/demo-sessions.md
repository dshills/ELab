# Connected demonstration sessions

The executable `elab-agent --connect` and the reusable Python `Client` helpers can inspect and control existing projects. Standalone demonstration programs are destructive test fixtures: they load/reset examples, alter values, save artifacts, and some close the supplied host. Their `--connect` argument now requires a **fresh owned blank document** before that demonstration can start.

Use a process you created for the test, with private recovery/appearance paths. An empty drawing alone is insufficient. Admission refuses a saved empty file, document edits/preferences/recordings, history, retained operations, pending work, previews, or a replaced/cleared document generation. Completed initial recovery configuration on an otherwise fresh document is allowed; active setup, failed setup, and file/recovery document work still refuse. This recognizes normal native startup without treating a restore or save as an empty experiment.

The guard is a read-only `project.inspect` check. On refusal, only the forwarding client is closed. The host receives no mutation or `app.close`; the demo's destructive cleanup is never entered. Raw-protocol LED/drawing demos perform the same check before their first mutation and only close their forwarding process on refusal. Recovery demos admit the target before starting a separate owned producer. No override bypasses this policy.

The check is **not an ownership credential or exclusive editing lease**. The caller must own the disposable process and must not run these demos against a person’s active project or concurrently edited session. Artifact directories may be created and an explicitly supplied legacy fixture may be read before remote admission; those existing files are not modified by refusal. Starting without `--connect` creates the demo's own headless process. Existing successful cleanup and `--keep-open` behavior remain.

The common persistence client now handles unexpected revision conflicts with at most eight mutation attempts. It inspects current guards and retains the exact original request ID, retry key, command and parameters. An intentionally expected `conflict` remains visible; other errors and EOF do not trigger retries. This is test-client behavior—the application's strict revision/document admission and successful retry retention are unchanged. Demo-only admission imports live in `main`, so the general client library remains importable independently.

`./scripts/check.sh` requires Python 3 in addition to Go and runs the standard-library Python admission/retry tests alongside the existing formatting, Staticcheck, root/native race tests and vet. To repeat the complete connected refusal audit with new output names:

```sh
python3 scripts/demo-session-audit.py \
  --artifacts build/demo-refusal-assets \
  --output build/demo-refusal-report.json \
  --native-binary build/ELab.app/Contents/MacOS/ELab
```

The audit creates its own hosts and an RLC fixture with a saved capture, advanced paused physics, unsaved edits and history. It discovers every CLI declaring `--connect`, runs its refusal, compares **all public snapshot fields**, checks host liveness and saved bytes, then re-exports the unchanged captured JSON. Native comparisons wait for actual canvas registration, preserving view/preferences in the comparison rather than excluding them. Owner cleanup closes only those audit processes after verification.

Current evidence also covers saved-empty, history-only empty and cleared-empty fixtures, including optimized-Python refusal and unrestricted general observers; successful LED/drawing, NMOS, PMOS, CMOS, LC and crash/recovery workflows run headlessly and natively. This is semantic API/process evidence, not physical keyboard/pointer/screen-reader or every historical demo's successful path. [Validation record](validation/demo-session-validation.json). The application bundle and circuit models are unchanged by this tooling checkpoint.

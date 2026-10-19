# Exact /165 macOS LaunchServices smoke launch

macOS `open -n -W -g -a` launched the exact ZIP-extracted /165 app bundle from its Unicode path with spaces. The observed native PID/command names that exact executable and the fresh owned endpoint. The smoke success event appears once, and all 50 stage-journal markers match the already-qualified exact copied native smoke log in order. These are log markers, not 50 independent tests. The run completes in 73.662 s; this single background launch is not a startup or visible-p95 benchmark.

The launcher exits 0, the owned native PID is absent, its endpoint is removed, and every original extracted file/mode remains unchanged. No forced cleanup was used. **The target native process exit status is unavailable:** LaunchServices owns that process, and `open -W` status is not its exit status. The success event, complete stage journal, process absence and cleanup are separate observations; no target status is inferred.

This runs `--smoke --smoke-timeout 2m`, which uses private temporary recovery/appearance storage and disables window-geometry saving. It does not exercise ordinary no-argument startup, default user configuration or persistence. The LaunchServices child is not claimed to inherit the earlier sandbox denial policy. Existing copied-runtime denial controls retain their original qualification scope. No user project, canonical app or sealed review was replaced.

This extends the local archive rehearsal to OS app launching. It does not qualify Finder interaction/Archive Utility, document association, downloaded quarantine/Gatekeeper, signing/notarization, a clean account/OS, another machine, physical UI input or owner acceptance. Those gates remain open; the private alpha is not promoted. Endpoint credentials are excluded, and the original logs are losslessly compressed.

Run the read-only verifier:

```sh
python3 -B research/ui-audit/first-version/verify-launchservices.py
```

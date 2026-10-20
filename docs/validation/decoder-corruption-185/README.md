# Structured decoder corruption diagnostic · current source

October 20, 2026. The unchanged current source passes **20,028 structured mutations across six valid fixtures**: 19,142 reject; 886 accepted inputs remain stable across decode/encode/decode. Eighteen rejected nested inputs also pass through the public in-process `project.load`, `verification.import` or `sweep.import` command, preserving the dirty full project/physics/history snapshot and all source/manual-save bytes. This is a bounded robustness result, not exhaustive malformed-file safety or native/agent-adapter qualification.

Fixtures cover an incomplete drawing with all 18 library parts, LED DC recording, RC transient recording, NPN pulse recording, LED DC sweep, and a project with both a recording and a sweep. Every nested JSON path receives twelve fixed replacement/deletion modes. Valid alternatives may be accepted; the oracle requires their decoder round trip to remain stable, rather than incorrectly demanding that every mutation reject. It does not authenticate imported history or certify recorded physics.

The separate coverage-guided Go fuzz run passes **24,200 executions in a 30-second budget**, retaining **475 new corpus inputs /30,598 bytes**. It uses 72 initial seeds, two workers, structure-aware path mutation plus raw-payload decoding, a 4,096-byte payload bound and 256-byte replacement-string bound. Corpus inputs are synthetic untrusted test data, not instructions. [Original corpus bytes](guided-corpus.json.gz).

Two method failures remain explicit:

- The first standalone test binary warned that it lacked coverage instrumentation. Its 175,612 executions pass as **unguided random evidence only**, with no coverage-guidance claim or pooled count. A separate declaration precedes the standard `go test -fuzz` invocation, which reports gathering baseline coverage and new interesting inputs.
- The initial full race pass exits 1 at its 120-second diagnostic timeout during active nested JSON decoding, with no data-race report observed. Its stack/log survive. One separately declared run of the **same full test and unchanged harness**, with a 300-second test deadline and no competing fuzz job, passes in **195.582 seconds**. No case reduction, product tolerance/budget change or timeout concealment. Go vet and gofmt checks pass.

[Terminal handles/outcomes](terminal-results.json.gz), [initial declaration](declaration.json.gz), [guided supplement](guided-declaration.json.gz), [race supplement](race-final-declaration.json.gz), [fixed-pass result](structured-result.json.gz), [byte manifest](manifest.json).

The diagnostic is compiled through an isolated Go overlay. Its source is archived as `.go.txt.gz`; the virtual test file never exists in the repository package. All **659 application Go hashes**, the current release receipt and canonical app stay exact. No application, SDK, model, command, schema/format, numerical limit, UI input, new release or promotion. Existing full release checks are not rerun or relabeled; this adds focused diagnostic race/vet/formatting evidence only. Current LED/RC/NPN timing blocks remain complete and unmodified.

Replay checks metadata integrity, exact source/tool identities, retained inputs, log/results consistency and corpus bytes. It does **not** rerun the fuzz campaign, execute tests, open an app, create files or claim broader gates:

```sh
python3 -B docs/validation/decoder-corruption-185/verify.py
```

To reproduce the actual diagnostic in a new private work folder, use the frozen harness/overlay and original fixtures with current source. Do not overwrite these original logs, declarations or corpus. [Harness](diagnostic_test.go.txt.gz), [overlay](overlay.json.gz), [source/artifact pins](artifacts.json). Owner walkthrough, actual study, broader corruption/power-loss, complete bindings/accessibility/environment and overall MVP acceptance remain open.

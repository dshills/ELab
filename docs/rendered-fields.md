# Rendered field context

Committed native field/selector callbacks now carry the command revision and document generation captured by their view. They submit those guards through the same public dispatcher that agents use. They cannot acquire a fresh revision at commit time and apply an older set of parameters over a newer actor's edit.

For example, a sine frequency field captures the complete waveform. If an agent changes amplitude before that field commits, the old frequency callback refuses rather than restoring the older amplitude. A fresh field still changes frequency while preserving the agent's amplitude. The same binding applies to value drafts, numeric settings, slider commits, probe/object selection, time fields and other `ui.Node.Change` callbacks in the assembled view.

Bindings are installed before macOS selector adaptation. The private native choice stores the already guarded callback; asynchronous selection retains document context even when command revision alone still matches. Callback context runs on the owner and is restored on return, including refused commands and nested callbacks. Ordinary solver publication does not by itself change command revision or document identity.

The dispatcher evaluates guards atomically. A stale commit returns the existing `conflict` response and preserves circuit/history/physics. It does not silently retry with current state. After the view refreshes, a current committed edit proceeds normally. Partial typing/IME and slider previews remain native editor/control behavior; this change concerns their committed semantic callbacks, not a new per-keystroke solver path.

The [release receipt](validation/field-context-validation.json) records stale waveform/draft refusal, fresh value preservation, document guard isolation, native selector retention and owner-context cleanup. Existing unit-text and native slider-gesture regressions still pass. Exact packaged smoke executes stale and fresh waveform callbacks in a real app owner alongside the established GPU checks.

Full mounted typing/IME/focus behavior, every raw Activate/dialog/gesture path, OS lifecycle interaction and accessibility remain separate gates. The [source inventory](../research/13-native-command-inventory.md) records the guard-binding site and unresolved source routes; it is not runtime coverage proof for every generated control.

Scope plot input is now separately bound to [prepared scope context](scope-input-context.md), including interval/cursor identity and optional recorded capture guards. This does not turn field/callback tests into complete physical gesture or OS display-completion qualification.

## Settling ordinary value entry before Apply

The unreleased component inspector uses Vellum's asynchronous `SettleTextField` barrier for numeric values and net-label names. Clicking Apply can precede the native field's final Change. Apply now waits, then uses the accepted draft through the same `component.set` command available to agents. Partial typing remains editor behavior; simulation changes on the ordinary committed command.

The action retains its rendered document/revision. Only its own successful `editor.value.set` result can advance that revision; it never refreshes from an unrelated snapshot. An agent edit before or after that commit, or a document replacement, refuses the obsolete action. Failed settlement/draft admission does not apply an old value, and duplicate callbacks do not replay it. The pending Apply control is disabled until completion. Unmounted semantic tests already have controlled Change values and complete synchronously.

The mounted smoke harness now waits for the controlled draft to reach the actual form and for asynchronous Apply to complete. These are callback/mounted tests; [physical editing evidence](../research/ui-audit/value-apply/README.md) comes from CUA on an isolated pointer-coordinate prototype. Shared Vellum and the canonical application remain unchanged; dependency, full input/IME and accessibility qualification remain required.

[Scoped validation and retained initial failures](validation/value-apply-validation.json) bind the final 45-stage native smoke, race/vet/lint checks, source hashes and dependency context.

## Settling coordinates before Place

Place now waits for explicit `SettleNumberField` barriers for X/Y, followed by label settlement when needed. Accessibility activation does not have to move pointer focus to commit a numeric draft. The final component still uses the shared guarded `component.place` command and only the coordinates accepted from its own field responses.

An agent edit, another field's change, document replacement or failed settlement refuses the pending placement. Duplicate completions cannot skip a field or place twice. The SDK keeps spinners on their legacy input path; the barrier orders and validates commitment without turning them into managed Cocoa text clients.

[Mounted regressions and actual native accessibility/pointer replay](validation/number-field-placement/README.md) verify the repaired source and independent agent parity. Batch smoke setup uses already accepted public coordinates before widgets mount; it does not qualify private input. Clean replacement packaging, exhaustive reader/IME and other unresolved input gates remain separate.

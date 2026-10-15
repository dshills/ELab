# Rendered field context

Committed native field/selector callbacks now carry the command revision and document generation captured by their view. They submit those guards through the same public dispatcher that agents use. They cannot acquire a fresh revision at commit time and apply an older set of parameters over a newer actor's edit.

For example, a sine frequency field captures the complete waveform. If an agent changes amplitude before that field commits, the old frequency callback refuses rather than restoring the older amplitude. A fresh field still changes frequency while preserving the agent's amplitude. The same binding applies to value drafts, numeric settings, slider commits, probe/object selection, time fields and other `ui.Node.Change` callbacks in the assembled view.

Bindings are installed before macOS selector adaptation. The private native choice stores the already guarded callback; asynchronous selection retains document context even when command revision alone still matches. Callback context runs on the owner and is restored on return, including refused commands and nested callbacks. Ordinary solver publication does not by itself change command revision or document identity.

The dispatcher evaluates guards atomically. A stale commit returns the existing `conflict` response and preserves circuit/history/physics. It does not silently retry with current state. After the view refreshes, a current committed edit proceeds normally. Partial typing/IME and slider previews remain native editor/control behavior; this change concerns their committed semantic callbacks, not a new per-keystroke solver path.

The [release receipt](validation/field-context-validation.json) records stale waveform/draft refusal, fresh value preservation, document guard isolation, native selector retention and owner-context cleanup. Existing unit-text and native slider-gesture regressions still pass. Exact packaged smoke executes stale and fresh waveform callbacks in a real app owner alongside the established GPU checks.

Full mounted typing/IME/focus behavior, every raw Activate/dialog/gesture path, OS lifecycle interaction and accessibility remain separate gates. The [source inventory](../research/13-native-command-inventory.md) records the guard-binding site and unresolved source routes; it is not runtime coverage proof for every generated control.

Scope plot input is now separately bound to [prepared scope context](scope-input-context.md), including interval/cursor identity and optional recorded capture guards. This does not turn field/callback tests into complete physical gesture or OS display-completion qualification.

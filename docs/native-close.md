# Closing a circuit safely

**File → Close window** and **Cmd+W** use the same native close policy as a window-close request. That policy submits the public `app.close` command. A clean session closes; unsaved schematic, preference or kept-record changes open the existing Save / Discard / Cancel decision. Cancel preserves the circuit. Discard explicitly admits close. Save completes and verifies its association/current clean state before continuing.

Repeated close requests do not replace another pending decision. A close request while a native picker or file operation is active leaves that operation in place; finish or cancel it, then close again. This is UI admission behavior, not a new requirement that headless agents open a picker. Agents use the same close command directly and receive structured `unsaved-changes`, with explicit discard and optional generation guards:

```json
{"command":"app.close","params":{"discard_changes":true,"expected_document_generation":7}}
```

The fragment also needs the ordinary transport version/request ID, revision guard and retry key. A successful semantic close marks the service closed. The native owner then tears down the application/window; it does not bypass the service's decision by closing widgets first.

The pending close retains its original document generation. If an agent replaces the document while the decision or Save picker is open, a later Discard or Save refuses; it cannot close or save the replacement on behalf of the old decision. The shared file worker also checks current association/dirty state before running a Save continuation, so later unsaved edits do not trigger automatic close.

Vellum's native Scene close gate consults `Window.CloseRequest`; the prepared multi-window quit path checks whether close requests actually removed all windows. Those framework paths were inspected read-only. Tests here exercise ELab's owner policy, File command, protected choices and actual packaged smoke callbacks. They do not claim physical title-bar click or OS Cmd+Q automation, a full screen-reader path, or all platform lifecycle behavior.

The [release receipt](validation/native-close-validation.json) records core/native tests, source-inventory refresh, public close semantics and packaged GPU smoke. Owned temporary test circuits cover clean close, dirty Cancel/Discard, Save-before-close, existing-decision/picker preservation and stale Save/Discard after agent replacement. Prior user windows and projects are preserved.

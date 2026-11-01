# Browser qualification

Development-only dependencies; the shipped app still needs neither Node nor npm.
The locked Playwright suite runs on GitHub's Linux runners with Chromium, Firefox
and WebKit. These are engine checks, not certification of installed Chrome or
Safari. Real macOS browser, VoiceOver and pointer testing remain manual gates.

The suite uses a separate temporary library, refuses an occupied port and never
reuses a running user host. It exercises keyboard tabs/dialogs, edits/undo,
save/reload/export, captured playback, narrow layouts and automated accessibility.
Failure reports include traces/screenshots of disposable test pages only. Video
recording is disabled. Serious/critical WCAG-tagged axe violations fail the gate;
all discovered violations are attached for inspection. Automation alone does not
establish accessibility conformance.

List tests without launching a server or browser:

```sh
npm --prefix exp-web/browser ci --ignore-scripts --no-audit --no-fund
npm --prefix exp-web/browser run list
```

When local browser launches are appropriate, explicitly opt in:

```sh
go build -o build/elab-web-browser ./exp-web
cd exp-web/browser
npx playwright install chromium firefox webkit
ELAB_BROWSER_TESTS=1 npm test
```

The process receives SIGTERM and saves/stops before its temporary library is
removed. Do not run these commands while desktop/browser testing is paused.

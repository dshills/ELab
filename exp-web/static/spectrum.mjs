import { escapeHTML as esc } from "./forms.mjs";
import { plotInstrument } from "./instrument-plot.mjs";
import { resultTable } from "./results.mjs";

export function spectrumView(state, source) {
  const live = source === "live",
    s = state.spectrum;
  const result = live ? s?.live?.result : s?.result;
  // A previous captured FFT must never impersonate a newly selected recording.
  const current =
    result && (live || result.settings.id === state.recording?.id)
      ? result
      : null;
  return {
    result: current,
    measurements: current
      ? live
        ? s?.live?.measurements
        : s?.measurements
      : null,
  };
}
export function spectrumEdit(values, state, source) {
  if (source === "recorded" && !state.recording?.id)
    throw new Error("Choose a completed recording before analyzing.");
  const common = {
    probe_id: values.probe_id,
    window: values.window,
    samples: Number(values.samples),
    fundamental_hz: values.fundamental_hz || "0",
    harmonics: Number(values.harmonics),
  };
  return source === "live"
    ? {
        name: "spectrum.live.set",
        params: { ...common, enabled: true, hold: false },
      }
    : {
        name: "spectrum.analyze",
        params: {
          ...common,
          id: state.recording?.id,
          start_s: values.start_s,
          end_s: values.end_s,
        },
      };
}

export function liveSpectrumToggle(state, action) {
  const cfg = state.spectrum?.live?.settings || {
    enabled: false,
    hold: false,
    window: "hann",
    samples: 0,
    harmonics: 5,
  };
  return {
    ...cfg,
    ...(action === "hold" ? { hold: !cfg.hold } : { enabled: !cfg.enabled }),
  };
}
export function spectrumWorkbench(api) {
  const root = document.getElementById("analysis-body");
  let source = "live",
    owner = "",
    domain = null,
    frame = null,
    formGuard,
    busy = false;
  const identity = () =>
    JSON.stringify([
      api.instance(),
      api.state().document.generation,
      source,
      source === "live"
        ? api.state().scope.selected
        : api.state().recording?.id,
    ]);
  const active = () => root.dataset.instrument === "spectrum";
  const choices = (options, value) =>
    options
      .map(
        (v) =>
          `<option value="${esc(v)}" ${String(v) === String(value) ? "selected" : ""}>${v === 0 ? "Automatic" : esc(v)}</option>`,
      )
      .join("");
  function build() {
    const state = api.state(),
      r = state.recording,
      live = source === "live";
    const cfg = live
      ? state.spectrum?.live?.settings || {}
      : state.spectrum?.settings?.id === r?.id
        ? state.spectrum.settings
        : {};
    const probes = live
      ? (state.scope.available || []).filter((p) =>
          state.scope.selected.includes(p.id),
        )
      : r?.scope.available || [];
    const defaultProbe =
      cfg.probe_id || (live ? state.scope.selected[0] : probes[0]?.id);
    root.innerHTML = `<div class="analysis-source"><label>Spectrum source<select id="spectrum-source"><option value="live" ${live ? "selected" : ""}>Live circuit</option><option value="recorded" ${!live ? "selected" : ""}>Recorded experiment</option></select></label>${
      !live
        ? `<label>Capture<select id="spectrum-capture">${(
            state.operations || []
          )
            .filter((o) => o.status === "completed")
            .map(
              (o) =>
                `<option value="${esc(o.id)}" ${o.id === r?.id ? "selected" : ""}>${esc(o.id)}</option>`,
            )
            .join("")}</select></label>`
        : '<button type="button" data-spectrum="hold"></button><button type="button" data-spectrum="enable"></button>'
    }</div><form id="spectrum-settings" class="analysis-toolbar"><label>Measurement<select name="probe_id" required>${probes.map((p) => `<option value="${esc(p.id)}" ${p.id === defaultProbe ? "selected" : ""}>${esc(p.label)} · ${esc(p.unit)}</option>`).join("")}</select></label><label>FFT window<select name="window">${choices(["hann", "rectangular"], cfg.window || "hann")}</select></label><button class="primary">${live ? "Apply live FFT" : "Analyze recording"}</button><details class="instrument-settings"><summary>FFT grid, interval & distortion</summary><div class="analysis-toolbar"><label>FFT samples<select name="samples">${choices(live ? [0, 32, 64, 128, 256] : [0, 32, 64, 128, 256, 512, 1024, 2048, 4096], cfg.samples || 0)}</select></label>${!live ? `<label>Start time<input name="start_s" value="${cfg.start_s || 0}" required></label><label>End time<input name="end_s" value="${cfg.end_s || 0}" required></label>` : ""}<label>Fundamental<input name="fundamental_hz" value="${esc(cfg.fundamental_hz || 0)}" required></label><label>Harmonics<select name="harmonics">${choices([2, 3, 4, 5, 6, 7, 8, 9, 10], cfg.harmonics || 5)}</select></label><p class="field-note">Fundamental accepts Hz/kHz; 0 leaves distortion unset. Recorded end time 0 uses the capture end. FFT values are native bins with mean removed and window gain corrected.</p></div></details><p class="settings-feedback error" role="status"></p></form><p id="spectrum-status" class="analysis-summary" role="status"></p><canvas id="instrument-chart" class="instrument-chart" aria-label="Frequency spectrum; click to place the active frequency cursor" tabindex="0"></canvas><div class="analysis-cursors"><button data-spectrum="a">A</button><button data-spectrum="b">B</button><form id="spectrum-cursor" class="analysis-toolbar"><label>Exact frequency<input name="frequency_hz" value="0" required></label><button>Set frequency cursor</button></form><button data-spectrum="clear">Clear cursors</button></div><div id="spectrum-readings"></div><div id="spectrum-peaks" class="analysis-peaks"></div><div class="analysis-export"><button data-spectrum="json">Download FFT JSON</button><button data-spectrum="csv">Download FFT CSV</button></div><details><summary>Distortion, harmonics & assumptions</summary><div id="spectrum-details"></div></details>`;
    formGuard = api.guards();
    domain = null;
    frame = null;
  }
  function update() {
    if (!active()) return;
    if (owner !== identity()) {
      owner = identity();
      build();
    }
    const state = api.state(),
      { result, measurements: m } = spectrumView(state, source),
      live = state.spectrum?.live;
    const status = root.querySelector("#spectrum-status");
    document.getElementById("scope-origin").textContent =
      source === "live" ? "LIVE FFT" : "RECORDED FFT";
    status.textContent = result
      ? `${source === "live" ? "Live FFT" : "Captured FFT · " + result.settings.id} · ${result.probe.label} · ${api.fmt(result.resolution_hz, "Hz")} resolution · ${api.fmt(result.nyquist_hz, "Hz")} Nyquist${source === "live" ? " · " + live.status : ""}`
      : source === "live"
        ? live?.reason || live?.status || "Apply live FFT to collect a frame."
        : "Choose a completed capture and analyze its stored samples.";
    for (const button of root.querySelectorAll("button"))
      button.disabled = busy;
    for (const input of root.querySelectorAll("input,select"))
      input.disabled = busy;
    for (const key of ["a", "b"]) {
      const button = root.querySelector(`[data-spectrum="${key}"]`);
      const selected = (m?.active_cursor || "a") === key;
      button.classList.toggle("active", selected);
      button.setAttribute("aria-pressed", String(selected));
    }
    for (const key of ["json", "csv"])
      root.querySelector(`[data-spectrum="${key}"]`).disabled =
        busy || source === "live" || !result;
    root.querySelector("#spectrum-settings button").disabled =
      busy || (source === "recorded" && !state.recording?.id);
    if (source === "live") {
      root.querySelector('[data-spectrum="hold"]').disabled =
        busy || !live?.settings.enabled;
      root.querySelector('[data-spectrum="hold"]').textContent = live?.settings
        .hold
        ? "Resume FFT"
        : "Hold FFT";
      root.querySelector('[data-spectrum="enable"]').textContent = live
        ?.settings.enabled
        ? "Turn FFT off"
        : "Turn FFT on";
    }
    root.querySelector("#spectrum-readings").innerHTML = m
      ? `${["a", "b"]
          .map((c) => {
            const v = m[c];
            return `<span><strong>${c.toUpperCase()}</strong> ${esc(v?.valid ? `${api.fmt(v.reading.frequency_hz, "Hz")} · ${api.fmt(v.reading.peak, m.unit)} · ${api.fmt(v.reading.db_reference_1_unit, "dB")}` : v?.reason || "Not set")}</span>`;
          })
          .join(
            "",
          )}<span>Δf ${esc(api.fmt(m.delta_hz, "Hz"))} · Δ ${esc(api.fmt(m.delta_db, "dB"))}</span>`
      : "";
    const peaks = root.querySelector("#spectrum-peaks"),
      html = (m?.peaks || [])
        .map(
          (p) =>
            `<button data-spectrum-peak="${p.rank}" title="Place active cursor at this native FFT bin">Peak ${p.rank} · ${esc(api.fmt(p.reading.frequency_hz, "Hz"))} · ${esc(api.fmt(p.reading.peak, m.unit))}</button>`,
        )
        .join("");
    if (peaks.innerHTML !== html) peaks.innerHTML = html;
    const details = root.querySelector("#spectrum-details"),
      detail = result
        ? resultTable({
            thd_percent: result.thd_percent,
            reason: result.thd_unavailable_reason,
            harmonics: result.harmonics,
            assumptions: result.assumptions,
          })
        : "";
    if (details.innerHTML !== detail) details.innerHTML = detail;
    frame = result;
    domain = plotInstrument(
      api,
      root.querySelector("#instrument-chart"),
      (result?.bins || []).map((p) => ({
        x: p.frequency_hz,
        y: p.db_reference_1_unit,
      })),
      {
        xUnit: "Hz",
        yUnit: "dB",
        markers: ["a", "b"]
          .filter((c) => m?.[c]?.valid)
          .map((c) => ({
            x: m[c].reading.frequency_hz,
            label: c.toUpperCase(),
          })),
      },
    );
  }
  async function run(edit, guard = api.guards(), expected = owner) {
    if (busy)
      throw new Error("Wait for the current spectrum action to finish.");
    if (expected !== identity())
      throw new Error("Spectrum source changed. Review the current controls.");
    busy = true;
    update();
    try {
      await api.command(edit.name, edit.params, { guard });
      formGuard = api.guards();
    } finally {
      busy = false;
      update();
    }
  }
  async function cursor(params, guard = api.guards()) {
    if (busy) return;
    const { measurements: m } = spectrumView(api.state(), source);
    if (!frame) throw new Error("A current spectrum frame is required.");
    await run(
      {
        name: "spectrum.cursor.set",
        params: {
          source,
          cursor: m?.active_cursor || "a",
          result_id: frame.id,
          ...params,
        },
      },
      guard,
    );
  }
  root.addEventListener("change", async (e) => {
    if (!active()) return;
    try {
      if (e.target.id === "spectrum-source") {
        source = e.target.value;
        owner = "";
        update();
      }
      if (e.target.id === "spectrum-capture")
        await run({ name: "operation.select", params: { id: e.target.value } });
    } catch (err) {
      api.toast(err.message, true);
    }
  });
  root.addEventListener("submit", async (e) => {
    if (!active()) return;
    e.preventDefault();
    const form = e.target,
      feedback = form.querySelector(".settings-feedback");
    try {
      const values = Object.fromEntries(new FormData(form));
      if (form.id === "spectrum-settings")
        await run(spectrumEdit(values, api.state(), source), formGuard);
      if (form.id === "spectrum-cursor")
        await cursor(
          { frequency_hz: values.frequency_hz },
          form._guard || api.guards(),
        );
      if (feedback) {
        feedback.textContent = "Applied.";
        feedback.classList.remove("error");
      }
    } catch (err) {
      if (feedback) {
        feedback.textContent = err.message;
        feedback.classList.add("error");
      } else api.toast(err.message, true);
    }
  });
  root.addEventListener("focusin", (e) => {
    if (active() && e.target.closest("#spectrum-cursor"))
      e.target.closest("form")._guard = api.guards();
  });
  root.addEventListener("click", async (e) => {
    if (!active()) return;
    const b = e.target.closest("button");
    try {
      if (e.target.id === "instrument-chart" && domain)
        await cursor({ frequency_hz: domain.at(e.clientX) });
      if (b?.dataset.spectrumPeak)
        await cursor({ peak: Number(b.dataset.spectrumPeak) });
      const action = b?.dataset.spectrum;
      if (!action) return;
      if (action === "json" || action === "csv") {
        await api.download("spectrum-" + action);
        return;
      }
      if (action === "a" || action === "b")
        await cursor({ cursor: action, select: true });
      if (action === "clear") {
        const guard = api.guards(),
          expected = owner;
        await run(
          {
            name: "spectrum.cursor.set",
            params: { source, cursor: "a", clear: true },
          },
          guard,
          expected,
        );
        await run(
          {
            name: "spectrum.cursor.set",
            params: { source, cursor: "b", clear: true },
          },
          api.guards(),
          expected,
        );
      }
      if (action === "hold" || action === "enable") {
        await run({
          name: "spectrum.live.set",
          params: liveSpectrumToggle(api.state(), action),
        });
      }
    } catch (err) {
      api.toast(err.message, true);
    }
  });
  return {
    show() {
      root.dataset.instrument = "spectrum";
      owner = "";
      update();
    },
    update,
  };
}

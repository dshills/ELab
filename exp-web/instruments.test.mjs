import test from "node:test";
import assert from "node:assert/strict";
import {
  spectrumView,
  spectrumEdit,
  liveSpectrumToggle,
} from "./static/spectrum.mjs";

test("recorded FFT never falls back to a live frame or a different captured source", () => {
  const result = { id: "fft-1", settings: { id: "capture-a" } };
  const state = {
    recording: { id: "capture-b" },
    spectrum: { result, live: { result: { id: "live-1" } } },
  };
  assert.equal(spectrumView(state, "recorded").result, null);
  assert.equal(
    spectrumView({ ...state, recording: { id: "capture-a" } }, "recorded")
      .result,
    result,
  );
  assert.equal(
    spectrumView({ ...state, spectrum: { result } }, "live").result,
    null,
  );
  assert.equal(spectrumView(state, "live").result.id, "live-1");
  assert.equal(spectrumView({}, "recorded").result, null);
  assert.equal(spectrumView({}, "live").result, null);
});
test("FFT requests retain native unit text and keep recorded and live settings separate", () => {
  const values = {
    probe_id: "v:C1",
    window: "hann",
    samples: "64",
    fundamental_hz: "1 kHz",
    harmonics: "5",
    start_s: "10 ms",
    end_s: "20 ms",
  };
  const recorded = spectrumEdit(
    values,
    { recording: { id: "capture-a" } },
    "recorded",
  );
  assert.deepEqual(recorded, {
    name: "spectrum.analyze",
    params: {
      id: "capture-a",
      probe_id: "v:C1",
      window: "hann",
      samples: 64,
      fundamental_hz: "1 kHz",
      harmonics: 5,
      start_s: "10 ms",
      end_s: "20 ms",
    },
  });
  const live = spectrumEdit(values, {}, "live");
  assert.deepEqual(live, {
    name: "spectrum.live.set",
    params: {
      enabled: true,
      hold: false,
      probe_id: "v:C1",
      window: "hann",
      samples: 64,
      fundamental_hz: "1 kHz",
      harmonics: 5,
    },
  });
  assert.throws(
    () => spectrumEdit(values, {}, "recorded"),
    /completed recording/,
  );
});

test("first live FFT toggle enables the engine and later toggles preserve frame settings", () => {
  assert.equal(liveSpectrumToggle({}, "enable").enabled, true);
  const state = {
    spectrum: {
      live: {
        settings: {
          enabled: true,
          hold: true,
          probe_id: "v:C1",
          window: "rectangular",
          samples: 64,
          harmonics: 3,
        },
      },
    },
  };
  const off = liveSpectrumToggle(state, "enable");
  assert.equal(off.enabled, false);
  assert.equal(off.probe_id, "v:C1");
  assert.equal(off.samples, 64);
  assert.equal(liveSpectrumToggle(state, "hold").hold, false);
  assert.equal(state.spectrum.live.settings.enabled, true);
});

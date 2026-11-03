import test from "node:test";
import assert from "node:assert/strict";
import { plotInstrument } from "./static/instrument-plot.mjs";

test("large plots avoid argument limits and all emitted coordinates remain finite", () => {
  const savedDocument = globalThis.document,
    savedStyle = globalThis.getComputedStyle;
  globalThis.document = { body: {} };
  globalThis.getComputedStyle = () => ({ getPropertyValue: () => "" });
  const ctx = new Proxy(
    {},
    {
      get(target, key) {
        return (
          target[key] ??
          ((...args) => {
            for (const v of args) {
              if (typeof v === "number")
                assert.ok(
                  Number.isFinite(v),
                  `${String(key)} received non-finite coordinate ${v}`,
                );
            }
            if (key === "fillText") assert.notEqual(args[0], undefined);
          })
        );
      },
    },
  );
  const api = { prepareCanvas: () => ({ ctx, w: 800, h: 180 }), fmt: String };
  let edgeLabel;
  ctx.measureText = (text) => ({ width: String(text).length * 6 });
  ctx.fillText = (text, x) => {
    assert.notEqual(text, undefined);
    if (text === "B") edgeLabel = x;
  };
  const canvas = { getBoundingClientRect: () => ({ left: 0 }) };
  try {
    const domain = plotInstrument(
      api,
      canvas,
      Array.from({ length: 150000 }, (_, x) => ({ x, y: Math.sin(x) })),
      { markers: [{ x: 75000 }, { x: 149999, label: "B" }] },
    );
    assert.equal(domain.at(66), 0);
    assert.equal(domain.at(772), 149999);
    assert.ok(
      edgeLabel >= 66 && edgeLabel + 6 <= 772,
      "right-edge cursor label remains inside the plot",
    );
    assert.ok(
      plotInstrument(
        api,
        canvas,
        [
          { x: 1, y: 1 },
          { x: 1 + Number.EPSILON, y: 2 },
        ],
        { log: true },
      ),
    );
    assert.equal(
      plotInstrument(api, canvas, [
        { x: -Number.MAX_VALUE, y: 1 },
        { x: Number.MAX_VALUE, y: 2 },
      ]),
      null,
    );
  } finally {
    globalThis.document = savedDocument;
    globalThis.getComputedStyle = savedStyle;
  }
});

export function plotInstrument(
  api,
  canvas,
  points,
  { xUnit = "", yUnit = "", log = false, markers = [] } = {},
) {
  const surface = api.prepareCanvas(canvas);
  if (!surface) return null;
  const { ctx, w, h } = surface;
  ctx.clearRect(0, 0, w, h);
  let minX = Infinity,
    maxX = -Infinity,
    minY = Infinity,
    maxY = -Infinity;
  for (const p of points) {
    if (!Number.isFinite(p.x) || !Number.isFinite(p.y) || (log && p.x <= 0))
      continue;
    minX = Math.min(minX, p.x);
    maxX = Math.max(maxX, p.x);
    minY = Math.min(minY, p.y);
    maxY = Math.max(maxY, p.y);
  }
  if (!Number.isFinite(minX)) return null;
  if (minX === maxX) maxX = minX + (Math.abs(minX) || 1) * 0.01;
  const span = maxY - minY || Math.abs(maxY) * 0.1 || 1;
  minY -= span * 0.08;
  maxY += span * 0.12;
  const xSpan = maxX - minX,
    ySpan = maxY - minY;
  const logSpan = log ? Math.log1p(xSpan / minX) : 0;
  if (
    !Number.isFinite(xSpan) ||
    xSpan <= 0 ||
    !Number.isFinite(ySpan) ||
    ySpan <= 0 ||
    (log && (!Number.isFinite(logSpan) || logSpan <= 0))
  )
    return null;
  const left = 66,
    right = w - 28,
    top = 16,
    bottom = h - 32,
    width = right - left,
    height = bottom - top;
  if (width <= 4 || height <= 0) return null;
  const fraction = (x) =>
    log ? Math.log1p((x - minX) / minX) / logSpan : (x - minX) / (maxX - minX);
  const px = (x) => left + fraction(x) * width,
    py = (y) => bottom - ((y - minY) / (maxY - minY)) * height;
  ctx.font = "11px system-ui";
  ctx.fillStyle = getComputedStyle(document.body).getPropertyValue("--muted");
  ctx.strokeStyle = "#8caa9833";
  ctx.lineWidth = 1;
  for (let i = 0; i < 4; i++) {
    const y = top + (height * i) / 3;
    ctx.beginPath();
    ctx.moveTo(left, y);
    ctx.lineTo(right, y);
    ctx.stroke();
    ctx.textAlign = "right";
    ctx.fillText(
      api.fmt(maxY - ((maxY - minY) * i) / 3, yUnit, 3),
      left - 7,
      y + 4,
    );
  }
  ctx.textAlign = "left";
  ctx.fillText(api.fmt(minX, xUnit), left, h - 9);
  ctx.textAlign = "right";
  ctx.fillText(api.fmt(maxX, xUnit), right, h - 9);
  ctx.save();
  ctx.beginPath();
  ctx.rect(left, top, width, height);
  ctx.clip();
  ctx.strokeStyle = getComputedStyle(document.body).getPropertyValue(
    "--accent",
  );
  ctx.lineWidth = 1.8;
  ctx.beginPath();
  let started = false;
  for (const p of points) {
    if (!Number.isFinite(p.x) || !Number.isFinite(p.y) || (log && p.x <= 0)) {
      started = false;
      continue;
    }
    if (started) ctx.lineTo(px(p.x), py(p.y));
    else {
      ctx.moveTo(px(p.x), py(p.y));
      started = true;
    }
  }
  ctx.stroke();
  for (const m of markers) {
    if (!Number.isFinite(m.x) || (log && m.x <= 0)) continue;
    ctx.strokeStyle = m.color || "#b98b51";
    ctx.fillStyle = ctx.strokeStyle;
    ctx.setLineDash([4, 3]);
    ctx.beginPath();
    ctx.moveTo(px(m.x), top);
    ctx.lineTo(px(m.x), bottom);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.textAlign = "left";
    if (m.label != null) {
      const label = String(m.label),
        labelWidth = Math.min(ctx.measureText(label).width, width - 4);
      const labelX = Math.max(
        left + 2,
        Math.min(right - labelWidth - 2, px(m.x) + 4),
      );
      ctx.fillText(label, labelX, top + 12, width - 4);
    }
  }
  ctx.restore();
  return {
    at: (clientX) => {
      const r = canvas.getBoundingClientRect(),
        f = Math.max(0, Math.min(1, (clientX - r.left - left) / width));
      return log ? minX * Math.exp(f * logSpan) : minX + f * (maxX - minX);
    },
  };
}

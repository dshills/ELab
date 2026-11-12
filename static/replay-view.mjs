const escapeReplay = value => String(value ?? '').replace(/[&<>"']/g, c => ({
 '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
})[c]);

export function replayView(root, capture, initialIndex = 0) {
 const esc = escapeReplay;
 const drawing = root.querySelector('[data-replay-drawing]');
 const slider = root.querySelector('[data-replay-position]');
 const sheet = root.querySelector('[data-replay-sheet]');
 const probe = root.querySelector('[data-replay-probe]');
 const reading = root.querySelector('[data-replay-reading]');
 const waveform = root.querySelector('[data-replay-waveform]');
 const play = root.querySelector('[data-replay-play]');
 let index = 0, timer;
 const end = capture.rows.length - 1;
 const format = (value, unit) => `${Number(value.toPrecision(5))} ${unit}`;
 const time = value => format(value * 1000, 'ms');
 const referencePart = name => name?.split('.')[0];
 sheet.innerHTML = capture.sheets.map((item, i) => `<option value="${i}">${esc(item.name)}</option>`).join('');
 probe.innerHTML = capture.probes.map((item, i) => `<option value="${i}">${esc(item.label)} · ${esc(item.unit)}</option>`).join('');
 slider.max = end;
 slider.disabled = end === 0;
 play.disabled = end === 0;
 const currentWithDirection = capture.probes.findIndex(item => item.unit === 'A' &&
  capture.sheets.some(({scene}) => scene.parts.some(part =>
   part.pins.some(pin => pin.ref === item.positive) && part.pins.some(pin => pin.ref === item.reference))));
 probe.value = String(Math.max(0, currentWithDirection));
 const stop = () => {
  clearInterval(timer);
  timer = undefined;
  play.textContent = 'Play';
 };
 function chooseSheet() {
  const id = referencePart(capture.probes[+probe.value].positive);
  const i = capture.sheets.findIndex(item => item.scene.parts.some(part => part.id === id));
  if (i >= 0) sheet.value = String(i);
 }
 const duration = capture.rows[end].time_s - capture.rows[0].time_s;
 const horizontal = sample => 8 + 584 * (duration ? (sample.time_s - capture.rows[0].time_s) / duration : .5);
 let overlay;
 function renderDrawing() {
  const scene = capture.sheets[+sheet.value].scene;
  const [left, top, right, bottom] = scene.bounds;
  const selected = capture.probes[+probe.value];
  drawing.setAttribute('viewBox', [left, top, right - left, bottom - top].join(' '));
  drawing.innerHTML = `<g fill="none" stroke="var(--wire, #6d827c)" stroke-width="2.4">${scene.wires.map(wire =>
   `<polyline points="${wire.points.map(point => `${point.x},${point.y}`).join(' ')}"/>`).join('')}</g>
   ${scene.parts.map(part => {
    const active = part.id === referencePart(selected.positive) || part.id === referencePart(selected.reference);
    return `<g transform="translate(${part.position.x} ${part.position.y})">
     <g transform="rotate(${part.rotation})" fill="none" stroke="${active ? 'var(--accent, #19735b)' : 'currentColor'}" stroke-width="${active ? 4 : 2.5}">${part.paths.map(path => `<path d="${esc(path)}"/>`).join('')}</g>
     <text x="0" y="45" text-anchor="middle">${esc(part.id)} ${esc(part.value_text)}</text></g>`;
   }).join('')}
   ${(scene.ports || []).map(port => `<text x="${port.x}" y="${port.y}" text-anchor="${port.side < 0 ? 'end' : 'start'}">${esc(port.name)}</text>`).join('')}
   <g data-replay-arrow></g>`;
  overlay = drawing.querySelector('[data-replay-arrow]');
 }
 function renderWaveform() {
  const channel = +probe.value;
  const values = capture.rows.map(row => row.values[channel]);
  const low = Math.min(...values), high = Math.max(...values);
  const coordinates = values.map((value, i) => `${horizontal(capture.rows[i])},${8 + 57 * (high === low ? .5 : (high - value) / (high - low))}`).join(' ');
  waveform.innerHTML = `<polyline fill="none" stroke="currentColor" stroke-width="2" points="${coordinates}"/><line data-replay-cursor y1="2" y2="72" stroke="var(--accent, #19735b)" stroke-width="2"/>`;
 }
 function renderSample(next) {
  index = Math.max(0, Math.min(end, Number(next) || 0));
  const row = capture.rows[index];
  const selected = capture.probes[+probe.value];
  const value = row.values[+probe.value];
  slider.value = index;
  slider.setAttribute('aria-valuetext', `Sample ${index + 1} of ${end + 1}, ${time(row.time_s)}`);
  const pins = capture.sheets[+sheet.value].scene.parts.flatMap(part => part.pins);
  const from = pins.find(pin => pin.ref === selected.positive);
  const to = pins.find(pin => pin.ref === selected.reference);
  const direction = selected.unit === 'A' && value !== 0 && from && to ? (value > 0 ? [from, to] : [to, from]) : null;
  reading.textContent = `${time(row.time_s)} · ${selected.label} ${format(value, selected.unit)}${direction ? ` · ${direction[0].ref} → ${direction[1].ref}` : ''}`;
  const cursor = waveform.querySelector('[data-replay-cursor]');
  const cursorX = horizontal(row);
  cursor.setAttribute('x1', cursorX);
  cursor.setAttribute('x2', cursorX);
  overlay.replaceChildren();
  if (!direction) return;
  const [a, b] = direction;
  const angle = Math.atan2(b.y - a.y, b.x - a.x) * 180 / Math.PI;
  const x = (a.x + b.x) / 2, y = (a.y + b.y) / 2 - 25;
  overlay.innerHTML = `<path transform="translate(${x} ${y}) rotate(${angle})" d="M-20 0H20M12 -7L20 0L12 7" fill="none" stroke="var(--accent, #19735b)" stroke-width="4" stroke-linecap="round"/>`;
 }
 chooseSheet();
 renderDrawing();
 renderWaveform();
 renderSample(initialIndex);
 sheet.onchange = () => {renderDrawing(); renderSample(index);};
 probe.onchange = () => {chooseSheet(); renderDrawing(); renderWaveform(); renderSample(index);};
 slider.oninput = () => {stop(); renderSample(slider.value);};
 waveform.onclick = event => {
  stop();
  const rect = waveform.getBoundingClientRect();
  const x = 600 * (event.clientX - rect.left) / rect.width;
  const nearest = capture.rows.reduce((best, row, i) =>
   Math.abs(horizontal(row) - x) <= Math.abs(horizontal(capture.rows[best]) - x) ? i : best, 0);
  renderSample(nearest);
 };
 play.onclick = () => {
  if (timer) {stop(); return;}
  if (index === end) renderSample(0);
  play.textContent = 'Pause';
  const start = performance.now(), first = capture.rows[index].time_s;
  timer = setInterval(() => {
   const target = first + (performance.now() - start) / 10000;
   let next = index;
   while (next < end && capture.rows[next + 1].time_s <= target) next++;
   renderSample(next);
   if (index === end) stop();
  }, 30);
 };
 return {stop};
}

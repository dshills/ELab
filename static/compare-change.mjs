import {escapeHTML as esc} from './forms.mjs';
import {experimentPlot, firstCrossing} from './experiment-plot.mjs';

export function compareChangeDialog(api) {
 const dialog = document.getElementById('compare-change-dialog');
 const form = dialog.querySelector('form');
 const output = dialog.querySelector('[data-result]');
 const status = dialog.querySelector('[role="status"]');
 const keep = dialog.querySelector('[data-keep]');
 const compare = form.querySelector('[type="submit"]');
 let source, result, request;

 function open(part) {
  const state = api.state();
  const probes = state.scope.available.filter(probe => ['V', 'A'].includes(probe.unit));
  if (!probes.length) {
   api.toast('Connect a circuit with a voltage or current to compare.', true);
   return;
  }
  source = {part: part.id, guard: api.guards(), instance: api.instance()};
  form.elements.value.value = '';
  form.elements.probe.innerHTML = probes.map(probe => `<option value="${esc(probe.id)}">${esc(probe.label)} (${esc(probe.unit)})</option>`).join('');
  const preferred = part.kind === 'capacitor' ? probes.find(probe => probe.id === 'v:' + part.id) : null;
  form.elements.probe.value = preferred?.id || state.scope.selected.find(id => probes.some(probe => probe.id === id)) || probes[0].id;
  const step = state.transient_settings.step_s;
  form.elements.duration.value = String(Math.min(.05, step * 1000));
  form.elements.duration.parentElement.hidden = !state.transient;
  form.elements.duration.disabled = !state.transient;
  dialog.querySelector('[data-part]').textContent = `${part.id} · ${part.name} · ${api.fmt(part.value, part.unit, 7)}`;
  status.textContent = '';
  output.replaceChildren();
  result = null;
  compare.disabled = false;
  keep.hidden = true;
  dialog.showModal();
  form.elements.value.focus();
 }

 function current() {
  const now = api.guards();
  return source.instance === api.instance() &&
   now.expected_command_revision === source.guard.expected_command_revision &&
   now.expected_document_generation === source.guard.expected_document_generation;
 }

 form.addEventListener('input', () => {
  if (result) {
   result = null;
   keep.hidden = true;
   output.replaceChildren();
  }
 });

 form.addEventListener('submit', async event => {
  event.preventDefault();
  if (request || !source) return;
  if (!current()) {
   status.textContent = 'The circuit changed. Close and reopen Compare a Change.';
   return;
  }
  const value = form.elements.value.value.trim();
  const params = {part: source.part, value, probe: form.elements.probe.value,
   duration_s: form.elements.duration.parentElement.hidden ? 0 : form.elements.duration.valueAsNumber};
  result = null;
  keep.hidden = true;
  output.replaceChildren();
  status.textContent = 'Measuring the original and trial circuits…';
  const pending = new AbortController();
  request = pending;
  form.querySelector('fieldset').disabled = true;
  try {
   const data = await api.experiment('/api/compare-change', params, source.guard, pending.signal);
   if (pending.signal.aborted) return;
   result = {id: source.part, value};
   const {before, after, probe} = data;
   const peak = rows => rows.reduce((max, row) => Math.max(max, row.values[0]), -Infinity);
   const a = peak(before), b = peak(after);
   let timing = '';
   const range = before.at(-1).values[0] - before[0].values[0];
   if (range !== 0 && before.length > 1) {
    const level = before[0].values[0] + range / 2;
    const oldTime = firstCrossing(before, level, Math.sign(range));
    const newTime = firstCrossing(after, level, Math.sign(range));
    if (oldTime > 0 && newTime !== null) {
     timing = `Both curves crossed ${api.fmt(level, probe.unit)}. The first crossing moved from ${api.fmt(oldTime, 's')} to ${api.fmt(newTime, 's')} (estimated between samples).`;
    }
   }
   output.innerHTML = `<h3>${esc(source.part)} · ${esc(data.original)} → ${esc(data.trial)}</h3>
    <p>${esc(probe.label)}. Original ${before.length === 1 ? 'reading' : 'maximum'} <strong>${esc(api.fmt(a, probe.unit))}</strong>.
    Trial <strong>${esc(api.fmt(b, probe.unit))}</strong>. Difference <strong>${esc(api.fmt(b - a, probe.unit))}</strong>.</p>
    <p>${esc(timing)}</p>${experimentPlot([{label: 'Original', rows: before}, {label: 'Trial', rows: after}], probe.unit)}
    <p class="field-note">Original: solid blue. Trial: dashed amber.</p>`;
   keep.hidden = false;
   status.textContent = current() ? 'Comparison ready.' : 'The circuit changed. Reopen to compare the current values.';
  } catch (error) {
   if (!pending.signal.aborted) status.textContent = error.message;
  } finally {
   if (request === pending) {
    request = null;
    form.querySelector('fieldset').disabled = false;
   }
  }
 });

 keep.addEventListener('click', async () => {
  if (!result || request) return;
  if (!current()) {
   status.textContent = 'The circuit changed. Reopen to compare the current values.';
   return;
  }
  keep.disabled = true;
  try {
   await api.command('component.set', result, {guard: source.guard, message: 'Compared value applied.'});
   dialog.close();
  } catch (error) {
   status.textContent = error.message;
  } finally {
   keep.disabled = false;
  }
 });

 dialog.addEventListener('close', () => {
  request?.abort();
  request = null;
  form.querySelector('fieldset').disabled = false;
  result = null;
 });
 return {open};
}

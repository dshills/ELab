import {escapeHTML as esc} from './forms.mjs';
import {experimentPlot} from './experiment-plot.mjs';

export function experimentChecks(api) {
 const dialog = document.getElementById('experiment-checks-dialog');
 const form = dialog.querySelector('form');
 const status = dialog.querySelector('[role="status"]');
 const list = dialog.querySelector('[data-checks-list]');
 const chart = dialog.querySelector('[data-check-chart]');
 const save = dialog.querySelector('[data-save-check]');
 const fields = form.querySelector('fieldset');
 let guard, pending, editIndex = -1, checks = [], probes = [];

 function ruleFields() {
  const probe = probes.find(item => item.id === form.elements.probe.value);
  const reversal = form.elements.rule.value === 'reverses';
  form.elements.level.disabled = reversal;
  form.elements.level.required = !reversal;
  form.elements.level.placeholder = probe?.unit === 'A' ? '2 mA' : '3 V';
 }

 function setEditor(check, index = -1) {
  editIndex = index;
  const selected = api.state().scope.selected.find(id => probes.some(item => item.id === id));
  const probe = check?.probe || selected || probes[0]?.id;
  form.elements.probe.innerHTML = probes.map(item => `<option value="${esc(item.id)}">${esc(item.label)} (${esc(item.unit)})</option>`).join('');
  if (probe && !probes.some(item => item.id === probe)) {
   form.elements.probe.add(new Option(probe + ' (unavailable)', probe));
  }
  if (probe) form.elements.probe.value = probe;
  form.elements.name.value = check?.name ?? '';
  form.elements.rule.value = check?.rule || 'rises';
  form.elements.level.value = check?.level ?? '';
  const duration = api.state().verification_settings?.duration_s || .05;
  form.elements.duration.value = check?.duration || api.fmt(duration, 's', 6);
  save.textContent = index < 0 ? 'Save test' : 'Update test';
  ruleFields();
 }

 function renderList() {
  list.replaceChildren();
  for (const [index, check] of checks.entries()) {
   const row = document.createElement('li');
   const name = document.createElement('span');
   name.textContent = check.name;
   row.append(name);
   for (const [action, label] of [['run', 'Run'], ['edit', 'Edit'], ['remove', 'Remove']]) {
    const button = document.createElement('button');
    button.type = 'button';
    button.dataset.action = action;
    button.dataset.index = index;
    button.textContent = label;
    button.setAttribute('aria-label', label + ' ' + check.name);
    row.append(button);
   }
   list.append(row);
  }
  list.hidden = !checks.length;
 }

 function open() {
  const state = api.state();
  if (!state) return;
  guard = api.guards();
  probes = state.scope.available;
  checks = state.experiment_checks || [];
  renderList();
  setEditor(null);
  chart.replaceChildren();
  status.textContent = '';
  dialog.showModal();
 }

 function draft() {
  const probe = probes.find(item => item.id === form.elements.probe.value);
  if (!probe) throw new Error('Choose a measurement on the current circuit.');
  const rule = form.elements.rule.value;
  return {
   name: form.elements.name.value.trim(), probe: probe.id, unit: probe.unit, rule,
   level: rule === 'reverses' ? '' : form.elements.level.value,
   duration: form.elements.duration.value,
  };
 }

 async function run(check, signal) {
  const {capture, evidence} = await api.experiment('/api/experiment-check', check, guard, signal);
  if (signal.aborted) return;
  const {unit} = capture.probe;
  const point = row => `${api.fmt(row.values[0], unit, 6)} at ${api.fmt(row.time_s, 's', 6)}`;
  const result = evidence.pass ? 'Pass' : 'Fail';
  const detail = evidence.after
   ? `Measured ${point(evidence.before)} and then ${point(evidence.after)}.`
   : check.rule === 'max'
    ? `Highest sample was ${point(evidence.maximum)}. Limit: ${api.fmt(evidence.level, unit, 6)}.`
    : `Samples ranged from ${point(evidence.minimum)} to ${point(evidence.maximum)}.`;
  status.textContent = `${result} · ${check.name}. ${detail}`;
  const series = [{label: capture.probe.label, rows: capture.rows}];
  if (check.rule !== 'reverses') {
   const first = capture.rows[0].time_s, last = capture.rows.at(-1).time_s;
   const rows = (first === last ? [first] : [first, last]).map(time_s => ({time_s, values: [evidence.level]}));
   series.push({label: 'Threshold', rows});
  }
  chart.innerHTML = experimentPlot(series, unit);
 }

 async function update(next, message) {
  await api.command('experiment.checks.set', {checks: next}, {guard});
  guard = api.guards();
  checks = api.state().experiment_checks || [];
  renderList();
  setEditor(null);
  status.textContent = message;
 }

 async function perform(work, message = '') {
  if (pending) return;
  const controller = new AbortController();
  pending = controller;
  fields.disabled = true;
  for (const button of list.querySelectorAll('button')) button.disabled = true;
  status.textContent = message;
  chart.replaceChildren();
  try {
   await work(controller.signal);
  } catch (error) {
   if (!controller.signal.aborted) status.textContent = error.message;
  } finally {
   if (pending === controller) {
    pending = null;
    fields.disabled = false;
    for (const button of list.querySelectorAll('button')) button.disabled = false;
    ruleFields();
   }
  }
 }

 form.addEventListener('submit', event => {
  event.preventDefault();
  if (event.submitter === save) {
   void perform(async () => {
    const check = draft();
    const next = checks.slice();
    next[editIndex < 0 ? next.length : editIndex] = check;
    await update(next, 'Test saved in this project.');
   });
  } else {
   void perform(signal => run(draft(), signal), 'Running test from startup…');
  }
 });
 form.elements.rule.addEventListener('change', ruleFields);
 form.elements.probe.addEventListener('change', ruleFields);
 form.querySelector('[data-new-check]').addEventListener('click', () => {
  setEditor(null);
  status.textContent = '';
  chart.replaceChildren();
  form.elements.name.focus();
 });
 list.addEventListener('click', event => {
  const button = event.target.closest('button[data-action]');
  if (!button || pending) return;
  const index = Number(button.dataset.index);
  const check = checks[index];
  switch (button.dataset.action) {
   case 'run': void perform(signal => run(check, signal), 'Running test from startup…'); break;
   case 'edit': setEditor(check, index); form.elements.name.focus(); break;
   case 'remove': void perform(() => update(checks.filter((_, i) => i !== index), 'Test removed.')); break;
  }
 });
 dialog.addEventListener('close', () => pending?.abort());
 return {open};
}

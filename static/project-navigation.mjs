import {escapeHTML as esc} from './forms.mjs';

const partID = ref => ref.split('.')[0];

export function projectMatches(state, query) {
  const needle = query.trim().toLowerCase();
  if (!needle) return [];
  const parts = new Map(state.components.map(part => [part.id, part]));
  const sheets = new Map((state.sheets || []).map(sheet => [sheet.id, sheet]));
  const sheetPath = id => {
    const sheet = sheets.get(id);
    return sheet ? `${sheetPath(sheet.parent)} / ${sheet.name}` : 'Main';
  };
  const matches = fields => fields.filter(Boolean).some(field => field.toLowerCase().includes(needle));
  const results = [];
  const add = (title, part, target, kind) => results.push({title, target, kind, sheet:sheetPath(part.sheet)});
  for (const part of parts.values()) {
    if (matches([part.id, part.name, part.label, part.kind, part.template_part])) {
      const designator = part.template_part ? `${part.id} (${part.template_part})` : part.id;
      add(`${designator} · ${part.label || part.name}`, part, part.id, 'Part');
    }
  }
  for (const net of state.nets || []) {
    const labels = net.labels || [];
    if (!matches([net.id, ...labels, ...net.terminals])) continue;
    const bySheet = new Map();
    for (const terminal of net.terminals) {
      const part = parts.get(partID(terminal));
      if (!part) continue;
      const sheet = part.sheet || '';
      const previous = bySheet.get(sheet);
      if (!previous || matches([terminal]) && !matches([previous])) bySheet.set(sheet, terminal);
    }
    for (const terminal of bySheet.values()) add(`${labels.join(', ') || net.id} · ${terminal}`, parts.get(partID(terminal)), terminal, 'Signal');
  }
  for (const wire of state.wires) {
    if (!wire.port || !matches([wire.port, wire.id])) continue;
    for (const terminal of [wire.from, wire.to]) {
      const part = parts.get(partID(terminal));
      if (part) add(`${wire.port} · ${terminal}`, part, terminal, 'Port');
    }
  }
  return results;
}

export function projectSearch(api) {
  const byID = id => document.getElementById(id);
  const dialog = byID('project-find-dialog');
  const input = byID('project-find-input');
  const list = byID('project-find-results');
  function update() {
    const results = projectMatches(api.state(), input.value);
    const visible = results.slice(0, 80);
    byID('project-find-status').textContent = !input.value.trim() ? 'Search all sheets.' : !results.length ? 'No matching parts or signals.' : results.length > visible.length ? `Showing ${visible.length} of ${results.length} matches. Narrow your search for more.` : `${results.length} ${results.length === 1 ? 'match' : 'matches'}`;
    list.innerHTML = visible.map(result => `<li><button type="button" data-find-target="${esc(result.target)}"><span>${esc(result.title)}</span><small>${esc(result.kind)} · ${esc(result.sheet)}</small></button></li>`).join('');
  }
  byID('project-find-button').onclick = () => {
    if (!api.state()) return;
    update();
    dialog.showModal();
    input.focus();
    input.select();
  };
  input.oninput = update;
  list.onclick = event => {
    const button = event.target.closest('[data-find-target]');
    if (!button) return;
    dialog.close();
    api.show(button.dataset.findTarget).catch(() => {});
  };
}

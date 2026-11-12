import {escapeHTML as esc} from './forms.mjs';

export function schematicSheets(api) {
  const byID = id => document.getElementById(id);
  const dialog = byID('sheets-dialog');
  const selector = byID('sheet-select');
  const portLayer = byID('sheet-ports');
  const root = {id: '', name: 'Main'};
  let signature;
  const hierarchy = () => [root, ...(api.state()?.sheets || [])];
  const name = id => hierarchy().find(sheet => sheet.id === id)?.name || 'Main';
  function nextSheetID() {
    const existing = new Set(hierarchy().map(sheet => sheet.id));
    let n = 1;
    while (existing.has(`Sheet${n}`)) n++;
    return `Sheet${n}`;
  }
  const open = id => api.command('sheet.open', {id});
  const show = () => {
    signature = null;
    update();
    dialog.showModal();
  };
  const options = (items, label) => items.map(item =>
    `<option value="${esc(item.id)}">${esc(label(item))}</option>`).join('');
  const path = sheet => sheet.id ? `${path(hierarchy().find(item => item.id === (sheet.parent || '')) || root)} / ${sheet.name}` : sheet.name;
  function terminals(parts) {
    return parts.flatMap(part => part.terminals.map(pin => ({id: `${part.id}.${pin}`, sheet: part.sheet || '', source:part.template_part ? `${part.template_part}.${pin}` : ''})));
  }
  function update() {
    const state = api.state();
    if (!state) return;
    const active = state.active_sheet || '';
    const sheets = hierarchy();
    const current = sheets.find(sheet => sheet.id === active) || root;
    const adjacent = sheets.filter(sheet => sheet.id !== active && ((sheet.parent || '') === active || sheet.id === current.parent || (active && !current.parent && !sheet.id)));
    const chosen = api.selection();
    const parts = state.components;
    const ports = api.scene()?.ports || [];
    const key = JSON.stringify([state.document.generation, state.command_revision, active, chosen]);
    if (signature === key) return;
    signature = key;
    selector.innerHTML = options(sheets, path);
    selector.value = active;
    selector.disabled = sheets.length === 1;
    byID('sheet-dialog-current').textContent = path(current);
    const linked = sheets.filter(sheet => sheet.template && sheet.template === active);
    const leafTemplates = sheets.filter(sheet => sheet.id && !sheet.template && !sheets.some(child => child.parent === sheet.id));
    const canAddChild = !current.template && linked.length === 0;
    byID('sheet-create-form').hidden = !canAddChild;
    byID('sheet-instance-form').hidden = !canAddChild || !leafTemplates.some(sheet => sheet.id !== active);
    byID('sheet-instance-template').innerHTML = options(leafTemplates.filter(sheet => sheet.id !== active), path);
    byID('sheet-instance-status').hidden = !current.template && !linked.length;
    byID('sheet-open-source').hidden = !current.template;
    byID('sheet-apply-source').hidden = !linked.length;
    byID('sheet-instance-description').textContent = current.template ? `Linked to ${name(current.template)}` : `${linked.length} linked ${linked.length === 1 ? 'sheet' : 'sheets'}`;
    byID('sheet-rename-form').hidden = !active;
    byID('sheet-delete').hidden = !active;
    byID('sheet-delete').disabled = parts.some(part => part.sheet === active) || sheets.some(sheet => sheet.parent === active || sheet.template === active);
    byID('sheet-move-form').hidden = !chosen.length || !adjacent.length;
    byID('sheet-move-target').innerHTML = options(adjacent, path);
    byID('sheet-move-count').textContent = `Move ${chosen.length} selected ${chosen.length === 1 ? 'component' : 'components'}`;
    const local = terminals(parts.filter(part => (part.sheet || '') === active));
    const remote = terminals(parts.filter(part => adjacent.some(sheet => sheet.id === (part.sheet || ''))));
    byID('sheet-port-form').hidden = !local.length || !remote.length;
    byID('sheet-port-from').innerHTML = options(local, item => item.source ? `${item.source} · ${item.id}` : item.id);
    byID('sheet-port-to').innerHTML = options(remote, item => `${name(item.sheet)} · ${item.source ? `${item.source} · ` : ''}${item.id}`);
    byID('sheet-port-list').innerHTML = ports.map(port => `<li>
      <button type="button" data-follow-sheet="${esc(port.target)}">${esc(port.name)} → ${esc(name(port.target))}</button>
      <span>${esc(port.terminal)}</span>
      <button type="button" data-remove-sheet-port="${esc(port.id)}" aria-label="Remove port ${esc(port.name)}">×</button>
    </li>`).join('');
    byID('sheet-ports-heading').hidden = !ports.length;
  }
  function renderPorts(scene) {
    const ports = scene?.ports || [];
    const pins = new Map(ports.length ? scene.parts.flatMap(part => part.pins.map(pin => [pin.ref, pin])) : []);
    const key = JSON.stringify(ports.map(port => [port, pins.get(port.terminal), name(port.target)]));
    if (portLayer.dataset.key === key) return;
    const focused = document.activeElement?.dataset?.sheetPort;
    portLayer.dataset.key = key;
    portLayer.innerHTML = ports.map(port => {
      const pin = pins.get(port.terminal);
      if (!pin) return '';
      const x = port.x + 9 * port.side;
      const label = `${port.name} to ${name(port.target)}${port.active ? '' : ', disconnected'}`;
      return `<g class="sheet-port${port.active ? '' : ' disconnected'}" role="button" tabindex="0" data-sheet-port="${esc(port.id)}" data-follow-sheet="${esc(port.target)}" aria-label="${esc(label)}">
        <title>${esc(label)} · ${esc(port.terminal)}</title>
        <path d="M${pin.x} ${pin.y} H${port.x - 10 * port.side} V${port.y} H${port.x}"/>
        <circle cx="${port.x}" cy="${port.y}" r="7"/>
        <text x="${x}" y="${port.y - 5}" text-anchor="${port.side < 0 ? 'end' : 'start'}">${esc(port.name)}</text>
      </g>`;
    }).join('');
    if (focused) [...portLayer.querySelectorAll('[data-sheet-port]')].find(port => port.dataset.sheetPort === focused)?.focus({preventScroll:true});
  }
  function form(id, submit, close = true) {
    byID(id).onsubmit = async event => {
      event.preventDefault();
      const fields = new FormData(event.currentTarget);
      try {
        await submit(fields);
        byID(id).reset();
        if (close) dialog.close();
        else update();
      } catch {
        // The command reports validation and conflict feedback.
      }
    };
  }
  selector.onchange = () => open(selector.value).catch(() => {signature = null; update();});
  byID('sheets-button').onclick = show;
  form('sheet-create-form', fields => api.command('sheet.create', {id:nextSheetID(), name:fields.get('name').trim(), parent:api.state().active_sheet || ''}));
  form('sheet-instance-form', fields => api.command('sheet.instance.create', {id:nextSheetID(), name:fields.get('name').trim(), parent:api.state().active_sheet || '', template:fields.get('template')}));
  byID('sheet-open-source').onclick = () => {
    const source = hierarchy().find(sheet => sheet.id === api.state().active_sheet)?.template;
    if (source) open(source).then(() => dialog.close()).catch(() => {});
  };
  byID('sheet-apply-source').onclick = () => api.command('sheet.template.apply', {id:api.state().active_sheet}, {message:'Applied source changes to linked sheets.'}).then(() => dialog.close()).catch(() => {});
  form('sheet-rename-form', fields => api.command('sheet.rename', {id:api.state().active_sheet, name:fields.get('name').trim()}));
  form('sheet-move-form', fields => api.command('sheet.move', {ids:api.selection(), sheet:fields.get('target')}));
  form('sheet-port-form', fields => api.command('sheet.port.connect', {from:fields.get('from'), to:fields.get('to'), name:fields.get('name').trim()}), false);
  byID('sheet-delete').onclick = () => api.command('sheet.delete', {id:api.state().active_sheet}).then(() => dialog.close()).catch(() => {});
  function follow(event) {
    const target = event.target.closest('[data-follow-sheet]');
    if (!target) return;
    event.stopPropagation();
    if (event.type === 'keydown') {
      if (event.key !== 'Enter' && event.key !== ' ') return;
      event.preventDefault();
      event.stopPropagation();
    }
    open(target.dataset.followSheet).then(() => {
      if (dialog.open) dialog.close();
      byID('schematic').focus();
    }).catch(() => {});
  }
  dialog.addEventListener('click', event => {
    follow(event);
    const button = event.target.closest('[data-remove-sheet-port]');
    if (button) api.command('object.delete', {id:button.dataset.removeSheetPort}).catch(() => {});
  });
  portLayer.addEventListener('click', follow);
  portLayer.addEventListener('keydown', follow);
  return {update, renderPorts};
}

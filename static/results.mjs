import {escapeHTML as esc,labelFor} from './forms.mjs';
export function resultTable(value,depth=0){
 if(value==null)return '<span class="muted">Unavailable</span>';
 if(typeof value!=='object')return esc(typeof value==='number'?Number(value.toPrecision(7)):value);
 if(Array.isArray(value)){
  if(!value.length)return '<span class="muted">None</span>';
  if(value.every(v=>v==null||typeof v!=='object'))return esc(value.join(' · '));
  const rows=value.slice(0,200),keys=[...new Set(rows.flatMap(row=>Object.keys(row||{})))];
  return `<div class="result-table-wrap"><table class="result-table"><thead><tr>${keys.map(k=>`<th>${esc(labelFor(k))}</th>`).join('')}</tr></thead><tbody>${rows.map(row=>`<tr>${keys.map(k=>`<td>${resultTable(row?.[k],depth+1)}</td>`).join('')}</tr>`).join('')}</tbody></table>${value.length>200?`<p class="field-note">Showing 200 of ${value.length} rows. Download the full result for all rows.</p>`:''}</div>`;
 }
 return `<dl class="result-fields">${Object.entries(value).map(([key,v])=>{if(v&&typeof v==='object')return `<div><details ${depth===0?'open':''}><summary>${esc(labelFor(key))}</summary>${depth>7?'<span class="muted">Download the complete result for deeper details.</span>':resultTable(v,depth+1)}</details></div>`;return `<div><dt>${esc(labelFor(key))}</dt><dd>${resultTable(v,depth+1)}</dd></div>`;}).join('')}</dl>`;
}
export function selectedReadings(state,id){
 const arrays=Object.values(state.mixed||{}).filter(Array.isArray),rows=arrays.flat().filter(r=>r?.id===id||r?.transistor_id===id);
 if(state.npn&&(state.npn.transistor_id===id||state.npn.led_id===id))rows.push(state.npn);
 return rows;
}

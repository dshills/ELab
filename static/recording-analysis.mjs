import {syncTabs} from './accessibility.mjs';
import {escapeHTML as esc} from './forms.mjs';
import {resultTable} from './results.mjs';

export function recordingAnalysisEdit(kind, v, id) {
 if(!id)throw new Error('Choose a recorded experiment first.');
 const params={id};
 switch(kind) {
  case 'compare':return {name:'recording.compare.set',params:{...params,reference_id:v.reference_id,time_s:v.time_s,show_overlay:!!v.show_overlay,show_statistics:!!v.show_statistics}};
  case 'timing':return {name:'recording.timing.set',params:{...params,probe_id:v.probe_id,level:v.level,...(v.second_level?.trim()?{second_level:v.second_level}:{}),direction:v.direction,start_s:v.start_s,end_s:v.end_s,offset:0,limit:Number(v.limit)}};
  case 'step':return {name:'recording.step-response.set',params:{...params,probe_id:v.probe_id,start_s:v.start_s,end_s:v.end_s,initial_value:Number(v.initial_value),final_value:Number(v.final_value),tolerance_fraction:Number(v.tolerance_percent)/100,minimum_hold_s:v.minimum_hold_s}};
  case 'clear-compare':return {name:'recording.compare.set',params:{id,reference_id:'',time_s:0}};
  case 'clear-timing':return {name:'recording.timing.set',params:{id,probe_id:'',level:'0',direction:'both',start_s:0,end_s:0,offset:0,limit:0}};
  case 'clear-step':return {name:'recording.step-response.set',params:{id,probe_id:'',start_s:0,end_s:0,initial_value:0,final_value:0,tolerance_fraction:0,minimum_hold_s:0}};
  default:throw new Error('Unknown recording analysis.');
 }
}

export function timingPage(timing) {
 const offset=timing?.settings.offset??0,total=timing?.total_crossings??0,end=timing?.next_offset??0;
 return {start:total?offset+1:0,end,total,previous:offset>0,next:end<total,previousOffset:Math.max(0,offset-(timing?.settings.limit||8))};
}

export function recordingAnalyses(api) {
 const $=id=>document.getElementById(id),dialog=$('recording-analysis-dialog'),content=$('recording-analysis-content');
 let owner='',id='',tab='compare',pending=false,openedState;
 const capture=()=>({owner:api.context().owner,guard:api.guards()});
 function chooseTab(next) {
  tab=next;
  for(const b of dialog.querySelectorAll('[data-recording-tab]')){const selected=b.dataset.recordingTab===tab;b.classList.toggle('active',selected);b.setAttribute('aria-selected',String(selected));}syncTabs();
  for(const panel of content.querySelectorAll('[data-analysis-panel]'))panel.classList.toggle('hidden',panel.dataset.analysisPanel!==tab);
 }
 function updateUnits(form) {
  const p=openedState.recording.scope.available.find(p=>p.id===form.querySelector('[name="probe_id"]').value);
  for(const span of form.querySelectorAll('[data-probe-unit]'))span.textContent=p?.unit||'SI';
 }
 function show() {
  const context=api.context(),r=context.recording;if(!r)return;
  openedState=api.state();owner=context.owner;id=r.id;
  $('recording-analysis-source').textContent=`${id} · ${api.time(r.duration_s)} · ${r.sample_count} stored samples. Analyses use captured data; the live circuit stays independent.`;
  const comparison=r.comparison?.settings||{},timing=r.timing?.settings||{},step=r.step_response?.settings||{},end=r.last_sample_time_s;
  const options=(value)=>r.scope.available.map(p=>`<option value="${esc(p.id)}" ${p.id===value?'selected':''}>${esc(p.label)} · ${esc(p.unit)}</option>`).join('');
  const field=(label,name,value,type='text',attrs='')=>`<label>${label}<input name="${name}" type="${type}" value="${esc(value)}" ${attrs} required></label>`;
  const feedback='<p class="settings-feedback error" role="status"></p>';
  content.innerHTML=`<section data-analysis-panel="compare"><form data-recording-analysis="compare" class="settings-form"><label>Reference capture<select name="reference_id" required><option value="">Choose another completed recording…</option>${(openedState.operations||[]).filter(o=>o.status==='completed'&&o.id!==id).map(o=>`<option value="${esc(o.id)}" ${o.id===comparison.reference_id?'selected':''}>${esc(o.id)}${o.kept_in_project?' · kept':''}</option>`).join('')}</select></label>${field('Comparison time','time_s',comparison.time_s??r.time_s)}<label class="checkbox-label"><input type="checkbox" name="show_overlay" ${(r.comparison?!!comparison.show_overlay:true)?'checked':''}>Overlay reference traces in Scope</label><label class="checkbox-label"><input type="checkbox" name="show_statistics" ${(r.comparison?!!comparison.show_statistics:true)?'checked':''}>Show whole-record statistics</label><p class="field-note">A is this capture, B is the reference, and Δ is B − A. Each uses its nearest stored row; offsets are reported, without resampling. Matching requires the same probe, units and terminal references.</p><div class="analysis-actions"><button class="primary">Compare captures</button><button type="button" data-recording-clear="compare">Clear comparison</button></div>${feedback}</form><div id="recording-compare-result" class="section"></div></section>
  <section data-analysis-panel="timing"><form data-recording-analysis="timing" class="settings-form"><label>Captured trace<select name="probe_id" aria-label="Timing trace">${options(timing.probe_id)}</select></label><div class="scope-settings-grid">${field('Threshold (<span data-probe-unit></span>)','level',timing.level??'2.5')}<label>Second threshold (<span data-probe-unit></span>, optional)<input name="second_level" value="${esc(timing.second_level||'')}" placeholder="For transition bounds"></label><label>Direction<select name="direction"><option value="both" ${!timing.direction||timing.direction==='both'?'selected':''}>Both</option><option value="rising" ${timing.direction==='rising'?'selected':''}>Rising</option><option value="falling" ${timing.direction==='falling'?'selected':''}>Falling</option></select></label>${field('Start time','start_s',timing.start_s??0)}${field('End time','end_s',timing.end_s??end)}${field('Crossings per page','limit',timing.limit||20,'number','min="1" max="100" step="1"')}</div><p class="field-note">Times accept units such as 10 ms. Thresholds accept the captured unit with m/µ prefixes. Crossings retain exact before/after sample brackets; spacing, pulse and cycle values are bounds. Pulse, cycle and transition summaries use both directions across the whole interval. Crossing lists and same-direction spacing follow the direction filter; summaries are independent of paging.</p><div class="analysis-actions"><button class="primary">Measure timing</button><button type="button" data-recording-clear="timing">Clear timing</button></div>${feedback}</form><div class="analysis-actions section"><button id="recording-timing-previous">Previous crossings</button><span id="recording-timing-page" role="status"></span><button id="recording-timing-next">Next crossings</button></div><div id="recording-timing-result"></div></section>
  <section data-analysis-panel="step"><form data-recording-analysis="step" class="settings-form"><label>Captured trace<select name="probe_id" aria-label="Step response trace">${options(step.probe_id)}</select></label><div class="scope-settings-grid">${field('Interval start','start_s',step.start_s??0)}${field('Interval end','end_s',step.end_s??end)}${field('Initial reference (<span data-probe-unit></span>)','initial_value',step.initial_value??0,'number','min="-1000000" max="1000000" step="any"')}${field('Final reference (<span data-probe-unit></span>)','final_value',step.final_value??'','number','min="-1000000" max="1000000" step="any" placeholder="Expected final value"')}${field('Tolerance (%)','tolerance_percent',(step.tolerance_fraction??.02)*100,'number','min="0.01" max="50" step="any"')}${field('Minimum continuous hold','minimum_hold_s',step.minimum_hold_s??Math.min(.005,end/4))}</div><p class="field-note">Supply distinct initial and final reference values for the selected trace. Overshoot and settling use the native piecewise-linear captured response. Peak time is absolute capture time; settling delay is measured from the interval start. A DC point has no step interval.</p><div class="analysis-actions"><button class="primary">Measure step response</button><button type="button" data-recording-clear="step">Clear step response</button></div>${feedback}</form><div id="recording-step-result" class="section"></div></section>`;
  for(const form of content.querySelectorAll('form')){form._context=capture();if(form.querySelector('[name="probe_id"]'))updateUnits(form);}
  chooseTab(tab);if(!dialog.open)dialog.showModal();dialog.scrollTop=0;update();
 }
 function render(id,html){const output=$(id);if(output.innerHTML!==html)output.innerHTML=html;}
 function update() {
  const context=api.context(),r=context.recording,button=$('recording-analysis-button');
  if(button){button.classList.toggle('hidden',!r);button.disabled=pending;}
  if(!dialog.open)return;
  $('recording-analysis-reload').disabled=pending||!r;
  const stale=!r||context.owner!==owner;
  $('recording-analysis-stale').classList.toggle('hidden',!stale);
  for(const control of content.querySelectorAll('input,select,button'))control.disabled=stale||pending;
  if(stale)return;
  content.querySelector('[data-recording-analysis="step"] button:not([type="button"])').disabled=pending||r.duration_s===0;
  const c=r.comparison;
  render('recording-compare-result',c?`${c.available?`<p>A ${esc(c.a.id)} · sample ${c.a.sample_index} at ${esc(api.time(c.a.time_s))}; B ${esc(c.b.id)} · sample ${c.b.sample_index} at ${esc(api.time(c.b.time_s))}.</p>`:`<p role="status">${esc(c.reason)}</p>`}<div class="result-table-wrap"><table class="result-table"><thead><tr><th>Captured trace</th><th>A</th><th>B</th><th>Δ B − A</th></tr></thead><tbody>${c.readings.map(row=>`<tr><th scope="row">${esc(row.probe.label)}</th>${[row.a,row.b,row.b_minus_a].map(value=>`<td>${esc(api.fmt(value,row.probe.unit,7))}</td>`).join('')}</tr>`).join('')}</tbody></table></div>${c.settings.show_statistics?`<details><summary>Whole-record statistics</summary>${c.readings.map(row=>`<h3>${esc(row.probe.label)} · ${esc(row.probe.unit)}</h3><div class="scope-settings-grid"><section><h4>A · current capture</h4>${resultTable(row.whole_record_a)}</section><section><h4>B · reference capture</h4>${resultTable(row.whole_record_b)}</section></div>`).join('')}</details>`:''}<details><summary>Exact sample provenance & unmatched channels</summary>${resultTable({current_capture:c.a,reference_capture:c.b,unmatched_current:c.unmatched_a,unmatched_reference:c.unmatched_b})}</details>`:'<p class="field-note">Choose a reference recording and compare its native readings.</p>');
  const t=r.timing,page=timingPage(t);
  $('recording-timing-previous').disabled=pending||!page.previous;
  $('recording-timing-next').disabled=pending||!page.next;
  $('recording-timing-page').textContent=t?`${page.start}–${page.end} of ${page.total} crossings`:'';
  render('recording-timing-result',t?resultTable(t):'<p class="field-note">Measure a threshold to inspect native sample brackets and timing bounds.</p>');
  const step=r.step_response;
  render('recording-step-result',step?`<div class="interval-measures">${[['Overshoot',api.fmt(step.overshoot_percent,'%')],['Peak',api.fmt(step.peak_value,step.probe.unit)],['Peak time',api.time(step.peak_time_s)],['Settling delay',step.settling_delay_s==null?'Unavailable':api.time(step.settling_delay_s)],['Observed hold',api.time(step.observed_hold_s)]].map(([label,value])=>`<span>${label}<strong>${esc(value)}</strong></span>`).join('')}</div>${step.settling_reason?`<p role="status">${esc(step.settling_reason)}</p>`:''}<details><summary>Native step-response result & settings</summary>${resultTable(step)}</details>`:'<p class="field-note">Set reference levels and a covered interval to measure the native response.</p>');
 }
 async function run(edit,context=capture()) {
  if(pending)throw new Error('Wait for the current analysis to finish.');
  if(context.owner!==api.context().owner)throw new Error('The captured source changed. Close and reopen these controls.');
  pending=true;update();
  try {
   const result=await api.command(edit.name,edit.params,{guard:context.guard});
   // Rebase only an acknowledged local edit, never an intervening external one.
   if(dialog.open&&owner===context.owner&&result?.command_revision===api.state().command_revision&&api.context().owner===owner)
    for(const form of content.querySelectorAll('form'))form._context=capture();
   return result;
  } finally {pending=false;update();}
 }
 const errorMessage=error=>{const message=typeof error?.message==='string'?error.message:typeof error==='string'?error:'The analysis could not be applied.';return /expected command revision|document generation/.test(message)?'The workbench changed while these entries were open. Use current settings to refresh them.':message;};
 $('scope-controls').addEventListener('click',e=>{if(e.target.closest('#recording-analysis-button'))show();});
 dialog.addEventListener('change',e=>{const form=e.target.closest('form');if(e.target.name==='probe_id'&&form)updateUnits(form);});
 dialog.addEventListener('submit',async e=>{
  const form=e.target;if(!form.dataset.recordingAnalysis)return;e.preventDefault();if(!form.reportValidity())return;
  const feedback=form.querySelector('.settings-feedback');
  try {const values=Object.fromEntries(new FormData(form));await run(recordingAnalysisEdit(form.dataset.recordingAnalysis,values,id),form._context);feedback.textContent='Applied.';feedback.classList.remove('error');}
  catch(error){feedback.textContent=errorMessage(error);feedback.classList.add('error');}
 });
 dialog.addEventListener('click',async e=>{
  const button=e.target.closest('button');if(!button||button.disabled)return;
  if(button.id==='recording-analysis-reload'){show();return;}
  if(button.dataset.recordingTab){chooseTab(button.dataset.recordingTab);return;}
  try {
   if(button.dataset.recordingClear){const form=button.closest('form');await run(recordingAnalysisEdit('clear-'+button.dataset.recordingClear,{},id),form._context);form.querySelector('.settings-feedback').textContent='Cleared.';form.querySelector('.settings-feedback').classList.remove('error');}
   if(button.id==='recording-timing-next'||button.id==='recording-timing-previous'){
    const t=api.context().recording?.timing;if(!t||t.settings.id!==id)return;
    await run({name:'recording.timing.set',params:{...t.settings,offset:button.id==='recording-timing-next'?t.next_offset:timingPage(t).previousOffset}});
   }
  }catch(error){api.toast(errorMessage(error),true);}
 });
 return {show,update};
}

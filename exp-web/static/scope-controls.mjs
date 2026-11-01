import {escapeHTML as esc} from './forms.mjs';
import {recordingAnalyses} from './recording-analysis.mjs';

export function scopeContext(state,instance='') {
 const recording=state.recording?.viewing?state.recording:null;
 return {recording,scope:recording?.scope||state.scope,owner:`${instance?instance+':':''}${state.document.generation}:${recording?.id||'live'}`};
}

// Keep engineering-unit text intact for the shared native parser. Indices and
// speeds are numbers; fractional indices must reach validation, never truncate.
export function scopeEdit(kind,values,state) {
 const {recording}=scopeContext(state),prefix=recording?'recording':'scope';
 switch(kind) {
  case 'index': return {name:'recording.seek',params:{index:Number(values.index)}};
  case 'seek': return {name:'recording.seek',params:{time_s:values.time_s}};
  case 'speed': return {name:'recording.speed.set',params:{seconds_per_wall_second:Number(values.speed)}};
  case 'window': return {name:prefix+'.window.set',params:{...(recording?{}:{mode:'fixed'}),start_s:values.start_s,end_s:values.end_s}};
  case 'cursor-a': case 'cursor-b': return {name:prefix+'.cursor.set',params:{cursor:kind.at(-1),time_s:values.time_s,...(recording?{operation_id:recording.id}:{})}};
  case 'probes': return {name:prefix+'.probes.set',params:{ids:values.ids}};
  case 'statistics': return {name:'recording.statistics.set',params:{id:recording.id,probe_id:values.probe_id,start_s:values.start_s,end_s:values.end_s}};
  default: throw new Error('Unknown scope control.');
 }
}

export function captureParams(values,state) {
 const probes=state.scope.selected||[];
 if(state.transient&&!probes.length)throw new Error('Choose at least one live scope trace before recording.');
 return {duration_s:values.duration,sample_step_s:values.step,
  ...(probes.length?{probe_ids:[...probes]}:{}),
  ...(values.input_mode?{input_mode:values.input_mode}:{}),
  ...(values.initial_voltage_v?.trim()?{initial_voltage_v:Number(values.initial_voltage_v)}:{})};
}

export function scopePlotDomain(scope,traces) {
 // Native windows describe the requested interval, including gaps between exact
 // rows. Using only plotted extrema hides empty margins and breaks cursor clicks.
 const window=scope?.window;
 if(window&&Number.isFinite(window.start_s)&&Number.isFinite(window.end_s)&&window.end_s>window.start_s)
  return {minT:window.start_s,maxT:window.end_s};
 const times=traces.flatMap(t=>[t.points?.[0]?.time_s,t.points?.at(-1)?.time_s]).filter(Number.isFinite);
 const start=times.length?Math.min(...times):0,end=times.length?Math.max(...times):.1;
 return {minT:start,maxT:end>start?end:start+.001};
}

export function scopeControls(api) {
 const $=id=>document.getElementById(id),root=$('scope-controls'),dialog=$('scope-settings-dialog');
 const analysisUI=recordingAnalyses({...api,context:()=>scopeContext(api.state(),api.instance?.())});
 let owner='',dialogOwner='',dialogState,rangeGuard,rangeOwner,pending=0;
 const capture=()=>({guard:api.guards(),owner:scopeContext(api.state(),api.instance?.()).owner});
 const assertOwner=expected=>{if(scopeContext(api.state(),api.instance?.()).owner!==expected)throw new Error('The scope source changed. Reopen these controls for the current source.');};
 async function run(edit,context=capture()) {
  assertOwner(context.owner);
  pending++;update();
  try {
   const result=await api.command(edit.name,edit.params,{guard:context.guard});
   // An acknowledged local edit is safe to continue from in this dialog. A
   // change in another tab still carries the old revision and is refused.
   if(dialog.open&&dialogOwner===context.owner)for(const form of dialog.querySelectorAll('form'))form._context=capture();
   return result;
  }
  finally {pending--;update();}
 }
 function update() {
  const state=api.state(),context=scopeContext(state,api.instance?.()),r=context.recording;
  if(context.owner!==owner) {
   owner=context.owner;rangeGuard=null;rangeOwner=null;
   root.innerHTML=`<div class="scope-source-row"><label>Scope source<select id="scope-source" aria-label="Scope source"></select></label><button id="scope-settings-button">Traces & interval</button><button id="recording-analysis-button" class="hidden">Analyze recording</button><button id="recording-keep" class="hidden"></button></div><div id="scope-capture-status" class="scope-capture-status hidden" role="status"><span id="scope-capture-label"></span><progress id="scope-capture-progress" max="1" aria-label="Recording progress"></progress><button id="scope-capture-cancel">Cancel recording</button></div><div id="recording-player" class="recording-player hidden"><div class="recording-playback"><button id="recording-previous" aria-label="Previous recorded sample" title="Previous recorded sample">‹</button><button id="recording-play" aria-label="Play recording">Play</button><button id="recording-next" aria-label="Next recorded sample" title="Next recorded sample">›</button><input id="recording-position" type="range" min="0" step="1" aria-label="Recorded sample position"><span id="recording-position-label"></span></div><details id="recording-exact"><summary>Position & speed</summary><div class="recording-exact-fields"><form data-scope-edit="index"><label>Sample index<input name="index" type="number" min="0" step="1" required></label><button>Go</button></form><form data-scope-edit="seek"><label>Seek time<input name="time_s" type="text" required></label><button>Seek</button></form><form data-scope-edit="speed"><label>Playback speed<input name="speed" type="number" min="0.001" max="10" step="any" required aria-describedby="recording-speed-unit"></label><button>Apply</button></form></div><button type="button" id="recording-reset-entries">Use current position</button><p id="recording-speed-unit">Speed is simulated seconds per real second. Playback leaves the live circuit unchanged.</p></details><p id="recording-edit-feedback" class="settings-feedback error" role="status"></p><div id="recording-sample-readings" class="recording-sample-readings"></div></div>`;
  }
  const completed=(state.operations||[]).filter(o=>o.status==='completed'),source=$('scope-source');
  const options='<option value="">Live circuit</option>'+completed.map(o=>`<option value="${esc(o.id)}">${esc(o.id)}${o.kept_in_project?' · kept':''}</option>`).join('');
  if(source.innerHTML!==options)source.innerHTML=options;
  source.value=r?.id||'';source.disabled=pending>0;
  const operation=(state.operations||[]).find(o=>['queued','running','canceling'].includes(o.status));
  $('scope-capture-status').classList.toggle('hidden',!operation);
  if(operation){$('scope-capture-label').textContent=`${operation.id} · ${operation.status} · ${Math.round((operation.progress||0)*100)}%`;$('scope-capture-progress').value=operation.progress||0;$('scope-capture-cancel').disabled=pending>0||operation.status==='canceling';$('scope-capture-cancel').dataset.operation=operation.id;}
  $('recording-player').classList.toggle('hidden',!r);
  $('recording-keep').classList.toggle('hidden',!r);
  if(r) {
   const kept=completed.find(o=>o.id===r.id)?.kept_in_project||false;
   $('recording-keep').textContent=kept?'Kept in project':'Keep in project';
   $('recording-keep').setAttribute('aria-pressed',String(kept));$('recording-keep').disabled=pending>0;
   $('recording-previous').disabled=pending>0||r.sample_index<=0;
   $('recording-next').disabled=pending>0||r.sample_index>=r.sample_count-1;
   $('recording-play').disabled=pending>0||r.duration_s===0;
   $('recording-play').textContent=r.playing?'Pause':'Play';$('recording-play').setAttribute('aria-label',r.playing?'Pause recording':'Play recording');
   const slider=$('recording-position');slider.max=Math.max(0,r.sample_count-1);slider.disabled=pending>0||r.sample_count<2;
   // Do not pull a scrubber out from under an active pointer/keyboard draft.
   if(!rangeGuard)slider.value=r.sample_index;
   slider.setAttribute('aria-valuetext',`Sample ${r.sample_index} of ${r.sample_count-1}, ${api.time(r.time_s)}`);
   $('recording-position-label').textContent=`${api.time(r.time_s)} · ${r.sample_index} / ${r.sample_count-1}`;
   for(const [name,value] of [['index',r.sample_index],['time_s',r.time_s],['speed',r.seconds_per_wall_second||.1]]) {
    const input=root.querySelector(`input[name="${name}"]`);
    if(document.activeElement!==input&&!input.closest('form')._draft)input.value=value;
   }
   root.querySelector('input[name="index"]').max=r.sample_count-1;
   $('recording-sample-readings').textContent=(r.readings||[]).map(v=>`${v.probe.label} ${api.fmt(v.value,v.probe.unit,5)}`).join(' · ')||'No captured traces selected.';
  }
  analysisUI.update();
  if(dialog.open) {
   const changed=dialogOwner!==context.owner;
   $('scope-settings-stale').classList.toggle('hidden',!changed);
   for(const form of dialog.querySelectorAll('form'))for(const el of form.elements)el.disabled=changed||pending>0||(form.dataset.scopeEdit==='window'&&r?.duration_s===0&&el.type==='submit');
   if(!changed)renderStatistics(state);
  }
 }
 function renderStatistics(state) {
  const {scope,recording:r}=scopeContext(state),output=$('scope-statistics');
  const stats=scope.statistics||[],probes=scope.available||[];
  const rows=stats.map(s=>{const p=probes.find(p=>p.id===s.probe_id);return `<tr><th scope="row">${esc(p?.label||s.probe_id)}</th><td>${s.count}</td>${[s.min,s.max,s.sample_mean,s.sample_rms].map(v=>`<td>${esc(api.fmt(v,p?.unit,5))}</td>`).join('')}</tr>`;}).join('');
  const html=`<p class="field-note">${esc(scope.window_reason||'Statistics use exact samples in the visible interval. Mean and RMS here are sample-weighted.')}</p>${rows?`<div class="result-table-wrap"><table class="result-table"><thead><tr><th>Trace</th><th>Samples</th><th>Min</th><th>Max</th><th>Sample mean</th><th>Sample RMS</th></tr></thead><tbody>${rows}</tbody></table></div>`:''}${r?.interval_statistics?`<p class="field-note">Time-weighted interval · ${esc(r.interval_statistics.probe.label)} · ${esc(api.time(r.interval_statistics.settings.start_s))} – ${esc(api.time(r.interval_statistics.settings.end_s))}</p><div class="interval-measures">${[['RMS',r.interval_statistics.time_rms],['AC RMS',r.interval_statistics.ac_rms],['Mean',r.interval_statistics.time_mean],['Peak to peak',r.interval_statistics.peak_to_peak]].map(([name,v])=>`<span>${name}<strong>${esc(api.fmt(v,r.interval_statistics.probe.unit,5))}</strong></span>`).join('')}</div>`:''}`;
  if(output.innerHTML!==html)output.innerHTML=html;
 }
 function showSettings() {
  const state=api.state(),{scope,recording:r,owner:nextOwner}=scopeContext(state,api.instance?.());
  dialogState=state;dialogOwner=nextOwner;
  const guard=api.guards(),window=scope.window||{start_s:0,end_s:0},probes=scope.available||[];
  const selected=scope.selected||[],limit=state.limits?.scope_traces||6;
  $('scope-settings-title').textContent=r?'Recorded traces & interval':'Live traces & interval';
  $('scope-settings-content').innerHTML=`<p class="scope-source-note">${r?'Measurements and playback use saved samples. The schematic and component inspector show the live circuit.':'Measurements follow accepted live samples. Fixed windows can expire as rolling history advances.'}</p><div class="scope-settings-grid"><section><h3>Measurements</h3><form data-scope-edit="probes" class="settings-form"><p class="field-note">Choose up to ${limit} ${r?'captured':'available'} traces.</p><div class="scope-probe-list">${probes.map(p=>`<label class="checkbox-label"><input type="checkbox" name="probe" value="${esc(p.id)}" ${selected.includes(p.id)?'checked':''}>${esc(p.label)} · ${esc(p.unit)}</label>`).join('')}</div><button class="primary">Apply traces</button><p class="settings-feedback error" role="status"></p></form></section><section><h3>Time window</h3><form data-scope-edit="window" class="settings-form"><label>Start time<input name="start_s" value="${window.start_s}" required></label><label>End time<input name="end_s" value="${window.end_s}" required></label><p class="field-note">Accepts seconds or units such as 20 ms. ${r?'Only stored samples in the interval are plotted.':'Fixed intervals must be inside retained live history.'}</p><button class="primary" ${r?.duration_s===0?'disabled':''}>Apply interval</button><button type="button" id="scope-window-reset">${r?'Show complete recording':'Follow live history'}</button><p class="settings-feedback error" role="status"></p></form><h3 class="scope-cursor-heading">Exact cursors</h3>${['a','b'].map(c=>`<form data-scope-edit="cursor-${c}" class="scope-cursor-form"><label>Cursor ${c.toUpperCase()}<input name="time_s" value="${scope[c]?.requested_s??scope[c]?.time_s??window.start_s}" required></label><button>Set ${c.toUpperCase()}</button><p class="settings-feedback error" role="status"></p></form>`).join('')}</section></div>${r&&r.duration_s>0?`<section class="section"><h3>Time-weighted interval measurement</h3><form data-scope-edit="statistics" class="analysis-toolbar"><label>Captured trace<select name="probe_id">${probes.map(p=>`<option value="${esc(p.id)}">${esc(p.label)}</option>`).join('')}</select></label><label>Start time<input name="start_s" value="${window.start_s}" required></label><label>End time<input name="end_s" value="${window.end_s}" required></label><button class="primary">Measure interval</button><p class="settings-feedback error" role="status"></p></form></section>`:''}<section class="section"><h3>Accepted measurements</h3><div id="scope-statistics"></div></section>`;
  for(const form of dialog.querySelectorAll('form'))form._context={guard,owner:dialogOwner};
  $('scope-settings-stale').classList.add('hidden');renderStatistics(state);dialog.showModal();
 }
 root.addEventListener('focusin',event=>{const form=event.target.closest('form');if(form&&!form._draft)form._context=capture();});
 root.addEventListener('input',event=>{const form=event.target.closest('form');if(form)form._draft=true;if(event.target.id==='recording-position'&&!rangeGuard){const ctx=capture();rangeGuard=ctx.guard;rangeOwner=ctx.owner;}});
 root.addEventListener('pointerdown',event=>{if(event.target.id==='recording-position'){const ctx=capture();rangeGuard=ctx.guard;rangeOwner=ctx.owner;}});
 root.addEventListener('pointercancel',()=>{rangeGuard=null;rangeOwner=null;update();});
 root.addEventListener('change',async event=>{
  try {
   if(event.target.id==='scope-source') {pending++;try {if(event.target.value)await api.viewRecording(event.target.value);else await api.command('recording.view.set',{mode:'live'},{guard:api.guards()});}finally {pending--;update();}}
   if(event.target.id==='recording-position') {const ctx={guard:rangeGuard||api.guards(),owner:rangeOwner||scopeContext(api.state(),api.instance?.()).owner};rangeGuard=null;rangeOwner=null;await run(scopeEdit('index',{index:event.target.value},api.state()),ctx);}
  }catch(error){api.toast(error.message,true);update();}
 });
 root.addEventListener('click',async event=>{
  const button=event.target.closest('button');if(!button||button.disabled)return;
  try {
   if(button.id==='scope-settings-button'){showSettings();return;}
   if(button.id==='recording-reset-entries'){for(const form of root.querySelectorAll('form')){form._draft=false;form._context=capture();}$('recording-edit-feedback').textContent='';update();return;}
   if(button.id==='scope-capture-cancel'){await run({name:'operation.cancel',params:{id:button.dataset.operation}});return;}
   const r=scopeContext(api.state(),api.instance?.()).recording;if(!r)return;
   if(button.id==='recording-play')await run({name:r.playing?'recording.pause':'recording.play',params:r.playing?{}:{seconds_per_wall_second:r.seconds_per_wall_second||.1}});
   if(button.id==='recording-previous'||button.id==='recording-next')await run({name:'recording.step',params:{direction:button.id==='recording-next'?'next':'previous'}});
   if(button.id==='recording-keep')await run({name:'recording.keep.set',params:{id:r.id,keep:button.getAttribute('aria-pressed')!=='true'}});
  }catch(error){api.toast(error.message,true);}
 });
 async function submit(event) {
  const form=event.target;if(!form.dataset.scopeEdit)return;event.preventDefault();
  const feedback=form.querySelector('.settings-feedback')||(root.contains(form)?$('recording-edit-feedback'):null),button=event.submitter;
  try {
   if(!form.reportValidity())return;
   if(button)button.disabled=true;
   const values=Object.fromEntries(new FormData(form));
   if(form.dataset.scopeEdit==='probes')values.ids=[...form.querySelectorAll('input[name="probe"]:checked')].map(el=>el.value);
   const edit=scopeEdit(form.dataset.scopeEdit,values,dialog.contains(form)?dialogState:api.state());
   await run(edit,form._context||capture());
   form._context=capture();form._draft=false;if(feedback){feedback.textContent=root.contains(form)?'':'Applied.';feedback.classList.remove('error');}
   document.activeElement?.blur();
  }catch(error){if(feedback){feedback.textContent=error.message+(root.contains(form)?' Use current position to refresh these entries.':'');feedback.classList.add('error');}else api.toast(error.message,true);}
  finally {if(button)button.disabled=false;update();}
 }
 root.addEventListener('submit',submit);dialog.addEventListener('submit',submit);
 dialog.addEventListener('click',async event=>{if(event.target.id!=='scope-window-reset')return;
  const {recording:r}=scopeContext(dialogState),form=event.target.closest('form');
  try {await run({name:r?'recording.window.set':'scope.window.set',params:r?{start_s:0,end_s:r.last_sample_time_s}:{mode:'live'}},form._context);const w=scopeContext(api.state(),api.instance?.()).scope.window;for(const side of ['start_s','end_s'])form.elements[side].value=w[side];form._context=capture();form.querySelector('.settings-feedback').textContent='Applied.';form.querySelector('.settings-feedback').classList.remove('error');}
  catch(error){form.querySelector('.settings-feedback').textContent=error.message;form.querySelector('.settings-feedback').classList.add('error');}
 });
 return {update,showSettings};
}

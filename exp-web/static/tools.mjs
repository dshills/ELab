import {resultTable} from './results.mjs';
import {commandSchema,schemaFields,readSchemaForm,bindSchemaArrays,escapeHTML as esc} from './forms.mjs';
export function workbenchTools(api){
 const dialog=document.getElementById('tools-dialog'),content=document.getElementById('tools-content');let section='analysis',active,activeSchema,playerID;
 const groups={
  analysis:[['tolerance.run','Tolerance / Monte Carlo'],['spectrum.analyze','Recorded FFT & distortion'],['spectrum.live.set','Live FFT settings'],['spectrum.cursor.set','Frequency cursors & peaks'],['sweep.start','DC, AC, noise & loop sweeps'],['sweep.select','Select a retained sweep'],['sweep.cancel','Cancel a sweep'],['scope.window.set','Live scope time window'],['scope.cursor.set','Exact scope cursor time'],['sweep.keep.set','Keep selected sweep in project'],['sweep.cursor.set','Exact sweep point cursor']],
  recordings:[['recording.view.set','Live / recorded view'],['recording.seek','Seek a captured sample'],['recording.cursor.set','Exact recorded cursor time'],['recording.play','Play recording'],['recording.pause','Pause recording'],['recording.step','Step recorded rows'],['recording.window.set','Recording time window'],['recording.statistics.set','Interval RMS & statistics'],['recording.step-response.set','Overshoot & settling'],['recording.compare.set','Compare two recordings'],['recording.timing.set','Edge timing & duty cycle'],['recording.keep.set','Keep / remove saved membership'],['operation.cancel','Cancel a capture']],
  display:[['display.preferences.set','Voltage colors & current cues'],['measurement.pin.add','Pin a measurement'],['measurement.pin.move','Move a pinned measurement'],['measurement.pin.remove','Remove a pinned measurement'],['scope.probes.set','Live scope measurements'],['recording.probes.set','Recorded measurements']],
  blocks:[['block.create','Create a reusable circuit block'],['block.copy','Copy block'],['block.paste','Paste block'],['block.parameter.set','Published block parameters'],['block.view.set','Expand / collapse internals'],['block.release','Release block into its components'],['wire.junction.insert','Insert an explicit junction']],
  audio:[['audio.render','Render an independent audio preview'],['audio.stop','Cancel audio rendering']],
  export:[['export.page.set','Paper, scale & page setup']],
 };
 function initial(name){const s=api.state();const id=s.selected_operation||s.last_operation||'';const probe=s.recording?.scope?.selected?.[0]||s.scope.selected[0]||'';switch(name){
  case 'tolerance.run':return s.tolerance?.settings||{};
  case 'spectrum.analyze':return {...s.spectrum?.settings,id};
  case 'spectrum.live.set':return s.spectrum?.live?.settings||{enabled:true,probe_id:probe,window:'hann'};
  case 'scope.window.set':return s.scope.window;
  case 'recording.statistics.set':return {id,probe_id:probe,start_s:0,end_s:s.recording?.duration_s||.1};
  case 'display.preferences.set':return s.display.preferences;
  case 'scope.probes.set':return {ids:s.scope.selected};
  case 'recording.probes.set':return {ids:s.recording?.scope?.selected||[]};
  case 'audio.render':return {id:s.components.some(p=>p.id===s.selected&&['speaker','piezo','active-buzzer'].includes(p.kind))?s.selected:s.components.find(p=>['speaker','piezo','active-buzzer'].includes(p.kind))?.id,start_s:0,duration_s:.02,gain:.1};
  case 'audio.stop':return {id:s.audio_preview?.id};
  case 'recording.compare.set':return {id,reference_id:s.operations.find(o=>o.id!==id&&o.status==='completed')?.id,time_s:0,show_overlay:true,show_statistics:true};
  case 'recording.step-response.set':return {id,probe_id:probe,start_s:0,end_s:s.recording?.duration_s||.1,initial_value:0,final_value:5,tolerance_fraction:.02,minimum_hold_s:.001};
  case 'spectrum.cursor.set':return {source:s.spectrum?.live?.settings?.enabled?'live':'recorded',cursor:'a',peak:1};
  case 'recording.cursor.set':case 'scope.cursor.set':return {cursor:'a',time_s:0};
  case 'sweep.select':case 'sweep.cancel':case 'sweep.keep.set':return {id:s.dc_sweep?.selected_operation||s.dc_sweep?.operations?.at(-1)?.id,keep:true};
  case 'recording.keep.set':return {id,keep:true};
  case 'recording.window.set':return {start_s:0,end_s:s.recording?.duration_s||.1};
  case 'recording.seek':return {time_s:0};
  case 'recording.timing.set':return {id,probe_id:probe,level:'2.5',direction:'rising',start_s:0,end_s:s.recording?.duration_s||.1,limit:20};
  case 'recording.play':return {seconds_per_wall_second:.02};
  case 'recording.step':return {direction:'next'};
  case 'recording.view.set':return {mode:'recorded'};
  case 'sweep.start':return {settings:s.dc_sweep?.settings};
  case 'export.page.set':return s.schematic_page;
  default:return {id:s.selected};
 }}
 function releaseAudio(){const player=document.getElementById('browser-audio');if(player?.src){player.pause();URL.revokeObjectURL(player.src);player.removeAttribute('src');}playerID=null;}
 dialog.addEventListener('close',releaseAudio);
 function show(next='analysis'){section=next;active=null;render();if(!dialog.open)dialog.showModal();}
 function render(){
  releaseAudio();
  const choices=(groups[section]||[]).filter(([name])=>api.catalog().commands.some(c=>c.id===name));
  content.innerHTML=`<div class="tools-navigation">${Object.keys(groups).map(key=>`<button data-tool-section="${key}" class="${key===section?'active':''}">${esc(key[0].toUpperCase()+key.slice(1))}</button>`).join('')}</div><div class="tools-choices">${choices.map(([name,title])=>`<button data-workbench-command="${name}">${esc(title)}</button>`).join('')}</div><form id="workbench-tool-form" class="settings-form hidden"></form><div id="tool-result" class="tool-results"></div>`;
  if(section==='export')content.insertAdjacentHTML('beforeend',`<div class="transfer-buttons">${['svg','pdf','png','csv','recording-json','recording-csv','sweep-json','sweep-csv','spectrum-json','spectrum-csv','tolerance-json','tolerance-csv','audio-wav'].map(kind=>`<button data-export-kind="${kind}">Download ${esc(kind.replaceAll('-',' '))}</button>`).join('')}<button data-open-print>Print schematic</button><button data-import-kind="recording">Import recording JSON</button><button data-import-kind="sweep">Import sweep JSON</button></div>`);
  if(section==='audio')content.insertAdjacentHTML('beforeend','<p class="muted">Choose a speaker, piezo, or buzzer before rendering. Playback is explicit and uses a separate captured experiment.</p><button data-load-audio class="primary">Load completed preview</button><div class="selection-actions"><button data-play-audio disabled>Play preview</button><button data-stop-audio disabled>Stop playback</button></div><audio id="browser-audio" controls hidden></audio>');
  if(section==='recordings')content.insertAdjacentHTML('beforeend',`<div class="retained-results">${(api.state().operations||[]).map(o=>`<button data-result-id="${esc(o.id)}">${esc(o.id+' · '+o.status+(o.kept_in_project?' · kept':''))}</button>`).join('')}</div>`);
 }
 content.onclick=async event=>{const sectionButton=event.target.closest('[data-tool-section]');if(sectionButton){section=sectionButton.dataset.toolSection;active=null;render();return;}
  const chosen=event.target.closest('[data-workbench-command]');try{
   if(chosen){active=api.catalog().commands.find(c=>c.id===chosen.dataset.workbenchCommand);activeSchema=commandSchema(active,api.state(),api.catalog().schemas);const form=document.getElementById('workbench-tool-form');form.classList.remove('hidden');form.innerHTML=`<h3>${esc(chosen.textContent)}</h3>${schemaFields(activeSchema,initial(active.id),api.catalog().schemas)}<p class="field-note">${esc(active.description)}</p><button class="primary" type="submit">Apply</button><p class="settings-feedback" role="status"></p>`;form._guard=api.guards();return;}
   const exp=event.target.closest('[data-export-kind]');if(exp){await api.download(exp.dataset.exportKind);return;}
   if(event.target.closest('[data-open-print]')){window.open('/print.html','_blank','noopener');return;}
   if(event.target.closest('[data-play-audio]')){await document.getElementById('browser-audio').play();return;}
   if(event.target.closest('[data-stop-audio]')){const player=document.getElementById('browser-audio');player.pause();player.currentTime=0;return;}
   const imp=event.target.closest('[data-import-kind]');if(imp){document.getElementById('result-file').dataset.kind=imp.dataset.importKind;document.getElementById('result-file').click();return;}
   const result=event.target.closest('[data-result-id]');if(result){await api.command('operation.select',{id:result.dataset.resultId});api.toast('Recording selected.');return;}
   if(event.target.closest('[data-load-audio]')){const audio=api.state().audio_preview;if(!audio||audio.status!=='completed'||!audio.source_current)throw new Error('Render a current audio preview first.');const result=await api.command('audio.read',{id:audio.id});const bytes=Uint8Array.from(atob(result.wav_base64),c=>c.charCodeAt(0));const player=document.getElementById('browser-audio');if(player.src)URL.revokeObjectURL(player.src);player.src=URL.createObjectURL(new Blob([bytes],{type:'audio/wav'}));playerID=audio.id;player.hidden=false;update();return;}
  }catch(e){api.toast(e.message,true);}
 };
 content.addEventListener('submit',async event=>{if(event.target.getAttribute('id')!=='workbench-tool-form')return;event.preventDefault();const form=event.target,button=event.submitter;button.disabled=true;try{if(!form.reportValidity())return;const params=readSchemaForm(form,activeSchema,api.catalog().schemas);const result=await api.command(active.id,params,{guard:form._guard,message:'Workbench settings applied.'});form._guard=api.guards();const output=document.getElementById('tool-result');output.innerHTML=active.mutation?'<p>Applied.</p>':resultTable(result);delete output.dataset.signature;update();}catch(e){form.querySelector('.settings-feedback').textContent=e.message;}finally{button.disabled=false;}});
 bindSchemaArrays(content,()=>activeSchema,api.catalog().schemas);
 document.getElementById('result-file').onchange=async event=>{const file=event.target.files[0];if(!file)return;const kind=event.target.dataset.kind;event.target.value='';try{await api.mutate(async()=>{const s=api.state(),query=new URLSearchParams({kind,revision:s.command_revision,generation:s.document.generation});const response=await api.fetch('/api/import-result?'+query,{method:'POST',body:file});if(response.error)throw new Error(response.error.message);await api.refresh();});api.toast('Result imported. Select it in Recordings or Sweep.');}catch(e){api.toast(e.message,true);}};
 function update(){const player=document.getElementById('browser-audio'),audio=api.state()?.audio_preview;if(player?.src&&(!audio||audio.id!==playerID||!audio.source_current)){player.pause();URL.revokeObjectURL(player.src);player.removeAttribute('src');}
  const load=content.querySelector('[data-load-audio]');if(load)load.disabled=!audio||audio.status!=='completed'||!audio.source_current;for(const button of content.querySelectorAll('[data-play-audio],[data-stop-audio]'))button.disabled=!player?.getAttribute('src');
  if(!dialog.open)return;
  const result=document.getElementById('tool-result');if(!result||active?.mutation===false)return;
  const s=api.state(),data=section==='analysis'?(active?.id.startsWith('tolerance.')?s.tolerance:active?.id.startsWith('spectrum.')?s.spectrum:active?.id.startsWith('sweep.')?s.dc_sweep:active?.id.startsWith('scope.')?s.scope:{tolerance:s.tolerance?.status,spectrum:s.spectrum?.live?.status,sweep:s.dc_sweep?.selected_operation}):section==='recordings'?s.recording:section==='audio'?s.audio_preview:null;
  if(data){const signature=JSON.stringify(data);if(result.dataset.signature!==signature){result.dataset.signature=signature;result.innerHTML=resultTable(data);}}
 }
 return {show,update};
}

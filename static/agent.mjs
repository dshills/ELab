export function agentConnection(api){
 const dialog=document.getElementById('agent-dialog');
 const status=document.getElementById('agent-status');
 const enable=document.getElementById('agent-enable'),disable=document.getElementById('agent-disable');
 const configuration=document.getElementById('agent-configuration');
 function render(info){
  status.textContent=info.enabled?'Agent access is enabled for this workspace.':'Agent access is disabled.';
  enable.hidden=!!info.enabled;disable.hidden=!info.enabled;
  configuration.parentElement.hidden=!info.enabled;
  configuration.value=info.configuration?JSON.stringify(info.configuration,null,2):'';
  document.getElementById('agent-last-command').textContent=info.last_command?`Last agent edit: ${info.last_command}`:'';
 }
 function update(info){
  document.getElementById('agent-button').textContent=info?.enabled?'Agent access on':'Connect agent';
  if(dialog.open){status.textContent=info?.enabled?'Agent access is enabled for this workspace.':'Agent access is disabled.';
   document.getElementById('agent-last-command').textContent=info?.last_command?`Last agent edit: ${info.last_command}`:'';
   if(info?.enabled&&!configuration.value)api.fetch('/api/agent').then(render).catch(e=>api.toast(e.message,true));
   if(!info?.enabled){configuration.value='';configuration.parentElement.hidden=true;enable.hidden=false;disable.hidden=true;}}
 }
 async function show(){try{render(await api.fetch('/api/agent'));if(!dialog.open)dialog.showModal();}catch(e){api.toast(e.message,true);}}
 async function set(enabled){
  enable.disabled=disable.disabled=true;
  try{await api.mutate(async()=>{render(await api.fetch('/api/agent',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled})}));await api.refresh();});}
  catch(e){api.toast(e.message,true);}finally{enable.disabled=disable.disabled=false;}
 }
 enable.onclick=()=>set(true);disable.onclick=()=>set(false);
 return {show,update};
}

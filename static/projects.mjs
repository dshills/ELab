import {syncTabs} from './accessibility.mjs';
const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export function projectBrowser(api){
 const dialog=document.getElementById('projects-dialog'),body=document.getElementById('projects-content');
 let tab='projects',listing,saveResolve;
 async function request(action,params={}){
  return api.mutate(async()=>{const query=new URLSearchParams({revision:api.state().command_revision,generation:api.state().document.generation});const result=await api.fetch('/api/projects?'+query,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action,...params})});if(result.error)throw new Error(result.error.message||result.error);await api.refresh();return result;});
 }
 async function save(copy=false){
  const current=api.workspace();
  if(current.project_id&&!copy){try{await request('save',{id:current.project_id,name:current.name});api.toast('Project saved locally.');return true;}catch(e){await api.refresh();api.toast(e.message,true);return false;}}
  const saveDialog=document.getElementById('save-project-dialog');document.getElementById('project-name').value=current.name||api.state().example?.info?.title||'Untitled circuit';
  saveDialog.showModal();document.getElementById('project-name').focus();return new Promise(resolve=>{saveResolve=resolve;});
 }
 document.getElementById('save-project-dialog').addEventListener('close',()=>{saveResolve?.(false);saveResolve=null;});
 document.getElementById('save-project-form').onsubmit=async event=>{event.preventDefault();const button=event.submitter;button.disabled=true;try{await request('save',{name:document.getElementById('project-name').value});saveResolve?.(true);saveResolve=null;document.getElementById('save-project-dialog').close();api.toast('Project saved locally.');}catch(e){document.getElementById('save-project-error').textContent=e.message;}finally{button.disabled=false;}};
 function render(){
  document.querySelectorAll('[data-project-tab]').forEach(el=>el.setAttribute('aria-selected',el.dataset.projectTab===tab));syncTabs();
  const items=tab==='history'?listing.history:listing.projects.filter(p=>Boolean(p.trashed)===(tab==='trash'));
  body.innerHTML=items.length?items.map(p=>`<article class="project-card"><div><strong>${esc(p.name)}</strong><p>${esc(new Date(p.updated_at).toLocaleString())}</p></div><button data-project-action="${tab==='history'?'recover':tab==='trash'?'untrash':'open'}" data-project-id="${p.id}" data-checkpoint="${p.checkpoint?'true':'false'}" data-sha="${p.sha256}">${tab==='history'?'Recover a copy':tab==='trash'?'Restore':'Open'}</button>${tab==='projects'?`<button data-project-action="trash" data-project-id="${p.id}" data-checkpoint="${p.checkpoint?'true':'false'}" data-sha="${p.sha256}">Move to Trash</button>`:''}</article>`).join(''):'<p class="muted">'+(tab==='history'?'Recovery copies appear before replacing or saving a circuit.':'No projects here yet. Save a project to keep it in this library.')+'</p>';
 }
 async function show(next='projects'){tab=next;try{listing=await api.fetch('/api/projects');render();if(!dialog.open)dialog.showModal();}catch(e){api.toast(e.message,true);}}
 document.querySelectorAll('[data-project-tab]').forEach(el=>el.onclick=()=>{tab=el.dataset.projectTab;render();});
 body.onclick=async event=>{const button=event.target.closest('[data-project-action]');if(!button)return;try{if(['open','recover'].includes(button.dataset.projectAction)&&!await api.confirm('Open this project in place of the current circuit?'))return;await request(button.dataset.projectAction,{id:button.dataset.projectId,discard_changes:true,checkpoint:button.dataset.checkpoint==='true',sha256:button.dataset.sha});if(['open','recover'].includes(button.dataset.projectAction))dialog.close();else await show(tab);}catch(e){api.toast(e.message,true);}};
 return {show,save};
}

export function freeSelectionOffset(positions,occupied,step=100){
 const used=new Set(occupied.map(p=>`${p.x},${p.y}`));
 for(let n=1;n<=100;n++)for(const [x,y] of [[1,1],[-1,1],[1,-1],[-1,-1],[1,0],[0,1],[-1,0],[0,-1]]){
 const offset={x:x*n*step,y:y*n*step};if(positions.every(p=>Math.abs(p.x+offset.x)<=10000&&Math.abs(p.y+offset.y)<=10000&&!used.has(`${p.x+offset.x},${p.y+offset.y}`)))return offset;
 }return null;
}
export function selectionEditor(api){
 let selected=new Set(),generation,clipboardID=0,clipboardPositions=[];
 const bar=document.getElementById('selection-toolbar'),label=document.getElementById('selection-count');
 function ids(){return [...selected].filter(id=>api.scene()?.parts.some(p=>p.id===id));}
 function render(){const count=ids().length;bar.hidden=count<2&&!clipboardID;label.textContent=count?`${count} component${count===1?'':'s'} selected`:'Selection clipboard ready';for(const button of bar.querySelectorAll('[data-selection]'))button.disabled=button.dataset.selection==='paste'?!clipboardID:!count;}
 function rootID(id){let owner;const seen=new Set();while((owner=api.state().components.find(p=>p.block?.members?.includes(id)))&&!seen.has(id)){seen.add(id);id=owner.id;}return id;}
 function set(next){const normalized=new Set([...next].map(rootID));if(normalized.size===selected.size&&[...normalized].every(id=>selected.has(id)))return;selected=normalized;render();api.redraw();}
 function select(id,add=false){id=rootID(id);if(!api.scene()?.parts.some(p=>p.id===id)){set([]);return;}const next=add?new Set(selected):selected.has(id)?new Set(selected):new Set();if(add&&next.has(id))next.delete(id);else next.add(id);set(next);}
 async function edit(kind,offset={x:80,y:80},guard){
  const chosen=ids();if(kind!=='paste'&&!chosen.length)return;
  const oldIDs=new Set(api.state().components.map(p=>p.id));const originalPositions=api.scene().parts.filter(p=>chosen.includes(p.id)).map(p=>({...p.position}));if(kind==='duplicate'){offset=freeSelectionOffset(originalPositions,api.scene().parts.map(p=>p.position),80);if(!offset){api.toast('There is no free position for this group. Move some parts first.',true);return;}}let params;
  if(kind==='paste')params={clipboard_id:clipboardID,offset};else params=kind==='copy'||kind==='delete'?{ids:chosen}:{ids:chosen,offset};
  const commandName='selection.'+kind;
  const result=await api.command(commandName,params,guard?{guard}:{});
  if(kind==='copy'){clipboardID=result.selection_clipboard_id;clipboardPositions=originalPositions;api.toast('Selection copied with its internal wires.');}
  if(kind==='duplicate'||kind==='paste')set(api.scene().parts.filter(p=>!oldIDs.has(p.id)).map(p=>p.id));
  if(kind==='delete')set([]);render();return result;
 }
 async function paste(){const offset=freeSelectionOffset(clipboardPositions,api.scene().parts.map(p=>p.position));if(!offset){api.toast('There is no free position for this group. Move some parts first.',true);return;}return edit('paste',offset);}
 bar.onclick=e=>{const kind=e.target.closest('[data-selection]')?.dataset.selection;if(kind)(kind==='paste'?paste():edit(kind)).catch(()=>{});};
 document.getElementById('selection-clear').onclick=()=>{set([]);api.command('component.select',{id:''}).catch(()=>{});};
 return {ids,has:id=>selected.has(id),select,set,edit,paste,clear:()=>set([]),render,
 update(state){if(generation!==undefined&&generation!==state.document.generation)selected.clear();generation=state.document.generation;selected=new Set(ids());if(!selected.size&&api.scene()?.parts.some(p=>p.id===state.selected))selected.add(state.selected);render();},
 box(start,end,initial=[]){const x0=Math.min(start.x,end.x),x1=Math.max(start.x,end.x),y0=Math.min(start.y,end.y),y1=Math.max(start.y,end.y);set([...initial,...api.scene().parts.filter(p=>p.bounds[0]>=x0&&p.bounds[2]<=x1&&p.bounds[1]>=y0&&p.bounds[3]<=y1).map(p=>p.id)]);}
 };
}

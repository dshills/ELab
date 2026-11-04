// Keep drawing shortcuts out of text composition and browser/OS shortcuts.
export function componentName(part) {
 return [...(part.labels||[]).map(label=>label.text),part.id,part.value_text,part.name].filter(Boolean).join(' ');
}

export function schematicFocus(root) {
 const active=root.ownerDocument.activeElement;
 if(!root.contains(active))return null;
 const object=active.closest('[data-object]'),pin=active.closest('[data-pin]');
 return object?{kind:'object',id:object.dataset.object}:pin?{kind:'pin',id:pin.dataset.pin}:null;
}

export function restoreSchematicFocus(root,focus) {
 if(!focus)return;
 const selector=focus.kind==='object'?'[data-object]':'[data-pin]';
 const item=[...root.querySelectorAll(selector)].find(node=>node.dataset[focus.kind]===focus.id);
 const target=focus.kind==='pin'?item?.querySelector('.pin-hit'):item;
 // A removed part or a terminal leaving wire mode returns focus to the sheet.
 (target&&target.tabIndex>=0?target:root).focus();
}

export function workbenchKey(event, modalOpen=false) {
 if(event.defaultPrevented||event.isComposing||event.keyCode===229||event.altKey||modalOpen)return null;
 const key=event.key.toLowerCase(),mod=event.metaKey||event.ctrlKey;
 if(mod&&['a','c','v','d'].includes(key)&&!event.target.closest('#schematic'))return null;
 if(mod&&!['k','s','o','z','a','c','v','d'].includes(key))return null;
 if(event.target.closest('input,textarea,select,[contenteditable]:not([contenteditable="false"])')&&!(mod&&key==='k'))return null;
 return key;
}

export function nextTabIndex(key,index,count,vertical=false) {
 if(key==='Home')return 0;
 if(key==='End')return count-1;
 const previous=vertical?'ArrowUp':'ArrowLeft',next=vertical?'ArrowDown':'ArrowRight';
 if(key===previous)return (index+count-1)%count;
 if(key===next)return (index+1)%count;
 return null;
}

export function syncTabs(root=document) {
 for(const list of root.querySelectorAll('[role="tablist"]')) {
  const tabs=[...list.querySelectorAll('[role="tab"]')];
  const selected=tabs.find(tab=>tab.getAttribute('aria-selected')==='true')||tabs[0];
  for(const tab of tabs)tab.tabIndex=tab===selected?0:-1;
  const panel=root.getElementById?.(selected?.getAttribute('aria-controls'));
  if(panel)panel.setAttribute('aria-labelledby',selected.id);
 }
}

export function bindTabs(root=document) {
 syncTabs(root);
 root.addEventListener('keydown',event=>{
  if(event.defaultPrevented||event.isComposing||event.altKey||event.ctrlKey||event.metaKey)return;
  const tab=event.target.closest('[role="tab"]'),list=tab?.closest('[role="tablist"]');
  if(!list)return;
  const tabs=[...list.querySelectorAll('[role="tab"]')].filter(item=>!item.disabled);
  const index=nextTabIndex(event.key,tabs.indexOf(tab),tabs.length,list.getAttribute('aria-orientation')==='vertical');
  if(index===null||!tabs.length)return;
  event.preventDefault();
  tabs[index].focus();
  tabs[index].click();
  syncTabs(root);
 });
 root.addEventListener('click',event=>{if(event.target.closest('[role="tab"]'))syncTabs(root);});
}

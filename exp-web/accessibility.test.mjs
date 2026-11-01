import {test} from 'node:test';
import assert from 'node:assert/strict';
import {componentName,workbenchKey,nextTabIndex,syncTabs,bindTabs} from './static/accessibility.mjs';

test('component spoken names contain the complete visible label in drawing order',()=>{
 assert.equal(componentName({id:'R1',name:'Resistor',value_text:'10 kΩ',labels:[{text:'R'},{text:'+'}]}),'R + R1 10 kΩ Resistor');
 assert.equal(componentName({id:'G1',name:'Ground'}),'G1 Ground');
});

const event=(key,options={})=>({key,target:{closest:()=>null},...options});
test('drawing shortcuts preserve browser modifiers, composition and text editing',()=>{
 for(const key of ['w','r','h','f','v','Backspace'])assert.equal(workbenchKey(event(key,{ctrlKey:true})),null);
 for(const options of [{isComposing:true},{keyCode:229},{altKey:true},{defaultPrevented:true}])assert.equal(workbenchKey(event('w',options)),null);
 assert.equal(workbenchKey(event('w'),true),null);
 assert.equal(workbenchKey(event('z',{metaKey:true,target:{closest:()=>({})}})),null);
 assert.equal(workbenchKey(event('k',{metaKey:true,target:{closest:()=>({})}})),'k');
 assert.equal(workbenchKey(event('Z',{metaKey:true,shiftKey:true})),'z');
 assert.equal(workbenchKey(event('w')),'w');
});
test('tabs wrap, support Home/End and leave scrolling keys alone',()=>{
 assert.equal(nextTabIndex('ArrowLeft',0,3),2);
 assert.equal(nextTabIndex('ArrowRight',2,3),0);
 assert.equal(nextTabIndex('Home',2,3),0);
 assert.equal(nextTabIndex('End',0,3),2);
 assert.equal(nextTabIndex('ArrowDown',0,3),null);
 assert.equal(nextTabIndex('ArrowDown',0,3,true),1);
 assert.equal(nextTabIndex('Enter',0,3),null);
});
test('delegated tabs activate through the existing click handler and maintain one tab stop',()=>{
 const listeners={},panel={setAttribute:(key,value)=>panel[key]=value};
 const tabs=[0,1,2].map(index=>({id:'tab-'+index,selected:index===0,focus(){this.focused=true;},click(){tabs.forEach(tab=>tab.selected=tab===this);},getAttribute(key){return key==='aria-selected'?String(this.selected):'panel';},closest(selector){return selector==='[role="tablist"]'?list:this;}}));
 const list={querySelectorAll:()=>tabs,getAttribute:()=>null};
 const root={querySelectorAll:()=>[list],getElementById:()=>panel,addEventListener:(key,handler)=>listeners[key]=handler};
 bindTabs(root);
 assert.deepEqual(tabs.map(tab=>tab.tabIndex),[0,-1,-1]);
 let prevented=false;
 listeners.keydown({key:'ArrowRight',target:tabs[0],preventDefault(){prevented=true;}});
 assert.ok(prevented&&tabs[1].focused&&tabs[1].selected);
 assert.deepEqual(tabs.map(tab=>tab.tabIndex),[-1,0,-1]);
 assert.equal(panel['aria-labelledby'],'tab-1');
 tabs[2].click();syncTabs(root);
 assert.deepEqual(tabs.map(tab=>tab.tabIndex),[-1,-1,0]);
});

import {test} from 'node:test';
import assert from 'node:assert/strict';
import {mergeSceneFrame,sceneStructureKey} from './static/scene-frame.mjs';
const part=(id,x=0)=>({id,kind:'resistor',position:{x,y:0},bounds:[x-10,-10,x+10,10],rotation:0,paths:['static'],labels:null,value_text:'1k'});
test('live frame preserves geometry identity and refreshes actual control paths and display segments',()=>{
 const resistor=part('R1'),button={...part('S1'),dynamic:true,paths:['open']},display={...part('D1'),dynamic:true,lit:[true,false]};const base={parts:[resistor,button,display],wires:[],bounds:[0,0,100,100]};
 const next=mergeSceneFrame(base,{markers:[{wire_id:'W1'}],parts:[{id:'S1',paths:['closed']},{id:'D1',lit:[false,true]}]});
 assert.equal(next.parts[0],resistor);assert.equal(next.wires,base.wires);assert.equal(next.bounds,base.bounds);assert.deepEqual(next.parts[1].paths,['closed']);assert.deepEqual(next.parts[2].lit,[false,true]);assert.deepEqual(button.paths,['open']);
});
test('scene structure preserves geometry-only identity but notices value and wiring changes',()=>{
 const make=x=>({parts:[part('R1',x)],wires:[{id:'W1',from:'R1.1',to:'R1.2',connected:true,points:[{x,y:0}]}]});
 const key=s=>sceneStructureKey(s,['R1'],'','select',null);assert.equal(key(make(0)),key(make(10)));
 const changed=make(10);changed.parts[0].value_text='2k';assert.notEqual(key(make(0)),key(changed));changed.parts[0].value_text='1k';changed.wires[0].connected=false;assert.notEqual(key(make(0)),key(changed));
});

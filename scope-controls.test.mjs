import test from 'node:test';
import assert from 'node:assert/strict';
import {scopeContext,scopeEdit,scopePlotDomain,captureParams} from './static/scope-controls.mjs';

const live={document:{generation:3},scope:{window:{mode:'live',start_s:1,end_s:1.5}}};
const recorded={...live,recording:{viewing:true,id:'capture-7',scope:{window:{mode:'fixed',start_s:0,end_s:.1}}}};

test('scope edits preserve time quantities and distinguish live and captured owners',()=>{
 assert.equal(scopeContext(live).owner,'3:live');
 assert.equal(scopeContext(recorded).owner,'3:capture-7');
 assert.equal(scopeContext({...recorded,recording:{...recorded.recording,viewing:false}}).scope,live.scope);
 assert.notEqual(scopeContext({...recorded,document:{generation:4}}).owner,scopeContext(recorded).owner);
 assert.notEqual(scopeContext(recorded,'before-restart').owner,scopeContext(recorded,'after-restart').owner);
 const interval={start_s:'20 us',end_s:'1 ms'};
 assert.deepEqual(scopeEdit('window',interval,live),{name:'scope.window.set',params:{mode:'fixed',...interval}});
 assert.deepEqual(scopeEdit('window',interval,recorded),{name:'recording.window.set',params:interval});
 assert.deepEqual(scopeEdit('cursor-b',{time_s:'50 µs'},recorded),{name:'recording.cursor.set',params:{cursor:'b',time_s:'50 µs',operation_id:'capture-7'}});
 assert.deepEqual(scopeEdit('statistics',{probe_id:'v:C1',...interval},recorded),{name:'recording.statistics.set',params:{id:'capture-7',probe_id:'v:C1',...interval}});
 assert.deepEqual(scopeEdit('probes',{ids:[]},recorded),{name:'recording.probes.set',params:{ids:[]}});
});

test('recorded seeks retain fractional indices for refusal and use exact source time text',()=>{
 assert.deepEqual(scopeEdit('index',{index:'1.5'},recorded),{name:'recording.seek',params:{index:1.5}});
 assert.deepEqual(scopeEdit('seek',{time_s:'20 ms'},recorded),{name:'recording.seek',params:{time_s:'20 ms'}});
 assert.deepEqual(scopeEdit('speed',{speed:'.025'},recorded),{name:'recording.speed.set',params:{seconds_per_wall_second:.025}});
});

test('plot domains retain requested margins and empty intervals without borrowing nearby samples',()=>{
 const trace={points:[{time_s:.02,value:1},{time_s:.08,value:2}]};
 assert.deepEqual(scopePlotDomain(recorded.recording.scope,[trace]),{minT:0,maxT:.1});
 assert.deepEqual(scopePlotDomain({window:{start_s:.021,end_s:.022}},[]),{minT:.021,maxT:.022});
 assert.deepEqual(scopePlotDomain(live.scope,[trace]),{minT:1,maxT:1.5});
 const dc=scopePlotDomain({window:{start_s:0,end_s:0}},[{points:[{time_s:0,value:5}]}]);
 assert.equal(dc.minT,0);assert.ok(dc.maxT>dc.minT);
});

test('captures preserve time units, native default DC probes and optional startup overrides',()=>{
 const fields={duration:'100 ms',step:'0',input_mode:'charge',initial_voltage_v:'2'};
 const state={...live,transient:{},scope:{selected:['v:C1']}};
 assert.deepEqual(captureParams(fields,state),{duration_s:'100 ms',sample_step_s:'0',probe_ids:['v:C1'],input_mode:'charge',initial_voltage_v:2});
 assert.deepEqual(captureParams({...fields,duration:'0',initial_voltage_v:''},{...live,scope:{selected:[]}}),{duration_s:'0',sample_step_s:'0',input_mode:'charge'});
 assert.throws(()=>captureParams(fields,{...state,scope:{selected:[]}}),/at least one live scope trace/);
});

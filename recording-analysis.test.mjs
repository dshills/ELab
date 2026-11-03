import test from 'node:test';
import assert from 'node:assert/strict';
import {recordingAnalysisEdit as edit,timingPage} from './static/recording-analysis.mjs';

test('recording analyses pin capture IDs and preserve native time/threshold quantities',()=>{
 assert.deepEqual(edit('compare',{reference_id:'capture-b',time_s:'11.4 ms',show_overlay:'on'},'capture-a'),{name:'recording.compare.set',params:{id:'capture-a',reference_id:'capture-b',time_s:'11.4 ms',show_overlay:true,show_statistics:false}});
 assert.deepEqual(edit('compare',{reference_id:'b',time_s:'1 ms',show_statistics:'on'},'a').params,{id:'a',reference_id:'b',time_s:'1 ms',show_overlay:false,show_statistics:true});
 const timing=edit('timing',{probe_id:'i:R1',level:'10 µA',second_level:'20 µA',direction:'rising',start_s:'5 ms',end_s:'50 ms',limit:'20'},'capture-a');
 assert.deepEqual(timing,{name:'recording.timing.set',params:{id:'capture-a',probe_id:'i:R1',level:'10 µA',second_level:'20 µA',direction:'rising',start_s:'5 ms',end_s:'50 ms',offset:0,limit:20}});
 assert.equal('second_level' in edit('timing',{second_level:'  '},'a').params,false);
 for(const action of ['compare','timing','step','clear-compare','clear-timing','clear-step'])
  assert.throws(()=>edit(action,{},''),/recorded experiment/,action);
});
test('step analysis converts only numeric references/tolerance and retains unit times',()=>{
 assert.deepEqual(edit('step',{probe_id:'v:C1',start_s:'0 ms',end_s:'49 ms',initial_value:'0',final_value:'5',tolerance_percent:'2',minimum_hold_s:'5 ms'},'capture-a'),{name:'recording.step-response.set',params:{id:'capture-a',probe_id:'v:C1',start_s:'0 ms',end_s:'49 ms',initial_value:0,final_value:5,tolerance_fraction:.02,minimum_hold_s:'5 ms'}});
});
test('clear commands obey native explicit-zero contracts and remain capture-bound',()=>{
 assert.deepEqual(edit('clear-compare',{},'a').params,{id:'a',reference_id:'',time_s:0});
 assert.deepEqual(edit('clear-timing',{},'a').params,{id:'a',probe_id:'',level:'0',direction:'both',start_s:0,end_s:0,offset:0,limit:0});
 assert.deepEqual(edit('clear-step',{},'a').params,{id:'a',probe_id:'',start_s:0,end_s:0,initial_value:0,final_value:0,tolerance_fraction:0,minimum_hold_s:0});
});

test('native omitted zero offsets produce valid first/empty timing pages',()=>{
 assert.deepEqual(timingPage({settings:{limit:1},total_crossings:4,next_offset:1}),{start:1,end:1,total:4,previous:false,next:true,previousOffset:0});
 assert.deepEqual(timingPage({settings:{offset:2,limit:1},total_crossings:4,next_offset:3}),{start:3,end:3,total:4,previous:true,next:true,previousOffset:1});
 assert.deepEqual(timingPage({settings:{offset:3,limit:5},total_crossings:4,next_offset:4}),{start:4,end:4,total:4,previous:true,next:false,previousOffset:0});
 assert.deepEqual(timingPage({settings:{},total_crossings:0,next_offset:0}),{start:0,end:0,total:0,previous:false,next:false,previousOffset:0});
});

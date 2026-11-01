import test from 'node:test';
import assert from 'node:assert/strict';
import {componentSettings, simulationSettings, settingsCommand, liveInputButtons} from './static/settings.mjs';
const state = {transient_settings:{step_s:.001,high_s:.03,low_s:.07,initial_voltage_v:2,seconds_per_wall_second:.02},pulse_timings:[]};

test('sampling changes preserve pulse phases and accept native time units', () => {
  const edit=settingsCommand('simulation-form',{step_s:'500 us',seconds_per_wall_second:'.01',initial_voltage_v:'3'},null,state);
  assert.equal(edit.name,'simulation.transient.set');
  assert.deepEqual(edit.params,{...state.transient_settings,step_s:'500 us',seconds_per_wall_second:.01,initial_voltage_v:3});
  assert.equal(state.transient_settings.step_s,.001);
  assert.deepEqual(settingsCommand('step-form',{duration_s:'21 ms'},null,state).params,{duration_s:'21 ms'});
});
test('startup reset and signed quantities reach the correct component commands',()=>{
  assert.deepEqual(settingsCommand('startup-form',{value:'default'},{id:'C1',kind:'capacitor'},state),{name:'capacitor.initial.set',params:{id:'C1',value:'default'}});
  assert.deepEqual(settingsCommand('startup-form',{value:'-5 mA'},{id:'L1',kind:'inductor'},state),{name:'inductor.initial.set',params:{id:'L1',value:'-5 mA'}});
});
test('loss model keeps unused fields zero and has an explicit disabled state',()=>{
  const cap=settingsCommand('loss-form',{enabled:'on',esr_ohm:'2',leakage_resistance_ohm:'1000000'},{id:'C1',kind:'capacitor'},state);
  assert.deepEqual(cap.params,{id:'C1',enabled:true,parameters:{version:'elab.passive-loss.v1',esr_ohm:2,leakage_resistance_ohm:1000000,winding_resistance_ohm:0}});
  const coil=settingsCommand('loss-form',{winding_resistance_ohm:'3'},{id:'L1',kind:'inductor'},state);
  assert.equal(coil.params.enabled,false);
  assert.deepEqual(coil.params.parameters,{version:'elab.passive-loss.v1',esr_ohm:0,leakage_resistance_ohm:0,winding_resistance_ohm:3});
});
test('waveform edits preserve exact engineering-unit text and identify their source',()=>{
  const wave={amplitude_v:'500 mV',offset_v:'1 V',frequency_hz:'100 Hz',phase_degrees:'90'};
  assert.deepEqual(settingsCommand('sine-form',wave,{id:'VS'},state),{name:'source.sine.set',params:{id:'VS',waveform:wave}});
  assert.deepEqual(settingsCommand('ideal-pulse-form',{frequency_hz:'25 Hz',duty_percent:'40 %'},{id:'DIR'},state),{name:'simulation.pulse.timing.set',params:{id:'DIR',frequency_hz:'25 Hz',duty_percent:'40 %'}});
});
test('rendered controls show saved values, shared-timing scope, and escaped attributes',()=>{
  const cap=componentSettings({id:'C"1',kind:'capacitor',initial_voltage_v:0,device_parameters:{passive:{esr_ohm:2,leakage_resistance_ohm:1e6}}},state);
  assert.match(cap,/data-component-id="C&quot;1"/);
  assert.match(cap,/name="value" value="0"/);
  assert.match(cap,/name="enabled" checked/);
  assert.match(cap,/name="esr_ohm" value="2"/);
  assert.match(componentSettings({id:'DIR',kind:'pulse-source'},state),/All ideal pulse sources share this timing/);
  assert.match(simulationSettings(state),/Default capacitor startup voltage/);
  assert.match(simulationSettings({...state,paused:true,transient:{stale:false}},'21 ms'),/name="duration_s" value="21 ms"/);
});

test('live source controls use native mode IDs and retain staged paused choices',()=>{
 const rc={live_input_modes:['auto','charge','discharge'],transient:{source_id:'V1',input_mode:'auto',pending_input_mode:'discharge'},components:[{id:'V1',kind:'pulse-source'}],status:'paused'};
 const html=liveInputButtons(rc);
 assert.match(html,/data-input="charge"/);assert.doesNotMatch(html,/data-input="high"/);
 assert.match(html,/data-input="discharge" class="active"/);
 assert.equal(liveInputButtons({...rc,components:[{id:'V1',kind:'current-source'}]}),'');
 assert.match(liveInputButtons({...rc,status:'error'}),/disabled/);
 const general=liveInputButtons({...rc,live_input_modes:['auto','high','low']});
 assert.match(general,/data-input="high"/);assert.doesNotMatch(general,/>Charge</);
});

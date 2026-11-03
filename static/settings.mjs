// Direct controls for common native workbench settings. Keep command construction
// separate from DOM events so partial forms cannot silently reset unrelated fields.
const escape = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const field = (name, label, value, options = '') => `<label>${label}<input name="${name}" value="${escape(value)}" ${options} required></label>`;
const number = (name, label, value, options = '') => field(name, label, value, `type="number" step="any" ${options}`);
const form = (id, title, fields, note, button = 'Apply settings', disabled = false) => `<section class="section advanced-settings"><h3>${title}</h3><form id="${id}" class="settings-form">${fields}<p class="field-note">${note}</p><button class="primary" type="submit" ${disabled ? 'disabled' : ''}>${button}</button><p class="settings-feedback" role="status" aria-live="polite"></p></form></section>`;

export function componentSettings(part, state) {
  let html = '';
  if (part.kind === 'capacitor' || part.kind === 'inductor') {
    const capacitor = part.kind === 'capacitor';
    html += form('startup-form', 'Startup condition',
      field('value', capacitor ? 'Initial voltage (V)' : 'Initial current (A)',
        (capacitor ? part.initial_voltage_v : part.initial_current_a) ?? 'default'),
      capacitor ? `Voltage from pin 1 to pin 2. “default” uses the experiment startup voltage (${state.transient_settings.initial_voltage_v} V). Restart or resume to apply while paused.` : 'Current from pin 1 to pin 2. “default” uses zero. Restart or resume to apply while paused.',
      'Apply startup condition');
    const loss = part.device_parameters?.passive;
    html += form('loss-form', 'Component losses',
      `<label class="checkbox-label"><input type="checkbox" name="enabled" ${loss ? 'checked' : ''}>Include electrical losses</label>` +
      (capacitor ? number('esr_ohm', 'Series ESR (Ω)', loss?.esr_ohm ?? 0, 'min="0" max="100000"') + number('leakage_resistance_ohm', 'Leakage resistance (Ω)', loss?.leakage_resistance_ohm ?? 0, 'min="0" max="1000000000000"') : number('winding_resistance_ohm', 'Winding resistance (Ω)', loss?.winding_resistance_ohm ?? 0, 'min="0" max="100000"')),
      'Zero disables a loss. Disabling the model restores ideal behavior. Restart or resume to apply while paused.', 'Apply losses');
  }
  if (part.kind === 'pulse-source') {
    const timing = state.pulse_timings?.find(t => t.id === part.id);
    const cfg = state.transient_settings;
    html += form('ideal-pulse-form', 'Pulse timing',
      field('frequency_hz', 'Frequency (Hz)', timing?.frequency_text ?? 1 / (cfg.high_s + cfg.low_s)) +
      field('duty_percent', 'Duty cycle (%)', timing?.duty_text ?? 100 * cfg.high_s / (cfg.high_s + cfg.low_s)),
      'All ideal pulse sources share this timing. Restart or resume to apply while paused.', 'Apply pulse timing');
  }
  if (part.kind === 'sine-source') {
    const wave = part.sine;
    if (wave) html += form('sine-form', 'Sine waveform',
      field('amplitude_v', 'Peak amplitude (V)', wave.amplitude_v) + field('offset_v', 'DC offset (V)', wave.offset_v) +
      field('frequency_hz', 'Frequency (Hz)', wave.frequency_hz) + field('phase_degrees', 'Phase (degrees)', wave.phase_degrees),
      'Accepts engineering units. Offset ± amplitude must stay within ±12 V. Restart or resume to apply while paused.', 'Apply sine waveform');
  }
  if (part.kind === 'finite-pulse-source') {
    const wave = part.pulse;
    if (wave) html += form('pulse-form', 'Pulse waveform',
      field('high_v', 'High voltage (V)', wave.high_v) + field('low_v', 'Low voltage (V)', wave.low_v) +
      field('high_s', 'High phase (s)', wave.high_s) + field('low_s', 'Low phase (s)', wave.low_s) +
      field('rise_s', 'Rise time (s)', wave.rise_s) + field('fall_s', 'Fall time (s)', wave.fall_s),
      'Accepts time units such as 1 ms. High phase includes rise; low phase includes fall. Restart or resume to apply while paused.', 'Apply pulse waveform');
  }
  return html.replaceAll('<form id=', `<form data-component-id="${escape(part.id)}" id=`);
}

export function simulationSettings(state, stepDuration = '20 ms') {
  const cfg = state.transient_settings;
  return form('simulation-form', 'Sampling & startup',
    field('step_s', 'Sample interval (s)', cfg.step_s) +
    number('seconds_per_wall_second', 'Playback speed (simulated s / real s)', cfg.seconds_per_wall_second, 'min="0.001" max="1"') +
    number('initial_voltage_v', 'Default capacitor startup voltage (V)', cfg.initial_voltage_v, 'min="0" max="12"'),
    'Sample interval accepts time units. Startup voltage applies to capacitors using “default”. Restart or resume to apply while paused.') +
    `<button class="quiet-button full-width sampling-fit" data-action="sampling-fit">Fit sampling to circuit</button><p class="field-note">Chooses a finer sample interval when sources or LC dynamics need it; playback may be reduced.</p>` +
    form('step-form', 'Advance while paused', field('duration_s', 'Step duration (s)', stepDuration),
      'Accepts time units. Pause the simulation, then advance by this duration to inspect a transition. Restart a changed circuit before stepping.', 'Advance simulation', !state.paused || !state.transient || state.transient.stale || state.status === 'updating');
}

// Values are plain form data, not a snapshot. Native commands retain final
// validation and guard checks. Numeric-only model fields are explicit SI units.
export function settingsCommand(id, values, part, state) {
  switch (id) {
    case 'startup-form':
      return {name: part.kind === 'capacitor' ? 'capacitor.initial.set' : 'inductor.initial.set', params: {id: part.id, value: values.value}};
    case 'loss-form':
      return {name: 'passive.configure', params: {id: part.id, enabled: values.enabled === 'on', parameters: {
        version: 'elab.passive-loss.v1', esr_ohm: part.kind === 'capacitor' ? Number(values.esr_ohm) : 0,
        leakage_resistance_ohm: part.kind === 'capacitor' ? Number(values.leakage_resistance_ohm) : 0,
        winding_resistance_ohm: part.kind === 'inductor' ? Number(values.winding_resistance_ohm) : 0,
      }}};
    case 'ideal-pulse-form':
      return {name: 'simulation.pulse.timing.set', params: {id: part.id, frequency_hz: values.frequency_hz, duty_percent: values.duty_percent}};
    case 'sine-form':
      return {name: 'source.sine.set', params: {id: part.id, waveform: {...values}}};
    case 'pulse-form':
      return {name: 'source.pulse.set', params: {id: part.id, waveform: {...values}}};
    case 'simulation-form':
      return {name: 'simulation.transient.set', params: {...state.transient_settings, step_s: values.step_s,
        seconds_per_wall_second: Number(values.seconds_per_wall_second), initial_voltage_v: Number(values.initial_voltage_v)}};
    case 'step-form':
      return {name: 'simulation.step', params: {duration_s: values.duration_s}};
    default: return null;
  }
}

export function liveInputModes(state){
 const source=state.components?.find(p=>p.id===state.transient?.source_id);
 return ['battery','current-source'].includes(source?.kind)?[]:(state.live_input_modes||[]);
}
export function liveInputButtons(state){
 const selected=state.transient?.pending_input_mode||state.transient?.input_mode;
 const disabled=!state.transient||state.transient.stale||state.status==='error';
 return liveInputModes(state).map(mode=>`<button data-input="${escape(mode)}" class="${mode===selected?'active':''}" ${disabled?'disabled':''}>${escape(({auto:'Auto',charge:'Charge',discharge:'Discharge',high:'High',low:'Low'})[mode]||mode)}</button>`).join('');
}

import {escapeHTML as esc} from './forms.mjs';

export function troubleshootingDialog(api) {
 const dialog = document.getElementById('troubleshoot-dialog');
 const form = dialog.querySelector('form');
 const status = dialog.querySelector('[role="status"]');
 const guide = dialog.querySelector('[data-troubleshooting-guide]');
 const meter = dialog.querySelector('[data-meter]');
 const tryFix = dialog.querySelector('[data-try-fix]');
 const outcome = dialog.querySelector('[data-trial-result]');
 let source, pending;

 async function inspect(places) {
  pending?.abort();
  const controller = new AbortController();
  pending = controller;
  form.querySelector('button').disabled = true;
  tryFix.disabled = true;
  status.textContent = 'Checking the circuit…';
  try {
   const data = await api.experiment('/api/troubleshoot', places, source, controller.signal);
   if (controller.signal.aborted) return;
   guide.replaceChildren(...data.hints.map(hint => {
    const paragraph = document.createElement('p');
    paragraph.textContent = hint.text;
    return paragraph;
   }));
   tryFix.hidden = !data.suggestion;
   tryFix.textContent = data.suggestion ? 'Try on a copy: ' + data.suggestion : '';
   if (places.trial) {
    outcome.textContent = data.trial_error || (data.trial ?
     'The trial circuit solved. ' + data.trial.reading + '.' : 'A trial is unavailable for this wiring.');
   }
   status.textContent = data.hints.length ? 'Review the checks below.' : 'No connection or startup LED polarity issue found.';
   if (places.red) {
    meter.textContent = data.voltage === null ? 'Reading unavailable. ' + data.reason :
     api.fmt(data.voltage, 'V', 5) + ' · red minus black. A negative reading means the red terminal is at a lower voltage.';
   }
  } catch (error) {
   if (!controller.signal.aborted) status.textContent = error.message;
  } finally {
   if (pending === controller) {
    pending = null;
    form.querySelector('button').disabled = false;
    tryFix.disabled = false;
   }
  }
 }

 function open() {
  const state = api.state();
  source = api.guards();
  const terminals = state.components.flatMap(part => part.terminals.map(pin => part.id + '.' + pin));
  const options = terminals.map(ref => `<option value="${esc(ref)}">${esc(ref)}</option>`).join('');
  form.elements.red.innerHTML = options;
  form.elements.black.innerHTML = '<option value="ground">Circuit ground</option>' + options;
  const led = state.components.find(part => part.kind === 'led');
  if (led) {
   form.elements.red.value = led.id + '.anode';
   form.elements.black.value = led.id + '.cathode';
  }
  meter.textContent = '';
  outcome.textContent = '';
  guide.replaceChildren();
  tryFix.hidden = true;
  dialog.showModal();
  void inspect({});
 }

 form.addEventListener('submit', event => {
  event.preventDefault();
  void inspect({red: form.elements.red.value, black: form.elements.black.value});
 });
 tryFix.addEventListener('click', () => void inspect({trial: true}));
 dialog.addEventListener('close', () => pending?.abort());
 return {open};
}

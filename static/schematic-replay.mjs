import {replayView} from './replay-view.mjs';

export function schematicReplay(api) {
 const dialog = document.getElementById('schematic-replay-dialog');
 const status = dialog.querySelector('[data-replay-status]');
 const player = dialog.querySelector('[data-replay-player]');
 const form = dialog.querySelector('[data-replay-share]');
 const shareStatus = form.querySelector('[data-share-status]');
 const download = form.querySelector('button');
 let pending, sharing, view, source;
 dialog.addEventListener('close', () => {
  pending?.abort();
  sharing?.abort();
  pending = null;
  sharing = null;
  view?.stop();
  view = null;
 });

 form.addEventListener('submit', async event => {
  event.preventDefault();
  if (!view || sharing) return;
  const controller = new AbortController();
  sharing = controller;
  download.disabled = true;
  shareStatus.textContent = 'Creating the offline replay…';
  try {
   if (api.instance() !== source.instance) throw new Error('ELab restarted. Open the recording again.');
   view.stop();
   const response = await fetch('/api/schematic-replay/share?' + source.query, {
    method: 'POST', headers: {'Content-Type': 'application/json'}, signal: controller.signal,
    body: JSON.stringify({
     position: view.position(), note: form.elements.note.value,
     include_project: form.elements.includeProject.checked,
    }),
   });
   if (!response.ok) {
    const detail = await response.json();
    throw new Error(detail.error || 'Could not create the offline replay.');
   }
   const blob = await response.blob();
   if (controller.signal.aborted) return;
   const link = document.createElement('a');
   link.href = URL.createObjectURL(blob);
   link.download = 'ELab-schematic-replay.html';
   link.click();
   setTimeout(() => URL.revokeObjectURL(link.href), 2000);
   shareStatus.textContent = 'Replay downloaded. Open the HTML file in a browser.';
  } catch (error) {
   if (!controller.signal.aborted) shareStatus.textContent = error.message;
  } finally {
   if (sharing === controller) {
    sharing = null;
    download.disabled = false;
   }
  }
 });

 return async function open() {
  const state = api.state();
  const recording = state.recording;
  if (!recording?.viewing || pending) return;
  const query = new URLSearchParams({id: recording.id, revision: state.command_revision, generation: state.document.generation});
  source = {query, instance: api.instance()};
  const request = new AbortController();
  pending = request;
  player.hidden = true;
  form.hidden = true;
  form.reset();
  shareStatus.textContent = '';
  download.disabled = false;
  status.textContent = 'Loading captured schematic and samples…';
  dialog.showModal();
  try {
   const capture = await api.fetch('/api/schematic-replay?' + query, {signal: request.signal});
   if (request.signal.aborted) return;
   view = replayView(player, capture, {index: recording.sample_index});
   status.textContent = 'Captured circuit · ' + recording.id;
   player.hidden = false;
   form.hidden = false;
  } catch (error) {
   if (!request.signal.aborted) status.textContent = error.message;
  } finally {
   if (pending === request) pending = null;
  }
 };
}

import {replayView} from './replay-view.mjs';

export function schematicReplay(api) {
 const dialog = document.getElementById('schematic-replay-dialog');
 const status = dialog.querySelector('[role="status"]');
 const player = dialog.querySelector('[data-replay-player]');
 let pending, view;
 dialog.addEventListener('close', () => {
  pending?.abort();
  pending = null;
  view?.stop();
 });
 return async function open() {
  const state = api.state();
  const recording = state.recording;
  if (!recording?.viewing || pending) return;
  const query = new URLSearchParams({id: recording.id, revision: state.command_revision, generation: state.document.generation});
  const request = new AbortController();
  pending = request;
  player.hidden = true;
  status.textContent = 'Loading captured schematic and samples…';
  dialog.showModal();
  try {
   const capture = await api.fetch('/api/schematic-replay?' + query, {signal: request.signal});
   if (request.signal.aborted) return;
   view = replayView(player, capture, recording.sample_index);
   status.textContent = 'Captured circuit · ' + recording.id;
   player.hidden = false;
  } catch (error) {
   if (!request.signal.aborted) status.textContent = error.message;
  } finally {
   if (pending === request) pending = null;
  }
 };
}

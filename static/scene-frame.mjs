// Live electrical presentation patches an existing drawing without replacing its
// geometry or object identity. New documents/commands receive a complete scene.
export function mergeSceneFrame(scene,frame){
 const parts=new Map((frame.parts||[]).map(p=>[p.id,p]));
 return {...scene,markers:frame.markers||[],parts:scene.parts.map(p=>parts.has(p.id)?{...p,...parts.get(p.id)}:p)};
}

export function sceneStructureKey(scene,selection,wire,mode,start){
 return JSON.stringify([scene.parts.map(p=>[p.id,p.kind,p.rotation,p.value_text,p.dynamic?null:p.paths,p.segments,(p.labels||[]).map(l=>[l.text,l.x-p.position.x,l.y-p.position.y]),p.bounds.map((n,i)=>n-(i%2?p.position.y:p.position.x))]),scene.wires.map(w=>[w.id,w.from,w.to,w.connected]),selection,wire,mode,start]);
}

const markupByLayer = new WeakMap();

// Replace only changed SVG objects. Preserve the nodes and focus of the other
// objects when a part is edited, selected or added elsewhere on the sheet.
export function updateSceneLayer(layer, entries) {
 const previous = markupByLayer.get(layer) || new Map();
 const next = new Map();
 const staging = layer.ownerDocument.createElementNS(layer.namespaceURI, 'g');
 for (const [id, markup] of entries) {
  const prior = previous.get(id);
  let node = prior?.node;
  if (prior?.markup !== markup || node?.parentNode !== layer) {
   staging.innerHTML = markup;
   const created = staging.firstElementChild;
   if (node?.parentNode === layer) node.replaceWith(created);
   node = created;
  }
  next.set(id, {markup, node});
 }
 for (const [id, {node}] of previous) if (!next.has(id)) node.remove();
 let at = layer.firstElementChild;
 for (const {node} of next.values()) {
  if (at !== node) layer.insertBefore(node, at);
  at = node.nextElementSibling;
 }
 markupByLayer.set(layer, next);
 return new Map([...next].map(([id, {node}]) => [id, node]));
}

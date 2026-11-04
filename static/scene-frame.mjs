// Live electrical presentation patches an existing drawing without replacing its
// geometry or object identity. New documents/commands receive a complete scene.
export function mergeSceneFrame(scene,frame){
 const parts=new Map((frame.parts||[]).map(p=>[p.id,p]));
 return {...scene,markers:frame.markers||[],parts:scene.parts.map(p=>parts.has(p.id)?{...p,...parts.get(p.id)}:p)};
}

export function sceneStructureKey(scene,selection,wire,mode,start){
 return JSON.stringify([scene.parts.map(p=>[p.id,p.kind,p.rotation,p.value_text,p.dynamic?null:p.paths,p.segments,(p.labels||[]).map(l=>[l.text,l.x-p.position.x,l.y-p.position.y]),p.bounds.map((n,i)=>n-(i%2?p.position.y:p.position.x))]),scene.wires.map(w=>[w.id,w.from,w.to,w.connected]),selection,wire,mode,start]);
}

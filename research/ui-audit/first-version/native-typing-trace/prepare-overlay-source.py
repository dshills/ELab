import json,os,subprocess,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-typing-trace';sdk=root/'build/.vellum-sdk-677548d'
sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import pinned_child_runner
_,run_child=pinned_child_runner()
replacements={}
def overlay(path,changes,imports=()):
 s=path.read_text()
 for old,new in changes:
  assert s.count(old)==1,(path,old[:100],s.count(old));s=s.replace(old,new)
 for name in imports:
  if '"'+name+'"' not in s:s=s.replace('import (','import (\n\t"'+name+'"',1)
 dest=base/'overlays'/path.name;assert not dest.exists();dest.write_text(s);subprocess.run(['gofmt','-w',str(dest)],check=True);replacements[str(path)]=str(dest)
overlay(root/'desktop/main.go',[( 'result := s.service.Dispatch(context.Background(), request, "user")', 'result := s.service.Dispatch(context.Background(), request, "user")\n\tif strings.HasPrefix(id, "scope.cursor") { log.Printf("ELAB_TRACE request at=%d command=%s params=%s expected=%d result_error=%+v result_help=%+v", time.Now().UnixNano(), id, raw, rev, result.Error, result.Help) }')])
overlay(sdk/'drivers/cogent/editor.go',[
 ('func (t *textWindow) apply(u textinput.Update) {','''func (t *textWindow) apply(u textinput.Update) {
 log.Printf("ELAB_TRACE input at=%d seq=%d client=%d kind=%s delta=%t before=%q text=%q anchor=%d focus=%d ack=%d",time.Now().UnixNano(),u.Sequence,u.ClientID,u.Kind,u.Delta,u.BeforeText,u.Text,u.Anchor,u.Focus,t.ack)
 defer func(){ if r:=t.clients[u.ClientID];r!=nil && r.node.ID=="cursor-time-a" { log.Printf("ELAB_TRACE applied at=%d seq=%d epoch=%d shadow=%+v widget=%+v",time.Now().UnixNano(),u.Sequence,r.control.textEpoch,t.shadows[u.ClientID],r.widget.(nativeTextClient).EditingState()) } }()''')],['log','time'])
overlay(sdk/'drivers/cogent/editor_geometry.go',[
 ('accepted, err := publishTextOnMain(t, r.control.client, state, start, end, active, t.ack, caret, geometry, maximum)', '''accepted, err := publishTextOnMain(t, r.control.client, state, start, end, active, t.ack, caret, geometry, maximum)
 if r.node.ID=="cursor-time-a" { value:=fmt.Sprintf("%t/%d/%t/%q/%d/%d/%d/%d",accepted,r.control.client,active,state.Text,state.Anchor,state.Focus,t.ack,r.control.textEpoch);if elabTracePublications[r.control.client]!=value {elabTracePublications[r.control.client]=value;log.Printf("ELAB_TRACE publish at=%d state=%s error=%v",time.Now().UnixNano(),value,err)} }'''),
 ('func textPoints(', 'var elabTracePublications=map[uint64]string{}\n\nfunc textPoints(')],['log','time'])
overlay(sdk/'drivers/cogent/controls.go',[
 ('field.SetNativeEditingListener(func(core.TextFieldEditingState) { requestTextPublicationScoped(c.windowRef) })', '''field.SetNativeEditingListener(func(state core.TextFieldEditingState) { if r.node.ID=="cursor-time-a" {log.Printf("ELAB_TRACE editing at=%d nativeactive=%t lost=%t state=%+v",time.Now().UnixNano(),c.fieldNativeActive.Load(),c.fieldFocusLost.Load(),state)};requestTextPublicationScoped(c.windowRef) })
 if r.node.ID=="cursor-time-a" {field.OnFirst(events.KeyChord,func(e events.Event){log.Printf("ELAB_TRACE legacy-key at=%d event=%v nativeactive=%t",time.Now().UnixNano(),e,c.fieldNativeActive.Load())})}'''),
 ('fieldEpoch := r.control.fieldChanges.Load()', '''if change.Kind==ui.TextField && r.node.ID=="cursor-time-a" {log.Printf("ELAB_TRACE queue-change at=%d text=%q",time.Now().UnixNano(),change.Text)}
 fieldEpoch := r.control.fieldChanges.Load()'''),
 ('callback(change)\n\t\tr.control.committed = true', '''if change.Kind==ui.TextField && r.node.ID=="cursor-time-a" {log.Printf("ELAB_TRACE deliver-change at=%d text=%q",time.Now().UnixNano(),change.Text)}
 callback(change)
 r.control.committed = true''')],['log','time'])
(base/'overlay.json').write_text(json.dumps({'Replace':replacements},indent=2)+'\n')
env=dict(os.environ,GOWORK='off',GOCACHE=str(root/'build/go-cache'),GOPROXY='off',GOTOOLCHAIN='local',CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native',PYTHONDONTWRITEBYTECODE='1')
with (base/'build.txt').open('x') as log:run_child(['go','build','-overlay',str(base/'overlay.json'),'-buildvcs=true','-o',str(base/'ELab-trace'),'.'],cwd=root/'desktop',env=env,stdout=log,stderr=-2,timeout=120,check=True)
print('diagnostic overlay build completed; source/SDK files unchanged')

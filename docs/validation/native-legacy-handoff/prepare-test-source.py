from pathlib import Path
import shutil
root=Path('/Users/dshills/Development/projects/ELab');dest=root/'build/native-handoff-repair/sdk'
assert not dest.exists();shutil.copytree(root/'build/.vellum-sdk-677548d',dest)
p=dest/'drivers/cogent/internal/textinput'
h=p/'input.h';h.write_text(h.read_text()+'\nvoid vellum_managed_input_fixture_key(void *input,const char *text,bool command);\nint vellum_managed_input_fixture_base_calls(void *input);\n')
m=p/'input_darwin.m';s=m.read_text();needle='- (void)insertText:(id)value replacementRange:(NSRange)range { baseCalls++; }';assert s.count(needle)==1;s=s.replace(needle,needle+'\n- (void)keyDown:(NSEvent *)event { baseCalls++; }');s+='''
void vellum_managed_input_fixture_key(void *input,const char *text,bool command) {
 VellumManagedInputState *s=(id)input;if(!s || s->closed)return;
 NSString *chars=[NSString stringWithUTF8String:text];
 NSEvent *event=[NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint modifierFlags:command ? NSEventModifierFlagCommand : 0 timestamp:0 windowNumber:0 context:nil characters:chars charactersIgnoringModifiers:chars isARepeat:NO keyCode:command ? 0 : 7];
 [s->view keyDown:event];
}
int vellum_managed_input_fixture_base_calls(void *input) {VellumManagedInputState *s=(id)input;return s && !s->closed ? ((VellumManagedFixtureView *)s->view)->baseCalls : -1;}
''';m.write_text(s)
g=p/'input_darwin.go';g.write_text(g.read_text()+'''
func fixtureInputKey(i *Input, text string, command bool) { v:=C.CString(text);defer C.free(unsafe.Pointer(v)); C.vellum_managed_input_fixture_key(i.native,v,C.bool(command)) }
func fixtureInputBaseCalls(i *Input) int { return int(C.vellum_managed_input_fixture_base_calls(i.native)) }
''')
t=p/'input_darwin_test.go';s=t.read_text();needle='\n\treturn results\n';assert s.count(needle)==1;s=s.replace(needle,'\n\trunLegacyHandoffFixtures(run,rect)'+needle);t.write_text(s)
(p/'legacy_handoff_test.go').write_text('''//go:build darwin && cgo

package textinput

import "testing"

func runLegacyHandoffFixtures(run func(string, func()), rect [4]float64) {
 run("legacy-handoff",func(){
  view:=fixtureInputView();defer fixtureReleaseInputView(view)
  var updates []Update
  i:=Attach(view,func(u Update) bool {updates=append(updates,u);return true},func(){})
  fixtureAssert(i!=nil,"attach failed");defer i.Close()
  fixtureInputKey(i,"8",false)
  fixtureAssert(fixtureInputBaseCalls(i)==1,"initial legacy key lost")
  fixtureAssert(!i.UpdateClient(13,"6 ms",4,4,-1,-1,true,0,rect),"activated Cocoa before the earlier legacy key was acknowledged")
  for _,s:=range []string{" ","m","s"} {fixtureInputKey(i,s,false)}
  fixtureAssert(fixtureInputBaseCalls(i)==4,"mixed legacy/native stream before acknowledgment")
  fixtureAssert(len(updates)==4,"missing ordered legacy acknowledgments")
  for n,u:=range updates {fixtureAssert(u.Kind=="legacy" && u.ClientID==0 && u.Sequence==uint64(n+1),"unexpected legacy marker")}
  fixtureAssert(i.UpdateClient(13,"8 ms",4,4,-1,-1,true,4,rect),"coherent final publication refused")
  fixtureInputEdit(i,"!",-1,0,0)
  fixtureAssert(len(updates)==5 && updates[4].Kind=="insert" && updates[4].BeforeText=="8 ms" && updates[4].Text=="8 ms!","native edit did not use acknowledged legacy baseline")
 })
 run("legacy-select-all",func(){
  view:=fixtureInputView();defer fixtureReleaseInputView(view)
  var updates []Update
  i:=Attach(view,func(u Update) bool {updates=append(updates,u);return true},func(){})
  fixtureAssert(i!=nil,"attach failed");defer i.Close()
  fixtureAssert(i.UpdateClient(13,"6 ms",4,4,-1,-1,true,0,rect),"initial active publication refused")
  fixtureInputKey(i,"a",true)
  fixtureAssert(!i.UpdateClient(13,"6 ms",0,4,-1,-1,true,0,rect),"selection publication bypassed queued select-all")
  fixtureInputKey(i,"8",false)
  fixtureAssert(fixtureInputBaseCalls(i)==2,"typing borrowed stale Cocoa selection")
  fixtureAssert(i.UpdateClient(13,"8",1,1,-1,-1,true,2,rect),"replacement baseline refused after acknowledgment")
  fixtureInputEdit(i," ms",-1,0,0)
  fixtureAssert(len(updates)==3 && updates[2].Text=="8 ms","native suffix used stale select-all baseline")
 })
 run("legacy-capacity-retry",func(){
  view:=fixtureInputView();defer fixtureReleaseInputView(view)
  allow:=false;var updates []Update
  i:=Attach(view,func(u Update) bool {if !allow{return false};updates=append(updates,u);return true},func(){})
  fixtureAssert(i!=nil,"attach failed");defer i.Close()
  fixtureInputKey(i,"8",false)
  fixtureAssert(fixtureInputBaseCalls(i)==1,"capacity refusal lost legacy key")
  fixtureAssert(!i.UpdateClient(13,"6 ms",4,4,-1,-1,true,0,rect),"capacity refusal permitted stale activation")
  allow=true
  fixtureAssert(!i.UpdateClient(13,"8",1,1,-1,-1,true,0,rect),"retry marker skipped owner acknowledgment")
  fixtureAssert(len(updates)==1 && updates[0].Kind=="legacy","publication did not retry bounded legacy marker")
  fixtureAssert(i.UpdateClient(13,"8",1,1,-1,-1,true,1,rect),"retry acknowledgment did not restore native input")
 })
}
func TestNativeLegacyHandoff(t *testing.T) {
 for _,name:=range []string{"legacy-handoff","legacy-select-all","legacy-capacity-retry"} {t.Run(name,func(t *testing.T){if err:=nativeFixtureResults[name];err!=nil{t.Fatal(err)}})}
}
''')
print('isolated SDK copy and three real-Cocoa ordering regressions prepared')

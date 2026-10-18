from pathlib import Path
base=Path('/Users/dshills/Development/projects/ELab/build/native-handoff-repair/sdk/drivers/cogent')
p=base/'internal/textinput/input_darwin.m';s=p.read_text()
s=s.replace('uint64_t sequence, clientID, requestedLimitClient;','uint64_t sequence, clientID, requestedLimitClient, legacyRequired;')
s=s.replace('BOOL deltas, active, forwarding, forwarded, closed;','BOOL deltas, active, forwarding, forwarded, closed, legacyUnsent;')
s=s.replace('return s && s->active && !s->closed && !s->forwarding;','return s && s->active && !s->closed && !s->forwarding && !s->legacyUnsent && s->legacyRequired==0;')
needle='static BOOL finish(VellumManagedInputState *s, BOOL cancel) {';assert s.count(needle)==1
s=s.replace(needle,'''// A fallback key reaches the toolkit before this marker enters its FIFO. Do
// not activate Cocoa with a snapshot captured before that key or its selection.
// Capacity refusal keeps fallback routing active; the next publication retries.
static BOOL queueLegacy(VellumManagedInputState *s) {
 if(!s || s->closed || !s->legacyUnsent || s->sequence==UINT64_MAX)return NO;
 if(!goVellumManagedPreflight(s->handle,s->sequence+1,0,"legacy","","",0,0,-1,-1))return NO;
 s->legacyUnsent=NO;s->legacyRequired=s->sequence+1;emit(s);return YES;
}
'''+needle)
needle='s->forwarded=YES;s->forwarding=YES;';assert s.count(needle)==1;s=s.replace(needle,needle+'s->legacyUnsent=YES;')
needle='s->forwarding=NO;\n}';assert s.count(needle)==1;s=s.replace(needle,'s->forwarding=NO;queueLegacy(s);\n}')
needle='if(!usable(s)){struct objc_super p=parent(self);((void(*)(struct objc_super *,SEL,id))objc_msgSendSuper)(&p,_cmd,event);return;}'
assert s.count(needle)==1;s=s.replace(needle,'''if(!usable(s)){
  if(s && !s->closed){s->keyEvent=event;s->forwarded=NO;forwardKey(self);s->keyEvent=nil;}
  else{struct objc_super p=parent(self);((void(*)(struct objc_super *,SEL,id))objc_msgSendSuper)(&p,_cmd,event);}
  return;
 }''')
needle='if(!s || s->closed)return NO;\n if(!text';assert s.count(needle)==1;s=s.replace(needle,'if(!s || s->closed)return NO;\n if(s->legacyUnsent){queueLegacy(s);return NO;}\n if(!text')
needle='if(acknowledged<s->sequence)return NO;\n s->clientID=clientID;';assert s.count(needle)==1;s=s.replace(needle,'if(acknowledged<s->sequence)return NO;\n s->legacyRequired=0;\n s->clientID=clientID;')
p.write_text(s)
p=base/'internal/textinput/encoding.go';s=p.read_text();needle='type RuneUpdate struct {';assert s.count(needle)==1;s=s.replace(needle,'''// IsLegacyBoundary recognizes an empty acknowledgment marker, never a text
// mutation or permission to apply a stale editor update. Its FIFO position is
// after the fallback key that changed toolkit text or selection.
func IsLegacyBoundary(u Update) bool {
 return u.Kind == "legacy" && u.Sequence != 0 && u.ClientID == 0 && !u.Delta &&
  u.Text == "" && u.BeforeText == "" && u.ReplaceStart == 0 && u.ReplaceEnd == 0 &&
  u.Anchor == 0 && u.Focus == 0 && u.MarkedStart == -1 && u.MarkedEnd == -1
}

'''+needle);p.write_text(s)
p=base/'editor.go';s=p.read_text();needle='\tr := t.clients[u.ClientID]\n\tshadow, known := t.shadows[u.ClientID]';assert s.count(needle)==1;s=s.replace(needle,''' // The fallback key already ran earlier in this window's FIFO. Acknowledge
 // its position and request publication through the existing defer; do not
 // decode text, mutate a field or weaken any stale-baseline guard.
 if textinput.IsLegacyBoundary(u) { return }
'''+needle)
needle='func (t *textWindow) gridInputAllowed(u textinput.Update) bool {\n';assert s.count(needle)==1;s=s.replace(needle,needle+'\tif u.Kind == "legacy" { return textinput.IsLegacyBoundary(u) }\n');p.write_text(s)
print('isolated native legacy boundary repair applied')

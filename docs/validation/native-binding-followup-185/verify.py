"""Read-only bounded audit replay; no app/input/capture or inventory output."""
from pathlib import Path
import gzip,hashlib,json,subprocess,sys
if sys.flags.optimize:raise SystemExit('Assertions required')
P=Path(__file__).resolve().parent;R=P.parents[2];h=lambda raw:hashlib.sha256(raw).hexdigest()
c=json.loads((P/'checkpoint.json').read_text())['files'];assert set(c)=={str(f.relative_to(P)) for f in P.rglob('*') if f.is_file() and f!=P/'checkpoint.json'}
for n,d in c.items():assert h((P/n).read_bytes())==d,n
for n,v in json.loads((P/'manifest.json').read_text()).items():
 raw=(P/n).read_bytes();decoded=gzip.decompress(raw);assert h(raw)==v['stored_sha256'] and h(decoded)==v['original_sha256'] and len(decoded)==v['original_bytes']
read=lambda n:json.loads(gzip.decompress((P/(n+'.gz')).read_bytes()))
a=json.loads((P/'artifacts.json').read_text());results=read('result.json');ledger=R/'research/13-native-command-inventory.json';assert h(ledger.read_bytes())==a['ledger_sha256'] and gzip.decompress((P/'current-candidate.json.gz').read_bytes())==ledger.read_bytes()
for n,d in a['application_go_sources'].items():assert h((R/n).read_bytes())==d,n
for n,d in a['reviewed_files'].items():assert h((R/n).read_bytes())==d,n
old=read('old-candidate.json');new=read('current-candidate.json');assert len(old['sites'])==375 and len(new['sites'])==400 and len(new['source_sha256'])==72 and len(new['public_commands'])==164 and len(new['commands_with_source_routes'])==132 and not new['issues']
assert old['source_sha256']==new['source_sha256'] and sorted(set(new['public_commands'])-set(old['public_commands']))==['appearance.work.cancel','appearance.work.inspect']
assert [s['exit_code'] for s in results['steps']]==[1,0,0,0]
tool=R/'build/elab-ui-audit';assert h(tool.read_bytes())==a['current_tool_sha256'] and h((R/'build/native-bindings-review-185/elab-ui-audit-before').read_bytes())==a['prior_tool_sha256']
r=subprocess.run([str(tool),'--verify',str(ledger)],cwd=R,capture_output=True,text=True,timeout=15);assert r.returncode==0,(r.stdout,r.stderr)
assert h((R/'research/ui-audit/first-version/inspector-width-review-checkpoint.json').read_bytes())==a['current_receipt_sha256'] and h((R/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==a['canonical_sha256']
print(json.dumps(dict(passed=True,ledger_unchanged=True,application_go_sources_unchanged=len(a['application_go_sources']),native_sources=72,sites=400,public_commands=164,source_routes=132,app_started=False,full_binding_qualification=False)))

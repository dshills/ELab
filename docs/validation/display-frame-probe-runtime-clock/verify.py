"""Read-only current-source, preserved-artifact and rational frame replay."""
from pathlib import Path
import gzip,hashlib,json,subprocess,sys
folder=Path(__file__).resolve().parent;root=folder.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
read=lambda p:json.loads(p.read_text())
cp=read(folder/'checkpoint.json')
actual={p.relative_to(folder).as_posix():sha(p) for p in folder.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
assert actual==cp['evidence_sha256']
assert len(cp['source_sha256'])==7 and all(sha(root/n)==d for n,d in cp['source_sha256'].items())
assert sha(root/'build/display-frame-probe-tools/display-frame-probe')==cp['current_tool_sha256']
assert sha(root/cp['preserved_initial_tool']['path'])==cp['preserved_initial_tool']['sha256']
oldfolder=root/'docs/validation/display-frame-probe-offline';old=read(oldfolder/'checkpoint.json')
assert sha(oldfolder/'checkpoint.json')==cp['initial_offline_checkpoint_sha256']
assert all(sha(oldfolder/n)==d for n,d in old['evidence_sha256'].items())
assert cp['preserved_initial_tool']['sha256']==old['tool_sha256']
changed=sorted(n for n,d in old['source_sha256'].items() if cp['source_sha256'][n]!=d)
assert changed==['tools/display-frame-probe/checks.swift','tools/display-frame-probe/collector.swift']
for name,digest in old['source_sha256'].items():
    if name.endswith('.swift'):
        archived=folder/'initial-clock-refusal/producers'/(Path(name).name+'.initial.txt.gz')
        assert hashlib.sha256(gzip.decompress(archived.read_bytes())).hexdigest()==digest
    else:assert sha(root/name)==digest
assert all(sha(root/n)==d for n,d in cp['preserved_artifacts'].items())
quality=read(root/'docs/validation/sweep-smoke-admission/quality-default.json');assert len(quality['source_sha256'])==650
assert all(sha(root/n)==d for n,d in quality['source_sha256'].items())
identity=json.loads(gzip.decompress((folder/'nominal/recorder-source-identities.json.gz').read_bytes()))
assert identity['source_sha256']==cp['source_sha256'] and identity['binary_sha256']==cp['current_tool_sha256'] and not identity['clean_source_claim']
worker=json.loads(gzip.decompress((folder/'nominal/capture-worker.json.gz').read_bytes()))
assert worker['exit_code']==0 and worker['source_sha256_unchanged'] and worker['eligible_latency_observations']==0
checks=json.loads(gzip.decompress((folder/'display-frame-probe-future-final.txt.gz').read_bytes()))
assert checks['self_tests_passed'] and len(checks['cases'])+len(checks['journal_cases'])==55 and not checks['capture_executed']
assert gzip.decompress((folder/'display-frame-probe-future-format.txt.gz').read_bytes())==b''
assert cp['eligible_latency_observations']==0 and cp['latency_gate_passed'] is None and not cp['physical_clock_or_visibility_qualified']
subprocess.run([sys.executable,'-B',str(folder/'analyze.py')],check=True)
print(dict(passed=True,current_offline_checks=55,source_and_original_recorder_bound=True,go_sources_unchanged=650,eligible_latency_observations=0))

"""Read-only source/artifact binding, cleanup checks and causal-image replay."""
from pathlib import Path
import gzip,hashlib,json,subprocess,sys

folder=Path(__file__).resolve().parent;root=folder.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def read(p):return json.loads(p.read_bytes() if p.exists() else gzip.decompress(p.with_name(p.name+'.gz').read_bytes()))
checkpoint=read(folder/'checkpoint.json')
assert {p.relative_to(folder).as_posix():sha(p) for p in folder.rglob('*') if p.is_file() and p.name!='checkpoint.json'}==checkpoint['evidence_sha256']
assert all(sha(root/name)==digest for name,digest in checkpoint['recorder_source_sha256'].items())
assert all(sha(Path(name))==digest for name,digest in checkpoint['preserved_artifacts'].items())
quality=read(root/'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256'])==650
assert all(sha(root/name)==digest for name,digest in quality['source_sha256'].items())
session=read(folder/'report.json')
assert all(session[k] for k in ['owned_process_absent','endpoint_removed','baseline_file_unchanged',
    'shared_geometry_unchanged','review_unchanged','copied_bundle_unchanged','canonical_unchanged'])
assert not session['forced_cleanup'] and session['open_exit_code']==0 and session['native_exit_status'] is None
closed=read(folder/'close-response.json')
assert closed['closed'] and closed['shutdown']['exit_ready'] and closed['shutdown']['status']=='completed'
assert read(folder/'cua-postflight.json')['running_elab']==[]
assert sha(folder/'baseline.elab')==session['baseline_file_sha256']
assert read(folder/'final.json')['result']==read(folder/'after-apply.json')['result']
assert read(folder/'trial-plan.json')['repeat_policy']=='One attempt; retain failures. Not a performance cohort.'
assert checkpoint['eligible_latency_observations']==0 and checkpoint['latency_gate_passed'] is None
subprocess.run([sys.executable,'-B',str(folder/'analyze.py'),str(folder),'--verify'],check=True)
print(dict(passed=True,causal_composition_sequence=True,go_sources_unchanged=650,
    eligible_latency_observations=0,latency_gate_passed=None))

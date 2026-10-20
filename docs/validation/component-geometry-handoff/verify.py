"""Read-only integrity replay of the component geometry value-draft repair."""
from pathlib import Path
from collections import Counter
import gzip
import hashlib
import json
import re
import subprocess
import sys

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')

HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]

def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def raw(name):return gzip.decompress((HERE/(name+'.gz')).read_bytes())
def read(name):return json.loads((HERE/name).read_text())

def main():
    cp=read('checkpoint.json')
    files={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
    assert files==set(cp['files'])
    for relative,digest in cp['files'].items():
        path=HERE/relative
        assert not path.is_symlink() and path.resolve().is_relative_to(HERE) and sha(path)==digest,relative
    report=read('report.json');quality=read('quality.json');native=read('native-smoke.json')
    assert report['passed'] and report['status']=='source_qualified_replacement_pending'
    assert report['owner_walkthrough']=='Not tried yet'
    assert not any(report[k] for k in ('app_release_created','canonical_promoted','new_physical_input','new_visible_latency','models_changed','commands_changed','format_changed','sdk_changed','owner_acceptance','full_mvp_complete'))
    assert len(quality['source_sha256'])==report['source_files']==654
    assert quality['passed'] and quality['exit_code']==0 and quality['source_unchanged']
    for relative,digest in quality['source_sha256'].items():assert sha(ROOT/relative)==digest,relative
    assert sha(ROOT/'scripts/check.sh')==cp['quality_runner_sha256']
    for archived,source in [('original-value_apply.go.txt','desktop/value_apply.go'),('original-component_inspector.go.txt','desktop/component_inspector.go'),('original-audit.go.txt','internal/uiaudit/audit.go')]:
        data=subprocess.check_output(['git','show',report['original_source_commit']+':'+source],cwd=ROOT,timeout=15)
        assert data==(HERE/archived).read_bytes()
    original=raw('original-final-result.txt')
    assert original.count(b'one geometry click must settle only its draft and act once:')==report['original_final_test_failures']==12
    assert b'build failed' not in original and b'undefined:' not in original
    assert raw('original-result.txt').count(b'one geometry click refused its own value draft:')==2
    assert raw('focused-result.txt').count(b'Could not finish value entry: field retired')==2
    assert re.search(rb'^ok\s+github.com/dshills/elab/desktop\s',raw('focused-final-result.txt'),re.M)
    output=raw('quality-output.txt')
    assert b'Ran 65 tests' in output and b'\nOK\n' in output
    assert len(re.findall(rb'^ok\s+github.com/dshills/elab/desktop\s',output,re.M))==2
    assert quality['elapsed_seconds']==report['source_quality_elapsed_seconds']
    assert native['passed'] and native['exit_code']==0 and native['endpoint_absent']
    assert native['source_unchanged'] and native['shared_geometry_unchanged'] and native['geometry_before']==native['geometry_after']
    assert native['binary_sha256']==native['runtime_binary_sha256']
    assert sha(ROOT/'build/component-copy-handoff/elab-native')==native['binary_sha256']
    assert sha(Path(native['argv'][3]))==native['runtime_binary_sha256']
    assert Path(native['argv'][2]).read_bytes()==raw('runtime.sb')
    assert raw('runtime.sb')==gzip.decompress((ROOT/'docs/validation/pinned-file-review-178/runtime.sb.gz').read_bytes())
    assert not Path(native['argv'][-1]).exists()
    assert native['elapsed_seconds']==report['general_native_smoke_elapsed_seconds']
    assert b'"event":"elab.native.smoke","passed":true' in raw('native-smoke-output.txt')
    assert b'vcs.modified=true' in raw('native-build-info.txt')
    before=json.loads(raw('original-inventory.json'));after=json.loads(raw('inventory-candidate.json'))
    assert after==json.loads((ROOT/'research/13-native-command-inventory.json').read_text())
    assert {n for n,v in before['source_sha256'].items() if after['source_sha256'][n]!=v}=={'desktop/value_apply.go','desktop/component_inspector.go'}
    for key in before.keys()-{'source_sha256','sites'}:assert before[key]==after[key],key
    assert len(after['source_sha256'])==72 and len(after['sites'])==400
    assert len(after['public_commands'])==164 and len(after['commands_with_source_routes'])==132
    normalize=lambda s:json.dumps({k:v for k,v in s.items() if k!='line'},sort_keys=True)
    a,b=Counter(map(normalize,before['sites'])),Counter(map(normalize,after['sites']))
    removed=[json.loads(s) for s,n in (a-b).items() for _ in range(n)]
    added=[json.loads(s) for s,n in (b-a).items() for _ in range(n)]
    review=read('inventory-review.json')
    assert removed==review['removed'] and added==review['added'] and len(removed)==3 and len(added)==6
    assert all(s['function']=='*screen.valueSetAction' for s in removed)
    assert len([s for s in added if s['kind']=='action-factory'])==3
    preservation=read('preservation.json')
    receipt_path=ROOT/'research/ui-audit/first-version/pin-reader-review-checkpoint.json'
    assert sha(receipt_path)==preservation['receipt_sha256']
    release=json.loads(receipt_path.read_text())
    changed={n for n,v in release['quality']['source_sha256'].items() if quality['source_sha256'][n]!=v}
    assert changed=={'desktop/component_inspector.go','desktop/value_apply.go','internal/uiaudit/audit.go','internal/uiaudit/audit_test.go'}
    assert set(quality['source_sha256'])-set(release['quality']['source_sha256'])=={'desktop/component_geometry_actions_test.go','desktop/component_geometry_actions_offscreen_test.go'}
    folder=Path(release['review_path']);manifest=folder/'RELEASE_MANIFEST.json'
    assert sha(manifest)==preservation['manifest_sha256']
    sys.path.insert(0,str(ROOT/'scripts'))
    from package_review import members
    actual=members(folder);actual.pop('RELEASE_MANIFEST.json')
    assert actual==json.loads(manifest.read_text())['files'] and len(actual)==preservation['manifest_entries']==529
    assert sha(ROOT/'build/ELab-0.161.0-178-review.zip')==preservation['archive_sha256']
    assert sha(ROOT/'build/ELab.app/Contents/MacOS/ELab')==preservation['canonical_native_sha256']
    assert sha(ROOT/preservation['sdk']['archive'])==preservation['sdk']['sha256']
    print('Verified GEOMETRY-DRAFT-01: 12 original failures, 36 repaired/guard scenarios, full source quality, development native smoke and preserved /178; replacement pending.')

if __name__=='__main__':main()

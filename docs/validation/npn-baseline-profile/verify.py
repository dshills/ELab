"""Read-only Go/Pillow NPN saved-image profile replay; no native operation."""
from pathlib import Path
import gzip,hashlib,io,json,subprocess,sys,tarfile,tempfile
if sys.flags.optimize:raise SystemExit('Assertions required')
from PIL import Image
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2];h=lambda raw:hashlib.sha256(raw).hexdigest()
def main():
 cp=json.loads((HERE/'checkpoint.json').read_text());assert set(cp['files'])=={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p!=HERE/'checkpoint.json'}
 for n,digest in cp['files'].items():assert not (HERE/n).is_symlink() and h((HERE/n).read_bytes())==digest
 report=json.loads((HERE/'report.json').read_text());cases=json.loads((HERE/'cases.json').read_text());profile_path=ROOT/'tools/baseline-admission/profiles/npn.json';profile=json.loads(profile_path.read_text());tool=ROOT/'build/baseline-admission-tools/baseline-admission'
 assert profile_path.read_bytes()==(HERE/'profile.json').read_bytes() and h(profile_path.read_bytes())==report['profile_sha256'] and h(tool.read_bytes())==report['tool_sha256']
 original=ROOT/'docs/validation/latency-npn-cohort-173';annotation=json.loads((original/'image-annotations.json').read_text());baseline=next(v for v in annotation['states'] if v['role']=='private-draft-original-result');assert profile==dict(version=1,width=1053,height=802,regions=[dict(name=n,rect=rect,rgb_sha256=baseline['hashes'][n]) for n,rect in annotation['regions'].items()])
 artifacts=json.loads((HERE/'artifacts.json').read_text());assert h((original/'checkpoint.json').read_bytes())==artifacts['prior_npn_checkpoint_sha256'] and h((ROOT/'docs/validation/native-baseline-admission-173/checkpoint.json').read_bytes())==artifacts['prior_native_admission_checkpoint_sha256']
 assert h((ROOT/'research/ui-audit/first-version/inspector-width-review-checkpoint.json').read_bytes())==artifacts['current_receipt_sha256'];assert len(artifacts['current_go_sources'])==659 and all(h((ROOT/n).read_bytes())==digest for n,digest in artifacts['current_go_sources'].items())
 assert h((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==artifacts['canonical_sha256']
 assert len(cases)==41 and sum(c['expected_admitted'] for c in cases)==20
 with tempfile.TemporaryDirectory(prefix='elab-npn-profile-replay-',dir='/private/tmp') as temp:
  for c in cases:
   if c['id']=='npn-explicit-stale':raw=(original/'stale.png').read_bytes()
   else:
    path=ROOT/c['source_archive'];assert h(path.read_bytes())==c['source_archive_sha256'];journal=gzip.decompress((path.parent/'frames.jsonl.gz').read_bytes());assert h(journal)==c['source_journal_sha256'];rows=[json.loads(s) for s in journal.splitlines()];frame=next(v for v in rows if v.get('png')==c['png'] and v.get('index')==c['frame_index']);assert frame['png_sha256']==c['png_sha256']
    with tarfile.open(path,'r:xz') as tar:
     m=tar.getmember(c['png']);assert m.isfile() and m.mode==0o600;raw=tar.extractfile(m).read()
   assert h(raw)==c['png_sha256'];rgba=Image.open(io.BytesIO(raw)).convert('RGBA');regions=[]
   for region in profile['regions']:
    crop=rgba.crop(region['rect']);assert crop.getchannel('A').getextrema()==(255,255);digest=h(crop.convert('RGB').tobytes());regions.append(dict(name=region['name'],rgb_sha256=digest,matches=digest==region['rgb_sha256']))
   assert regions==c['pillow_regions']==c['decision']['regions'] and all(v['matches'] for v in regions)==c['expected_admitted']
   image=Path(temp)/'original.png';image.write_bytes(raw);p=subprocess.run([str(tool),'--profile',str(profile_path),'--image',str(image),'--sha256',c['png_sha256']],capture_output=True,text=True,timeout=2);assert p.returncode==c['tool_exit_code'] and json.loads(p.stdout)==c['decision']
 assert report['passed'] and report['region_comparisons']==246 and not any(report[k] for k in ('actual_capture','actual_input','native_profile_qualified','app_code_changed'))
 print(json.dumps(dict(passed=True,cases=41,baseline_admitted=20,fresh_refused=20,stale_refused=1,go_pillow_regions=246,native_profile_qualified=False)))
if __name__=='__main__':main()

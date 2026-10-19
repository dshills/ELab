from pathlib import Path
import hashlib,json,stat
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/ax-recovery-diagnostic';prior=json.loads((root/'docs/validation/number-field-placement/preservation.json').read_text());rows=prior['qualified_reviews']+[dict(folder='build/ELab-0.142.0-159-review',entries=257,manifest_sha256='4322975a12d56b48ee40f2d5ebb0d9432a4ae55653422f6bce84130eb10eaab2')]
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def verify(folder,entries,digest,manifest='RELEASE_MANIFEST.json'):
 p=root/folder;m=p/manifest;assert sha(m)==digest;data=json.loads(m.read_text());assert len(data['files'])==entries
 for name,entry in data['files'].items():
  f=p/name;assert f.is_file() and not f.is_symlink() and sha(f)==entry['sha256'],str(f);assert oct(stat.S_IMODE(f.stat().st_mode))==entry['mode'],str(f)
 return dict(folder=folder,manifest=manifest,entries=entries,manifest_sha256=digest)
verified=[verify(row['folder'],row['entries'],row['manifest_sha256']) for row in rows]
original=verify('build/ELab-0.130.0-147-review',prior['original_147_candidate_entries'],prior['original_147_candidate_manifest_sha256'],'CANDIDATE_MANIFEST.json');canonical=root/'build/ELab.app/Contents/MacOS/ELab';assert sha(canonical)==prior['canonical_native_sha256'];r=dict(passed=True,qualified_reviews=verified,original_147_candidate=original,canonical_native_sha256=sha(canonical),canonical_promoted=False,scope='All listed sealed manifest entries and file modes verified; current candidate remains development-only')
(base/'preservation.json').write_text(json.dumps(r,indent=2)+'\n');print('All 20 qualified reviews, original /147 and canonical preserved')

import gzip,hashlib,io,json,shutil,sys,tarfile
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).resolve().parent;out=base/'observations'
destination=root/'research/ui-audit/first-version/native-recovery-161';destination.mkdir()
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json');sha=lambda path:hashlib.sha256(path.read_bytes()).hexdigest()

def compressed(source,target):
    with target.open('wb') as f:
        with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write(source.read_bytes())
    assert gzip.decompress(target.read_bytes())==source.read_bytes()

r=json.loads((out/'report.json').read_text());assert not r['passed'] and r['saved_fields_except_view_match'];reopen=json.loads((out/'reopen/report.json').read_text());assert reopen['passed'] and reopen['native_exit_code']==0
for name in ['identity.json','setup.py','observe.py','finish.py','preflight.py','preflight.json','preflight-read.json','reopen.py','observer-output.txt','reopen-output.txt']:
    shutil.copyfile(base/name,destination/name)
for name in ['Agent baseline.elab','Agent recovered.elab','Original automatic snapshot.elab-recovery','cleanup.json','independent-led-checks.json','corrected-close.json','close-protocol-correction.txt','fresh-agent-reopen.json.gz']:
    shutil.copyfile(out/name,destination/name)
compressed(out/'report.json',destination/'original-session.json.gz');compressed(out/'close-correction-before.json',destination/'close-correction-before.json.gz')
for phase in ['first','restored','reopen']:
    target=destination/phase;target.mkdir()
    for path in (out/phase).iterdir():
        if path.suffix=='.elab' or path.name=='native-host.txt':shutil.copyfile(path,target/path.name)
        elif path.name.endswith('-completion.json'):compressed(path,target/(path.name+'.gz'))
    if phase=='reopen':compressed(out/phase/'report.json',target/'session.json.gz')
captures=[]
with (destination/'ui-captures.tar.gz').open('wb') as f:
    with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:
        with tarfile.open(fileobj=z,mode='w') as tar:
            for phase in ['first','restored','reopen']:
                for path in sorted((out/phase).iterdir()):
                    if not path.is_file() or not(path.name.endswith('.ax.txt') or path.suffix in ['.jpg','.png']):continue
                    payload=path.read_bytes();name=path.stem+'.jpg' if payload[:2]==b'\xff\xd8' else path.name;actual=phase+'/'+name
                    info=tarfile.TarInfo(actual);info.size=len(payload);info.mode=0o644;tar.addfile(info,io.BytesIO(payload));captures.append(dict(original_name=phase+'/'+path.name,archive_name=actual,bytes=len(payload),sha256=sha(path)))
with tarfile.open(destination/'ui-captures.tar.gz') as tar:
    for item in captures:assert hashlib.sha256(tar.extractfile(item['archive_name']).read()).hexdigest()==item['sha256']
(destination/'ui-capture-index.json').write_text(json.dumps(dict(scope='Unmodified CUA images and accessibility observations; JPEG naming follows bytes. No endpoint credentials or executable backups.',captures=captures),indent=2)+'\n')
for phase,name in [('first','edited-fresh'),('restored','recovery-inspected'),('restored','recovered-saved'),('reopen','reopen-loaded')]:
    for suffix in ['.jpg','.ax.txt']:shutil.copyfile(out/phase/(name+suffix),destination/(name+suffix))
capsule=json.loads((out/'Original automatic snapshot.elab-recovery').read_text());lease=base/'recovery'/('session-'+capsule['session']+'.json');assert json.loads(lease.read_text())['pid']==r['hosts'][0]['pid'];shutil.copyfile(lease,destination/'crashed-owner-lease.json')
canonical=root/'build/ELab.app/Contents/MacOS/ELab';assert sha(canonical)=='914c1ba7547d9d1608cc1c8a0ff1bdeca42918e9343108b712e933c43e5d5b7c'
(destination/'preservation.json').write_text(json.dumps(dict(passed=True,current_review=release,canonical_native_sha256=sha(canonical),canonical_promoted=False,manual_save_sha256=r['manual_save_sha256'],original_capsule_sha256=r['automatic_capsule_sha256'],independent_generator_sha256=sha(root/'scripts/freeze-electrical-reference.py')),indent=2)+'\n')
shutil.copyfile(base/'package.py',destination/'package.py')
print('Packaged native automatic recovery, original failed controller, corrected close, successful fresh native reopen, preserved files/capsule and raw observations')

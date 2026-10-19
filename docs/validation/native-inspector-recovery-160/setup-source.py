from pathlib import Path
import hashlib,json,plistlib,shutil,subprocess,sys
root=Path('/Users/dshills/Development/projects/ELab');stage=root/'build/first-version-160-tools';review=root/'build/ELab-0.143.0-160-review';bundle=root/'build/native-files-156-ui/ELab-value-recovery.app';binary=bundle/'Contents/MacOS/ELab-value-recovery';sys.path.insert(0,str(root/'scripts'));from runtime_isolation import bundle_tree
live=[row for row in subprocess.check_output(['/bin/ps','-axo','pid=,comm='],text=True).splitlines() if 'ELab-value-recovery' in row];assert not live,live
before=stage/'harness-before';before.mkdir();saved={}
for rel in ['Contents/MacOS/ELab-value-recovery','Contents/Info.plist','Contents/Resources/APP_MANIFEST.json']:
 p=bundle/rel;shutil.copyfile(p,before/p.name);saved[rel]=hashlib.sha256(p.read_bytes()).hexdigest()
assert saved['Contents/MacOS/ELab-value-recovery']=='570b805b42a9f3ee01acb6fecd3bdea5420c57a0b38dd247d1c52689127d82b5'
shutil.copyfile(review/'ELab.app/Contents/MacOS/ELab',binary);binary.chmod(0o755);info=plistlib.loads((bundle/'Contents/Info.plist').read_bytes());info['CFBundleShortVersionString']='0.143.0';info['CFBundleVersion']='160';(bundle/'Contents/Info.plist').write_bytes(plistlib.dumps(info));shutil.copyfile(review/'ELab.app/Contents/Resources/APP_MANIFEST.json',bundle/'Contents/Resources/APP_MANIFEST.json')
out=stage/'native-observations';out.mkdir();shutil.copyfile(root/'docs/validation/inspector-accessibility/Native corrected LED.elab',out/'Baseline.elab')
identity=dict(scope='Exact clean /160 executable in a disposable custom-identifier/name bundle; modified plist, not intact sealed-bundle acceptance',source_commit='4403097eb81cdc558d8e00e49c387149022a1c75',clean_build_vcs=True,version='0.143.0 /160',native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),agent_sha256=hashlib.sha256((review/'elab-agent').read_bytes()).hexdigest(),harness_before=saved,sdk='b22478cf8cec8dda1ccdcd93d6ff6b62e2cc11de',package_tree=bundle_tree(review/'ELab.app'))
assert identity['native_sha256']=='4158b9b1332e36a458c03c40c1075c72127a6aadd728dafc5b8db5aebb512eef';assert identity['agent_sha256']=='bf177e5f3d136196745f09bb4f090bc87ce7b45c82218fd5991fc890acd86cc4';(stage/'native-identity.json').write_text(json.dumps(identity,indent=2)+'\n');print('Stopped test bundle prepared for exact /160 replay; original bytes preserved')

import hashlib,importlib.util,json,re,shlex,subprocess
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');stage=Path(__file__).resolve().parent;out=Path('/private/tmp/elab-readme-packaging-165');out.mkdir(mode=0o700)
readme=(root/'README.md').read_text();blocks=[b for b in re.findall(r'```sh\n(.*?)```',readme,re.S) if 'scripts/package-app.py' in b];assert len(blocks)==1
binary=root/'build/first-version-165-tools/build/elab';bundle=out/'ELab-local.app';snippet=blocks[0];assert snippet.count('--binary build/elab')==1 and snippet.count('--output build/ELab-local.app')==1
snippet=snippet.replace('--binary build/elab','--binary '+shlex.quote(str(binary))).replace('--output build/ELab-local.app','--output '+shlex.quote(str(bundle)))
(stage/'executed-package-snippet.sh').write_text(snippet)
r=dict(passed=False,scope='The current README SDK resolver and development packager CLI, with only binary/output arguments substituted to use the clean /165 binary and a new private destination. No source rebuild, application launch or release qualification claim.',binary=str(binary),binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),output=str(bundle),substitutions=['--binary uses existing clean /165 executable','--output uses new temporary test destination'])
try:
 p=subprocess.run(['/bin/sh','-c',snippet],cwd=root,capture_output=True,text=True,timeout=120);(stage/'packaging-stdout.txt').write_text(p.stdout);(stage/'packaging-stderr.txt').write_text(p.stderr);r['exit_code']=p.returncode;p.check_returncode()
 import plistlib
 info=plistlib.loads((bundle/'Contents/Info.plist').read_bytes());manifest=json.loads((bundle/'Contents/Resources/APP_MANIFEST.json').read_text());assert info['CFBundleShortVersionString']=='0.0.0' and info['CFBundleVersion']=='1' and info['CFBundleIdentifier']=='com.dshills.elab';assert manifest['source_binary_sha256']==r['binary_sha256']==hashlib.sha256((bundle/'Contents/MacOS/ELab').read_bytes()).hexdigest()
 s=importlib.util.spec_from_file_location('prepare',root/'scripts/prepare-vellum.py');m=importlib.util.module_from_spec(s);s.loader.exec_module(m);sdk=m.prepare_sdk();r.update(passed=True,verified_sdk=str(sdk),sdk_source=m.load_pin(root)['source_commit'],sdk_reverified_after_packaging=True,private_build=manifest['private_build'],development_version='0.0.0 /1')
 (stage/'packaged-Info.plist').write_bytes((bundle/'Contents/Info.plist').read_bytes());(stage/'packaged-APP_MANIFEST.json').write_bytes((bundle/'Contents/Resources/APP_MANIFEST.json').read_bytes())
finally:(stage/'packaging-report.json').write_text(json.dumps(r,indent=2)+'\n')
print(json.dumps({k:r.get(k) for k in ['passed','exit_code','sdk_source','development_version']}))

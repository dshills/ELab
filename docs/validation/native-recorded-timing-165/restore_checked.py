import datetime
import hashlib
import json
import os
from pathlib import Path
import plistlib
import subprocess
import sys

root=Path('/Users/dshills/Development/projects/ELab')
base=Path(__file__).resolve().parent
out=base/'observations'
bundle=root/'build/native-files-156-ui/ELab-value-recovery.app'
identity=json.loads((base/'identity.json').read_text())
report=json.loads((out/'report.json').read_text())
assert report['pid']==91137 and report['final_host_exit_code']==0 and report['normal_closure'] and report.get('ended_utc')
processes=subprocess.check_output(['/bin/ps','-axo','pid=,comm='],text=True)
assert not [line for line in processes.splitlines() if 'ELab-value-recovery' in line], 'Test bundle is still running; refusing to restore'
status=subprocess.run(['/bin/ps','-p',str(report['pid']),'-o','pid=,comm='],capture_output=True,text=True)
assert status.returncode==1 and not status.stdout.strip(), 'Original PID exists; inspect possible PID reuse before proceeding'

def digest(data):return hashlib.sha256(data).hexdigest()
backups={}
for relative,expected in identity['harness_before'].items():
    data=(base/'harness-before'/Path(relative).name).read_bytes()
    assert digest(data)==expected, relative
    backups[relative]=data
review=root/'build/ELab-0.148.0-165-review'
info=plistlib.loads(backups['Contents/Info.plist'])
info['CFBundleShortVersionString']='0.148.0'
info['CFBundleVersion']='165'
prepared={
    'Contents/MacOS/ELab-value-recovery':identity['native_sha256'],
    'Contents/Info.plist':digest(plistlib.dumps(info)),
    'Contents/Resources/APP_MANIFEST.json':digest((review/'ELab.app/Contents/Resources/APP_MANIFEST.json').read_bytes()),
}
before={relative:digest((bundle/relative).read_bytes()) for relative in backups}
assert before==prepared or before==identity['harness_before'], 'Unexpected bundle edits; refusing to overwrite'
if before!=identity['harness_before']:
    for relative,data in backups.items():
        destination=bundle/relative
        temporary=destination.with_name(destination.name+'.timing165-restore')
        assert not temporary.exists()
        temporary.write_bytes(data)
        temporary.chmod(destination.stat().st_mode & 0o777)
        os.replace(temporary,destination)
after={relative:digest((bundle/relative).read_bytes()) for relative in backups}
assert after==identity['harness_before']
sys.path.insert(0,str(root/'scripts'))
subprocess.run([sys.executable,'-B',str(root/'docs/validation/recorded-timing-feedback/verify.py')],check=True,stdout=subprocess.DEVNULL)
release=identity['release']
result=dict(passed=True,recorded_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),pid=report['pid'],observer_exit_code=0,native_host_exit_code=0,normal_closure=True,original_process_absent=True,no_test_bundle_processes=True,restored_from_verified_backups=True,before_sha256=before,restored_sha256=after,release_reverified=release,native_threshold_menu_test_completed=True,scope='Owned exact /165 native host closed normally after all threshold/menu and paging refusal/correction stages passed. Original capture pages/metadata and saved data remain intact. This native proof does not itself qualify the complete review package.')
(out/'cleanup.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(dict(passed=True,original_process_absent=True,bundle_restored=True,native_host_exit_code=0,native_interaction_completed=True,release=release['release'])))

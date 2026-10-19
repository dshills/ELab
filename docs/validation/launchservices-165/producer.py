import datetime,hashlib,json,os,re,signal,subprocess,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');out=Path('/private/tmp/elab-165-launchservices');out.mkdir(mode=0o700)
package=json.loads((root/'docs/validation/review-archive-165/checkpoint.json').read_text());source=json.loads(__import__('gzip').decompress((root/'docs/validation/review-archive-165/package-report.json.gz').read_bytes()));review=Path(source['extracted_review']);app=review/'ELab.app';binary=app/'Contents/MacOS/ELab';native_hash=hashlib.sha256(binary.read_bytes()).hexdigest();assert native_hash=='f75a72427b4556b163437132a46f0694b5845f6753db5aa9bf61f0d18e6a27de'
expected=source['expected_files']
def verify_review():
 files={str(p.relative_to(review)) for p in review.rglob('*') if p.is_file()};assert files==set(expected)
 assert not any(p.is_symlink() for p in review.rglob('*'))
 for name,e in expected.items():
  p=review/name;assert hashlib.sha256(p.read_bytes()).hexdigest()==e['sha256'] and oct(p.stat().st_mode&0o777)==e['mode'],name

def instances():
 rows=subprocess.check_output(['/bin/ps','-axo','pid=,comm='],text=True).splitlines();result=[]
 for row in rows:
  fields=row.strip().split(None,1)
  if len(fields)==2 and fields[1]==str(binary):result.append(int(fields[0]))
 return result

def command(pid):
 p=subprocess.run(['/bin/ps','-p',str(pid),'-o','command='],capture_output=True,text=True)
 return p.stdout.strip() if p.returncode==0 else None

verify_review();assert not instances(),'Extracted test app already running; no existing app will be operated'
endpoint=out/'owned-endpoint.json';stdout=out/'native-stdout.txt';stderr=out/'native-stderr.txt'
argv=['/usr/bin/open','-n','-W','-g','-a',str(app),'--stdout',str(stdout),'--stderr',str(stderr),'--args','--smoke','--smoke-timeout','2m','--agent-endpoint',str(endpoint)]
r=dict(passed=False,started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),scope='Exact extracted /165 bundle launched through macOS open/LaunchServices into its bounded native smoke mode, with private smoke recovery/appearance storage and window-geometry saving disabled. No direct executable launch, code signature/quarantine, Finder interaction, document association, default user configuration, clean-account/OS or owner acceptance claim.',argv=argv,archive_sha256=package['archive_sha256'],native_sha256=native_hash,owned_pids=[],observed_commands={},launcher_exit_code=None,native_process_exit_status=None,native_process_exit_status_reason='LaunchServices owns the native process; open -W exit status is not the target application exit status.',stages=[],forced_cleanup=False)
launch=None;start=time.monotonic()
def save():
 (out/'report.json').write_text(json.dumps(r,indent=2)+'\n')
try:
 with (out/'open-output.txt').open('x') as log:launch=subprocess.Popen(argv,stdout=log,stderr=-2)
 r['launcher_pid']=launch.pid;save();deadline=time.monotonic()+155
 while True:
  for pid in instances():
   if pid not in r['owned_pids']:
    cmd=command(pid);assert cmd and str(endpoint) in cmd and '--smoke' in cmd,(pid,cmd)
    r['owned_pids'].append(pid);r['observed_commands'][str(pid)]=cmd;save()
  if endpoint.exists():r['private_endpoint_observed']=True
  if launch.poll() is not None:break
  assert time.monotonic()<deadline,'LaunchServices wait exceeded bounded smoke deadline'
  time.sleep(.05)
 r['launcher_exit_code']=launch.returncode;assert launch.returncode==0
 assert len(r['owned_pids'])==1,'Exact launched app PID was not observed'
 assert not instances() and all(command(pid) is None for pid in r['owned_pids'])
 r['owned_native_process_absent']=True;r['endpoint_removed']=not endpoint.exists();assert r['endpoint_removed']
 text=stdout.read_text();err=stderr.read_text();assert [json.loads(line) for line in text.splitlines() if line.startswith('{"event":"elab.native.smoke"')]==[dict(event='elab.native.smoke',passed=True)]
 reference=(Path('/private/tmp/elab-first-version-165-runtime')/'native-smoke.log.txt').read_text()
 def stages(text):return [m.group(1) for line in text.splitlines() if (m:=re.search(r'smoke: (.*)$',line))]
 r['stages']=stages(err);r['qualified_reference_stages']=stages(reference);assert r['stages'] and r['stages']==r['qualified_reference_stages']
 assert not any(message in err for message in ['shutdown cleanup failed:','Recovery unavailable:','Appearance unavailable:','Application menus unavailable:'])
 verify_review();r.update(passed=True,native_smoke_success_event=True,smoke_stages_match_qualified_reference=True,extracted_review_unchanged=True,elapsed_s=time.monotonic()-start)
except BaseException as e:r['error']=type(e).__name__+': '+str(e);raise
finally:
 if launch and launch.poll() is None:launch.terminate();launch.wait(timeout=10)
 for pid in r['owned_pids']:
  cmd=command(pid)
  if cmd:
   assert cmd==r['observed_commands'][str(pid)] and str(endpoint) in cmd,'Cannot establish test-process ownership for cleanup'
   os.kill(pid,signal.SIGTERM);r['forced_cleanup']=True
 if r['forced_cleanup']:
  deadline=time.monotonic()+10
  while instances() and time.monotonic()<deadline:time.sleep(.05)
  r['owned_native_process_absent_after_cleanup']=not instances()
 r['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
print(json.dumps({key:r.get(key) for key in ['passed','launcher_exit_code','native_process_exit_status','owned_native_process_absent','native_smoke_success_event','elapsed_s']}))

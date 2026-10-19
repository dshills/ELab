from pathlib import Path
from PIL import Image
import gzip,hashlib,io,json,math,sys,tarfile
folder=Path(sys.argv[1])
read=lambda p:json.loads(p.read_bytes() if p.exists() else gzip.decompress(p.with_name(p.name+'.gz').read_bytes()))
ready=read(folder/'fixture/ready.json'); closed=read(folder/'fixture/closed.json')
archives={}
def image_bytes(stream,name):
    assert Path(name).name==name and name.endswith('.png')
    if stream not in archives:
        archives[stream]=tarfile.open(folder/stream/'frames.tar.xz','r:xz')
        manifest=read(folder/stream/'image-manifest.json')
        members=archives[stream].getmembers()
        assert len(members)==len(manifest) and {m.name for m in members}==set(manifest)
        for member in members:
            assert member.isfile() and member.mode==0o600
            raw=archives[stream].extractfile(member).read()
            assert hashlib.sha256(raw).hexdigest()==manifest[member.name]['sha256']
            assert len(raw)==manifest[member.name]['bytes']
    return archives[stream].extractfile(name).read()
W,H=1053,802
patterns=[('tl',(2,2,18,18),['1100','1010','0111','0001'],4),('tr',(W-18,2,W-2,18),['1011','0100','1101','0010'],4),('bl',(2,H-18,18,H-2),['0011','1110','1000','0101'],4),('br',(W-18,H-18,W-2,H-2),['0110','1001','0010','1101'],4),('occluder',(W//2-48,int(H*.65)-32,W//2+48,int(H*.65)+32),[''.join(str(int((r+c)%2==0)) for c in range(12)) for r in range(8)],8)]
def patch_score(image,box,bits,cell):
    x,y,right,bottom=box
    assert 0<=x<right<=W and 0<=y<bottom<=H,'Clipped ROI'
    values=list(image.crop(box).convert('L').get_flattened_data())
    expected=[bits[r//cell][c//cell]=='1' for r in range(bottom-y) for c in range(right-x)]
    assert len(values)==len(expected)
    score=sum((v>=128)==e for v,e in zip(values,expected))/len(values)
    groups=[[v for v,e in zip(values,expected) if e==b] for b in [False,True]]
    return dict(match_fraction=score,color_ranges=[max(g)-min(g) for g in groups],color_extrema=[[min(g),max(g)] for g in groups])
def correlation(a,b):
    assert len(a)==len(b) and a
    aa=sum(a)/len(a);bb=sum(b)/len(b)
    va=sum((v-aa)**2 for v in a);vb=sum((v-bb)**2 for v in b)
    return sum((x-aa)*(y-bb) for x,y in zip(a,b))/math.sqrt(va*vb) if va>0 and vb>0 else None
baseline=Image.open(folder/'cua-baseline.png').convert('RGB');assert baseline.size==(W,H)
truth=list(baseline.crop(patterns[-1][1]).convert('L').get_flattened_data())
answer=dict(streams={},eligible_latency_observations=0,latency_gate_passed=None)
for stream in ['capture','window-capture']:
    report=read(folder/stream/'report.json')
    assert report['command_error'] is None and report['collector_error'] is None
    assert report['eligible_latency_observations']==0 and report['latency_gate_passed'] is None
    rows=[json.loads(line) for line in (folder/stream/'frames.jsonl').read_text().splitlines()]
    cache={}; count=0; ticks=[]; rejected=0; outside=0
    for row in rows:
        assert row.get('eligible_for_latency') is False
        if not row.get('png'):
            rejected+=1;continue
        assert row['status']==0
        if not int(ready['ready_mach_ticks'])<int(row['display_mach_ticks'])<int(closed['closed_mach_ticks']):outside+=1
        ticks.append(int(row['display_mach_ticks']))
        raw=image_bytes(stream,row['png']);assert hashlib.sha256(raw).hexdigest()==row['png_sha256']
        if row['png'] not in cache:
            image=Image.open(io.BytesIO(raw)).convert('RGB');assert image.size==(W,H)
            scores={name:patch_score(image,box,bits,cell) for name,box,bits,cell in patterns}
            corr=correlation(truth,list(image.crop(patterns[-1][1]).convert('L').get_flattened_data()))
            if stream=='capture':
                assert all(p['match_fraction']==1 and max(p['color_ranges'])<=3 for p in scores.values()),scores
            else:
                assert all(p['match_fraction']<.85 for p in scores.values()),scores
                assert corr is not None and corr>=.95,corr
            cache[row['png']]=dict(patches=scores,underlying_correlation=corr)
        count+=1
    assert count==report['complete_frames'] and count>0
    answer['streams'][stream]=dict(complete_frames=count,metadata_only_rows=rejected,unique_images=len(cache),outside_fixture_lifetime=outside,
        first_display_mach_ticks=str(min(ticks)),last_display_mach_ticks=str(max(ticks)),image_metrics=cache)
sample_name=next(iter(read(folder/'capture/image-manifest.json')))
sample=Image.open(io.BytesIO(image_bytes('capture',sample_name))).convert('RGB')
negative={}
for name,box,bits,cell in patterns:
    shifted=(box[0]+1,box[1],box[2]+1,box[3])
    negative[name+'_one_pixel_shift_refused']=patch_score(sample,shifted,bits,cell)['match_fraction']<1
negative['wrong_scale_refused']=sample.resize((W*2,H*2)).size!=(W,H)
try:patch_score(sample,(W-8,2,W+8,18),patterns[0][2],4)
except AssertionError:negative['clipping_refused']=True
negative['blank_underlying_refused']=correlation(truth,[255]*len(truth)) is None
assert all(negative.values())
answer['negative_checks']=negative
answer['helper_error']=closed['error'];answer['calibration_attempt_accepted']=closed['error'] is None
assert answer==read(folder/'pixel-analysis.json'),'Independent archive replay differs'
for archive in archives.values():archive.close()
print(json.dumps({k:({n:{key:v for key,v in x.items() if key!='image_metrics'} for n,x in value.items()} if k=='streams' else value) for k,value in answer.items()},indent=2))

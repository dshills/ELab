"""Independent rational/byte replay of stored crops, without UI or native calls."""
from copy import deepcopy
from fractions import Fraction
from pathlib import Path
import gzip,hashlib,io,json,lzma,math,re,struct,tarfile

folder=Path(__file__).resolve().parent
read=lambda p:json.loads(gzip.decompress(p.read_bytes()) if p.suffix=='.gz' else p.read_bytes())

def require(ok,message):
    if not ok:raise ValueError(message)

def validate(rows,report,config):
    ratio=Fraction(report['timebase_numerator'],report['timebase_denominator'])
    start=int(config['start_mach_ticks']);deadline=int(config['deadline_mach_ticks'])
    require((deadline-start)*ratio==3_000_000_000,'Output interval differs from predeclared three seconds')
    require(config['seconds']==3 and not config['audio'] and not config['microphone'] and not config['permission_requested'],'Capture scope changed')
    require(config['filter']=='display-excluding-no-windows','Filter changed')
    require(report['command_error'] is None and report['collector_error'] is None and report['journal_close_error'] is None,'Collector failed')
    require(report['eligible_latency_observations']==0 and report['latency_gate_passed'] is None and not report['clock_calibrated_for_latency'] and not report['pixel_mapping_calibrated'] and not report['display_visibility_calibrated'],'Uncalibrated data promoted')
    plan=config['capture_plan'];require(plan['width_pixels']==1053 and plan['height_pixels']==802 and plan['point_pixel_scale']==1,'Recorded workstation geometry changed')
    require(plan['crop_is_explicit'] and not plan['crop_follows_target'],'Crop policy changed')
    require(len(rows)==report['frames_received_within_deadline']==164,'Incomplete callback data')
    previous=0;future=0;names=set();leads=[]
    for index,row in enumerate(rows,1):
        require(row['index']==index and row['event']=='frame' and row['status']==0,'Frame sequence/status changed')
        received=int(row['received_mach_ticks']);displayed=int(row['display_mach_ticks'])
        require(start<=received<=deadline and previous<=displayed<=deadline and displayed>0,'Frame outside frozen interval or regressing clock')
        previous=displayed
        presentation=Fraction(int(row['presentation_value']),row['presentation_timescale'])*1_000_000_000
        expected=(displayed*ratio).__floor__()
        require(presentation==expected and row['presentation_matches_converted_display_time'],'Presentation does not match host conversion at nanosecond resolution')
        ahead=displayed>received;future+=ahead
        require(row['display_time_after_receipt']==ahead,'Future-at-receipt diagnostic changed')
        if ahead:
            lead=(displayed-received)*ratio/1_000_000;leads.append(lead)
            require(abs(row['display_lead_at_receipt_ms']-float(lead))<1e-9,'Serialized lead differs')
        require(not row['eligible_for_latency'] and 'since_acknowledged_marker_ms' not in row,'No-input baseline promoted to latency evidence')
        require(row['target_geometry_matches_at_receipt'] and row['frame_geometry_validated'],'Geometry validation missing')
        require(row['screen_rect_points']==plan['window_global_points'] and row['content_rect_points']==dict(x=0,y=0,width=1053,height=802),'Frame crop/content differs')
        require(row['scale_factor']==1 and row['content_scale']==1 and row['width_pixels']==1053 and row['height_pixels']==802,'Native pixel scale changed')
        require(re.fullmatch(r'frame-[0-9a-f]{64}\.png',row['png']) and row['png_sha256']==row['png'][6:-4],'Unsafe or inconsistent image identity')
        names.add(row['png'])
    require(future==report['frames_with_display_time_after_receipt']==143,'Future count changed')
    require(report['complete_frames']==164 and report['unique_images']==len(names)==115 and report['marker_count']==0,'Accounting changed')
    return dict(frames=164,unique_images=115,future_at_receipt=143,min_future_lead_ms=float(min(leads)),max_future_lead_ms=float(max(leads)),eligible_latency_observations=0)


def main():
    path=folder/'nominal/capture'
    rows=[json.loads(line) for line in gzip.decompress((path/'frames.jsonl.gz').read_bytes()).splitlines()]
    report=read(path/'report.json.gz');config=read(path/'configuration.json.gz')
    result=validate(rows,report,config)
    require(hashlib.sha256(gzip.decompress((path/'configuration.json.gz').read_bytes())).hexdigest()==report['configuration_sha256'],'Configuration hash changed')
    manifest=read(folder/'nominal/png-manifest.json');expected={row['png'] for row in rows}
    require(set(manifest)==expected,'PNG manifest differs')
    raw=lzma.decompress((folder/'nominal/frames.tar.xz').read_bytes())
    with tarfile.open(fileobj=io.BytesIO(raw),mode='r:') as archive:
        members=archive.getmembers();require(len(members)==len(expected) and {m.name for m in members}==expected,'Archive file set differs')
        for member in members:
            require(member.isfile() and member.mode==0o600 and re.fullmatch(r'frame-[0-9a-f]{64}\.png',member.name),'Unsafe archived image')
            data=archive.extractfile(member).read();entry=manifest[member.name]
            require(hashlib.sha256(data).hexdigest()==entry['sha256']==member.name[6:-4] and len(data)==entry['bytes'],'Image bytes differ')
            require(data[:8]==b'\x89PNG\r\n\x1a\n' and data[12:16]==b'IHDR' and struct.unpack('>II',data[16:24])==(1053,802),'PNG dimensions differ')
    require(sum(x['bytes'] for x in manifest.values())==report['image_bytes']==15623335,'Unique image byte accounting differs')
    initial=read(folder/'initial-clock-refusal/capture/report.json.gz')
    first=json.loads(gzip.decompress((folder/'initial-clock-refusal/capture/frames.jsonl.gz').read_bytes()).splitlines()[0])
    lead=Fraction(int(first['display_mach_ticks'])-int(first['received_mach_ticks']))*Fraction(initial['timebase_numerator'],initial['timebase_denominator'])/1_000_000
    require(lead==Fraction(26935,3000) and initial['unique_images']==0 and initial['complete_frames']==0,'Initial future refusal changed')
    for label,mutate in [('shifted-display-clock',lambda row:row.update(display_mach_ticks=str(int(row['display_mach_ticks'])+24_000_000))),('wrong-screen-crop',lambda row:row['screen_rect_points'].update(x=row['screen_rect_points']['x']+1)),('forged-eligibility',lambda row:row.update(eligible_for_latency=True))]:
        changed=deepcopy(rows);mutate(changed[0])
        try:validate(changed,report,config)
        except ValueError:continue
        raise ValueError('Invalid evidence admitted: '+label)
    for name in ['launch-helper-failure','initial-clock-refusal','nominal']:
        session=read(folder/name/'report.json.gz')
        require(session['owned_process_absent'] and session['endpoint_removed'] and not session['forced_cleanup'] and session['shared_geometry_unchanged'] and session['review_unchanged'] and session['copied_bundle_unchanged'] and session['canonical_unchanged'],'Session cleanup or preservation failed')
        close=read(folder/name/'close-response.json.gz')
        require(close['closed'] and close['shutdown']['status']=='completed' and close['shutdown']['workers_joined'] and close['shutdown']['exit_ready'],'Incomplete native cleanup')
    require(read(folder/'initial-clock-refusal/wrong-crop-report.json.gz')['passed'],'Wrong-crop refusal missing')
    require(read(folder/'nominal/nominal-agent-match.json.gz')['passed'],'Independent baseline agent match failed')
    print(json.dumps(dict(passed=True,**result,raw_png_archive_verified=True,rejected_mutations=3,physical_timing_or_visibility_qualified=False)))

if __name__=='__main__':main()

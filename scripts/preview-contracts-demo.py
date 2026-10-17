#!/usr/bin/env python3
"""Owned preview lifecycle, exact PNG chunks and source/default-page boundaries."""
import argparse
import base64
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import struct
import time

from demo_session import connect_demo_client
from recovery_timing import settle_native_view

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
COMMANDS = ['print.preview.start','print.preview.cancel','print.preview.close','print.preview.image']


def protected(state):
    return {key:state.get(key) for key in ['components','wires','document','view','transient','mixed','npn',
            'scope','paused','run_id','electrical_revision','undo_entry','redo_entry']}


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--artifacts',type=Path,required=True)
    args=parser.parse_args();assets=args.artifacts.resolve();assets.mkdir(parents=True,mode=0o700,exist_ok=False)
    client=connect_demo_client(helper.Client,args.connect,command_preconditions=True)
    refusals=[];chunks=[];previews=[]
    try:
        initial=client.ask('project.inspect')
        if initial['recovery'].get('directory'):
            client.ask('recovery.configure',{'directory':initial['recovery']['directory'],'enabled':False,'interval_s':2},True)
        contracts={command:client.ask('capabilities.list',{'command':command})['capabilities'][0]['input_schema'] for command in COMMANDS}
        assert all(s['x-command-contract']['scope']=='captured-print-preview' for s in contracts.values())
        client.ask('example.rc',{},True);client.ask('simulation.pause',{},True)
        client.ask('simulation.restart',{},True);client.ask('simulation.wait')
        client.ask('simulation.step',{'duration_s':.003},True)
        if args.connect:
            settle_native_view(client);client.ask('view.pan',{'dx':0,'dy':0},True)
        source=assets/'source.elab';client.ask('project.save',{'path':str(source)},True)
        source_hash=hashlib.sha256(source.read_bytes()).hexdigest();baseline=client.ask('project.inspect');original_live=protected(baseline)
        sample=baseline['transient']['sample'];assert sample['time_s']==.003
        expected=5*(-math.expm1(-.003/.01));assert abs(sample['voltage_v']-expected)<1e-10

        def inspect():return client.ask('project.inspect')

        def preserve(state):
            assert protected(state)==protected(baseline)
            assert hashlib.sha256(source.read_bytes()).hexdigest()==source_hash

        def wait(identity,expected_status='completed'):
            deadline=time.monotonic()+10
            while True:
                state=inspect();p=state['print_preview'];assert p['id']==identity
                if p['status'] not in ('queued','running','canceling'):
                    assert p['status']==expected_status,p
                    preserve(state);previews.append(p);return p
                assert time.monotonic()<deadline
                time.sleep(.005)

        def start(params,expected_status='completed'):
            state=client.ask('print.preview.start',params,True);preserve(state)
            return wait(state['print_preview']['id'],expected_status)

        def refuse(command,params,code,mutate=False):
            before=inspect();client.ask(command,params,mutate,code);after=inspect()
            assert protected(before)==protected(after) and before.get('print_preview')==after.get('print_preview')
            preserve(after);refusals.append(dict(command=command,params=params,code=code))

        def pixels(preview,name):
            data=bytearray();count=0
            while len(data)<preview['bytes']:
                part=client.ask('print.preview.image',{'id':preview['id'],'offset':len(data),'limit':997})
                raw=base64.b64decode(part['png_base64'],validate=True)
                assert part['id']==preview['id'] and part['offset']==len(data) and part['sha256']==preview['sha256']
                data.extend(raw);count+=1
                assert part['next_offset']==len(data) and part['total']==preview['bytes']
            assert hashlib.sha256(data).hexdigest()==preview['sha256'] and data[:8]==b'\x89PNG\r\n\x1a\n'
            assert struct.unpack('>II',data[16:24])==(preview['width_px'],preview['height_px'])==(1650,1275)
            eof=client.ask('print.preview.image',{'id':preview['id'],'offset':len(data),'limit':0})
            assert eof['png_base64']=='' and eof['next_offset']==eof['total']==len(data) and eof['sha256']==preview['sha256']
            defaults=client.ask('print.preview.image',{'id':preview['id'],'offset':None,'limit':None})
            assert base64.b64decode(defaults['png_base64'],validate=True)==data[:65536]
            (assets/name).write_bytes(data);chunks.append(dict(id=preview['id'],name=name,chunks=count,bytes=len(data),sha256=preview['sha256']))
            preserve(inspect());return bytes(data)

        refuse('print.preview.image',{'id':'unknown','offset':-1},'not-found')
        first=start({'page':None,'page_index':None});assert first['source_current'] and first['visible']
        captured=pixels(first,'captured-default.png')
        for params in [dict(id=first['id'],offset=-1),dict(id=first['id'],offset=first['bytes']+1),
                       dict(id=first['id'],limit=-1),dict(id=first['id'],limit=65537),dict(id=None)]:
            refuse('print.preview.image',params,'invalid-params')
        refuse('print.preview.cancel',{'id':first['id']},'unavailable',True)
        client.ask('export.page.set',{'paper':'a4','orientation':'portrait'},True)
        assert not inspect()['print_preview']['source_current']
        stale_file=assets/'shown-stale.png'
        client.ask('export.schematic.png',{'preview_id':first['id'],'path':str(stale_file)},True)
        assert stale_file.read_bytes()==captured;preserve(inspect())
        explicit={'paper':'letter','orientation':'landscape','margin_mm':10}
        second=start({'page':explicit});assert second['id']!=first['id'] and second['source_current']
        pixels(second,'captured-explicit.png')
        refuse('print.preview.image',{'id':first['id']},'not-found')
        refuse('print.preview.close',{'id':first['id']},'not-found',True)
        client.ask('export.page.set',{'paper':'a4','margin_mm':20},True)
        assert inspect()['print_preview']['source_current'];preserve(inspect())
        client.ask('component.set',{'id':'R1','value':'20k'},True);baseline=inspect()
        assert not baseline['print_preview']['source_current']
        stale=client.ask('print.preview.image',{'id':second['id']})
        assert stale['sha256']==second['sha256']
        failed=start({'page':explicit,'page_index':1},'failed')
        assert failed['error']['code']=='preview-failed' and failed['bytes']==0 and failed['page_count']==1
        refuse('print.preview.image',{'id':second['id']},'not-found')
        refuse('print.preview.image',{'id':failed['id']},'not-ready')
        refuse('print.preview.cancel',{'id':failed['id']},'unavailable',True)
        third=start({'page':explicit});assert third['source_current'] and third['id']!=failed['id']
        edited=pixels(third,'captured-edited.png');assert edited!=captured
        client.ask('print.preview.close',{'id':third['id']},True)
        closed=inspect()['print_preview']
        assert not closed['visible'] and closed['status']=='completed' and closed['bytes']==0 and not closed.get('sha256')
        assert closed['width_px']==third['width_px'] and closed['height_px']==third['height_px']
        refuse('print.preview.image',{'id':third['id'],'offset':-1},'not-ready')
        refuse('print.preview.cancel',{'id':third['id']},'unavailable',True)
        client.ask('print.preview.close',{'id':third['id']},True)
        assert inspect()['print_preview']==closed;preserve(inspect())
        report=dict(passed=True,transport='native-socket' if args.connect else 'headless-stdio',contracts=contracts,refusals=refusals,
                    previews=previews,closed=closed,chunks=chunks,source_sha256=source_hash,original_live=original_live,edited_live=protected(baseline),
                    rc_reference_v=expected,checks=['EOF/default chunks, stale/default versus explicit captures, exact stale PNG export, failed refresh replaces old ID, repeated close and source preservation'],
                    limits=['Active cancellation and non-returned worker capacity use controlled core tests; public cancel checks terminal refusal',
                            'No new renderer/model/equation/file format or physical input/AX/pixel/printer qualification'])
        args.output.parent.mkdir(parents=True,exist_ok=True)
        with args.output.open('x') as output:json.dump(report,output,indent=2);output.write('\n')
        print(json.dumps(dict(passed=True,transport=report['transport'],contracts=len(contracts),refusals=len(refusals),images=len(chunks))))
    finally:
        try:
            if args.connect:client.ask('app.close',{'discard_changes':True},True)
        finally:client.close()


if __name__=='__main__':main()

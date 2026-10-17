#!/usr/bin/env python3
"""Verify recorded viewer contracts over exact RC rows in fresh owned sessions."""
import argparse
import copy
import importlib.util
import json
from pathlib import Path
import time

from demo_session import connect_demo_client
from recovery_timing import digest, settle_native_view

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
COMMANDS = ['recording.view.set','recording.seek','recording.step','recording.speed.set','recording.play',
            'recording.pause','recording.window.set','recording.probes.set','recording.cursor.select',
            'recording.cursor.clear','recording.keep.set']


def live(state):
    return {key:copy.deepcopy(state.get(key)) for key in
            ['components','wires','run_id','electrical_revision','result','transient','mixed','npn','undo_entry','view']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    root = args.artifacts.resolve();root.mkdir(mode=0o700,parents=True,exist_ok=False)
    client = connect_demo_client(helper.Client,args.connect,command_preconditions=True)
    fresh = None
    try:
        initial = client.ask('project.inspect')
        if initial['recovery'].get('directory'):
            client.ask('recovery.configure',{'directory':initial['recovery']['directory'],'enabled':False,'interval_s':2},True)
        before = client.ask('project.inspect')
        contracts = {command:client.ask('capabilities.list',{'command':command})['capabilities'][0]['input_schema'] for command in COMMANDS}
        assert all(schema['x-command-contract']['scope']=='recorded-viewer-and-kept-membership' for schema in contracts.values())
        assert before==client.ask('project.inspect')
        client.ask('example.rc',{},True);client.ask('simulation.pause',{},True);client.ask('simulation.wait')
        accepted = client.ask('simulation.verify',{'duration_s':.06,'probe_ids':['v:C1','i:C1']},True)
        identifier = accepted['last_operation'];deadline = time.monotonic()+10
        while True:
            record = client.ask('operation.inspect',{'id':identifier})
            assert record['operation']['status'] not in ('failed','canceled')
            if record['operation']['status']=='completed':break
            assert time.monotonic()<deadline
            time.sleep(.005)
        rows = client.ask('operation.samples',{'id':identifier,'offset':0,'limit':100})['rows']
        edge = next(i for i in range(len(rows)-1) if rows[i]['time_s']==rows[i+1]['time_s'])
        right = edge+1
        while right+1<len(rows) and rows[right+1]['time_s']==rows[edge]['time_s']:right+=1
        original = root/'original-record.json'
        client.ask('verification.export',{'id':identifier,'path':str(original),'format':'json'},True)
        original_bytes = original.read_bytes()
        if args.connect:
            settle_native_view(client);client.ask('view.pan',{'dx':0,'dy':0},True)
        project = root/'recorded-experiment.elab'
        client.ask('project.save',{'path':str(project)},True)
        baseline_bytes = project.read_bytes();baseline = client.ask('project.inspect')
        refusal_checks = []

        def refuse(command, params, code='invalid-params'):
            before = client.ask('project.inspect')
            client.ask(command,params,True,code)
            assert before==client.ask('project.inspect'),command
            refusal_checks.append({'command':command,'params':params,'code':code})

        for window in [dict(start_s=-1e-13,end_s=0),dict(start_s=.06,end_s=.06+1e-13),
                       dict(start_s=-2e-13,end_s=-1e-13),dict(start_s=.06+1e-13,end_s=.06+2e-13)]:
            refuse('recording.window.set',window)
        refuse('recording.window.set',{'start_s':None,'end_s':.01})
        client.ask('recording.window.set',{'start_s':-1e-13,'end_s':.06+1e-13},True)
        clamped = client.ask('project.inspect')['recording']['scope']['window']
        assert clamped['start_s']==0 and clamped['end_s']==.06
        client.ask('recording.seek',{'index':None,'time_s':rows[edge]['time_s']},True)
        right_edge = client.ask('project.inspect')['recording']
        assert right_edge['sample_index']==right
        client.ask('recording.seek',{'index':edge,'time_s':None},True)
        client.ask('recording.step',{'direction':'next'},True)
        stepped = client.ask('project.inspect')['recording']
        assert stepped['sample_index']==edge+1 and stepped['time_s']==rows[edge]['time_s']
        refuse('recording.seek',{'index':None,'time_s':None})
        client.ask('recording.seek',{'index':0},True)
        refuse('recording.step',{'direction':'previous'},'unavailable')
        client.ask('recording.window.set',{'start_s':'10.1 ms','end_s':'10.2 ms'},True)
        empty = client.ask('project.inspect')['recording']
        assert empty['window_sample_count']==0 and not empty['scope']['window_valid']
        client.ask('recording.window.set',{'start_s':'5 ms','end_s':'15 ms'},True)
        client.ask('recording.seek',{'index':len(rows)-1},True)
        playing = client.ask('recording.play',{'seconds_per_wall_second':.001},True)
        assert playing['recording']['playing'] and not playing['recording']['viewing'] and playing['recording']['sample_index']==0
        assert playing['recording']['scope']['window']['start_s']==.005
        client.ask('recording.speed.set',{'seconds_per_wall_second':.002},True)
        client.ask('recording.view.set',{'mode':'live'},True)
        assert not client.ask('project.inspect')['recording']['playing']
        client.ask('recording.pause',{},True)
        client.ask('recording.view.set',{'mode':'recorded'},True)
        client.ask('recording.probes.set',{'ids':[]},True)
        assert client.ask('project.inspect')['recording']['scope']['traces']==[]
        refuse('recording.probes.set',{'ids':None})
        refuse('recording.probes.set',{'ids':['v:C1','v:C1']})
        client.ask('recording.cursor.select',{'cursor':'b'},True)
        placed = client.ask('recording.cursor.set',{'operation_id':identifier,'cursor':'a','time_s':'10 ms'},True)
        assert placed['recording']['scope']['a']['valid']
        client.ask('recording.cursor.clear',{'cursor':'a'},True)
        client.ask('recording.cursor.clear',{'cursor':'a'},True)
        viewed = client.ask('project.inspect')
        assert viewed['recording']['scope'].get('a') is None and viewed['recording']['scope']['active_cursor']=='b'
        refuse('recording.keep.set',{'id':identifier,'keep':None})
        client.ask('recording.keep.set',{'id':identifier,'keep':False},True)
        excluded = client.ask('project.inspect')
        assert excluded['document']['recordings_dirty'] and excluded['recording']['id']==identifier
        client.ask('recording.keep.set',{'id':identifier,'keep':True},True)
        restored = client.ask('project.inspect')
        assert restored['document']==baseline['document'] and live(restored)==live(baseline)
        assert project.read_bytes()==baseline_bytes
        after_record = root/'record-after-controls.json'
        client.ask('verification.export',{'id':identifier,'path':str(after_record),'format':'json'},True)
        assert after_record.read_bytes()==original_bytes
        assert client.ask('operation.samples',{'id':identifier,'offset':0,'limit':100})['rows']==rows
        client.ask('project.save',{},True)
        saved = json.loads(project.read_bytes())
        assert saved['recordings']['viewing'] and saved['recordings']['entries'][0]['data']['rows']==rows
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load',{'path':str(project)},True)
        reopened = fresh.ask('simulation.wait')
        viewer = reopened['recording']
        assert not reopened['document']['dirty'] and reopened['paused'] and reopened['transient']['sample']['time_s']==0
        assert viewer['viewing'] and not viewer['playing'] and viewer['sample_index']==0
        assert viewer['scope']['window']['start_s']==0 and viewer['scope']['window']['end_s']==.06
        assert viewer['scope']['selected']==['v:C1','i:C1']
        assert viewer['scope'].get('a') is None and viewer['scope'].get('b') is None
        assert fresh.ask('operation.samples',{'id':viewer['id'],'offset':0,'limit':100})['rows']==rows
        report = {'passed':True,'transport':'native-socket' if args.connect else 'headless-stdio','contracts':contracts,
                  'rows':rows,'edge_first_index':edge,'edge_rightmost_index':right,'time_seek_index':right_edge['sample_index'],
                  'row_step_index':stepped['sample_index'],'refusals':refusal_checks,'clamped_window':clamped,
                  'no_sample_window':empty,'record_sha256':digest(original_bytes),'project_sha256':digest(project.read_bytes()),
                  'live_semantic':live(restored),'viewer_after_controls':restored['recording'],'reopened_viewer':viewer,
                  'checks':['all eleven contracts/query purity','normalized boundary collapse refused atomically','rightmost time seek and actual equal-time row step',
                            'empty exact-sample interval','whole-capture Play independent of window/view','live-view stops recorded clock',
                            'hidden trace subset/cursor selection and repeat clear','null required fields refused','kept membership dirty restoration',
                            'exact live/source/history and captured byte preservation','explicit save/fresh reopen retains only supported viewer preferences'],
                  'limits':['Named paused RC capture; playback start/stop semantics rather than real-time frame performance',
                            'Controlled core tests separately cover DC and missing selection; full physical input/accessibility remains open']}
        args.output.write_text(json.dumps(report,indent=2)+'\n')
        print(json.dumps({'passed':True,'transport':report['transport'],'contracts':len(contracts),'rows':len(rows),'refusals':len(refusal_checks)}))
        if args.connect:client.ask('app.close',{'discard_changes':True},True)
    finally:
        if fresh:fresh.close()
        client.close()


if __name__=='__main__':main()

#!/usr/bin/env python3
"""Build, measure and compare through generated MCP setup in a NEW owned session."""
import argparse
import json
import math
from pathlib import Path
import subprocess
import sys
import tempfile
import time

from mcp_client import MCPClient


def exercise(client):
    assert not client.inspect()['components'], 'demo needs a new blank session'
    catalog = client.tool('elab_discover', {})['result']['capabilities']
    assert {'component.place', 'simulation.verify', 'recording.compare', 'analysis.page'} <= {c['id'] for c in catalog}
    for kind, identifier, x, y in [('dc-source','V1',-200,0),('resistor','R1',0,-120),
                                   ('led','D1',200,0),('ground','GND',-200,120)]:
        client.ask('component.place',dict(kind=kind,id=identifier,position=dict(x=x,y=y)),True)
    for a,b in [('V1.positive','R1.1'),('R1.2','D1.anode'),('D1.cathode','GND.reference'),('GND.reference','V1.negative')]:
        client.ask('wire.connect',{'from':a,'to':b},True)
    led = client.ask('simulation.wait')
    assert math.isclose(led['result']['current_a'],.002916796577441,rel_tol=0,abs_tol=5e-12)
    client.ask('component.set',dict(id='R1',value='2k'),True)
    dimmed=client.ask('simulation.wait')['result']['current_a']
    assert .00147 < dimmed < .00149
    client.ask('edit.undo',mutate=True)
    assert client.ask('simulation.wait')['result']['current_a']==led['result']['current_a']
    client.ask('example.load',{'id':'rc','discard_changes':True},True)
    client.ask('simulation.pause',mutate=True)
    recordings=[]
    for resistance in ['10k','20k']:
        client.ask('component.set',{'id':'R1','value':resistance},True)
        state=client.ask('simulation.verify',{'duration_s':.02,'sample_step_s':.001,'probe_ids':['v:C1'],'input_mode':'charge'},True)
        identifier=state['last_operation'];deadline=time.monotonic()+5
        while True:
            operation=client.ask('operation.inspect',{'id':identifier})['operation']
            if operation['status']=='completed':break
            if operation['status']=='failed' or time.monotonic()>deadline:raise RuntimeError('verification did not complete')
            time.sleep(.01)
        recordings.append(identifier)
    listing=client.ask('analysis.list')
    recording=next(r for r in listing['analyses'] if r['kind']=='recording' and r['id']==recordings[0])
    page=client.ask('analysis.page',dict(kind='recording',id=recording['id'],document_generation=listing['document_generation'],limit=5))
    assert len(page['samples'])==5 and not page['done'] and page['next_offset']==5
    before=client.inspect()
    comparison=client.ask('recording.compare',dict(id=recordings[0],reference_id=recordings[1],time_s=.01,show_statistics=True))
    assert comparison['available'] and len(comparison['readings'])==1
    reading=comparison['readings'][0]
    assert math.isclose(reading['a'],5*(1-math.exp(-1)),rel_tol=0,abs_tol=1e-9)
    assert math.isclose(reading['b'],5*(1-math.exp(-.5)),rel_tol=0,abs_tol=1e-9)
    assert reading['b_minus_a']<0 and before['components']==client.inspect()['components']
    return dict(passed=True,commands=len(catalog),tools=[t['name'] for t in client.tools],
                built_led_current_a=led['result']['current_a'],bounded_rows=len(page['samples']),
                compared_voltages_v=[reading['a'],reading['b']],automatic_mutation_retries=0)


def exercise_first_use_files(client, folder):
    """Continue the four-task kit in this owned headless circuit, never owner files."""
    client.ask('example.load',dict(id='flasher',discard_changes=True),True)
    client.ask('simulation.pause',mutate=True)
    client.ask('simulation.step',dict(duration_s=.02),True)
    state=client.ask('simulation.wait')
    graph={k:state[k] for k in ('components','wires')}
    assert {d['id'] for d in state['mixed']['leds']}=={'D1','D2'}
    for section in ['component','selection','measurement','pins','live','recordings','spectrum','sweeps','tolerances','circuit']:
        before=client.inspect()
        after=client.ask('workspace.details.set',dict(section=section,visible=True),True)
        assert after['workspace_details'][section]
        for key in ['components','wires','transient','mixed','document','run_id','electrical_revision']:
            assert before.get(key)==after.get(key),(section,key)
    project=folder/'practice.elab'
    client.ask('project.save',dict(path=str(project)),True)
    client.ask('project.clear',mutate=True)
    client.ask('project.load',dict(path=str(project)),True)
    loaded=client.ask('simulation.wait')
    assert not loaded['document']['dirty'] and all(loaded[k]==v for k,v in graph.items())
    exports = ['svg', 'pdf'] if sys.platform == 'darwin' else ['svg']
    for fmt in exports:
        output=folder/('practice.'+fmt)
        client.ask('export.schematic.'+fmt,dict(path=str(output)),True)
        data=output.read_bytes()
        assert data.startswith(b'%PDF-') if fmt=='pdf' else all(p['id'].encode() in data for p in graph['components'])
    if 'pdf' not in exports:
        pdf = folder/'practice.pdf'
        client.ask('export.schematic.pdf', dict(path=str(pdf)), True, expect='unsupported')
        assert not pdf.exists()
    return dict(passed=True,shared_disclosures=10,saved_graph_exact=True,exports=exports,scope='MCP engineering walkthrough; no participant or native OS-input claim')


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent',type=Path,required=True)
    parser.add_argument('--socket',action='store_true',help='also launch an owned private headless host')
    parser.add_argument('--artifacts',type=Path,required=True,help='new output directory')
    args=parser.parse_args();agent=args.agent.resolve(strict=True)
    args.artifacts.mkdir(parents=True,exist_ok=False)
    host=None;log=None;client=None
    try:
        endpoint=None
        if args.socket:
            endpoint=(args.artifacts/'host.endpoint.json').resolve()
            log=(args.artifacts/'host.txt').open('xb')
            host=subprocess.Popen([str(agent),'--listen',str(endpoint)],stdout=log,stderr=subprocess.STDOUT)
            deadline=time.monotonic()+8
            while not endpoint.exists():
                if host.poll() is not None or time.monotonic()>deadline:raise RuntimeError('owned host startup failed')
                time.sleep(.02)
        # A different cwd proves generated executable/descriptor paths are usable.
        with tempfile.TemporaryDirectory(prefix='elab-mcp-client-') as cwd:
            client=MCPClient(agent,endpoint,cwd=cwd)
            report=exercise(client)
            report['first_use_files']=exercise_first_use_files(client,args.artifacts.resolve())
            client.ask('app.close',{'discard_changes':True},True)
            client.close();client=None
            report['bridge_exit_code']=0
        if host is not None:
            assert host.wait(timeout=8)==0
            report['host_exit_code']=host.returncode
        report['transport']='MCP to owned private socket' if args.socket else 'MCP headless stdio'
        (args.artifacts/'report.json').write_text(json.dumps(report,indent=2)+'\n')
        print(json.dumps(report))
    finally:
        if client is not None and client.process.poll() is None:client.close()
        if host is not None and host.poll() is None:
            # Only this newly owned host, never a supplied live endpoint.
            host.terminate();host.wait(timeout=8)
        if log is not None:log.close()


if __name__=='__main__':main()

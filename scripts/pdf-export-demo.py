#!/usr/bin/env python3
"""Produce PDF/PNG reference schematics through the shared export interface."""
import argparse
import importlib.util
import json
from pathlib import Path


spec=importlib.util.spec_from_file_location('helper',Path(__file__).with_name('persistence-demo.py'))
helper=importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser=argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--format',choices=['pdf','png'],default='pdf')
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--artifacts',type=Path,required=True)
    args=parser.parse_args()
    directory=args.artifacts.resolve();directory.mkdir(parents=True,exist_ok=False)
    client=connect_demo_client(helper.Client, args.connect,command_preconditions=True)
    try:
        exported=[]
        for info in client.ask('examples.list'):
            client.ask('example.load',{'id':info['id'],'discard_changes':True},True)
            client.ask('simulation.pause',{},True)
            client.ask('simulation.wait')
            client.ask('export.page.set',{'paper':'a4' if info['id']=='rc' else 'letter','orientation':'portrait' if info['id']=='rc' else 'landscape','margin_mm':10},True)
            before=client.ask('project.inspect')
            path=directory/(info['id']+'.'+args.format)
            client.ask('export.schematic.'+args.format,{'path':str(path)},True)
            after=client.ask('project.inspect')
            for key in ['document','components','wires','run_id','electrical_revision','undo_entry','transient','mixed','npn','result']:
                assert before.get(key)==after.get(key),key
            signature=b'%PDF-' if args.format=='pdf' else b'\x89PNG\r\n\x1a\n'
            assert path.read_bytes().startswith(signature) and after['last_export']['format']==args.format
            exported.append({'id':info['id'],'dpi':after['last_export'].get('dpi'),'page':after['last_export']['page'],'bytes':path.stat().st_size,'sha256':after['last_export']['sha256']})
        client.ask('project.clear',{'discard_changes':True},True)
        report={'passed':True,'format':args.format,'transport':'native-socket' if args.connect else 'headless-stdio','exports':exported,
                'checks':[str(len(exported))+' reference schematics','shared session page settings','paper/orientation capture','source/history/physics preservation'],
                'limits':['Document/image generation; no printer job or physical printing qualification','rendered inspection is recorded separately']}
        args.output.parent.mkdir(parents=True,exist_ok=True);args.output.write_text(json.dumps(report,indent=2)+'\n')
        print(json.dumps(report))
    finally:client.close()


if __name__=='__main__':main()

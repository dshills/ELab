import {test,expect} from '@playwright/test';
import {readFileSync,statSync,existsSync} from 'node:fs';

test('enabled workspace shares agent edits, guarded undo and revocation',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/');
 await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 await page.locator('[data-object="R1"]').press('Enter');
 const resistance=page.getByRole('textbox',{name:'Resistance',exact:true});
 await expect(resistance).toHaveValue(/10/);
 await page.getByRole('button',{name:'Connect agent',exact:true}).click();
 const dialog=page.getByRole('dialog',{name:'Agent connection',exact:true});
 await expect(dialog.getByRole('status')).toHaveText('Agent access is disabled.');
 await dialog.getByRole('button',{name:'Enable agent access',exact:true}).click();
 const field=dialog.getByRole('textbox',{name:'MCP client configuration'});
 await expect(field).toBeVisible();
 const config=JSON.parse(await field.inputValue()).mcpServers.elab;
 expect(config.args[0]).toBe('--mcp-workspace');
 const path=config.args[1],endpoint=JSON.parse(readFileSync(path,'utf8'));
 expect(statSync(path).mode&0o077).toBe(0);
 expect(await field.inputValue()).not.toContain(endpoint.token);
 await dialog.getByRole('button',{name:'Close agent connection',exact:true}).click();
 const send=async(key,value)=>{
  const state=(await (await page.request.get('/api/state')).json()).state;
  const response=await page.request.post('/api/agent/command',{headers:{
   'X-ELab-Agent-Token':endpoint.token,'X-ELab-Instance':endpoint.instance,'X-ELab-Workspace':endpoint.workspace},
   data:{version:1,id:key,key,command:'component.set',params:{id:'R1',value},
    expected_command_revision:state.command_revision,expected_document_generation:state.document.generation}});
  return response.json();
 };
 expect((await send('browser-agent-edit','22k')).error).toBeUndefined();
 await expect(resistance).toHaveValue(/22/);
 await page.getByRole('button',{name:'Undo',exact:true}).click();
 const history=page.getByRole('dialog',{name:'Undo shared edit?',exact:true});
 await expect(history).toBeVisible();
 await history.getByRole('button',{name:'Keep editing',exact:true}).click();
 await expect(resistance).toHaveValue(/22/);
 await page.getByRole('button',{name:'Undo',exact:true}).click();
 await history.getByRole('button',{name:'Undo shared edit',exact:true}).click();
 await expect(resistance).toHaveValue(/10/);
 await page.getByRole('button',{name:'Agent access on',exact:true}).click();
 await dialog.getByRole('button',{name:'Disconnect agent',exact:true}).click();
 await expect(dialog.getByRole('status')).toHaveText('Agent access is disabled.');
 expect(existsSync(path)).toBe(false);
 expect((await send('revoked-edit','33k')).error.code).toBe('agent-disconnected');
 await dialog.getByRole('button',{name:'Close agent connection',exact:true}).click();
 await expect(resistance).toHaveValue(/10/);
 expect(errors).toEqual([]);
});

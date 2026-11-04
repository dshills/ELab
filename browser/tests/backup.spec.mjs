import {test,expect} from '@playwright/test';
import {readFileSync} from 'node:fs';

test('library backup previews, cancels and restores copies without replacing the open circuit',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/');await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 await page.getByRole('button',{name:'Save project',exact:true}).click();
 const save=page.getByRole('dialog',{name:'Save project',exact:true});
 await save.getByRole('textbox',{name:'Project name'}).fill('Backup browser fixture');
 await save.getByRole('button',{name:'Save project',exact:true}).click();await expect(save).not.toBeVisible();
 const capacitance=page.getByRole('textbox',{name:'Capacitance',exact:true});
 await capacitance.fill('2.2 uF');await page.getByRole('button',{name:'Apply',exact:true}).click();await expect(capacitance).toHaveValue(/2\.2/);
 await page.getByRole('button',{name:'Projects',exact:true}).click();
 const projects=page.getByRole('dialog',{name:'Projects & recovery'});
 const download=page.waitForEvent('download');await projects.getByRole('button',{name:'Back up library',exact:true}).click();
 const file=await download;expect(file.suggestedFilename()).toMatch(/^ELab-library-.*\.zip$/);
 const bytes=readFileSync(await file.path());expect(bytes.subarray(0,2).toString()).toBe('PK');
 await expect(projects.getByRole('status')).toContainText('Library backup downloaded');
 const cards=projects.locator('.project-card');const before=await cards.count();
 const input=projects.locator('input[type=file]');
 await input.setInputFiles({name:'Library.zip',mimeType:'application/zip',buffer:bytes});
 const confirm=page.getByRole('dialog',{name:'Restore library backup?',exact:true});
 await expect(confirm).toBeVisible();await expect(confirm.locator('#library-restore-summary')).toContainText('saved projects');
 await confirm.getByRole('button',{name:'Cancel restore',exact:true}).click();await expect(cards).toHaveCount(before);
 await input.setInputFiles({name:'Library.zip',mimeType:'application/zip',buffer:bytes});
 await expect(confirm).toBeVisible();await confirm.getByRole('button',{name:'Restore copies',exact:true}).click();
 await expect(projects.getByRole('status')).toContainText('Restored copies added');
 await expect(cards).toHaveCount(before*2);
 await projects.getByRole('tab',{name:'Recovery history',exact:true}).click();
 await expect(projects.locator('.project-card').filter({hasText:'restored workspace'}).first()).toBeVisible();
 await projects.getByRole('button',{name:'Close projects',exact:true}).click();await expect(capacitance).toHaveValue(/2\.2/);
 // A malformed file shows a useful error and never reaches confirmation.
 await page.getByRole('button',{name:'Projects',exact:true}).click();
 await input.setInputFiles({name:'Bad.zip',mimeType:'application/zip',buffer:Buffer.from('not a backup')});
 await expect(projects.getByRole('status')).toContainText('Restore failed');await expect(confirm).not.toBeVisible();
 // Native loading marks a document clean, but a recovery copy still needs Save.
 // A projected selection reply must retain that browser recovery status.
 await projects.getByRole('tab',{name:'Recovery history',exact:true}).click();
 await projects.locator('.project-card').filter({hasText:'Backup browser fixture (restored workspace)'}).first().getByRole('button',{name:'Recover a copy',exact:true}).click();
 const replace=page.getByRole('dialog',{name:'Replace this circuit?',exact:true});if(await replace.isVisible())await replace.getByRole('button',{name:'Replace circuit',exact:true}).click();
 await expect(projects).not.toBeVisible();await page.locator('[data-object="R1"]').press('Enter');await expect(page.locator('.selected-heading h2')).toHaveText('R1');
 await expect(page.locator('#document-status')).toContainText('changes to save');
 expect(errors).toEqual([]);
});

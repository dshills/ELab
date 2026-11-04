export function libraryBackup(api){
 const backup=document.getElementById('library-backup'),restore=document.getElementById('library-restore');
 const input=document.getElementById('library-backup-file'),status=document.getElementById('library-backup-status');
 const dialog=document.getElementById('library-restore-dialog');
 const query=()=>new URLSearchParams({revision:api.state().command_revision,generation:api.state().document.generation});
 function busy(on){backup.disabled=restore.disabled=on;}
 backup.onclick=async()=>{
  backup.focus();busy(true);status.textContent='Creating library backup…';
  try{await api.mutate(async()=>{
   const response=await fetch('/api/library/backup?'+query(),{method:'POST',headers:{'Content-Type':'application/json'}});
   if(!response.ok){const error=await response.json();throw new Error(error.error||'Library backup failed.');}
   const blob=await response.blob(),url=URL.createObjectURL(blob),link=document.createElement('a');
   link.href=url;link.download=/filename="([^"]+)"/.exec(response.headers.get('Content-Disposition')||'')?.[1]||'ELab-library.zip';
   link.click();setTimeout(()=>URL.revokeObjectURL(url),30_000);
   status.textContent='Library backup downloaded. Keep a copy outside this Mac.';
  });}catch(e){status.textContent='Backup failed: '+e.message;api.toast(e.message,true);}finally{busy(false);}
 };
 restore.onclick=()=>{restore.focus();input.click();};
 input.onchange=async()=>{
  const file=input.files?.[0];input.value='';if(!file)return;
  busy(true);status.textContent='Checking backup…';
  try{
   if(file.size>64*1024*1024)throw new Error('Library backups must be 64 MiB or smaller.');
   const contents=(await api.fetch('/api/library/restore?'+query()+'&preview=true',{method:'POST',headers:{'Content-Type':'application/zip'},body:file})).contents;
   document.getElementById('library-restore-summary').textContent=`${contents.projects} saved projects (${contents.trash} in Trash), ${contents.history} history copies and ${contents.workspaces} workspaces.`;
   dialog.returnValue='cancel';dialog.showModal();
   const choice=await new Promise(resolve=>dialog.addEventListener('close',()=>resolve(dialog.returnValue),{once:true}));
   if(choice!=='restore'){status.textContent='Restore cancelled. Your library is unchanged.';return;}
   await api.mutate(async()=>{
    status.textContent='Restoring copies…';
    const result=await api.fetch('/api/library/restore?'+query(),{method:'POST',headers:{'Content-Type':'application/zip'},body:file});
    status.textContent=result.message;await api.reload();
   });
  }catch(e){status.textContent='Restore failed: '+e.message;api.toast(e.message,true);}finally{busy(false);}
 };
}

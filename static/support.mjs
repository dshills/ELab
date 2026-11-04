export function browserEnvironment(userAgent='',width=0,height=0,ratio=1){
 const candidates=[['Edge',/Edg\/([\d.]{1,30})/],['Firefox',/Firefox\/([\d.]{1,30})/],['Chrome',/Chrome\/([\d.]{1,30})/],['Safari',/Version\/([\d.]{1,30}).*Safari\//]];
 const match=candidates.map(([family,re])=>({family,match:re.exec(userAgent)})).find(v=>v.match);
 const platform=/Android/.test(userAgent)?'Android':/iPhone|iPad/.test(userAgent)?'iOS':/Macintosh|Mac OS X/.test(userAgent)?'macOS':/Windows/.test(userAgent)?'Windows':/Linux/.test(userAgent)?'Linux':'Other';
 const bounded=(n,max)=>Number.isFinite(n)?Math.max(0,Math.min(max,n)):0;
 return {family:match?.family||'Other',version:match?.match[1]||'unknown',platform,
  viewport:{width:bounded(width,100000),height:bounded(height,100000)},device_pixel_ratio:bounded(ratio,10)};
}

export function supportPanel(api){
 const $=id=>document.getElementById(id),dialog=$('support-dialog'),button=$('about-button');let report='',pending=false,session=0,infoRequest=0;
 function label(text){button.textContent=text;button.setAttribute('aria-label',`${text} · About ELab and diagnostic report`);}
 function describe(info){label(info.version==='development'?'Development':`v${info.version}`);
  $('support-version').textContent=`ELab ${info.version}${info.build?` · build ${info.build}`:''}`;
  $('support-source').textContent=`Source: ${info.source_commit}${info.source_modified?' · modified source':''}`;
  $('support-runtime').textContent=`${info.os} / ${info.architecture} · ${info.go_version}`;
 }
 async function load(){const request=++infoRequest;try{const info=await api.fetch('/api/about');if(request===infoRequest)describe(info);}catch{if(request!==infoRequest)return;$('support-source').textContent='';$('support-runtime').textContent='';label('Version unavailable');$('support-version').textContent='Build information is unavailable. Reconnect to ELab and reopen this window.';}}
 function controls(){ $('support-create').disabled=pending;$('support-download').disabled=pending||!report;$('support-copy').disabled=pending||!report;}
 function clear(){report='';pending=false;$('support-preview').value='';$('support-preview-container').hidden=true;$('support-status').textContent='';$('support-create').textContent='Create report preview';controls();}
 async function create(){if(pending)return;const requested=session;pending=true;controls();$('support-status').textContent='Creating report…';try{
  const snapshot=await api.fetch('/api/diagnostic-report');if(requested!==session||!dialog.open)return;
  report=JSON.stringify({...snapshot,browser:browserEnvironment(navigator.userAgent,innerWidth,innerHeight,devicePixelRatio)},null,2)+'\n';
  $('support-preview').value=report;$('support-preview-container').hidden=false;$('support-status').textContent='Report ready to inspect. Nothing has been sent.';$('support-create').textContent='Refresh report';
 }catch{if(requested!==session)return;report='';$('support-preview').value='';$('support-preview-container').hidden=true;$('support-status').textContent='Could not create a report. Reconnect to ELab and try again.';}finally{if(requested===session){pending=false;controls();}}}
 button.onclick=async()=>{session++;clear();if(!dialog.open)dialog.showModal();await load();};
 dialog.addEventListener('close',()=>{session++;clear();});
 $('support-create').onclick=create;
 $('support-download').onclick=()=>{if(!report||pending)return;const url=URL.createObjectURL(new Blob([report],{type:'application/json'}));const link=document.createElement('a');link.href=url;link.download='ELab-diagnostic-report.json';link.click();setTimeout(()=>URL.revokeObjectURL(url),1000);$('support-status').textContent='Report downloaded. Share the file only when you choose.';};
 $('support-copy').onclick=async()=>{if(!report||pending)return;try{await navigator.clipboard.writeText(report);$('support-status').textContent='Report copied. Share it only when you choose.';}catch{$('support-status').textContent='Copy is unavailable. Select the preview text to copy it, or download the file.';}};
 controls();load();return {show:()=>button.click()};
}

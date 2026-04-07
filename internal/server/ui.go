package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Probe</title>
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#c45d2c;--rl:#e8753a;--leather:#a0845c;--ll:#c4a87a;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c44040;--blue:#4a7ec4;--mono:'JetBrains Mono',Consolas,monospace;--serif:'Libre Baskerville',Georgia,serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);font-size:13px;line-height:1.6}
a{color:var(--rl);text-decoration:none}a:hover{color:var(--gold)}
.hdr{padding:.6rem 1.2rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}
.hdr h1{font-family:var(--serif);font-size:1rem}.hdr h1 span{color:var(--rl)}
.main{max-width:1000px;margin:0 auto;padding:1rem 1.2rem}
.btn{font-family:var(--mono);font-size:.68rem;padding:.3rem .6rem;border:1px solid;cursor:pointer;background:transparent;transition:.15s;white-space:nowrap}
.btn-p{border-color:var(--rust);color:var(--rl)}.btn-p:hover{background:var(--rust);color:var(--cream)}
.btn-d{border-color:var(--bg3);color:var(--cm)}.btn-d:hover{border-color:var(--red);color:var(--red)}
.bin-card{background:var(--bg2);border:1px solid var(--bg3);padding:.7rem;margin-bottom:.4rem;cursor:pointer;transition:.1s}
.bin-card:hover{background:var(--bg3)}
.bin-card h3{font-size:.8rem;margin-bottom:.2rem}.bin-meta{font-size:.65rem;color:var(--cm);display:flex;gap:.7rem}
.method{font-size:.6rem;padding:.1rem .3rem;border-radius:2px;font-weight:600}
.m-GET{background:rgba(74,158,92,.15);color:var(--green)}.m-POST{background:rgba(74,126,196,.15);color:var(--blue)}.m-PUT{background:rgba(212,168,67,.15);color:var(--gold)}.m-DELETE{background:rgba(196,64,64,.15);color:var(--red)}.m-PATCH{background:rgba(160,132,92,.15);color:var(--leather)}
.req-row{display:flex;align-items:center;gap:.5rem;padding:.3rem .5rem;border-bottom:1px solid var(--bg3);font-size:.72rem;cursor:pointer;transition:.1s}
.req-row:hover{background:var(--bg2)}
.req-time{color:var(--cm);font-size:.6rem;width:65px;flex-shrink:0}
.req-path{color:var(--cd);flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.req-size{color:var(--cm);font-size:.6rem}
.modal-bg{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.65);display:flex;align-items:center;justify-content:center;z-index:100}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:95%;max-width:700px;max-height:90vh;overflow-y:auto}
.modal h2{font-family:var(--serif);font-size:.95rem;margin-bottom:1rem}
label.fl{display:block;font-size:.65rem;color:var(--leather);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem;margin-top:.5rem}
input[type=text],input[type=number],textarea,select{background:var(--bg);border:1px solid var(--bg3);color:var(--cream);padding:.35rem .5rem;font-family:var(--mono);font-size:.78rem;width:100%;outline:none}
textarea{resize:vertical;min-height:60px}
.form-row{display:flex;gap:.5rem}.form-row>*{flex:1}
.empty{text-align:center;padding:2rem;color:var(--cm);font-style:italic;font-family:var(--serif)}
pre.detail{background:var(--bg);border:1px solid var(--bg3);padding:.5rem;font-size:.7rem;overflow-x:auto;white-space:pre-wrap;color:var(--cd);margin:.3rem 0;max-height:200px;overflow-y:auto}
</style>
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital@0;1&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
</head><body>
<div class="hdr"><h1><span>Probe</span></h1><div style="display:flex;gap:.5rem;align-items:center"><span style="font-size:.7rem;color:var(--leather)">Bins: <b id="sBins">-</b></span><button class="btn btn-p" onclick="showNewBin()">+ Bin</button></div></div>
<div class="main"><div id="upgrade-banner" style="display:none;background:#241e18;border:1px solid #8b3d1a;border-left:3px solid #c45d2c;padding:.6rem 1rem;font-size:.78rem;color:#bfb5a3;margin-bottom:.8rem"><strong style="color:#f0e6d3">Free tier</strong> — 10 items max. <a href="https://stockyard.dev/probe/" target="_blank" style="color:#e8753a">Upgrade to Pro →</a></div>
<div id="binList"></div>
<div id="reqPane" style="display:none;margin-top:1rem">
<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:.5rem">
<div style="font-size:.7rem;color:var(--leather)">Requests for: <b id="curBinName" style="color:var(--cream)">-</b></div>
<div style="display:flex;gap:.3rem">
<span style="font-size:.65rem;color:var(--cm)">Capture URL: <code id="captureUrl" style="color:var(--rl)">-</code></span>
<button class="btn btn-d" style="font-size:.6rem" onclick="clearReqs()">Clear</button>
</div>
</div>
<div id="reqList"></div>
</div>
</div>
<div id="modal"></div>
<script>
let bins=[],curBin=null;
async function api(url,opts){return(await fetch(url,opts)).json()}
function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function timeAgo(d){if(!d)return'';const s=Math.floor((Date.now()-new Date(d))/1e3);if(s<60)return s+'s';if(s<3600)return Math.floor(s/60)+'m';if(s<86400)return Math.floor(s/3600)+'h';return Math.floor(s/86400)+'d'}

async function init(){
  const[bd,sd]=await Promise.all([api('/api/bins'),api('/api/stats')]);
  bins=bd.bins||[];document.getElementById('sBins').textContent=sd.bins;
  document.getElementById('binList').innerHTML=bins.length?bins.map(b=>
    '<div class="bin-card" onclick="selectBin(\''+b.id+'\')"><h3>'+esc(b.name)+'</h3>'+
    '<div class="bin-meta"><span>/b/'+esc(b.slug)+'</span><span>'+b.request_count+' requests</span><span>Response: '+b.response_code+'</span>'+
    '<span onclick="event.stopPropagation();editBin(\''+b.id+'\')" style="color:var(--rl);cursor:pointer">edit</span>'+
    '<span onclick="event.stopPropagation();if(confirm(\'Delete?\'))delBin(\''+b.id+'\')" style="color:var(--red);cursor:pointer">del</span></div></div>'
  ).join(''):'<div class="empty">No bins yet. Create one to start capturing requests.</div>';
  if(curBin)loadReqs(curBin);
}

async function selectBin(id){curBin=id;const b=bins.find(x=>x.id===id);
  document.getElementById('reqPane').style.display='block';
  document.getElementById('curBinName').textContent=b?b.name:'';
  document.getElementById('captureUrl').textContent=location.origin+'/b/'+(b?b.slug:'');
  loadReqs(id);
}

async function loadReqs(binID){
  const d=await api('/api/bins/'+binID+'/requests?limit=100');
  const reqs=d.requests||[];
  document.getElementById('reqList').innerHTML=reqs.length?reqs.map(r=>
    '<div class="req-row" onclick="showReq(\''+r.id+'\')"><span class="method m-'+r.method+'">'+r.method+'</span>'+
    '<span class="req-path">'+esc(r.path)+(r.query?'?'+esc(r.query):'')+'</span>'+
    '<span class="req-size">'+r.size+'B</span><span class="req-time">'+timeAgo(r.created_at)+' ago</span></div>'
  ).join(''):'<div class="empty" style="padding:1rem">No requests captured yet.</div>';
}

async function showReq(id){
  const r=await api('/api/requests/'+id);
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal">'+
    '<h2><span class="method m-'+r.method+'">'+r.method+'</span> '+esc(r.path)+'</h2>'+
    '<div style="font-size:.65rem;color:var(--cm)">From: '+esc(r.ip)+' · '+r.size+' bytes · '+r.created_at+'</div>'+
    (r.query?'<label class="fl">Query</label><pre class="detail">'+esc(r.query)+'</pre>':'')+
    '<label class="fl">Headers</label><pre class="detail">'+esc(JSON.stringify(r.headers,null,2))+'</pre>'+
    (r.body?'<label class="fl">Body</label><pre class="detail">'+esc(r.body)+'</pre>':'')+
    '<button class="btn btn-d" style="margin-top:.5rem" onclick="closeModal()">Close</button></div></div>';
}

function showNewBin(){
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal">'+
    '<h2>New Bin</h2><label class="fl">Name</label><input type="text" id="nb-name"><label class="fl">Slug (URL path)</label><input type="text" id="nb-slug" placeholder="my-webhook">'+
    '<div class="form-row"><div><label class="fl">Response Code</label><input type="number" id="nb-code" value="200"></div><div><label class="fl">Content-Type</label><input type="text" id="nb-type" value="application/json"></div></div>'+
    '<label class="fl">Response Body</label><textarea id="nb-body" rows="3" placeholder=\'{"ok":true}\'></textarea>'+
    '<div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveNewBin()">Create</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>';
}
async function saveNewBin(){
  const body={name:document.getElementById('nb-name').value,slug:document.getElementById('nb-slug').value,response_code:parseInt(document.getElementById('nb-code').value)||200,response_type:document.getElementById('nb-type').value,response_body:document.getElementById('nb-body').value};
  if(!body.name||!body.slug){alert('Name and slug required');return}
  await api('/api/bins',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});closeModal();init()
}
function editBin(id){const b=bins.find(x=>x.id===id);if(!b)return;
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal">'+
    '<h2>Edit Bin</h2><label class="fl">Name</label><input type="text" id="eb-name" value="'+esc(b.name)+'">'+
    '<div class="form-row"><div><label class="fl">Response Code</label><input type="number" id="eb-code" value="'+b.response_code+'"></div><div><label class="fl">Content-Type</label><input type="text" id="eb-type" value="'+esc(b.response_type)+'"></div></div>'+
    '<label class="fl">Response Body</label><textarea id="eb-body" rows="3">'+esc(b.response_body)+'</textarea>'+
    '<div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveEditBin(\''+id+'\')">Save</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>';
}
async function saveEditBin(id){
  const body={name:document.getElementById('eb-name').value,response_code:parseInt(document.getElementById('eb-code').value)||200,response_type:document.getElementById('eb-type').value,response_body:document.getElementById('eb-body').value};
  await api('/api/bins/'+id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});closeModal();init()
}
async function delBin(id){await api('/api/bins/'+id,{method:'DELETE'});if(curBin===id){curBin=null;document.getElementById('reqPane').style.display='none'}init()}
async function clearReqs(){if(!curBin)return;await api('/api/bins/'+curBin+'/clear',{method:'POST'});loadReqs(curBin);init()}
function closeModal(){document.getElementById('modal').innerHTML=''}
init();setInterval(()=>{if(curBin)loadReqs(curBin)},5000)
fetch('/api/tier').then(r=>r.json()).then(j=>{if(j.tier==='free'){var b=document.getElementById('upgrade-banner');if(b)b.style.display='block'}}).catch(()=>{var b=document.getElementById('upgrade-banner');if(b)b.style.display='block'});
</script><script>
(function(){
  fetch('/api/config').then(function(r){return r.json()}).then(function(cfg){
    if(!cfg||typeof cfg!=='object')return;
    if(cfg.dashboard_title){
      document.title=cfg.dashboard_title;
      var h1=document.querySelector('h1');
      if(h1){
        var inner=h1.innerHTML;
        var firstSpan=inner.match(/<span[^>]*>[^<]*<\/span>/);
        if(firstSpan){h1.innerHTML=firstSpan[0]+' '+cfg.dashboard_title}
        else{h1.textContent=cfg.dashboard_title}
      }
    }
  }).catch(function(){});
})();
</script>
</body></html>`

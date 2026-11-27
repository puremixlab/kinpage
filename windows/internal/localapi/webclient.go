package localapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const builtinWebClientHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Kinpage Web Client</title>
<style>
  body{font-family:system-ui,-apple-system,Segoe UI,sans-serif;margin:0;background:#f7f8fa;color:#111827;display:flex;height:100vh}
  aside{width:250px;background:#f1f2f5;border-right:1px solid rgba(15,23,42,.08);padding:14px;overflow:auto}
  main{flex:1;padding:24px;overflow:auto;max-width:860px}
  h1{font-size:18px;margin:0 0 12px}
  .item{padding:8px 10px;border-radius:7px;cursor:pointer;margin-bottom:4px}
  .item:hover,.item.active{background:#fff;box-shadow:0 1px 2px rgba(15,23,42,.07)}
  .card{background:#fff;border:1px solid rgba(15,23,42,.08);border-radius:10px;padding:16px;margin-bottom:12px}
  input,textarea,button{font:inherit;border-radius:6px;border:1px solid #d1d5db;padding:7px 10px}
  button{background:#007aff;color:#fff;border:none;cursor:pointer}
  button.sec{background:#e5e7eb;color:#111827}
  .row{display:flex;gap:8px;align-items:center;margin-bottom:10px}
</style>
</head>
<body>
<aside>
  <h1>邻页 · Kinpage Web</h1>
  <div id="sites"></div>
</aside>
<main>
  <div class="row">
    <h1 id="siteTitle" style="flex:1">选择一个站点</h1>
    <button class="sec" onclick="publishSite()">发布到 IPNS</button>
  </div>
  <div class="card">
    <div class="row"><input id="newTitle" placeholder="新文章标题" style="flex:1"><button onclick="createPost()">发文</button></div>
    <textarea id="newBody" rows="3" placeholder="输入 Markdown 正文…" style="width:100%;box-sizing:border-box"></textarea>
  </div>
  <div id="articles"></div>
</main>
<script>
let currentSite = null;
async function api(p, opts){ const r = await fetch(p, opts); if(!r.ok) throw new Error(await r.text()); return r.json(); }
async function loadSites(){
  const sites = await api('/v0/planets/my');
  const el = document.getElementById('sites');
  el.innerHTML = '';
  sites.forEach(s => {
    const d = document.createElement('div');
    d.className = 'item' + (currentSite && currentSite.id === s.id ? ' active' : '');
    d.textContent = s.name || s.id;
    d.onclick = () => selectSite(s);
    el.appendChild(d);
  });
  if(!currentSite && sites.length) selectSite(sites[0]);
}
async function selectSite(s){
  currentSite = s;
  document.getElementById('siteTitle').textContent = s.name || s.id;
  loadSites();
  const arts = await api('/v0/planets/my/' + encodeURIComponent(s.id) + '/articles');
  const el = document.getElementById('articles');
  el.innerHTML = arts.map(a => '<div class="card"><strong>' + (a.title||'无标题') + '</strong><p style="white-space:pre-wrap;color:#4b5563">' + (a.content||'') + '</p></div>').join('');
}
async function createPost(){
  if(!currentSite) return;
  const title = document.getElementById('newTitle').value.trim();
  const content = document.getElementById('newBody').value;
  if(!title && !content) return;
  await api('/v0/planets/my/' + encodeURIComponent(currentSite.id) + '/articles', {
    method:'POST', headers:{'Content-Type':'application/json'},
    body: JSON.stringify({title, content})
  });
  document.getElementById('newTitle').value = '';
  document.getElementById('newBody').value = '';
  selectSite(currentSite);
}
async function publishSite(){
  if(!currentSite) return;
  await api('/v0/planets/my/' + encodeURIComponent(currentSite.id) + '/publish', {method:'POST'});
  alert('已触发发布');
}
loadSites();
</script>
</body>
</html>`

func (srv *Server) handleID(w http.ResponseWriter) {
	host, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       "Kinpage",
		"platform":   "windows",
		"apiVersion": "v0",
		"webClient":  "/app/",
		"hostname":   host,
	})
}

func (srv *Server) handleWebClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method Not Allowed"})
		return
	}
	srv.mu.Lock()
	root := srv.root
	srv.mu.Unlock()
	if root != "" {
		customDir := filepath.Join(root, "Public", "app")
		if info, err := os.Stat(filepath.Join(customDir, "index.html")); err == nil && !info.IsDir() {
			http.StripPrefix("/app", http.FileServer(http.Dir(customDir))).ServeHTTP(w, r)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(builtinWebClientHTML))
}

func (srv *Server) handleSitePublic(w http.ResponseWriter, r *http.Request, siteID, rel string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method Not Allowed"})
		return
	}
	srv.mu.Lock()
	root := srv.root
	srv.mu.Unlock()
	if root == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Public directory unavailable"})
		return
	}
	baseDir := filepath.Join(root, "sites", siteID)
	cleanRel := strings.TrimLeft(path.Clean("/"+rel), "/")
	if cleanRel == "" {
		cleanRel = "index.html"
	}
	target := filepath.Join(baseDir, filepath.FromSlash(cleanRel))
	relCheck, err := filepath.Rel(baseDir, target)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid path"})
		return
	}
	info, err := os.Stat(target)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not Found"})
		return
	}
	if info.IsDir() {
		target = filepath.Join(target, "index.html")
	}
	http.ServeFile(w, r, target)
}

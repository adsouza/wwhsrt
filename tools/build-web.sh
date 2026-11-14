#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
GOMEMLIMIT=768MiB GOMAXPROCS=2 GOOS=js GOARCH=wasm go build -p 1 -trimpath -o web/main.wasm ./cmd/wwhsrt
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js
cp assets/fonts/rail-ui.ttf web/rail-ui.ttf
cp assets/branding/favicon.svg web/favicon.svg
cat assets/terrain/ATTRIBUTION.md assets/scenario/NETWORK_ATTRIBUTION.md > web/terrain-credits.txt
python3 - <<'PY'
import hashlib,json,pathlib
root=pathlib.Path('web')
files=['index.html','map.html','app.js','style.css','wasm_exec.js','main.wasm','rail-ui.ttf','favicon.svg','terrain-credits.txt']
version=hashlib.sha256(b''.join((root/f).read_bytes() for f in files)).hexdigest()[:16]
source="const CACHE='wwhsrt-m1-"+version+"';\nconst FILES="+json.dumps(['./']+files)+";\n"
source+='''self.addEventListener('install',e=>e.waitUntil(caches.open(CACHE).then(c=>c.addAll(FILES))));
self.addEventListener('activate',e=>e.waitUntil(caches.keys().then(keys=>Promise.all(keys.filter(k=>k.startsWith('wwhsrt-m1-')&&k!==CACHE).map(k=>caches.delete(k)))).then(()=>self.clients.claim())));
self.addEventListener('fetch',e=>{if(e.request.method!=='GET'||new URL(e.request.url).origin!==self.location.origin)return;e.respondWith(caches.match(e.request).then(r=>r||fetch(e.request)));});
'''
(root/'sw.js').write_text(source)
print('Built offline web bundle',version)
PY

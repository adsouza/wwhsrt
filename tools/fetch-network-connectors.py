"""Explicit source refresh. Offline compilation reads the pinned output instead.

Fetch active passenger-capable mainline geometry in operator-backed network bounds,
with a per-network gauge whitelist. Preserve ordered OSM node identities. Cache
responses under build/ so interrupted fetches resume without repeating requests.
"""
import concurrent.futures,gzip,hashlib,json,pathlib,re,subprocess,sys
from shapely import from_wkt
ROOT=pathlib.Path(__file__).resolve().parents[1]
CACHE=ROOT/'build/network-completion/source';CACHE.mkdir(parents=True,exist_ok=True)
ENDPOINT='https://qlever.dev/api/osm-planet'
def query(q,file):
 if file.exists():return json.loads(file.read_bytes())
 request=file.with_suffix('.sparql');request.write_text(q)
 b=subprocess.check_output(['curl','-L','--max-time','55','-sS','-H','Accept: application/sparql-results+json','--data-urlencode','query@'+str(request),ENDPOINT]);value=json.loads(b);assert 'results' in value,value
 file.write_bytes(b);return value

def main():
 bridges='--bridge-approaches' in sys.argv
 regions_file='data/imports/bridge-approach-regions.v1.json' if bridges else 'data/imports/connector-regions.v2.json'
 regions=json.loads((ROOT/regions_file).read_bytes())['regions'];old=set()
 sources=['data/imports/rail-topology-source.v1.jsonl.gz']
 if bridges:sources+=['data/imports/rail-connectors.v2.jsonl.gz','data/imports/rail-regional-links.v1.jsonl.gz']
 for source in sources:
  with gzip.open(ROOT/source,'rt') as f:
   next(f)
   for row in f:old.add(json.loads(row)[0])
 ways={};receipts=[]
 for region in regions:
  w,s,e,n=region['bounds'];q=f'''PREFIX k:<https://www.openstreetmap.org/wiki/Key:>
PREFIX geo:<http://www.opengis.net/ont/geosparql#>
PREFIX geof:<http://www.opengis.net/def/function/geosparql/>
SELECT ?way ?wkt ?speed ?gauge ?usage ?service ?traffic WHERE {{
 ?way k:railway "rail"; geo:hasGeometry/geo:asWKT ?wkt.
 FILTER(STRSTARTS(STR(?way),"https://www.openstreetmap.org/way/"))
 FILTER(geof:sfIntersects(?wkt,"POLYGON(({w} {s},{e} {s},{e} {n},{w} {n},{w} {s}))"^^geo:wktLiteral))
 FILTER NOT EXISTS {{?way k:service ?sv. FILTER(?sv IN ("yard","siding","spur"))}}
 FILTER NOT EXISTS {{?way k:disused "yes"}} FILTER NOT EXISTS {{?way k:abandoned "yes"}} FILTER NOT EXISTS {{?way k:construction ?c}}
 FILTER NOT EXISTS {{?way k:usage ?u. FILTER(?u IN ("military","industrial","test","freight"))}}
 FILTER NOT EXISTS {{?way k:railway:traffic_mode ?t. FILTER(?t IN ("freight","test"))}}
 OPTIONAL {{?way k:maxspeed ?speed}} OPTIONAL {{?way k:gauge ?gauge}} OPTIONAL {{?way k:usage ?usage}} OPTIONAL {{?way k:service ?service}} OPTIONAL {{?way k:railway:traffic_mode ?traffic}}
}}'''
  file=CACHE/(region['id']+'.json');value=query(q,file);admitted=[]
  for r in value['results']['bindings']:
   id=int(r['way']['value'].split('/')[-1]);shape=from_wkt(r['wkt']['value']);gauge=r.get('gauge',{}).get('value','')
   # Unknown gauge is not silently assumed compatible.
   if shape.geom_type!='LineString':continue
   known=bool(set(gauge.split(';'))&set(region['gauges']))
   inferred=not gauge and region.get('allow_untagged_active_rail',False)
   if not known and not inferred:continue
   admitted.append(id)
   if id in old or id in ways:continue
   speed=r.get('speed',{}).get('value','100');m=re.match(r'^(\d+)',speed);v=int(m[1]) if m else 100
   if 'mph' in speed:v=round(v*1.609344)
   ways[id]=[id,max(20,min(160,v)),True,r['wkt']['value']]
  receipts.append(dict(region=region['id'],query=q,response_sha256=hashlib.sha256(file.read_bytes()).hexdigest(),gauge_whitelist=region['gauges'],admitted_ways=len(admitted),operator_source=region['source'],unknown_gauge_policy=region.get('gauge_inference','reject')))
  print(region['id'],len(admitted),'admitted;',len(ways),'new ways',flush=True)
 ids=sorted(ways)
 def members(start):
  chunk=ids[start:start+5000];fingerprint=hashlib.sha256(','.join(map(str,chunk)).encode()).hexdigest()[:12];file=CACHE/f'nodes-{start//5000:03d}-{fingerprint}.json'
  q='SELECT ?way (GROUP_CONCAT(CONCAT(STR(?pos),":",STRAFTER(STR(?node),"/node/"));separator=",") AS ?members) WHERE { VALUES ?way {'+' '.join('<https://www.openstreetmap.org/way/'+str(id)+'>' for id in chunk)+'} ?way <https://www.openstreetmap.org/way/member> ?m. ?m <https://www.openstreetmap.org/way/member_id> ?node; <https://www.openstreetmap.org/way/member_pos> ?pos. } GROUP BY ?way'
  x=query(q,file);assert len(x['results']['bindings'])==len(chunk)
  for r in x['results']['bindings']:
   id=int(r['way']['value'].split('/')[-1]);pairs=sorted(tuple(map(int,p.split(':'))) for p in r['members']['value'].split(','));assert [p for p,n in pairs]==list(range(len(pairs)))
   row=ways[id];nodes=[n for p,n in pairs]
   if len(nodes)!=row[3].count(',')+1:
    # Restore collinear nodes omitted by the geometry export using their IDs.
    q='SELECT ?pos ?wkt WHERE {<https://www.openstreetmap.org/way/'+str(id)+'> <https://www.openstreetmap.org/way/member> ?m. ?m <https://www.openstreetmap.org/way/member_id> ?node; <https://www.openstreetmap.org/way/member_pos> ?pos. ?node <http://www.opengis.net/ont/geosparql#hasGeometry>/<http://www.opengis.net/ont/geosparql#asWKT> ?wkt} ORDER BY ?pos'
    pts=query(q,CACHE/f'expanded-{id}.json')['results']['bindings'];assert len(pts)==len(nodes)
    row[3]='LINESTRING('+','.join(' '.join(map(str,from_wkt(p['wkt']['value']).coords[0])) for p in pts)+')'
   row.append(nodes)
  print('node batch',start//5000,flush=True)
  return hashlib.sha256(file.read_bytes()).hexdigest()
 with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:node_hashes=list(pool.map(members,range(0,len(ids),5000)))
 header=dict(format='wwhsrt-rail-connectors-v2',retrieved_on='2026-11-13',endpoint=ENDPOINT,license='ODbL 1.0; OpenStreetMap contributors',base_sha256=hashlib.sha256((ROOT/'data/imports/rail-topology-source.v1.jsonl.gz').read_bytes()).hexdigest(),regions=receipts,node_response_sha256=node_hashes)
 if bridges:
  header['format']='wwhsrt-bridge-approaches-v1'
  header['excluded_source_sha256']={p:hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in sources}
 out=ROOT/('data/imports/rail-bridge-approaches.v1.jsonl.gz' if bridges else 'data/imports/rail-connectors.v2.jsonl.gz')
 with out.open('wb') as target:
  with gzip.GzipFile(filename='',fileobj=target,mode='wb',mtime=0) as f:
   for row in [header,*[ways[id] for id in ids]]:f.write((json.dumps(row,ensure_ascii=False,separators=(',',':'))+'\n').encode())
 print('Pinned',len(ids),'connector ways;',out.stat().st_size,'bytes',flush=True)
if __name__=='__main__':main()

"""Contract pinned, unsimplified source track into station-to-station corridors.

No geometric intersection/snap joins: only shared OSM node IDs form
junctions. Fixed city station pins attach to the nearest source vertex <=400 m.
Slow connector geometry is admitted only by the pinned service-region extract.
Run separately from the ordinary compiler; no network or player-save access.
"""
import array,collections,gzip,hashlib,heapq,json,math,pathlib
import numpy as np
from scipy.spatial import cKDTree
from scipy.sparse import csr_matrix
from scipy.sparse.csgraph import dijkstra,connected_components
from shapely import from_wkt,STRtree
from shapely.geometry import LineString,Point
ROOT=pathlib.Path(__file__).resolve().parents[1]
def km(a,b):
 x,y,z,t=map(math.radians,(*a,*b));return 12742*math.asin(min(1,math.sqrt(math.sin((t-y)/2)**2+math.cos(y)*math.cos(t)*math.sin((z-x)/2)**2)))
def main():
 from rail_source import rows as source_rows,catalog,sha,SOURCE,CONNECTORS,REGIONAL,BRIDGES,MANIFEST,BASE
 base,manifest=catalog()
 platforms={r['city_id']:r for r in manifest.get('platform_hubs',[])}
 platform_ways={wid:id for id,r in platforms.items() for wid in r['way_ids']}
 citymap={c['id']:c for c in base['cities']}
 platform_links=[]
 # Project every station onto its actual source edge, including long edges.
 # Geometry proximity chooses a station adapter; it never joins rail crossings.
 shapes=[];wids=[]
 for wid,speed,connector,wkt,node_ids in source_rows():
  shape=from_wkt(wkt);shapes.append(shape);wids.append(wid)
  if wid in platform_ways:
   id=platform_ways[wid];c=citymap[id];p=Point(c['station_lon'],c['station_lat']);q=shape.interpolate(shape.project(p));distance=km((p.x,p.y),(q.x,q.y))*1000
   assert not connector and speed>=200 and distance<=platforms[id]['max_adapter_m'],('Unqualified station-platform adapter',id,wid,distance)
   platform_links.append((id,wid,(q.x,q.y),round(distance,2)))
 tree=STRtree(shapes);projections={};adapter_info={}
 for c in base['cities']:
  if 'station_lon' not in c:continue
  p=Point(c['station_lon'],c['station_lat']);i=int(tree.nearest(p));shape=shapes[i];q=shape.interpolate(shape.project(p));d=km((p.x,p.y),(q.x,q.y))
  adapter_info[c['id']]=round(d*1000,2)
  if d<=.4:projections.setdefault(wids[i],set()).add((q.x,q.y))
 del tree,shapes,wids
 assert {wid for id,wid,p,d in platform_links}==set(platform_ways),'Missing reviewed HSR platform way'
 for id,wid,p,d in platform_links:projections.setdefault(wid,set()).add(p)
 lookup={};coords=[];rows=array.array('i');cols=array.array('i');seconds=array.array('f');length=array.array('i');slow=array.array('i')
 projection_nodes={}
 def node(p,key):
  p=tuple(p)
  if key not in lookup:lookup[key]=len(coords);coords.append(p)
  else:assert max(abs(a-b) for a,b in zip(coords[lookup[key]],p))<=.000001,("Source node geometry disagrees",key,coords[lookup[key]],p,wid)
  return lookup[key]
 ways=0
 for wid,speed,connector,wkt,node_ids in source_rows():
  shape=from_wkt(wkt);last=None;ways+=1
  path=list(zip(shape.coords,node_ids));existing={p for p,id in path}
  if wid in projections:path=sorted(path+[(p,(wid,p)) for p in sorted(projections[wid]) if p not in existing],key=lambda row:shape.project(Point(row[0])))
  way_projections=projections.get(wid,())
  for p,key in path:
   n=node(p,key)
   if p in way_projections:projection_nodes[wid,p]=n
   if last is not None and last!=n:
    m=max(1,round(km(coords[last],p)*1000));s=max(.01,m*3.6/speed)
    for a,b in [(last,n),(n,last)]:rows.append(a);cols.append(b);seconds.append(s);length.append(m);slow.append(m if connector else 0)
   last=n
 platform_nodes={}
 for id,wid,p,d in platform_links:
  city=citymap[id];hub=node((city['station_lon'],city['station_lat']),('station-platform-hub',id));platform_nodes[id]=hub
  target=projection_nodes[wid,p];m=max(1,round(d))
  for a,b in [(hub,target),(target,hub)]:rows.append(a);cols.append(b);seconds.append(max(.01,m*3.6/60));length.append(m);slow.append(m)
 print('Source vertices',len(coords),'ways',ways,flush=True)
 # Drop duplicate source arcs by fastest traversal, never sum parallel tracks.
 r=np.frombuffer(rows,dtype=np.int32);c=np.frombuffer(cols,dtype=np.int32);s=np.frombuffer(seconds,dtype=np.float32)
 order=np.lexsort((s,c,r));rr,cc=r[order],c[order];keep=np.r_[True,(rr[1:]!=rr[:-1])|(cc[1:]!=cc[:-1])];ix=order[keep]
 rr,cc=rr[keep],cc[keep];ss=s[ix];mm=np.frombuffer(length,dtype=np.int32)[ix];ll=np.frombuffer(slow,dtype=np.int32)[ix]
 g=csr_matrix((ss,(rr,cc)),shape=(len(coords),len(coords)));meters=csr_matrix((mm,(rr,cc)),shape=g.shape);connector=csr_matrix((ll,(rr,cc)),shape=g.shape)
 del rows,cols,seconds,length,slow,r,c,s,order,keep,ix,rr,cc,ss,mm,ll,lookup
 points=np.asarray(coords);rad=np.radians(points);xyz=np.column_stack((np.cos(rad[:,1])*np.cos(rad[:,0]),np.cos(rad[:,1])*np.sin(rad[:,0]),np.sin(rad[:,1])));tree=cKDTree(xyz)
 pins={};claimed=set()
 for city in base['cities']:
  if 'station_lon' not in city:continue
  lon,lat=map(math.radians,[city['station_lon'],city['station_lat']]);_,n=tree.query([math.cos(lat)*math.cos(lon),math.cos(lat)*math.sin(lon),math.sin(lat)]);n=platform_nodes.get(city['id'],int(n))
  if km(coords[n],(city['station_lon'],city['station_lat']))<=.4 and n not in claimed:pins[city['id']]=n;claimed.add(n)
 del tree,xyz,rad,points
 hubs=sorted(pins);indices=[pins[id] for id in hubs];owners={pins[id]:id for id in hubs}
 ds,prev,nearest=dijkstra(g,directed=False,indices=indices,min_only=True,return_predecessors=True)
 pairs={}
 for n in range(len(coords)):
  a=int(nearest[n])
  if a<0:continue
  for j in range(g.indptr[n],g.indptr[n+1]):
   nxt=int(g.indices[j]);b=int(nearest[nxt])
   if a>=b or b<0:continue
   key=tuple(sorted((owners[a],owners[b])));value=float(ds[n]+g.data[j]+ds[nxt])
   if key not in pairs or value<pairs[key][0]:pairs[key]=(value,n,nxt)
 out=[]
 for (a,b),(cost,n,nxt) in sorted(pairs.items()):
  left=[n];right=[nxt]
  while prev[left[-1]]>=0:left.append(int(prev[left[-1]]))
  while prev[right[-1]]>=0:right.append(int(prev[right[-1]]))
  path=list(reversed(left))+right
  if owners[int(nearest[n])]!=a:path.reverse()
  distance=sum(int(meters[u,v]) for u,v in zip(path,path[1:]));sl=sum(int(connector[u,v]) for u,v in zip(path,path[1:]));shape=LineString([coords[n] for n in path]).simplify(.0005,preserve_topology=False)
  pts=[dict(lon=round(x,7),lat=round(y,7)) for x,y in shape.coords]
  # Station adapters are local and visible; never connect different terminals.
  citymap={c['id']:c for c in base['cities']}
  for end,id in [(0,a),(-1,b)]:
   city=citymap[id];pt=dict(lon=city['station_lon'],lat=city['station_lat'])
   if pts[end]!=pt:
    delta=km((pts[end]['lon'],pts[end]['lat']),(pt['lon'],pt['lat']));distance+=round(delta*1000);cost+=delta*60
    if end==0:pts.insert(0,pt)
    else:pts.append(pt)
  k=max(1,round(distance/1000));out.append(dict(id='public_'+a+'__'+b,from_city=a,to_city=b,existing=True,daily_market=5000,outside_cost=15000,alignments=[dict(id='prototype',km=k,minutes=max(5,math.ceil(cost/60)),months=0,surface_km=k,hill_km=0,viaduct_km=0,tunnel_km=0,connector_km=min(k,round(sl/1000)),path=pts)]))
 count,labels=connected_components(g,directed=False)
 # Reassess the old review queue without auto-joining any candidate.
 points=np.radians(np.asarray(coords));xyz=np.column_stack((np.cos(points[:,1])*np.cos(points[:,0]),np.cos(points[:,1])*np.sin(points[:,0]),np.sin(points[:,1])));tree=cKDTree(xyz)
 review=[]
 for site in json.loads((ROOT/'docs/validation/network-topology-audit.json').read_bytes())['proximity_review_sites']:
  found=[];distances=[]
  for p in [site['a'],site['b']]:
   lon,lat=map(math.radians,p);_,i=tree.query([math.cos(lat)*math.cos(lon),math.cos(lat)*math.sin(lon),math.sin(lat)]);i=int(i);found.append(int(labels[i]));distances.append(round(km(p,coords[i])*1000,2))
  review.append(dict(a=site['a'],b=site['b'],kind=site['kind'],status='connected_in_source_node_topology' if found[0]==found[1] and max(distances)<=400 else 'requires_junction_or_source_review',source_distance_m=distances))
 network_checks=[]
 for network in manifest['networks']:
  missing=[id for id in network['cities'] if id not in pins]
  comps={int(labels[pins[id]]) for id in network['cities'] if id in pins}
  network_checks.append(dict(id=network['id'],missing_hubs=missing,components=sorted(comps),connected=not missing and len(comps)==1))
 result=dict(id='source-topology-repair-v2',source_sha256=sha(SOURCE),connectors_sha256=sha(CONNECTORS),regional_links_sha256=sha(REGIONAL),bridge_approaches_sha256=sha(BRIDGES),manifest_sha256=sha(MANIFEST),base_catalog_sha256=hashlib.sha256(gzip.decompress(BASE.read_bytes())).hexdigest(),cities=base['cities'],market_cities=base['network']['cities'],network_checks=network_checks,station_adapter_m=adapter_info,platform_adapters=[dict(city_id=id,way_id=wid,distance_m=d) for id,wid,p,d in sorted(platform_links)],source_vertices=len(coords),source_ways=ways,source_components=count,review_sites=review,pins={id:dict(lon=coords[n][0],lat=coords[n][1],component=int(labels[n])) for id,n in sorted(pins.items())},corridors=out)
 target=ROOT/'data/imports/network-repairs.v2.json';data=(json.dumps(result,ensure_ascii=False,separators=(',',':'))+'\n').encode()
 import sys
 failed=[r for r in network_checks if not r['connected']]
 if failed:
  (ROOT/'build/network-completion/candidate-repairs.json').write_bytes(data)
  raise SystemExit('Source network coverage failed: '+json.dumps(failed))
 if '--check' in sys.argv:assert target.read_bytes()==data,'Repair output differs'
 else:target.write_bytes(data)
 print('Corridors',len(out),'hubs',len(pins),flush=True)
if __name__=='__main__':main()

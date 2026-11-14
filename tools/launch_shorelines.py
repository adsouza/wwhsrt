"""Replace coarse coastal countries that conflict with the new game alignments."""
import gzip,json,math,pathlib
from shapely.geometry import Polygon,LineString,shape
from shapely.ops import unary_union,transform
ROOT=pathlib.Path(__file__).resolve().parents[1]
def project(x,y,z=None):
 try:return tuple(project(a,b)[0] for a,b in zip(x,y)),tuple(project(a,b)[1] for a,b in zip(x,y))
 except TypeError:pass
 t=math.asin(math.sqrt(3)/2*math.sin(math.radians(y)));t2=t*t;t6=t**6
 return 2*math.sqrt(3)*math.radians(x)*math.cos(t)/(3*(1.340264-3*.081106*t2+t6*(7*.000893+9*.003796*t2))),t*(1.340264-.081106*t2+t6*(.000893+.003796*t2))
def refine(rings,source,launch):
 mask=json.loads(gzip.decompress((ROOT/'data/construction/route-land.v1.json.gz').read_bytes()));countries={c['name']:shape(c['geometry']) for c in mask['countries']};targets={'Australia','South Africa','Chile'}
 key=lambda r:json.dumps(r,separators=(',',':'))
 removed=set()
 for f in source['countries']['features']:
  if f['properties']['name'] not in targets:continue
  g=f['geometry'];polys=g['coordinates'] if g['type']=='MultiPolygon' else [g['coordinates']];removed.update(key(p[0]) for p in polys)
 assert removed<={key(r) for r in rings}
 rings=[r for r in rings if key(r) not in removed];reviews={r['corridor_id']:r for r in launch['review']};metadata={}
 for name in sorted(targets):
  country=countries[name];polys=list(country.geoms) if hasattr(country,'geoms') else [country]
  affected=[c for c in launch['corridors'] if any(p['country']==name for p in c['alignments'][0]['cost_intervals'])]
  paths=[]
  for c in affected:
   path=LineString([(p['lon'],p['lat']) for p in c['alignments'][0]['path']]);allowed=unary_union([shape(a['geometry']) for a in reviews[c['id']]['crossings']]);paths.append((path,allowed))
  for tolerance in [.003,.001,.0003,.0001,0]:
   candidates=[Polygon(p.exterior).simplify(tolerance,preserve_topology=True) for p in polys];ground=unary_union(candidates);projected=transform(project,ground)
   if all(ground.union(a).covers(p) and projected.union(transform(project,a)).covers(LineString([project(*q) for q in p.coords])) for p,a in paths):break
  else:raise ValueError('Detailed drawing does not cover construction paths: '+name)
  extra=[[[x,y] for x,y in p.exterior.coords] for p in candidates];rings.extend(extra);metadata[name]=dict(simplification_degrees=tolerance,rings=len(extra),vertices=sum(map(len,extra)))
 assert len(rings)==len({key(r) for r in rings}),'Duplicate refined shoreline rings'
 return rings,dict(replaced_coarse_rings=len(removed),refinements=metadata,source_mask_sha256=launch['land_mask_sha256'])

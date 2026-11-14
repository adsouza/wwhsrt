"""Pinned source inputs shared by offline preparation and topology checks."""
import gzip,json,pathlib,hashlib,copy
ROOT=pathlib.Path(__file__).resolve().parents[1]
BASE=ROOT/'assets/scenario/public-network-v2.json.gz'
SOURCE=ROOT/'data/imports/rail-topology-source.v1.jsonl.gz'
CONNECTORS=ROOT/'data/imports/rail-connectors.v2.jsonl.gz'
REGIONAL=ROOT/'data/imports/rail-regional-links.v1.jsonl.gz'
BRIDGES=ROOT/'data/imports/rail-bridge-approaches.v1.jsonl.gz'
MANIFEST=ROOT/'data/imports/operating-networks.v2.json'
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rows():
 with gzip.open(CONNECTORS,'rt') as f:header=json.loads(next(f));assert header['base_sha256']==sha(SOURCE)
 with gzip.open(BRIDGES,'rt') as f:
  header=json.loads(next(f))
  assert all(sha(ROOT/p)==value for p,value in header['excluded_source_sha256'].items())
 for p in [SOURCE,CONNECTORS,REGIONAL,BRIDGES]:
  with gzip.open(p,'rt') as f:
   next(f)
   for line in f:yield json.loads(line)
def catalog():
 base=json.loads(gzip.decompress(BASE.read_bytes()));manifest=json.loads(MANIFEST.read_bytes());cities={c['id']:c for c in base['cities']};markets={c['id']:c for c in base['network']['cities']}
 metros={c['id']:c for c in json.loads((ROOT/'data/prototypes/playable-metros.v1.json').read_bytes())}
 for h in manifest['hubs']:
  id=h['city_id']
  if id not in cities:
   seed=metros.get(id)
   c=dict(id=id,name=seed['name'].removesuffix(' Metropolitan Area') if seed else h['name'],lat=seed['lat'] if seed else h['lat'],lon=seed['lng'] if seed else h['lon']);base['cities'].append(c);cities[id]=c
   demand=max(500,min(50000,round(seed['population']/1000))) if seed else 500
   m=dict(c,daily_trips=demand,attraction=demand,road_region=h['road_region']);base['network']['cities'].append(m);markets[id]=m
  c=cities[id];c.update(station_name=h['name'],station_lat=h['lat'],station_lon=h['lon']);markets[id].update(c);markets[id]['road_region']=h['road_region']
 return base,manifest

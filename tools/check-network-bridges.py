"""Independent admission checks for the three prospects and inherited rail repairs."""
import copy,gzip,hashlib,json,pathlib
from network_coverage import checks,require,require_map
ROOT=pathlib.Path(__file__).resolve().parents[1]
def read(p):return json.loads((ROOT/p).read_bytes())
def sha(p):return hashlib.sha256((ROOT/p).read_bytes()).hexdigest()
def main():
 c=read('assets/scenario/public-network.json');prior=json.loads(gzip.decompress((ROOT/'assets/scenario/public-network-v5.json.gz').read_bytes()))
 rows={r['id']:r for r in c['corridors']};old={r['id']:r for r in prior['corridors']};cities={r['id']:r for r in c['cities']}
 assert set(cities)=={r['id'] for r in prior['cities']}|{'maibara','camp_de_tarragona','figueres'}
 assert set(old)<=set(rows),'An accepted corridor disappeared'
 for id,r in old.items():
  assert (r['from_city'],r['to_city'],r['existing'])==(rows[id]['from_city'],rows[id]['to_city'],rows[id]['existing']),('Accepted identity changed',id)
  assert len(r['alignments'])==len(rows[id]['alignments'])
  for a,b in zip(r['alignments'],rows[id]['alignments']):
   assert {k:v for k,v in a.items() if k not in {'path','connector_km'}}=={k:v for k,v in b.items() if k not in {'path','connector_km'}},('Accepted economics changed',id)
 routes={r['id']:r for r in c['network']['routes']}
 for r in prior['network']['routes']:assert routes.get(r['id'])==r,('Accepted service redirected',r['id'])
 old_markets={m['id']:m for m in prior['network']['cities']}
 for m in c['network']['cities']:
  if m['id'] in old_markets:
   assert all(m[k]==old_markets[m['id']][k] for k in ['daily_trips','attraction','road_region','lon','lat']),('Unrelated demand changed',m['id'])
  else:assert m['daily_trips']==m['attraction']==500
 manifest=read('data/imports/operating-networks.v2.json');networks=require(c,manifest)
 scene=read('assets/scenario/playable-rail-map.json');require_map(c,scene,manifest)
 expected={'launch_72_tsuruga__maibara':('tsuruga','maibara'),'launch_73_valencia__camp_de_tarragona':('valencia','camp_de_tarragona'),'launch_74_porto__vigo':('porto','stop_node_12819737430')}
 assert {id for id,r in rows.items() if not r['existing']}=={r['id'] for r in prior['corridors'] if not r['existing']}|set(expected)
 for id,pair in expected.items():
  r=rows[id];assert not r['existing'] and (r['from_city'],r['to_city'])==pair
  a=r['alignments'][0];assert a['months']>=36 and a['cost_intervals']
  for end,city in [(0,pair[0]),(-1,pair[1])]:assert a['path'][end]=={'lon':cities[city]['station_lon'],'lat':cities[city]['station_lat']}
 # These inherited links are present immediately, never offered as new works.
 public=['public_milan__turin','public_figueres__perpignan','public_barcelona__figueres','public_barcelona__camp_de_tarragona','public_kyoto__maibara','public_maibara__nagoya']
 for id in public:assert rows[id]['existing'] and id in scene['corridor_ids']
 assert rows['public_kyoto__nagoya']['shared_sections']==['public_kyoto__maibara','public_maibara__nagoya'] or set(rows['public_kyoto__nagoya']['shared_sections'])=={'public_kyoto__maibara','public_maibara__nagoya'}
 for authored in read('data/construction/launch-prospects.v1.json')['services']:
  r=routes[authored['id']];assert r['cities']==r['stops']==authored['cities'] and any(not rows[id]['existing'] for id in r['sections'])
  if 'valencia' in r['cities']:assert 'public_barcelona__camp_de_tarragona' in r['sections']
  if 'lisbon' in r['cities']:assert 'public_lisbon__porto' in r['sections'] and 'launch_26_lisbon__porto' not in r['sections']
 negative=[]
 for cut,network in [('public_milan__turin','italian-hsr'),('public_figueres__perpignan','catalan-hsr'),('public_kyoto__maibara','tokaido-junction')]:
  broken=copy.deepcopy(c);broken['corridors']=[r for r in broken['corridors'] if r['id']!=cut]
  assert not next(n for n in checks(broken,manifest) if n['id']==network)['connected'],('A severed source link passed',cut)
  negative.append(cut+' severed')
 patch=read('data/imports/network-repairs.v2.json');assert patch['bridge_approaches_sha256']==c['bridge_approaches_sha256']==sha('data/imports/rail-bridge-approaches.v1.jsonl.gz')
 assert all(a['distance_m']<=100 for a in patch['platform_adapters'])
 result=dict(passed=True,catalog_sha256=sha('assets/scenario/public-network.json'),cities=len(cities),existing_corridors=sum(r['existing'] for r in rows.values()),prospects=len(expected)+71,service_templates=len(routes),mapped_stations=len(scene['station_ids']),station_inventory=len(c['stations']),networks=networks,public_repairs=public,new_prospects=list(expected),platform_adapters=patch['platform_adapters'],negative_fixtures=negative,limits=['Pinned source topology and bounded strategic station aggregation, not physical crossover engineering.','New construction uses authored game terrain quantities and durations; profitability depends on fares, feeder services, grants and financing.'])
 (ROOT/'docs/validation/network-bridge-checks.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
 print('Passed all three connectors, inherited network repairs, accepted identities, capacity aliases, station markers and severed-link regressions')
if __name__=='__main__':main()

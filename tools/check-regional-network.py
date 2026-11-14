"""Independent admission checks for Yeongcheon, Taipei and Houston–Austin."""
import copy,gzip,hashlib,json,math,pathlib
from network_coverage import require,require_map,checks

ROOT=pathlib.Path(__file__).resolve().parents[1]
def read(p):return json.loads((ROOT/p).read_bytes())
def sha(p):return hashlib.sha256((ROOT/p).read_bytes()).hexdigest()
def length(path):
 total=0
 for a,b in zip(path,path[1:]):
  x,y,z,t=map(math.radians,[a['lon'],a['lat'],b['lon'],b['lat']]);total+=12742*math.asin(min(1,math.sqrt(math.sin((t-y)/2)**2+math.cos(y)*math.cos(t)*math.sin((z-x)/2)**2)))
 return total
def admit_daegu(path):
 assert max(p['lat'] for p in path)<35.9,'Yeongcheon turnaround remains'
 assert 70<length(path)<95,'Daegu–Ulsan source route is missing or still detours'
def main():
 catalog=read('assets/scenario/public-network.json');rows={c['id']:c for c in catalog['corridors']};cities={c['id']:c for c in catalog['cities']}
 prior=json.loads(gzip.decompress((ROOT/'assets/scenario/public-network-v4.json.gz').read_bytes()))
 assert set(cities)=={c['id'] for c in prior['cities']}|{'taipei','maibara','camp_de_tarragona','figueres'}
 old={c['id']:c for c in prior['corridors']}
 assert set(old)|{'public_taichung__taipei','launch_71_houston__austin'}<=set(rows)
 for id,co in old.items():
  for a,b in zip(co['alignments'],rows[id]['alignments']):
   ignored={'path','connector_km'}
   if 'ulsan' in (co['from_city'],co['to_city']):
    ignored|={'km','minutes','surface_km'}
    assert abs(length(b['path'])-b['km'])<1,'Ulsan service still charges the detoured distance'
   assert {k:v for k,v in a.items() if k not in ignored}=={k:v for k,v in b.items() if k not in ignored},'Unrelated economic inputs changed: '+id
 patterns={r['id']:r for r in catalog['network']['routes']}
 for r in prior['network']['routes']:assert patterns.get(r['id'])==r,'Accepted service path changed: '+r['id']
 assert cities['ulsan']['station_lon']==129.13793 and cities['ulsan']['station_lat']==35.55131
 assert cities['taipei']['station_name']=='Taipei' and cities['taipei']['station_lon']==121.517 and cities['taipei']['station_lat']==25.04793
 manifest=read('data/imports/operating-networks.v2.json');require(catalog,manifest)
 scene=read('assets/scenario/playable-rail-map.json');require_map(catalog,scene,manifest)
 assert 'node/3799089099' in scene['station_ids']
 taiwan=next(r for r in catalog['network']['routes'] if r['id']=='network_taiwan_hsr_all_stops')
 assert taiwan['stops']==['taipei','taichung','tainan','kaohsiung'] and taiwan['cities']==taiwan['stops']
 route=next(r for r in catalog['network']['routes'] if r['id']=='launch_route_houston__san_antonio_regional')
 assert route['cities']==['houston','austin','san_antonio'] and 'launch_71_houston__austin' in route['sections']
 repair=read('data/imports/network-repairs.v2.json');assert repair['regional_links_sha256']==sha('data/imports/rail-regional-links.v1.jsonl.gz')==catalog['regional_links_sha256']
 with gzip.open(ROOT/'data/imports/rail-regional-links.v1.jsonl.gz','rt') as stream:
  metadata=json.loads(next(stream));links=[json.loads(line) for line in stream]
 assert {r[0]for r in links}=={1019974746,1019974747,1019974748,1025593622,710482736,710482737,968527286,968527287}
 assert all(len(r[4])==r[3].count(',')+1 and len(set(r[4]))==len(r[4]) for r in links)
 assert len(repair['platform_adapters'])==35 and all(a['distance_m']<=100 for a in repair['platform_adapters'])
 path=rows['public_daegu__ulsan']['alignments'][0]['path'];admit_daegu(path)
 try:admit_daegu(old['public_daegu__ulsan']['alignments'][0]['path'])
 except AssertionError:pass
 else:raise AssertionError('Old Yeongcheon detour was admitted')
 broken=copy.deepcopy(catalog);broken['corridors']=[c for c in broken['corridors'] if c['id']!='public_taichung__taipei']
 assert not next(r for r in checks(broken,manifest) if r['id']=='taiwan-hsr')['connected']
 result=dict(passed=True,catalog_sha256=sha('assets/scenario/public-network.json'),cities=len(cities),existing_corridors=sum(c['existing']for c in rows.values()),prospects=sum(not c['existing']for c in rows.values()),daegu_ulsan_drawn_km=round(length(path),3),prior_detour_drawn_km=round(length(old['public_daegu__ulsan']['alignments'][0]['path']),3),regional_source_way_ids=[r[0] for r in links],platform_adapters=repair['platform_adapters'],taiwan_stops=taiwan['stops'],houston_austin_km=rows['launch_71_houston__austin']['alignments'][0]['km'],ulsan_gateway=cities['ulsan']['station_name'],ulsan_operating_inputs=[dict(id=id,km=rows[id]['alignments'][0]['km'],minutes=rows[id]['alignments'][0]['minutes'])for id in ['public_daegu__ulsan','public_busan__ulsan','public_seoul__ulsan']],negative_fixtures=['Old Yeongcheon detour rejected','Severed Taipei service rejected'],limits=['Station-platform aggregation is a reviewed strategic station resource, not a new physical track junction.','HSR construction alignment and engineering shares are authored game estimates.'])
 (ROOT/'docs/validation/regional-network-checks.json').write_text(json.dumps(result,indent=2)+'\n')
 print('Passed repaired Yeongcheon alignment, Taipei HSR chain, Houston–Austin/through-running and scoped prospective Ulsan distance corrections')
if __name__=='__main__':main()

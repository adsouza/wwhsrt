"""Independent end-to-end and ordered-service checks over the compiled catalog."""
import collections

def checks(catalog,manifest):
 cities={c['id']:c for c in catalog['cities']};corridors={c['id']:c for c in catalog['corridors']};adj=collections.defaultdict(set)
 def valid(id,seen):
  if id not in corridors or id in seen:return False
  c=corridors[id]
  return c['existing'] and all(valid(k,seen|{id}) for k in c.get('shared_sections',[]))
 for id,c in corridors.items():
  if valid(id,set()):a,b=c['from_city'],c['to_city'];adj[a].add(b);adj[b].add(a)
 result=[]
 for network in manifest['networks']:
  ids=network['cities'];seen={ids[0]};q=[ids[0]]
  for n in q:
   for other in sorted(adj[n]-seen):seen.add(other);q.append(other)
  missing=[id for id in ids if id not in cities or id not in seen]
  row=dict(id=network['id'],source=network['source'],cities=ids,missing=missing,connected=not missing)
  if network.get('ordered'):
   pattern=next((r for r in catalog['network']['routes'] if r['id']=='network_'+network['id'].replace('-','_')+'_all_stops'),None)
   if pattern is None and len(ids)==2:
    co=next((c for c in corridors.values() if {c['from_city'],c['to_city']}==set(ids) and valid(c['id'],set())),None)
    if co:pattern=dict(id=co['id'],sections=[co['id']],cities=ids,stops=ids)
   row['service_id']=pattern['id'] if pattern else None
   chain=bool(pattern and len(pattern['sections'])==len(pattern['cities'])-1 and len(set(pattern['cities']))==len(pattern['cities']))
   if chain:
    for a,b,section in zip(pattern['cities'],pattern['cities'][1:],pattern['sections']):
     co=corridors.get(section,{})
     chain &= valid(section,set()) and {a,b}=={co.get('from_city'),co.get('to_city')}
    chain &= [c for c in pattern['cities'] if c in ids]==ids
   row['ordered_service']=bool(chain and pattern['stops']==ids)
   row['connected'] &= row['ordered_service']
  result.append(row)
 return result

def require(catalog,manifest):
 result=checks(catalog,manifest);failed=[r for r in result if not r['connected']]
 if failed:raise ValueError('Operating-network coverage failed: '+repr(failed))
 return result

def require_map(catalog,map_data,manifest):
 """Required station markers and city-to-city paths must ship with the services."""
 city={c['id']:c for c in catalog['cities']};stations={s['id']:s for s in catalog['stations']}
 visible=set(map_data['station_ids']);drawn=dict(zip(map_data['corridor_ids'],map_data['lines']))
 assert len(drawn)==len(map_data['lines'])==len(map_data['corridor_ids'])
 for hub in manifest['hubs']:
  id=hub['station_id'];c=city[hub['city_id']]
  assert id in stations and id in visible,'Missing operating station marker: '+id
  assert (c['station_lon'],c['station_lat'])==(stations[id]['lon'],stations[id]['lat']),'Station anchor mismatch: '+id
 for co in catalog['corridors']:
  if not co['existing']:continue
  path=drawn.get(co['id']);assert path and len(path)>1,'Missing mapped corridor: '+co['id']
  for end,id in [(0,co['from_city']),(-1,co['to_city'])]:
   c=city[id];assert path[end]==[c['station_lon'],c['station_lat']],'Detached track endpoint: '+co['id']
 return dict(required_station_markers=len(manifest['hubs']),drawn_corridors=len(drawn))

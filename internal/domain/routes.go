package domain

import (
	"slices"
	"sort"
	"wwhsrt/pkg/gameapi"
)

func (c Catalog) WithNetwork(r gameapi.NetworkRules) (Catalog, error) {
	if r.ID == "" || len(r.Cities) < 2 || len(r.Cities) > 1024 || len(r.Routes) > 8192 || r.PublicPaths < 1 ||
		r.PublicPaths > 64 ||
		r.PrivatePaths < 1 ||
		r.PrivatePaths > 64 ||
		r.StationCalls < 2 ||
		r.StationCalls > 256 ||
		r.DwellMinutes < 0 ||
		r.DwellMinutes > 10 ||
		r.TransferMinutes < 1 ||
		r.TransferMinutes > 60 {
		return c, errCode("invalid_scenario")
	}
	seen := map[gameapi.CityID]bool{}
	for _, city := range r.Cities {
		if city.ID == "" || seen[city.ID] || city.DailyTrips < 1 || city.DailyTrips > 100000 || city.Attraction < 1 ||
			city.Attraction > 100000 ||
			city.RoadRegion == "" {
			return c, errCode("invalid_scenario")
		}
		seen[city.ID] = true
	}
	c.cityIndex = map[gameapi.CityID]int{}
	for i, city := range r.Cities {
		c.cityIndex[city.ID] = i
	}
	n := len(r.Cities)
	c.distanceKM = make([]int, n*n)
	if len(r.DistanceKM) > 0 {
		if len(r.DistanceKM) != n*n {
			return c, errCode("invalid_scenario")
		}
		for i, v := range r.DistanceKM {
			if (i/n == i%n && v != 0) || (i/n != i%n && (v < 1 || v > 25000)) || v != r.DistanceKM[(i%n)*n+i/n] {
				return c, errCode("invalid_scenario")
			}
			c.distanceKM[i] = int(v)
		}
	}
	for _, co := range c.corridors {
		if !seen[co.FromCity] || !seen[co.ToCity] {
			return c, errCode("invalid_scenario")
		}
	}
	distances := map[string]bool{}
	for _, d := range r.Distances {
		key := marketKey(d.From, d.To)
		if d.From == d.To || !seen[d.From] || !seen[d.To] || d.KM < 1 || d.KM > 25000 || distances[key] {
			return c, errCode("invalid_scenario")
		}
		distances[key] = true
		a, b := c.cityIndex[d.From], c.cityIndex[d.To]
		c.distanceKM[a*n+b], c.distanceKM[b*n+a] = d.KM, d.KM
	}
	if len(r.DistanceKM) == 0 && len(distances) != len(r.Cities)*(len(r.Cities)-1)/2 {
		return c, errCode("invalid_scenario")
	}
	for _, pair := range r.ConventionalPairs {
		if len(pair) != 2 || pair[0] == pair[1] || !seen[pair[0]] || !seen[pair[1]] {
			return c, errCode("invalid_scenario")
		}
	}
	ids := map[string]bool{}
	for _, co := range c.corridors {
		ids[co.ID] = true
	}
	for _, route := range r.Routes {
		if route.ID == "" || ids[route.ID] || len(route.Sections) < 2 || len(route.Sections) > 64 ||
			len(route.Cities) != len(route.Sections)+1 ||
			len(route.Stops) < 2 ||
			route.Stops[0] != route.Cities[0] ||
			route.Stops[len(route.Stops)-1] != route.Cities[len(route.Cities)-1] {
			return c, errCode("invalid_scenario")
		}
		ids[route.ID] = true
		visited := map[gameapi.CityID]bool{}
		for i, city := range route.Cities {
			if !seen[city] || visited[city] {
				return c, errCode("invalid_scenario")
			}
			visited[city] = true
			if i < len(route.Sections) {
				co, ok := c.find(route.Sections[i])
				if !ok ||
					!(co.FromCity == city && co.ToCity == route.Cities[i+1] || co.ToCity == city && co.FromCity == route.Cities[i+1]) {
					return c, errCode("invalid_scenario")
				}
			}
		}
		stop := 0
		for _, city := range route.Cities {
			if stop < len(route.Stops) && route.Stops[stop] == city {
				stop++
			}
		}
		if stop != len(route.Stops) {
			return c, errCode("invalid_scenario")
		}
	}
	for _, id := range r.SupersededRoutes {
		if _, corridor := c.find(id); !ids[id] || corridor {
			return c, errCode("invalid_scenario")
		}
	}
	r.SupersededRoutes = slices.Clone(r.SupersededRoutes)
	r.Cities = append([]gameapi.MarketCity{}, r.Cities...)
	r.DistanceKM = append([]uint16{}, r.DistanceKM...)
	r.Distances = append([]gameapi.MarketDistance{}, r.Distances...)
	r.Routes = append([]gameapi.Route{}, r.Routes...)
	for i := range r.Routes {
		r.Routes[i].Sections = append([]string{}, r.Routes[i].Sections...)
		r.Routes[i].Cities = append([]gameapi.CityID{}, r.Routes[i].Cities...)
		r.Routes[i].Stops = append([]gameapi.CityID{}, r.Routes[i].Stops...)
	}
	pairs := make([][]gameapi.CityID, len(r.ConventionalPairs))
	for i, p := range r.ConventionalPairs {
		pairs[i] = append([]gameapi.CityID{}, p...)
	}
	r.ConventionalPairs = pairs
	c.network = r
	return c, nil
}
func marketKey(a, b gameapi.CityID) string {
	if a > b {
		a, b = b, a
	}
	return string(a) + "/" + string(b)
}
func (c Catalog) distance(a, b gameapi.CityID) int {
	i, ok := c.cityIndex[a]
	j, found := c.cityIndex[b]
	if ok && found {
		return c.distanceKM[i*len(c.cityIndex)+j]
	}
	return 0
}

// The immutable prior catalog validates reports written before the import.
func (c Catalog) WithPrior(prior Catalog) Catalog { c.prior = &prior; return c }
func (w World) CatalogSince() int                 { return w.catalogSince }
func (w World) Expanded() bool                    { return w.catalog.network.Sparse }
func (w World) UpgradeCatalog(c Catalog) (World, error) {
	at := Cutovers{
		Network:   w.networkSince,
		Catalog:   w.months,
		Economy:   w.months,
		Regional:  regionalCutover(c, w.months),
		Connector: connectorCutover(c, w.months),
	}
	return Restore(
		c,
		w.revision,
		w.months,
		w.company,
		w.reports,
		w.projects,
		w.services,
		w.fleet,
		w.debt,
		w.pending,
		w.nextTrain,
		at,
	)
}
func corridorRoute(co gameapi.Corridor) gameapi.Route {
	return gameapi.Route{
		InfrastructureKind: co.InfrastructureKind,
		ID:                 co.ID,
		Sections:           []string{co.ID},
		Cities:             []gameapi.CityID{co.FromCity, co.ToCity},
		Stops:              []gameapi.CityID{co.FromCity, co.ToCity},
	}
}
func (w World) route(id string) (gameapi.Route, bool) {
	if co, ok := w.catalog.find(id); ok {
		return w.resolveRoute(corridorRoute(co)), true
	}
	for i := range w.catalog.network.Routes {
		if w.catalog.network.Routes[i].ID == id {
			return w.resolveRoute(w.catalog.network.Routes[i]), true
		}
	}
	return gameapi.Route{}, false
}

// resolveRoute fills in the length, timing and open state of a catalog route
// for this world, on private copies of its slices.
func (w World) resolveRoute(route gameapi.Route) gameapi.Route {
	route.Sections = append([]string{}, route.Sections...)
	route.Cities = append([]gameapi.CityID{}, route.Cities...)
	route.Stops = append([]gameapi.CityID{}, route.Stops...)
	route.KM, route.Minutes, route.Open = 0, 0, true
	route.ConnectorKM = 0
	for _, id := range route.Sections {
		co, _ := w.catalog.find(id)
		a, _ := w.alignment(co, "")
		if a.ID == "" {
			a = co.Alignments[0]
		}
		if co.ConstructionKind != "" {
			route.InfrastructureKind = "new_hsr"
		}
		route.KM += a.KM
		route.ConnectorKM += a.ConnectorKM
		route.Minutes += a.Minutes
		route.Open = route.Open && w.operational(co)
	}
	route.Minutes += (len(route.Stops) - 2) * w.catalog.network.DwellMinutes
	return route
}

// Routes resolves every catalog entry directly. Looking each one up by ID
// would rescan the route list once per route.
func (w World) Routes() []gameapi.Route {
	result := make([]gameapi.Route, 0, len(w.catalog.corridors)+len(w.catalog.network.Routes))
	for _, co := range w.catalog.corridors {
		r := w.resolveRoute(corridorRoute(co))
		w.suggest(&r)
		result = append(result, r)
	}
	for i := range w.catalog.network.Routes {
		pattern := w.catalog.network.Routes[i]
		if slices.Contains(w.catalog.network.SupersededRoutes, pattern.ID) && w.FleetCount(pattern.ID) == 0 &&
			!slices.ContainsFunc(w.services, func(s gameapi.Service) bool { return s.CorridorID == pattern.ID }) {
			continue
		}
		r := w.resolveRoute(pattern)
		w.suggest(&r)
		result = append(result, r)
	}
	return result
}
func sortedServices(in []gameapi.Service) []gameapi.Service {
	out := append([]gameapi.Service{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].CorridorID < out[j].CorridorID })
	return out
}
func withService(in []gameapi.Service, s gameapi.Service) []gameapi.Service {
	out := append([]gameapi.Service{}, in...)
	for i := range out {
		if out[i].CorridorID == s.CorridorID {
			out[i] = s
			return out
		}
	}
	return append(out, s)
}
func (w World) sectionLimit(id string) int {
	co, _ := w.catalog.find(id)
	if co.Existing {
		return w.catalog.network.PublicPaths
	}
	return w.catalog.network.PrivatePaths
}
func (c Catalog) resources(id string) []string {
	co, _ := c.find(id)
	if len(co.SharedSections) > 0 {
		return co.SharedSections
	}
	return []string{id}
}

// Private infrastructure and stations cannot be deliberately overbooked.
// Public capacity is provisional and fairly apportioned at monthly resolution.
func (w World) validateOwnedCapacity(plans []gameapi.Service, draftIDs ...string) error {
	id := ""
	if len(draftIDs) > 0 {
		id = draftIDs[0]
	}
	usage, err := w.ownedCapacityUsage(plans, id)
	if err != nil {
		return err
	}
	return capacityError(usage)
}

// Historical topology selection is independent of the catalog and economy epochs.
func (c Catalog) WithPrevious(previous Catalog) Catalog { c.previous = &previous; return c }
func (w World) TopologySince() int                      { return w.topologySince }
func (w World) UpgradeTopology(c Catalog) (World, error) {
	since := 0
	if c.connectivityPrior != nil {
		since = w.months
	}
	at := w.cutovers()
	at.Topology, at.Connectivity, at.Construction = w.months, since, since
	at.Regional, at.Connector = regionalCutover(c, w.months), connectorCutover(c, w.months)
	return Restore(
		c,
		w.revision,
		w.months,
		w.company,
		w.reports,
		w.projects,
		w.services,
		w.fleet,
		w.debt,
		w.pending,
		w.nextTrain,
		at,
	)
}
func (c Catalog) WithConnectivityPrior(prior Catalog) Catalog { c.connectivityPrior = &prior; return c }
func (w World) ConnectivitySince() int                        { return w.connectivitySince }
func (w World) UpgradeConnectivity(c Catalog) (World, error) {
	at := w.cutovers()
	at.Connectivity, at.Construction = w.months, w.months
	at.Regional, at.Connector = regionalCutover(c, w.months), connectorCutover(c, w.months)
	return Restore(
		c,
		w.revision,
		w.months,
		w.company,
		w.reports,
		w.projects,
		w.services,
		w.fleet,
		w.debt,
		w.pending,
		w.nextTrain,
		at,
	)
}
func (w World) atMonth(month int) World {
	if w.catalog.rhinePrior != nil && month < w.rhineSince {
		w.catalog = *w.catalog.rhinePrior
	}
	if w.catalog.connectorPrior != nil && month < w.connectorSince {
		w.catalog = *w.catalog.connectorPrior
	}
	if w.catalog.regionalPrior != nil && month < w.regionalSince {
		w.catalog = *w.catalog.regionalPrior
	}
	if w.catalog.constructionPrior != nil && month < w.constructionSince {
		w.catalog = *w.catalog.constructionPrior
	}
	if w.catalog.connectivityPrior != nil && month < w.connectivitySince {
		w.catalog = *w.catalog.connectivityPrior
	}
	if w.catalog.previous != nil && month < w.topologySince {
		w.catalog = *w.catalog.previous
	}
	if w.catalog.prior != nil && month < w.catalogSince {
		w.catalog = *w.catalog.prior
	}
	w.months = month
	return w
}

func (c Catalog) WithConstructionPrior(prior Catalog) Catalog { c.constructionPrior = &prior; return c }
func (w World) ConstructionSince() int                        { return w.constructionSince }

// The withdrawn conventional Moroccan links are not public HSR assets. Retire
// their unserviceable plans/trains on upgrade while replaying all past months
// with the exact archived catalog. Unrelated projects and fleets stay intact.
func (w World) UpgradeConstruction(c Catalog) (World, error) {
	probe := w
	probe.catalog = c
	plans := []gameapi.Service{}
	fleet := []gameapi.FleetLease{}
	for _, s := range w.services {
		if _, ok := probe.route(s.CorridorID); ok {
			plans = append(plans, s)
		}
	}
	for _, f := range w.fleet {
		if _, ok := probe.route(f.CorridorID); ok {
			fleet = append(fleet, f)
		}
	}
	at := w.cutovers()
	at.Construction = w.months
	at.Regional, at.Connector = regionalCutover(c, w.months), connectorCutover(c, w.months)
	return Restore(
		c,
		w.revision,
		w.months,
		w.company,
		w.reports,
		w.projects,
		plans,
		fleet,
		w.debt,
		w.pending,
		w.nextTrain,
		at,
	)
}

func (c Catalog) WithRegionalPrior(prior Catalog) Catalog { c.regionalPrior = &prior; return c }
func (w World) RegionalSince() int                        { return w.regionalSince }
func regionalCutover(c Catalog, month int) int {
	if c.regionalPrior != nil {
		return month
	}
	return 0
}

// New markets affect demand, so retain the exact pre-expansion catalog for past
// months. Existing company assets/plans and completed financial reports survive.
func (w World) UpgradeRegional(c Catalog) (World, error) {
	at := w.cutovers()
	at.Regional, at.Connector = w.months, connectorCutover(c, w.months)
	return Restore(
		c,
		w.revision,
		w.months,
		w.company,
		w.reports,
		w.projects,
		w.services,
		w.fleet,
		w.debt,
		w.pending,
		w.nextTrain,
		at,
	)
}

// Connector additions introduce junction markets and capacity decompositions.
// Replay past accounts against the exact prior catalog; keep accepted plans.
func (c Catalog) WithConnectorPrior(prior Catalog) Catalog { c.connectorPrior = &prior; return c }
func (w World) ConnectorSince() int                        { return w.connectorSince }
func connectorCutover(c Catalog, month int) int {
	if c.connectorPrior != nil {
		return month
	}
	return 0
}
func (c Catalog) WithRhinePrior(prior Catalog) Catalog { c.rhinePrior = &prior; return c }
func (w World) RhineSince() int                        { return w.rhineSince }
func (w World) UpgradeRhine(c Catalog) (World, error) {
	at := w.cutovers()
	at.Rhine = w.months
	return Restore(c, w.revision, w.months, w.company, w.reports, w.projects, w.services, w.fleet, w.debt, w.pending, w.nextTrain, at)
}

func (w World) UpgradeConnectors(c Catalog) (World, error) {
	at := w.cutovers()
	at.Connector = w.months
	return Restore(
		c,
		w.revision,
		w.months,
		w.company,
		w.reports,
		w.projects,
		w.services,
		w.fleet,
		w.debt,
		w.pending,
		w.nextTrain,
		at,
	)
}

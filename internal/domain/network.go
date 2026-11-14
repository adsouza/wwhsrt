package domain

import (
	"sort"
	"wwhsrt/pkg/gameapi"
)

func (w World) schedule(plans []gameapi.Service) map[string]int {
	requests := []capacityRequest{}
	remaining := map[string]int{}
	for _, plan := range sortedServices(plans) {
		r, ok := w.route(plan.CorridorID)
		if !ok || !r.Open || plan.Frequency <= 0 {
			continue
		}
		resources := []string{}
		used := map[string]bool{}
		for _, id := range r.Sections {
			for _, physical := range w.catalog.resources(id) {
				key := "track/" + physical
				if !used[key] {
					resources = append(resources, key)
					used[key] = true
				}
				remaining[key] = w.sectionLimit(physical)
			}
		}
		for _, city := range r.Stops {
			key := "station/" + string(city)
			resources = append(resources, key)
			remaining[key] = w.catalog.network.StationCalls / 2
		}
		requests = append(requests, capacityRequest{id: r.ID, need: plan.Frequency, legs: resources})
	}
	ration(requests, remaining)
	delivered := map[string]int{}
	for _, r := range requests {
		delivered[r.id] = r.served
	}
	return delivered
}

type marketRequest struct {
	market             int
	choices            []itinerary
	wanted             int
	primary, alternate int
	alternateChoice    int
}

func (w World) simulate(plans []gameapi.Service, month int) gameapi.NetworkReport {
	w.months = month
	result := gameapi.NetworkReport{
		Available: true,
		Month:     Calendar(month),
		Markets:   []gameapi.MarketRun{},
		Journeys:  []gameapi.JourneyFlow{},
		Services:  []gameapi.NetworkService{},
		Sections:  []gameapi.SectionUsage{},
		Stations:  []gameapi.StationActivity{},
	}
	d := days(result.Month)
	delivered := w.schedule(plans)
	graph := w.rides(plans, delivered)
	remaining := map[string]int{}
	serviceIndex := map[string]int{}
	legIndex := map[string][2]int{}
	sections := map[string]*gameapi.SectionUsage{}
	stations := map[gameapi.CityID]*gameapi.StationActivity{}
	for _, station := range w.Stations() {
		if station.Open {
			stations[station.CityID] = &gameapi.StationActivity{
				CityID:   station.CityID,
				Capacity: w.catalog.network.StationCalls,
			}
		}
	}
	for _, plan := range sortedServices(plans) {
		route, ok := w.route(plan.CorridorID)
		if !ok || !route.Open {
			continue
		}
		f := delivered[route.ID]
		s := gameapi.NetworkService{ID: route.ID, Requested: plan.Frequency, Delivered: f, Legs: []gameapi.LegLoad{}}
		usedSections := map[string]bool{}
		for i, id := range route.Sections {
			co, _ := w.catalog.find(id)
			a, _ := w.alignment(co, "")
			for _, physical := range w.catalog.resources(id) {
				if usedSections[physical] {
					continue
				}
				usedSections[physical] = true
				if sections[physical] == nil {
					sections[physical] = &gameapi.SectionUsage{CorridorID: physical, Capacity: w.sectionLimit(physical)}
				}
				sections[physical].Requested += plan.Frequency
				sections[physical].Delivered += f
			}
			for _, direction := range []int{1, -1} {
				from, to := route.Cities[i], route.Cities[i+1]
				if direction < 0 {
					from, to = to, from
				}
				leg := gameapi.LegLoad{
					CorridorID: id,
					From:       from,
					To:         to,
					KM:         a.KM,
					Minutes:    a.Minutes,
					Capacity:   f * trainSeats * d,
				}
				key := legKey(route.ID, from, to)
				remaining[key] = leg.Capacity
				legIndex[key] = [2]int{len(result.Services), len(s.Legs)}
				s.Legs = append(s.Legs, leg)
				result.SeatKM += int64(leg.Capacity) * int64(a.KM)
			}
			s.Running += gameapi.Money(2*f*a.KM*d) * w.runningRate()
			if co.Existing {
				s.Access += gameapi.Money(2*f*a.KM*d) * w.accessRate()
			}
		}
		for _, city := range route.Stops {
			if stations[city] != nil {
				stations[city].Calls += f * 2
			}
		}
		serviceIndex[route.ID] = len(result.Services)
		result.Services = append(result.Services, s)
	}
	cities := append([]gameapi.MarketCity{}, w.catalog.network.Cities...)
	sort.Slice(cities, func(i, j int) bool { return cities[i].ID < cities[j].ID })
	requests := []marketRequest{}
	primary := []capacityRequest{}
	for _, from := range cities {
		// An origin without a company departure cannot offer any rail itinerary.
		// Preserve its entire finite budget without allocating thousands of empty ODs.
		if w.catalog.network.Sparse && len(graph[from.ID]) == 0 {
			demand := w.dailyTrips(from) * d
			result.Demand += demand
			result.UnservedDemand += demand
			continue
		}
		destinations := []gameapi.MarketCity{}
		weights := []weighted{}
		for _, to := range cities {
			if to.ID == from.ID {
				continue
			}
			destinations = append(destinations, to)
			weights = append(weights, weighted{string(to.ID), w.destinationWeight(from, to)})
		}
		for _, class := range []string{"business", "leisure"} {
			itineraries := w.itinerariesFrom(from.ID, class, graph)
			budget := w.dailyTrips(from) * d * 30 / 100
			if class == "leisure" {
				budget = w.dailyTrips(from)*d - budget
			}
			demands := shares(budget, weights)
			for i, to := range destinations {
				choices := itineraries[to.ID]
				if w.catalog.network.Sparse && len(choices) == 0 {
					result.UnservedDemand += demands[i]
					result.Demand += demands[i]
					continue
				}
				offers := w.outside(from, to, class)
				m := gameapi.MarketRun{
					From:   from.ID,
					To:     to.ID,
					Class:  class,
					Demand: demands[i],
					Offers: append([]gameapi.ModeOffer{}, offers...),
				}
				if len(choices) > 0 {
					it := choices[0]
					m.RailFare, m.RailMinutes = it.fare, it.minutes
					offers = append(
						offers,
						gameapi.ModeOffer{Mode: "rail", Fare: it.fare, Minutes: it.minutes, Generalized: it.cost},
					)
					m.Offers = append([]gameapi.ModeOffer{}, offers...)
				}
				counts := shares(m.Demand, offerWeights(offers))
				for i, o := range offers {
					switch o.Mode {
					case "rail":
						m.Requested = counts[i]
					case "road":
						m.Road = counts[i]
					case "air":
						m.Air = counts[i]
					case "conventional":
						m.Conventional = counts[i]
					case "no_trip":
						m.NoTrip = counts[i]
					}
				}
				index := len(result.Markets)
				result.Markets = append(result.Markets, m)
				result.Demand += m.Demand
				result.Requested += m.Requested
				if m.Requested > 0 {
					requests = append(
						requests,
						marketRequest{market: index, choices: choices, wanted: m.Requested, alternateChoice: -1},
					)
					primary = append(
						primary,
						capacityRequest{
							id:   requestID(from.ID, to.ID, class),
							need: m.Requested,
							legs: itineraryLegs(choices[0]),
						},
					)
				}
			}
		}
	}
	ration(primary, remaining)
	alternates := []capacityRequest{}
	altIndex := []int{}
	for i := range requests {
		r := &requests[i]
		r.primary = primary[i].served
		if r.primary == r.wanted {
			continue
		}
		for choice := 1; choice < len(r.choices); choice++ {
			legs := itineraryLegs(r.choices[choice])
			possible := true
			for _, leg := range legs {
				if remaining[leg] <= 0 {
					possible = false
					break
				}
			}
			if possible {
				r.alternateChoice = choice
				alternates = append(
					alternates,
					capacityRequest{id: primary[i].id, need: r.wanted - r.primary, legs: legs},
				)
				altIndex = append(altIndex, i)
				break
			}
		}
	}
	ration(alternates, remaining)
	for i, alt := range alternates {
		requests[altIndex[i]].alternate = alt.served
	}
	addFlow := func(m gameapi.MarketRun, it itinerary, count int, alternate bool) {
		if count == 0 {
			return
		}
		flow := gameapi.JourneyFlow{
			From:       m.From,
			To:         m.To,
			Class:      m.Class,
			Passengers: count,
			Fare:       it.fare,
			Minutes:    it.minutes,
			Services:   []string{},
			Transfers:  []gameapi.CityID{},
			Alternate:  alternate,
		}
		for i, r := range it.rides {
			flow.Services = append(flow.Services, r.service)
			s := &result.Services[serviceIndex[r.service]]
			s.Boardings += count
			s.Revenue += gameapi.Money(count) * r.fare
			s.Handling += gameapi.Money(count) * stationPerJourney
			stations[r.from].Boardings += count
			stations[r.to].Alightings += count
			if i > 0 {
				flow.Transfers = append(flow.Transfers, r.from)
				stations[r.from].Transfers += count
				result.Transfers += count
			}
			for _, key := range r.legs {
				indices := legIndex[key]
				leg := &result.Services[indices[0]].Legs[indices[1]]
				leg.Passengers += count
				result.PassengerKM += int64(count) * int64(leg.KM)
			}
		}
		result.Revenue += gameapi.Money(count) * it.fare
		result.Journeys = append(result.Journeys, flow)
	}
	for _, request := range requests {
		m := &result.Markets[request.market]
		m.Carried = request.primary + request.alternate
		m.AlternateServed = request.alternate
		m.Rejected = m.Requested - m.Carried
		result.Carried += m.Carried
		result.AlternateServed += m.AlternateServed
		result.Rejected += m.Rejected
		addFlow(*m, request.choices[0], request.primary, false)
		if request.alternate > 0 {
			addFlow(*m, request.choices[request.alternateChoice], request.alternate, true)
		}
		outside := []gameapi.ModeOffer{}
		for _, o := range m.Offers {
			if o.Mode != "rail" {
				outside = append(outside, o)
			}
		}
		counts := shares(m.Rejected, offerWeights(outside))
		for i, o := range outside {
			switch o.Mode {
			case "road":
				m.Road += counts[i]
			case "air":
				m.Air += counts[i]
			case "conventional":
				m.Conventional += counts[i]
			case "no_trip":
				m.NoTrip += counts[i]
			}
		}
	}
	for _, co := range w.catalog.corridors {
		if usage := sections[co.ID]; usage != nil {
			result.Sections = append(result.Sections, *usage)
		}
	}
	for _, city := range cities {
		if activity := stations[city.ID]; activity != nil && (!w.catalog.network.Sparse || activity.Calls > 0) {
			result.Stations = append(result.Stations, *activity)
		}
	}
	return result
}

func (w World) NetworkSince() int { return w.networkSince }
func (w World) NetworkReport() gameapi.NetworkReport {
	if len(w.reports) == 0 || len(w.reports) <= w.networkSince {
		return gameapi.NetworkReport{
			Markets:  []gameapi.MarketRun{},
			Journeys: []gameapi.JourneyFlow{},
			Services: []gameapi.NetworkService{},
			Sections: []gameapi.SectionUsage{},
			Stations: []gameapi.StationActivity{},
		}
	}
	plans := []gameapi.Service{}
	for _, run := range w.reports[len(w.reports)-1].Services {
		frequency := run.RequestedFrequency
		if frequency == 0 {
			frequency = run.Frequency
		}
		plans = append(plans, gameapi.Service{CorridorID: run.CorridorID, Frequency: frequency, Fare: run.Fare})
	}
	month := w.months - 1
	w = w.atMonth(month)
	return w.simulate(plans, month)
}

func (w World) networkForecast(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
	r, ok := w.route(q.CorridorID)
	if !ok {
		return gameapi.Forecast{}, errCode("no_route")
	}
	if q.Frequency < 0 || q.Frequency > MaxFrequency || q.Fare < MinFare || q.Fare > MaxFare {
		return gameapi.Forecast{}, errCode("invalid_service")
	}
	result := gameapi.Forecast{
		FleetNeeded:    required(q.Frequency, r.Minutes),
		FleetAvailable: w.FleetCount(r.ID),
		Open:           r.Open,
		KM:             r.KM,
		Minutes:        r.Minutes,
	}
	base := w.simulate(w.services, w.months)
	plans := append([]gameapi.Service{}, w.services...)
	if q.ReplaceServiceID != "" {
		if !w.validExtension(r.ID, q.ReplaceServiceID) {
			return result, errCode("invalid_extension")
		}
		result.FleetAvailable += w.FleetCount(q.ReplaceServiceID)
		for i := range plans {
			if plans[i].CorridorID == q.ReplaceServiceID {
				plans[i].Frequency = 0
			}
		}
	}
	hypothesis := w
	if !r.Open {
		// Independent construction preview: include the proposed line/stations
		// only in the counterfactual, never mutate the accepted world.
		co, found := w.catalog.find(q.CorridorID)
		if !found {
			return result, errCode("line_not_open")
		}
		a, found := alternative(co, q.AlignmentID)
		if !found && q.AlignmentID == "" {
			a = co.Alignments[0]
			found = true
		}
		if !found {
			return result, errCode("no_route")
		}
		hypothesis.projects = append(
			w.Projects(),
			gameapi.Project{
				CorridorID:  co.ID,
				AlignmentID: a.ID,
				Owner:       w.company.ID,
				Started:     w.months,
				Ready:       w.months,
			},
		)
		r, _ = hypothesis.route(r.ID)
		result.KM, result.Minutes = r.KM, r.Minutes
		result.FleetNeeded = required(q.Frequency, r.Minutes)
	}
	plans = withService(plans, gameapi.Service{CorridorID: q.CorridorID, Frequency: q.Frequency, Fare: q.Fare})
	usage, err := hypothesis.ownedCapacityUsage(plans, q.CorridorID)
	if err != nil {
		return result, err
	}
	result.CapacityUsage = []gameapi.CapacityUsage{}
	for _, c := range usage {
		if c.Draft {
			result.CapacityUsage = append(result.CapacityUsage, c)
		}
	}
	if err := capacityError(usage); err != nil {
		return result, err
	}
	next := hypothesis.simulate(plans, w.months)
	var nextVariable, baseVariable, baseOther, nextOther gameapi.Money
	for _, s := range base.Services {
		baseVariable += s.Running + s.Access + s.Handling
		if s.ID != r.ID {
			baseOther += s.Revenue
		}
	}
	for _, s := range next.Services {
		nextVariable += s.Running + s.Access + s.Handling
		if s.ID != r.ID {
			nextOther += s.Revenue
			continue
		}
		result.Passengers = s.Boardings
		result.Revenue = s.Revenue
		result.DeliveredFrequency = s.Delivered
		result.RunningCosts, result.AccessCosts, result.HandlingCosts = s.Running, s.Access, s.Handling
		result.Costs = s.Running + s.Access + s.Handling
		// A through service can reuse seats after intermediate alightings;
		// capacity here counts leg-seats, not a fictitious end-to-end seat pool.
		for _, leg := range s.Legs {
			result.Capacity += leg.Capacity
		}
	}
	result.LeaseCosts = gameapi.Money(max(result.FleetNeeded, result.FleetAvailable)) * w.leaseMonthly()
	result.Costs += result.LeaseCosts
	for _, id := range r.Sections {
		co, _ := w.catalog.find(id)
		if !co.Existing {
			a, _ := hypothesis.alignment(co, "")
			result.MaintenanceCosts += a.Cost / maintenanceDivisor
			result.Costs += a.Cost / maintenanceDivisor
		}
	}
	result.Contribution = result.Revenue - result.Costs
	result.CompanyPassengers = next.Carried
	result.CompanyRevenue = next.Revenue
	additional := max(0, result.FleetNeeded-result.FleetAvailable)
	result.AdditionalLeases = additional
	result.CompanyChange = next.Revenue - base.Revenue - (nextVariable - baseVariable) - gameapi.Money(
		additional,
	)*w.leaseMonthly()
	if !result.Open {
		for _, id := range r.Sections {
			co, _ := w.catalog.find(id)
			if !co.Existing {
				a, _ := hypothesis.alignment(co, "")
				result.CompanyChange -= a.Cost / maintenanceDivisor
			}
		}
	}
	breakEven(&result, q.Fare, q.Frequency, len(r.Sections) == 1)
	result.FeederRevenue = nextOther - baseOther
	return result, nil
}

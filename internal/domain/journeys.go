package domain

import (
	"sort"
	"strconv"
	"strings"
	"wwhsrt/pkg/gameapi"
)

type ride struct {
	service       string
	from, to      gameapi.CityID
	legs          []string
	cities        []gameapi.CityID
	fare          gameapi.Money
	minutes, wait int
}
type itinerary struct {
	id      string
	rides   []ride
	cost    gameapi.Money
	fare    gameapi.Money
	minutes int
}

func legKey(service string, from, to gameapi.CityID) string {
	return service + "/" + string(from) + "/" + string(to)
}
func containsCity(rows []gameapi.CityID, city gameapi.CityID) bool {
	for _, id := range rows {
		if id == city {
			return true
		}
	}
	return false
}

func (w World) rides(plans []gameapi.Service, delivered map[string]int) map[gameapi.CityID][]ride {
	out := map[gameapi.CityID][]ride{}
	for _, plan := range sortedServices(plans) {
		frequency := delivered[plan.CorridorID]
		if frequency == 0 {
			continue
		}
		route, _ := w.route(plan.CorridorID)
		fares := map[string]gameapi.Money{}
		km := 0
		for _, id := range route.Sections {
			co, _ := w.catalog.find(id)
			a, _ := w.alignment(co, "")
			next := km + a.KM
			fares[id] = plan.Fare*gameapi.Money(next)/gameapi.Money(route.KM) - plan.Fare*gameapi.Money(km)/gameapi.Money(route.KM)
			km = next
		}
		for _, direction := range []int{1, -1} {
			cities := append([]gameapi.CityID{}, route.Cities...)
			sections := append([]string{}, route.Sections...)
			if direction < 0 {
				for i, j := 0, len(cities)-1; i < j; i, j = i+1, j-1 {
					cities[i], cities[j] = cities[j], cities[i]
				}
				for i, j := 0, len(sections)-1; i < j; i, j = i+1, j-1 {
					sections[i], sections[j] = sections[j], sections[i]
				}
			}
			cumulative := []gameapi.Money{0}
			timings := []int{0}
			for i, id := range sections {
				co, _ := w.catalog.find(id)
				a, _ := w.alignment(co, "")
				cumulative = append(cumulative, cumulative[i]+fares[id])
				timings = append(timings, timings[i]+a.Minutes)
			}
			for start, from := range cities {
				if !containsCity(route.Stops, from) {
					continue
				}
				for end := start + 1; end < len(cities); end++ {
					to := cities[end]
					if !containsCity(route.Stops, to) {
						continue
					}
					legs := []string{}
					dwell := 0
					for i := start; i < end; i++ {
						legs = append(legs, legKey(route.ID, cities[i], cities[i+1]))
						if i > start && containsCity(route.Stops, cities[i]) {
							dwell += w.catalog.network.DwellMinutes
						}
					}
					// Cumulative fare rounding makes every full ride equal the
					// displayed terminal fare, in either direction, to the cent.
					fare := cumulative[end] - cumulative[start]
					out[from] = append(out[from], ride{route.ID, from, to, legs, append([]gameapi.CityID{}, cities[start+1:end+1]...), fare, timings[end] - timings[start] + dwell, w.serviceWait(frequency)})
				}
			}
		}
	}
	return out
}

// Enumerate once per origin/class rather than once per possible destination.
// Keep exactly the same three best acyclic, at-most-three-service itineraries.
func (w World) itinerariesFrom(from gameapi.CityID, class string, graph map[gameapi.CityID][]ride) map[gameapi.CityID][]itinerary {
	vot := gameapi.Money(15)
	if class == "business" {
		vot = 50
	}
	result := map[gameapi.CityID][]itinerary{}
	var visit func(gameapi.CityID, []ride, map[gameapi.CityID]bool, map[string]bool)
	visit = func(at gameapi.CityID, path []ride, visited map[gameapi.CityID]bool, used map[string]bool) {
		if len(path) > 0 {
			fare := gameapi.Money(0)
			minutes := 45 + (len(path)-1)*w.catalog.network.TransferMinutes
			ids := []string{}
			for _, r := range path {
				fare += r.fare
				minutes += r.minutes + r.wait
				ids = append(ids, r.service+":"+string(r.from)+">"+string(r.to))
			}
			rows := append(result[at], itinerary{strings.Join(ids, "|"), append([]ride{}, path...), fare + vot*gameapi.Money(minutes), fare, minutes})
			sort.Slice(rows, func(i, j int) bool {
				if rows[i].cost != rows[j].cost {
					return rows[i].cost < rows[j].cost
				}
				return rows[i].id < rows[j].id
			})
			if len(rows) > 3 {
				rows = rows[:3]
			}
			result[at] = rows
		}
		if len(path) == 3 {
			return
		}
		for _, r := range graph[at] {
			if used[r.service] {
				continue
			}
			loop := false
			for _, city := range r.cities {
				if visited[city] {
					loop = true
					break
				}
			}
			if loop {
				continue
			}
			for _, city := range r.cities {
				visited[city] = true
			}
			used[r.service] = true
			visit(r.to, append(path, r), visited, used)
			delete(used, r.service)
			for _, city := range r.cities {
				delete(visited, city)
			}
		}
	}
	visit(from, nil, map[gameapi.CityID]bool{from: true}, map[string]bool{})
	return result
}
func (w World) outside(from, to gameapi.MarketCity, class string) []gameapi.ModeOffer {
	km := w.catalog.distance(from.ID, to.ID)
	vot := gameapi.Money(15)
	noTrip := gameapi.Money(18000)
	if class == "business" {
		vot = 50
		noTrip = 30000
	}
	out := []gameapi.ModeOffer{}
	add := func(mode string, fare gameapi.Money, minutes int) {
		out = append(out, gameapi.ModeOffer{Mode: mode, Fare: fare, Minutes: minutes, Generalized: fare + vot*gameapi.Money(minutes)})
	}
	if from.RoadRegion == to.RoadRegion {
		add("road", 800+gameapi.Money(km)*12, 30+(km*60+79)/80)
	}
	if km >= 250 {
		add("air", 7000+gameapi.Money(km)*9, 160+(km*60+699)/700)
	}
	for _, p := range w.catalog.network.ConventionalPairs {
		if marketKey(p[0], p[1]) == marketKey(from.ID, to.ID) {
			add("conventional", 1200+gameapi.Money(km)*14, 60+(km*60+89)/90)
			break
		}
	}
	out = append(out, gameapi.ModeOffer{Mode: "no_trip", Generalized: noTrip})
	return out
}
func offerWeights(offers []gameapi.ModeOffer) []weighted {
	result := make([]weighted, len(offers))
	for i, o := range offers {
		cost := max(1000, int64(o.Generalized))
		result[i] = weighted{o.Mode, max(1, 1_000_000_000_000/(cost*cost))}
	}
	return result
}
func requestID(from, to gameapi.CityID, class string) string {
	return string(from) + "/" + string(to) + "/" + class
}
func itineraryLegs(it itinerary) []string {
	out := []string{}
	for _, r := range it.rides {
		out = append(out, r.legs...)
	}
	return out
}
func departureID(service string, direction, departure int) string {
	return service + "/" + strconv.Itoa(direction) + "/" + strconv.Itoa(departure)
}

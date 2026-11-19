package ui

import "wwhsrt/pkg/gameapi"

// ProspectMap describes the line advertised on the construction map. A tunnel
// contract can open a longer passenger route over existing approach tracks.
type ProspectMap struct {
	Paths             [][]gameapi.GeoPoint
	Stops             []gameapi.CityID
	AlternateStations []gameapi.City
}

func MapForProspect(f gameapi.Frame, co gameapi.Corridor, alignment gameapi.Alignment) ProspectMap {
	works := ProspectMap{
		Paths: [][]gameapi.GeoPoint{alignment.Path},
		Stops: []gameapi.CityID{co.FromCity, co.ToCity},
	}
	works.addTerminalStations(f, co, alignment.Path)
	if co.ConstructionKind != "base_tunnel" || co.OpeningService == "" {
		return works
	}
	route, ok := ServiceRoute(f, co.OpeningService)
	if !ok || len(route.Stops) < 2 || len(route.Sections) == 0 {
		return works
	}
	line := ProspectMap{Stops: route.Stops}
	for _, id := range route.Sections {
		if id == co.ID {
			line.Paths = append(line.Paths, alignment.Path)
			continue
		}
		var approach []gameapi.GeoPoint
		for _, section := range f.Corridors {
			if section.ID == id && section.Existing && len(section.Alignments) > 0 {
				approach = section.Alignments[0].Path
				line.addTerminalStations(f, section, approach)
				break
			}
		}
		if len(approach) < 2 {
			return works
		}
		line.Paths = append(line.Paths, approach)
	}
	return line
}

// Keep named construction terminals visible even after the contract is built.
func ConstructionTerminals(f gameapi.Frame) map[gameapi.CityID]gameapi.City {
	terminals := map[gameapi.CityID]gameapi.City{}
	for _, co := range f.Corridors {
		if co.Existing || co.Retired || len(co.Alignments) == 0 || len(co.StationNames) != 2 {
			continue
		}
		var line ProspectMap
		line.addTerminalStations(f, co, co.Alignments[0].Path)
		for _, station := range line.AlternateStations {
			terminals[station.ID] = station
		}
	}
	return terminals
}

// Some works and approaches end at a different station from the city marker.
func (line *ProspectMap) addTerminalStations(f gameapi.Frame, section gameapi.Corridor, path []gameapi.GeoPoint) {
	if len(path) < 2 || len(section.StationNames) != 2 {
		return
	}
	for _, city := range f.Cities {
		var name string
		var point gameapi.GeoPoint
		switch city.ID {
		case section.FromCity:
			name, point = section.StationNames[0], path[0]
		case section.ToCity:
			name, point = section.StationNames[1], path[len(path)-1]
		default:
			continue
		}
		lat, lon := city.StationLat, city.StationLon
		if city.StationName == "" {
			lat, lon = city.Lat, city.Lon
		}
		if name != "" && (lat != point.Lat || lon != point.Lon) {
			city.StationName, city.StationLat, city.StationLon = name, point.Lat, point.Lon
			line.AlternateStations = append(line.AlternateStations, city)
		}
	}
}

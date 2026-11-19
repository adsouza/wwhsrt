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
				line.addApproachStations(f, section, approach)
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

// Some approaches use a different passenger station from the city's primary map pin.
func (line *ProspectMap) addApproachStations(f gameapi.Frame, section gameapi.Corridor, path []gameapi.GeoPoint) {
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
		if name != "" && city.StationName != name {
			city.StationName, city.StationLat, city.StationLon = name, point.Lat, point.Lon
			line.AlternateStations = append(line.AlternateStations, city)
		}
	}
}

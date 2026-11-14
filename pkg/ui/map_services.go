package ui

import "wwhsrt/pkg/gameapi"

func routeUsesSections(r gameapi.Route, sections []string) bool {
	for _, id := range sections {
		for _, section := range r.Sections {
			if id == section {
				return true
			}
		}
	}
	return false
}

// MapServiceRoutes preserves the catalog's stable order. A clicked section can
// support several direct/through patterns; the named selector disambiguates them.
func MapServiceRoutes(f gameapi.Frame, s State) []gameapi.Route {
	var out []gameapi.Route
	for _, r := range f.Routes {
		if r.Open && (len(s.ServiceMapSections) == 0 || routeUsesSections(r, s.ServiceMapSections)) {
			out = append(out, r)
		}
	}
	return out
}

func SelectMapService(s *State, f gameapi.Frame, sections []string) bool {
	valid := []string{}
	for _, id := range sections {
		for _, c := range f.Corridors {
			if c.ID == id && Open(f, id) {
				valid = append(valid, id)
				break
			}
		}
	}
	next := *s
	next.ServiceMapSections = valid
	if len(valid) == 0 {
		return false
	}
	choices := MapServiceRoutes(f, next)
	if len(choices) == 0 {
		return false
	}
	id := choices[0].ID
	// Prefer a direct pattern for the nearest clicked section.
	for _, section := range valid {
		found := false
		for _, r := range choices {
			if r.ID == section {
				id, found = r.ID, true
				break
			}
		}
		if found {
			break
		}
	}
	// Reclicking the current route preserves its pending frequency/fare draft.
	for _, r := range choices {
		if r.ID == s.ServiceCorridor {
			id = r.ID
			break
		}
	}
	if id != s.ServiceCorridor {
		DraftService(&next, f, id)
	}
	next.View, next.ServicePane = "services", "plan"
	next.Selected, next.SourceStation = "", ""
	next.ServiceQuery = ""
	next.ReviewMap, next.Expanded = true, false
	*s = next
	return true
}

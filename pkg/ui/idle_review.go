package ui

import "wwhsrt/pkg/gameapi"

// Physical warnings highlight the idle corridor's committed alignment. Fleet
// warnings highlight the service's route with a distinct style: shared track
// can still carry other services even when this company's trains are idle.
type IdleHighlight struct {
	ReviewID, CorridorID string
	FromCity, ToCity     gameapi.CityID
	Path                 []gameapi.GeoPoint
	FleetOnly            bool
}

func IdleFleetOnly(line gameapi.IdleLine) bool { return line.Maintenance == 0 && line.Leases > 0 }
func IdleHighlights(f gameapi.Frame) []IdleHighlight {
	out := []IdleHighlight{}
	for _, line := range f.IdleLines {
		sections := []string{line.CorridorID}
		if IdleFleetOnly(line) {
			if route, ok := ServiceRoute(f, line.CorridorID); ok {
				sections = route.Sections
			}
		}
		for _, id := range sections {
			for _, co := range f.Corridors {
				if co.ID != id {
					continue
				}
				alignment := ""
				if co.Existing && len(co.Alignments) > 0 {
					alignment = co.Alignments[0].ID
				}
				for _, p := range f.Projects {
					if p.CorridorID == co.ID && p.Ready <= f.MonthsResolved {
						alignment = p.AlignmentID
					}
				}
				for _, a := range co.Alignments {
					if a.ID == alignment && len(a.Path) > 1 {
						out = append(out, IdleHighlight{line.CorridorID, co.ID, co.FromCity, co.ToCity, a.Path, IdleFleetOnly(line)})
						break
					}
				}
			}
		}
	}
	return out
}
func ReviewIdle(s *State, f gameapi.Frame, id string) bool {
	if len(f.IdleLines) == 0 {
		return false
	}
	if id == "" {
		id = f.IdleLines[0].CorridorID
	}
	found := false
	for _, line := range f.IdleLines {
		if line.CorridorID == id {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	DraftService(s, f, id)
	s.View = "services"
	s.ServicePane = "plan"
	s.ReviewMap = true
	SetDisclosure(s, "idle", true)
	s.Expanded = false
	s.Help.Open = false
	s.Help.ReturnExpanded = false
	return true
}

func IdleName(f gameapi.Frame, locale string, line gameapi.IdleLine) string {
	kind := "idleLineLabel"
	name := Text(locale, string(line.FromCity)) + " – " + Text(locale, string(line.ToCity))
	if IdleFleetOnly(line) {
		kind = "idleFleetLabel"
		if route, ok := ServiceRoute(f, line.CorridorID); ok {
			name = RouteName(route, locale)
		}
	}
	return Text(locale, kind) + ": " + name
}

package ui

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"wwhsrt/pkg/gameapi"
)

// RouteChoiceLimit caps the service route dropdown. The selected route is
// listed in addition, so a player never loses the route being edited.
const RouteChoiceLimit = 25

// Box is an axis-aligned rectangle in the map's projected coordinates.
type Box struct{ MinX, MinY, MaxX, MaxY float64 }

// Contains reports whether o lies entirely inside b, edges included.
func (b Box) Contains(o Box) bool {
	return o.MinX >= b.MinX && o.MinY >= b.MinY && o.MaxX <= b.MaxX && o.MaxY <= b.MaxY
}

// Span is the length of the box's diagonal.
func (b Box) Span() float64 { return math.Hypot(b.MaxX-b.MinX, b.MaxY-b.MinY) }

// RouteChoices is what the service route dropdown offers.
type RouteChoices struct {
	Routes   []gameapi.Route // ranked by span; the selection comes last when it was added
	Matching int             // routes that passed the filter, before the cap
	InView   bool            // the view rectangle was the filter that applied
}

// ServiceRouteChoices narrows the open routes to a short, ranked list. A track
// tap and a search look across every route; otherwise a route must fit entirely
// inside view. A nil view lets every open route compete.
func ServiceRouteChoices(f gameapi.Frame, s State, boxes map[string]Box, view *Box) RouteChoices {
	words := strings.Fields(strings.ToLower(s.ServiceQuery))
	out := RouteChoices{InView: view != nil && len(words) == 0 && len(s.ServiceMapSections) == 0}
	kept := []gameapi.Route{}
	var selected *gameapi.Route
	for i, r := range f.Routes {
		if !r.Open {
			continue
		}
		if r.ID == s.ServiceCorridor {
			selected = &f.Routes[i]
		}
		if len(s.ServiceMapSections) > 0 && !routeUsesSections(r, s.ServiceMapSections) {
			continue
		}
		if len(words) > 0 {
			name := strings.ToLower(RouteName(r, s.Locale))
			matches := true
			for _, word := range words {
				matches = matches && strings.Contains(name, word)
			}
			if !matches {
				continue
			}
		}
		if out.InView {
			if box, ok := boxes[r.ID]; !ok || !view.Contains(box) {
				continue
			}
		}
		kept = append(kept, r)
	}
	out.Matching = len(kept)
	sort.SliceStable(kept, func(i, j int) bool {
		a, b := boxes[kept[i].ID].Span(), boxes[kept[j].ID].Span()
		if a != b {
			return a > b
		}
		return kept[i].ID < kept[j].ID
	})
	if len(kept) > RouteChoiceLimit {
		kept = kept[:RouteChoiceLimit]
	}
	if selected != nil {
		listed := false
		for _, r := range kept {
			listed = listed || r.ID == selected.ID
		}
		if !listed {
			kept = append(kept, *selected)
		}
	}
	out.Routes = kept
	return out
}

// RouteChoiceHint explains a list that was cut short or came up empty. With no
// map on screen the list follows the chosen region, so the advice changes.
func RouteChoiceHint(locale string, c RouteChoices, mapShown bool) string {
	switch {
	case c.Matching > RouteChoiceLimit && !c.InView:
		return fmt.Sprintf(Text(locale, "routeChoicesMoreSearch"), RouteChoiceLimit, c.Matching)
	case c.Matching > RouteChoiceLimit && mapShown:
		return fmt.Sprintf(Text(locale, "routeChoicesMore"), RouteChoiceLimit, c.Matching)
	case c.Matching > RouteChoiceLimit:
		return fmt.Sprintf(Text(locale, "routeChoicesMoreRegion"), RouteChoiceLimit, c.Matching)
	case c.InView && c.Matching == 0 && mapShown:
		return Text(locale, "routeChoicesNone")
	case c.InView && c.Matching == 0:
		return Text(locale, "routeChoicesNoneRegion")
	case c.Matching == 0:
		return Text(locale, "routeChoicesNoneSearch")
	}
	return ""
}

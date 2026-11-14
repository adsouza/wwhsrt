//go:build !js

package app

import (
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

func growBox(b ui.Box, p mapview.Point, first bool) ui.Box {
	if first {
		return ui.Box{MinX: p.X, MinY: p.Y, MaxX: p.X, MaxY: p.Y}
	}
	return ui.Box{
		MinX: min(b.MinX, p.X), MinY: min(b.MinY, p.Y),
		MaxX: max(b.MaxX, p.X), MaxY: max(b.MaxY, p.Y),
	}
}

// routeBoxes projects every route's track once. A built line uses the
// alignment its project chose, as the map and route focus do.
func routeBoxes(f gameapi.Frame) map[string]ui.Box {
	chosen := map[string]string{}
	for _, p := range f.Projects {
		chosen[p.CorridorID] = p.AlignmentID
	}
	sections := map[string]ui.Box{}
	for _, co := range f.Corridors {
		if len(co.Alignments) == 0 {
			continue
		}
		a := co.Alignments[0]
		for _, candidate := range co.Alignments {
			if candidate.ID == chosen[co.ID] {
				a = candidate
			}
		}
		var box ui.Box
		for i, p := range a.Path {
			box = growBox(box, mapview.Project(p.Lat, p.Lon), i == 0)
		}
		if len(a.Path) > 0 {
			sections[co.ID] = box
		}
	}
	out := make(map[string]ui.Box, len(f.Routes))
	for _, r := range f.Routes {
		var box ui.Box
		complete := len(r.Sections) > 0
		for i, id := range r.Sections {
			part, ok := sections[id]
			if !ok {
				complete = false
				break
			}
			box = growBox(box, mapview.Point{X: part.MinX, Y: part.MinY}, i == 0)
			box = growBox(box, mapview.Point{X: part.MaxX, Y: part.MaxY}, false)
		}
		if complete {
			out[r.ID] = box
		}
	}
	return out
}

// viewBox is the rectangle of the map the camera currently shows.
func viewBox(c mapview.Camera) ui.Box {
	box := growBox(ui.Box{}, c.World(mapview.Point{}), true)
	return growBox(box, c.World(mapview.Point{X: c.Width, Y: c.Height}), false)
}

// regionFrame is the area the map shows when it frames a region, at a
// standard window shape so the hidden map's real size does not matter. A
// region is defined by a few points, and the projection can make their own
// bounding box far narrower than the region looks on screen.
func (g *Game) regionFrame(id string, portrait bool) ui.Box {
	width, height := 1200.0, 800.0
	if portrait {
		width, height = 600, 900
	}
	if id != "" && id != "east-asia" {
		return viewBox(mapview.RegionCamera(width, height, id, portrait))
	}
	return viewBox(g.Map.Camera(width, height, portrait))
}

// boxesFor reuses the projected boxes until the frame's revision changes.
func (h *host) boxesFor(f gameapi.Frame) map[string]ui.Box {
	if !h.routeBoxReady || h.routeBoxRevision != f.Revision {
		h.routeBoxCache, h.routeBoxRevision, h.routeBoxReady = routeBoxes(f), f.Revision, true
	}
	return h.routeBoxCache
}

// regionPicker is the region dropdown shared by the Map panel and, when no
// map is on screen, the Services panel. offer filters the regions listed.
func (h *host) regionPicker(g *Game, label nativeLabel, offer func(id string) bool) {
	label(ui.Text(g.State.Locale, "region"), 13)
	regions := []choice{}
	for _, r := range mapview.VisibleRegions(g.State.Portrait) {
		if r.ID == "world" && g.State.Portrait || !offer(r.ID) {
			continue
		}
		name := r.Name
		if g.State.Locale == "zh-Hans" {
			name = r.Chinese
		}
		regions = append(regions, choice{r.ID, name})
	}
	h.selectChoice(g, currentRegion(g), regions, func(id string) {
		g.State.Region = id
		g.Camera = g.cameraForRegion(id, g.State.Portrait)
		g.dirty = true
		h.persist(g)
	})
}

func currentRegion(g *Game) string {
	if g.State.Region == "" {
		return "east-asia"
	}
	return g.State.Region
}

// serviceRegionPicker offers the regions that have at least one open route
// fitting their frame, and always the current region.
func (h *host) serviceRegionPicker(g *Game, label nativeLabel) {
	boxes := h.boxesFor(g.Frame)
	h.regionPicker(g, label, func(id string) bool {
		if id == currentRegion(g) {
			return true
		}
		frame := g.regionFrame(id, g.State.Portrait)
		for _, r := range g.Frame.Routes {
			if box, ok := boxes[r.ID]; ok && r.Open && frame.Contains(box) {
				return true
			}
		}
		return false
	})
}

// serviceRouteChoices picks the routes the Services dropdown offers: those in
// the map view, or in the chosen region when no map is on screen.
func (h *host) serviceRouteChoices(g *Game) (ui.RouteChoices, bool) {
	mapShown := !h.workspace.Map.Empty()
	view := viewBox(g.Camera)
	if !mapShown {
		view = g.regionFrame(g.State.Region, g.State.Portrait)
	}
	h.routeChoices = ui.ServiceRouteChoices(g.Frame, g.State, h.boxesFor(g.Frame), &view)
	h.routeView, h.routeViewUsed = view, mapShown
	return h.routeChoices, mapShown
}

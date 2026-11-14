//go:build !js

package app

import (
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

// regionFrameMargin widens a region's defining points on every side, as the
// map does when it frames the region.
const regionFrameMargin = 0.1

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

// regionFrame is the area a region camera is built to show, independent of
// the window: its defining points plus a margin.
func regionFrame(c mapview.Camera) ui.Box {
	dx, dy := (c.Max.X-c.Min.X)*regionFrameMargin, (c.Max.Y-c.Min.Y)*regionFrameMargin
	return ui.Box{MinX: c.Min.X - dx, MinY: c.Min.Y - dy, MaxX: c.Max.X + dx, MaxY: c.Max.Y + dy}
}

// boxesFor reuses the projected boxes until the frame's revision changes.
func (h *host) boxesFor(f gameapi.Frame) map[string]ui.Box {
	if !h.routeBoxReady || h.routeBoxRevision != f.Revision {
		h.routeBoxCache, h.routeBoxRevision, h.routeBoxReady = routeBoxes(f), f.Revision, true
	}
	return h.routeBoxCache
}

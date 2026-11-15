package mapview

import (
	"encoding/json"
	"wwhsrt/assets"
)

type RegionView struct {
	Points [][2]float64 `json:"points"`
	Label  string       `json:"label"`
	AlignY string       `json:"align_y"`
	// Exclude lists metros (lon, lat) the frame pans off screen; see FitAvoiding.
	Exclude [][2]float64 `json:"exclude,omitempty"`
}
type Region struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Chinese   string     `json:"chinese"`
	Available []string   `json:"available_in"`
	Landscape RegionView `json:"landscape"`
	Portrait  RegionView `json:"portrait"`
}

var Regions = func() []Region {
	var rows []Region
	if err := json.Unmarshal(assets.Regions, &rows); err != nil {
		panic(err)
	}
	return rows
}()

// RegionCamera frames a region for a w x h map. In landscape it is limited by
// the World view (see LimitToWorld); in portrait by its own frame.
func RegionCamera(w, h float64, id string, portrait bool) Camera {
	c := regionFit(w, h, id, portrait)
	if !portrait {
		LimitToWorld(&c)
	}
	return c
}

// regionFit is the region's opening frame, before any limit is applied.
func regionFit(w, h float64, id string, portrait bool) Camera {
	for _, r := range Regions {
		if r.ID == id {
			v := r.Landscape
			if portrait {
				v = r.Portrait
			}
			points := []Point{}
			for _, p := range v.Points {
				points = append(points, Project(p[1], p[0]))
			}
			if len(points) < 2 {
				return New(w, h)
			}
			c := Fit(w, h, points)
			if v.AlignY == "north" {
				top := h
				for _, p := range points {
					top = min(top, c.Screen(p).Y)
				}
				c.PanY = 40 - top
			}
			for _, p := range v.Exclude {
				c.panClear(points, Project(p[1], p[0]))
			}
			return c
		}
	}
	return New(w, h)
}

// LimitToWorld bounds a landscape camera by the World view at its own fit: it
// zooms out that far and no further, and never shows beyond what that view
// shows. Every city and station lies inside it, so choosing a region frames the
// region without fencing off the rest of the map. Portrait keeps each region's
// own limits.
func LimitToWorld(c *Camera) {
	c.LimitTo(regionFit(c.Width, c.Height, "world", false))
}

func VisibleRegions(portrait bool) []Region {
	mode := "landscape"
	if portrait {
		mode = "portrait"
	}
	result := []Region{}
	for _, r := range Regions {
		for _, m := range r.Available {
			if m == mode {
				if portrait {
					r.Name = r.Portrait.Label
				} else {
					r.Name = r.Landscape.Label
				}
				result = append(result, r)
				break
			}
		}
	}
	return result
}

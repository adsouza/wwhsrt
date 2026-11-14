package render

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"math"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

type TrainPosition struct {
	ID, ServiceID       string
	Point               mapview.Point
	Next                gameapi.CityID
	LoadPermille, Speed int
	Dwelling            bool
}

func TrainPositions(c mapview.Camera, f gameapi.Frame, minute float64) []TrainPosition {
	out := []TrainPosition{}
	for _, trip := range f.Day {
		elapsed := math.Mod(minute-float64(trip.Departure)+1440, 1440)
		if elapsed > float64(trip.Arrival-trip.Departure) {
			continue
		}
		at := float64(trip.Departure) + elapsed
		position := TrainPosition{ID: trip.ID, ServiceID: trip.ServiceID}
		for _, call := range trip.Calls {
			if float64(call.Arrival) >= at {
				position.Next = call.CityID
				break
			}
		}
		found := false
		for _, leg := range trip.Legs {
			if at < float64(leg.Start) || at > float64(leg.End) {
				continue
			}
			for _, co := range f.Corridors {
				if co.ID != leg.CorridorID {
					continue
				}
				for _, a := range co.Alignments {
					if a.ID != leg.AlignmentID {
						continue
					}
					u := (at - float64(leg.Start)) / float64(leg.End-leg.Start)
					if leg.From != co.FromCity {
						u = 1 - u
					}
					position.Point = routePoint(c, a.Path, u)
					position.LoadPermille = leg.LoadPermille
					position.Speed = a.KM * 60 / a.Minutes
					found = true
				}
			}
			break
		}
		if !found {
			for _, call := range trip.Calls {
				if at >= float64(call.Arrival) && at <= float64(call.Departure) {
					for _, city := range f.Cities {
						if city.ID == call.CityID {
							position.Point = CityPoint(c, city)
							position.Dwelling = true
							position.Next = call.CityID
							found = true
						}
					}
				}
			}
		}
		if found {
			out = append(out, position)
		}
	}
	return out
}
func PickTrain(c mapview.Camera, f gameapi.Frame, minute float64, at mapview.Point) string {
	best := 20.0
	id := ""
	for _, p := range TrainPositions(c, f, minute) {
		distance := math.Hypot(p.Point.X-at.X, p.Point.Y-at.Y)
		if distance < best {
			best = distance
			id = p.ID
		}
	}
	return id
}
func TrainDescription(c mapview.Camera, f gameapi.Frame, s ui.State, minute float64) string {
	for _, p := range TrainPositions(c, f, minute) {
		if p.ID == s.SelectedTrain {
			return fmt.Sprintf("%s: %s · %s: %d km/h · %s: %.1f%%", ui.Text(s.Locale, "nextStop"), ui.Text(s.Locale, string(p.Next)), ui.Text(s.Locale, "averageSpeed"), p.Speed, ui.Text(s.Locale, "networkLoad"), float64(p.LoadPermille)/10)
		}
	}
	for _, trip := range f.Day {
		if trip.ID == s.SelectedTrain {
			return ui.Text(s.Locale, "departs") + " " + ui.ReplayClock(float64(trip.Departure))
		}
	}
	return ""
}
func (m *Map) drawDay(dst *ebiten.Image, c mapview.Camera, f gameapi.Frame, s ui.State, minute float64) {
	p := ui.PaletteFor(s)
	positions := TrainPositions(c, f, minute)
	stride := max(1, (len(positions)+95)/96)
	for i, train := range positions {
		if i%stride != 0 && train.ID != s.SelectedTrain {
			continue
		}
		at := train.Point
		if at.X < 0 || at.Y < 0 || at.X > c.Width || at.Y > c.Height {
			continue
		}
		if train.ID == s.SelectedTrain {
			vector.StrokeCircle(dst, float32(at.X), float32(at.Y), 17, 2, p.Draft, true)
		}
		vector.FillRect(dst, float32(at.X-13), float32(at.Y-7), 26, 14, p.Action, true)
		vector.FillRect(dst, float32(at.X-10), float32(at.Y-4), 20, 5, p.Surface, true)
		vector.FillRect(dst, float32(at.X-10), float32(at.Y-4), 20*float32(train.LoadPermille)/1000, 5, p.Construction, true)
	}
	if s.Watching {
		vector.FillRect(dst, float32(c.Width-78), 12, 66, 28, p.Surface, true)
		m.Label(dst, ui.ReplayClock(minute), c.Width-72, 16, 16, p.Text)
	}
}

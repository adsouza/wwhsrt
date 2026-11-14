package render

import (
	"math"
	"testing"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
)

func TestRepresentativeTrainTimingDwellReverseAndPick(t *testing.T) {
	c := mapview.Fit(800, 600, []mapview.Point{mapview.Project(35, 135), mapview.Project(37, 137)})
	f := gameapi.Frame{
		Cities: []gameapi.City{
			{ID: "a", Lat: 35, Lon: 135},
			{ID: "b", Lat: 36, Lon: 136},
			{ID: "c", Lat: 37, Lon: 137},
		},
		Corridors: []gameapi.Corridor{
			{
				ID:       "ab",
				FromCity: "a",
				ToCity:   "b",
				Alignments: []gameapi.Alignment{
					{ID: "x", KM: 100, Minutes: 60, Path: []gameapi.GeoPoint{{Lat: 35, Lon: 135}, {Lat: 36, Lon: 136}}},
				},
			},
			{
				ID:       "bc",
				FromCity: "b",
				ToCity:   "c",
				Alignments: []gameapi.Alignment{
					{ID: "y", KM: 100, Minutes: 60, Path: []gameapi.GeoPoint{{Lat: 36, Lon: 136}, {Lat: 37, Lon: 137}}},
				},
			},
		},
		Day: []gameapi.DayTrip{
			{
				ID:        "t",
				ServiceID: "through",
				Departure: 0,
				Arrival:   124,
				Calls: []gameapi.DayCall{
					{CityID: "a"},
					{CityID: "b", Arrival: 60, Departure: 64},
					{CityID: "c", Arrival: 124, Departure: 124},
				},
				Legs: []gameapi.DayLeg{
					{CorridorID: "ab", AlignmentID: "x", From: "a", To: "b", Start: 0, End: 60, LoadPermille: 500},
					{CorridorID: "bc", AlignmentID: "y", From: "b", To: "c", Start: 64, End: 124, LoadPermille: 750},
				},
			},
		},
	}
	p := TrainPositions(c, f, 30)
	if len(p) != 1 || p[0].Next != "b" || p[0].Speed != 100 || p[0].LoadPermille != 500 {
		t.Fatal(p)
	}
	if PickTrain(c, f, 30, p[0].Point) != "t" {
		t.Fatal("train pick differs from drawing")
	}
	p = TrainPositions(c, f, 62)
	if len(p) != 1 || !p[0].Dwelling || p[0].Point != CityPoint(c, f.Cities[1]) {
		t.Fatal("station dwell not retained", p)
	}
	if len(TrainPositions(c, f, 125)) != 0 {
		t.Fatal("completed train still running")
	}
	f.Day[0].Legs = f.Day[0].Legs[:1]
	f.Day[0].Legs[0].From = "b"
	f.Day[0].Legs[0].To = "a"
	f.Day[0].Arrival = 60
	p = TrainPositions(c, f, 0)
	if len(p) != 1 || p[0].Point != CityPoint(c, f.Cities[1]) {
		t.Fatal("reverse train starts at wrong end", p)
	}
	if len(TrainPositions(c, f, 1440+30)) != 1 {
		t.Fatal("repeating day differs")
	}
}

func TestImportedOsakaTrainArrivalsAndDeparturesMeetStation(t *testing.T) {
	port, e := application.Open(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	f := port.Snapshot()
	camera := mapview.Fit(1000, 700, []mapview.Point{mapview.Project(34.65, 135.4), mapview.Project(35.1, 135.85)})
	var osaka gameapi.City
	for _, city := range f.Cities {
		if city.ID == "osaka" {
			osaka = city
		}
	}
	for _, id := range []string{"nagoya-osaka", "public_kyoto__osaka", "tsuruga-osaka"} {
		for _, co := range f.Corridors {
			if co.ID != id {
				continue
			}
			for _, a := range co.Alignments {
				for _, reverse := range []bool{false, true} {
					from, to := co.FromCity, co.ToCity
					minute := 60.0
					if reverse {
						from, to = to, from
						minute = 0
					}
					f.Day = []gameapi.DayTrip{
						{
							ID:        "test",
							Departure: 0,
							Arrival:   60,
							Legs: []gameapi.DayLeg{
								{CorridorID: id, AlignmentID: a.ID, From: from, To: to, Start: 0, End: 60},
							},
						},
					}
					positions := TrainPositions(camera, f, minute)
					if len(positions) != 1 {
						t.Fatal("missing train")
					}
					p, q := positions[0].Point, CityPoint(camera, osaka)
					if math.Hypot(p.X-q.X, p.Y-q.Y) > 0.001 {
						t.Fatalf("%s/%s reverse=%v train missed Shin-Osaka", id, a.ID, reverse)
					}
				}
			}
		}
	}
}

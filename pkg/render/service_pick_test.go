package render

import (
	"reflect"
	"testing"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
)

func TestServicePickingUsesOpenCommittedGeometryAndOverlaps(t *testing.T) {
	c := mapview.Fit(1000, 700, []mapview.Point{mapview.Project(30, 100), mapview.Project(40, 110)})
	valley := []gameapi.GeoPoint{{Lat: 31, Lon: 101}, {Lat: 36, Lon: 102}, {Lat: 39, Lon: 109}}
	direct := []gameapi.GeoPoint{valley[0], valley[2]}
	f := gameapi.Frame{Corridors: []gameapi.Corridor{
		{ID: "public", Existing: true, Alignments: []gameapi.Alignment{{Path: valley}}},
		{ID: "overlap", Existing: true, Alignments: []gameapi.Alignment{{Path: valley}}},
		{ID: "project", Alignments: []gameapi.Alignment{{ID: "valley", Path: valley}, {ID: "tunnel", Path: direct}}},
	}, Projects: []gameapi.Project{{CorridorID: "project", AlignmentID: "tunnel", Ready: 5}}}
	mid := func(a, b gameapi.GeoPoint) mapview.Point {
		x, y := c.Screen(mapview.Project(a.Lat, a.Lon)), c.Screen(mapview.Project(b.Lat, b.Lon))
		return mapview.Point{X: (x.X + y.X) / 2, Y: (x.Y + y.Y) / 2}
	}
	if ids := PickServiceSections(c, f, mid(valley[0], valley[1])); !reflect.DeepEqual(ids, []string{"public", "overlap"}) {
		t.Fatal(ids)
	}
	if ids := PickServiceSections(c, f, mid(direct[0], direct[1])); len(ids) != 0 {
		t.Fatal("unfinished project/endpoint chord pickable", ids)
	}
	f.MonthsResolved = 5
	if ids := PickServiceSections(c, f, mid(direct[0], direct[1])); !reflect.DeepEqual(ids, []string{"project"}) {
		t.Fatal("opened project did not use its actual tunnel", ids)
	}
	if ids := PickServiceSections(c, f, mapview.Point{X: -1, Y: 300}); len(ids) != 0 {
		t.Fatal("outside map was selectable")
	}
}

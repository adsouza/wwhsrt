package ui

import (
	"fmt"
	"testing"

	"wwhsrt/pkg/gameapi"
)

func choiceRoute(id string, open bool, cities ...gameapi.CityID) gameapi.Route {
	if len(cities) == 0 {
		cities = []gameapi.CityID{gameapi.CityID(id + "-a"), gameapi.CityID(id + "-b")}
	}
	return gameapi.Route{ID: id, Open: open, Sections: []string{id}, Cities: cities, Stops: cities}
}

func choiceIDs(c RouteChoices) []string {
	ids := []string{}
	for _, r := range c.Routes {
		ids = append(ids, r.ID)
	}
	return ids
}

func TestRouteChoicesKeepOnlyRoutesFullyInsideView(t *testing.T) {
	f := gameapi.Frame{Routes: []gameapi.Route{choiceRoute("inside", true), choiceRoute("straddles", true)}}
	boxes := map[string]Box{"inside": {1, 1, 5, 5}, "straddles": {8, 8, 12, 12}}
	view := Box{0, 0, 10, 10}
	got := ServiceRouteChoices(f, State{}, boxes, &view)
	if ids := choiceIDs(got); len(ids) != 1 || ids[0] != "inside" || !got.InView || got.Matching != 1 {
		t.Fatalf("want only the route fully inside the view, got %v %+v", ids, got)
	}
	edge := Box{1, 1, 5, 5}
	if got = ServiceRouteChoices(f, State{}, boxes, &edge); len(got.Routes) != 1 {
		t.Fatal("a route exactly filling the view must count as inside", choiceIDs(got))
	}
}

func TestRouteChoicesRankBySpanThenIDAndCapAt25(t *testing.T) {
	f := gameapi.Frame{}
	boxes := map[string]Box{}
	for i := 1; i <= 40; i++ {
		id := fmt.Sprintf("r%02d", i)
		f.Routes = append(f.Routes, choiceRoute(id, true))
		boxes[id] = Box{0, 0, float64(i), 0}
	}
	// Two routes share the largest span; the ID settles their order.
	boxes["r39"] = Box{0, 0, 40, 0}
	view := Box{-1, -1, 100, 100}
	got := ServiceRouteChoices(f, State{}, boxes, &view)
	ids := choiceIDs(got)
	if len(ids) != RouteChoiceLimit || RouteChoiceLimit != 25 || got.Matching != 40 {
		t.Fatalf("want 25 of 40, got %d of %d", len(ids), got.Matching)
	}
	if ids[0] != "r39" || ids[1] != "r40" || ids[2] != "r38" || ids[24] != "r16" {
		t.Fatal("want largest span first with ties by ID", ids)
	}
}

func TestRouteChoicesAlwaysListOpenSelection(t *testing.T) {
	f := gameapi.Frame{
		Routes: []gameapi.Route{choiceRoute("near", true), choiceRoute("far", true), choiceRoute("shut", false)},
	}
	boxes := map[string]Box{"near": {1, 1, 2, 2}, "far": {50, 50, 60, 60}, "shut": {1, 1, 3, 3}}
	view := Box{0, 0, 10, 10}
	got := ServiceRouteChoices(f, State{ServiceCorridor: "far"}, boxes, &view)
	if ids := choiceIDs(got); len(ids) != 2 || ids[0] != "near" || ids[1] != "far" || got.Matching != 1 {
		t.Fatal("selection outside the view must be listed last", ids, got.Matching)
	}
	got = ServiceRouteChoices(f, State{ServiceCorridor: "near"}, boxes, &view)
	if ids := choiceIDs(got); len(ids) != 1 {
		t.Fatal("selection already listed must not repeat", ids)
	}
	for _, stale := range []string{"shut", "gone"} {
		got = ServiceRouteChoices(f, State{ServiceCorridor: stale}, boxes, &view)
		if ids := choiceIDs(got); len(ids) != 1 || ids[0] != "near" {
			t.Fatal("closed or missing selection must not be listed", stale, ids)
		}
	}
}

func TestRouteChoicesSearchAndTrackTapIgnoreView(t *testing.T) {
	f := gameapi.Frame{Routes: []gameapi.Route{
		choiceRoute("near", true, "alpha", "beta"),
		choiceRoute("far", true, "gamma", "delta"),
		choiceRoute("boxless", true, "gamma", "epsilon"),
	}}
	boxes := map[string]Box{"near": {1, 1, 2, 2}, "far": {50, 50, 60, 60}}
	view := Box{0, 0, 10, 10}
	got := ServiceRouteChoices(f, State{Locale: "en", ServiceQuery: " GAMMA "}, boxes, &view)
	if ids := choiceIDs(got); len(ids) != 2 || ids[0] != "far" || ids[1] != "boxless" || got.InView {
		t.Fatal("search must look beyond the view and rank a route without a box last", ids, got.InView)
	}
	got = ServiceRouteChoices(f, State{Locale: "en", ServiceMapSections: []string{"far"}}, boxes, &view)
	if ids := choiceIDs(got); len(ids) != 1 || ids[0] != "far" || got.InView {
		t.Fatal("a track tap must look beyond the view", ids, got.InView)
	}
	got = ServiceRouteChoices(
		f,
		State{Locale: "en", ServiceMapSections: []string{"far", "near"}, ServiceQuery: "alpha"},
		boxes,
		&view,
	)
	if ids := choiceIDs(got); len(ids) != 1 || ids[0] != "near" {
		t.Fatal("track tap and search must both apply", ids)
	}
	if got = ServiceRouteChoices(f, State{}, boxes, &view); len(got.Routes) != 1 || got.Routes[0].ID != "near" {
		t.Fatal("a route without a box cannot be in view", choiceIDs(got))
	}
}

func TestRouteChoicesWithoutViewRankAllOpenRoutes(t *testing.T) {
	f := gameapi.Frame{Routes: []gameapi.Route{choiceRoute("shut", false)}}
	boxes := map[string]Box{"shut": {0, 0, 99, 0}}
	for i := 1; i <= 30; i++ {
		id := fmt.Sprintf("r%02d", i)
		f.Routes = append(f.Routes, choiceRoute(id, true))
		boxes[id] = Box{0, 0, float64(i), 0}
	}
	got := ServiceRouteChoices(f, State{}, boxes, nil)
	ids := choiceIDs(got)
	if len(ids) != 25 || ids[0] != "r30" || got.InView || got.Matching != 30 {
		t.Fatal("without a view every open route competes", len(ids), ids[0], got.InView, got.Matching)
	}
}

func TestRouteChoiceHint(t *testing.T) {
	more := RouteChoices{Matching: 40, InView: true}
	none := RouteChoices{Matching: 0, InView: true}
	if got := RouteChoiceHint("en", more, true); got != "Showing 25 of 40 routes. Zoom in or search to narrow the list." {
		t.Fatal(got)
	}
	if got := RouteChoiceHint("en", none, true); got != "No route fits in this map view. Zoom out or search by name." {
		t.Fatal(got)
	}
	want := "Showing 25 of 40 routes. Choose a smaller region or search to narrow the list."
	if got := RouteChoiceHint("en", more, false); got != want {
		t.Fatal(got)
	}
	if got := RouteChoiceHint("en", none, false); got != "No route fits in this region. Choose another region or search by name." {
		t.Fatal(got)
	}
	// Search and track taps ignore the view, so zooming or changing region would not narrow them.
	for _, mapShown := range []bool{true, false} {
		got := RouteChoiceHint("en", RouteChoices{Matching: 40}, mapShown)
		if got != "Showing 25 of 40 matching routes. Add a word to the search to narrow the list." {
			t.Fatal(got)
		}
	}
	for _, quiet := range []RouteChoices{{Matching: 3, InView: true}, {Matching: 25}, {Matching: 0}} {
		if got := RouteChoiceHint("en", quiet, true); got != "" {
			t.Fatal("no hint expected", quiet, got)
		}
	}
	seen := map[string]bool{}
	for _, key := range []string{
		"routeChoicesMore", "routeChoicesNone", "routeChoicesMoreRegion", "routeChoicesNoneRegion",
		"routeChoicesMoreSearch",
	} {
		zh := Text("zh-Hans", key)
		if zh == key || zh == Text("en", key) || seen[zh] {
			t.Fatal("missing or duplicated Chinese hint", key, zh)
		}
		seen[zh] = true
	}
}

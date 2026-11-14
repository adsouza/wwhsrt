package ui

import (
	"reflect"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestMapServiceSelectionOffersOpenDirectAndThroughRoutes(t *testing.T) {
	f := gameapi.Frame{Corridors: []gameapi.Corridor{{ID: "ab", Existing: true}, {ID: "bc", Existing: true}, {ID: "future"}},
		Routes:   []gameapi.Route{{ID: "through", Sections: []string{"ab", "bc"}, Open: true}, {ID: "ab", Sections: []string{"ab"}, Open: true}, {ID: "bc", Sections: []string{"bc"}, Open: true}, {ID: "closed", Sections: []string{"ab", "future"}}},
		Services: []gameapi.Service{{CorridorID: "ab", Frequency: 6, Fare: 8000}}}
	s := State{View: "build", Expanded: true, FromCity: "x", ToCity: "y", AlignmentID: "valley", ServiceCorridor: "unrelated", ServiceQuery: "unrelated search"}
	if !SelectMapService(&s, f, []string{"ab"}) || s.View != "services" || s.ServicePane != "plan" || s.Expanded || !s.ReviewMap || s.ServiceCorridor != "ab" || s.Frequency != 6 || s.Fare != 8000 || s.ServiceQuery != "" {
		t.Fatal(s)
	}
	if s.FromCity != "x" || s.ToCity != "y" || s.AlignmentID != "valley" {
		t.Fatal("service picking lost construction draft")
	}
	choices := MapServiceRoutes(f, s)
	if len(choices) != 2 || choices[0].ID != "through" || choices[1].ID != "ab" {
		t.Fatal(choices)
	}
	DraftService(&s, f, "through")
	s.Frequency, s.Fare = 9, 12345
	if !SelectMapService(&s, f, []string{"bc"}) || s.ServiceCorridor != "through" || s.Frequency != 9 || s.Fare != 12345 {
		t.Fatal("reclick lost through-service draft", s)
	}
	for _, ids := range [][]string{{"future"}, {"missing"}, nil} {
		before := s
		if SelectMapService(&s, f, ids) || !reflect.DeepEqual(s, before) {
			t.Fatal("invalid click changed state")
		}
	}
	DraftService(&s, f, "ab")
	if len(s.ServiceMapSections) != 0 {
		t.Fatal("unrelated route retained stale map filter")
	}
	s.ServiceMapSections = []string{"ab", "bc"}
	if len(MapServiceRoutes(f, s)) != 3 {
		t.Fatal("overlapping sections lost route choices")
	}
	if len(NewGameState(s, f).ServiceMapSections) != 0 {
		t.Fatal("new game retained map selection")
	}
}

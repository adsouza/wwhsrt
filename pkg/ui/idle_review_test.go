package ui

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestIdleHighlightsUseOnlyIdleCommittedGeometry(t *testing.T) {
	a := []gameapi.GeoPoint{{Lat: 34, Lon: 135}, {Lat: 36, Lon: 136}}
	b := []gameapi.GeoPoint{{Lat: 34, Lon: 135}, {Lat: 37, Lon: 132}}
	f := gameapi.Frame{MonthsResolved: 48, Corridors: []gameapi.Corridor{{ID: "idle", FromCity: "a", ToCity: "b", Alignments: []gameapi.Alignment{{ID: "unbuilt", Path: a}, {ID: "built", Path: b}}}, {ID: "active", Existing: true, Alignments: []gameapi.Alignment{{ID: "public", Path: a}}}, {ID: "future", Alignments: []gameapi.Alignment{{ID: "x", Path: a}}}}, Projects: []gameapi.Project{{CorridorID: "idle", AlignmentID: "built", Ready: 48}, {CorridorID: "future", AlignmentID: "x", Ready: 60}}, IdleLines: []gameapi.IdleLine{{CorridorID: "idle", FromCity: "a", ToCity: "b"}}}
	before, _ := json.Marshal(f)
	h := IdleHighlights(f)
	if len(h) != 1 || h[0].CorridorID != "idle" || !reflect.DeepEqual(h[0].Path, b) {
		t.Fatal("wrong or extra highlighted track", h)
	}
	s := State{View: "report", Expanded: true, Help: HelpState{Version: 1, Open: true}}
	if !ReviewIdle(&s, f, "") || s.View != "services" || s.ServiceCorridor != "idle" || s.Expanded || s.Help.Open {
		t.Fatal("review missed idle service", s)
	}
	snapshot := s
	if ReviewIdle(&s, f, "active") || !reflect.DeepEqual(snapshot, s) {
		t.Fatal("non-idle review changed selection")
	}
	f.IdleLines = nil
	if len(IdleHighlights(f)) != 0 || ReviewIdle(&s, f, "") {
		t.Fatal("resolved warnings left highlights")
	}
	f.IdleLines = []gameapi.IdleLine{{CorridorID: "idle", FromCity: "a", ToCity: "b"}}
	after, _ := json.Marshal(f)
	if string(before) != string(after) {
		t.Fatal("review mutated business state")
	}
}

func TestIdleThroughFleetHighlightsItsSectionsWithoutCallingThemIdleTrack(t *testing.T) {
	f := gameapi.Frame{Routes: []gameapi.Route{{ID: "through", Sections: []string{"ab", "bc"}, Cities: []gameapi.CityID{"a", "b", "c"}}}, Corridors: []gameapi.Corridor{{ID: "ab", Existing: true, FromCity: "a", ToCity: "b", Alignments: []gameapi.Alignment{{ID: "public", Path: []gameapi.GeoPoint{{Lat: 35, Lon: 135}, {Lat: 36, Lon: 136}}}}}, {ID: "bc", Existing: true, FromCity: "b", ToCity: "c", Alignments: []gameapi.Alignment{{ID: "public", Path: []gameapi.GeoPoint{{Lat: 36, Lon: 136}, {Lat: 37, Lon: 137}}}}}}, IdleLines: []gameapi.IdleLine{{CorridorID: "through", Leases: 100}}}
	h := IdleHighlights(f)
	if len(h) != 2 {
		t.Fatal("through fleet did not map to both sections", h)
	}
	for _, line := range h {
		if line.ReviewID != "through" || !line.FleetOnly {
			t.Fatal("idle fleet mislabeled as idle infrastructure", line)
		}
	}
	s := State{}
	if !ReviewIdle(&s, f, "through") || s.ServiceCorridor != "through" {
		t.Fatal("through fleet not reviewable")
	}
}

func TestIdleFleetNamesDistinguishThroughAndExpressPatterns(t *testing.T) {
	f := gameapi.Frame{Routes: []gameapi.Route{{ID: "express", Cities: []gameapi.CityID{"tokyo", "nagoya", "osaka"}, Stops: []gameapi.CityID{"tokyo", "osaka"}}}}
	line := gameapi.IdleLine{CorridorID: "express", FromCity: "tokyo", ToCity: "osaka", Leases: 100}
	for _, locale := range []string{"en", "zh-Hans"} {
		name := IdleName(f, locale, line)
		if !strings.Contains(name, Text(locale, "idleFleetLabel")) || !strings.Contains(name, Text(locale, "express")) || !strings.Contains(name, Text(locale, "nagoya")) {
			t.Fatal("ambiguous idle fleet name", name)
		}
	}
}

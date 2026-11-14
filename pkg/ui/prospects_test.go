package ui

import (
	"reflect"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestSelectProspectUsesOnlyCurrentUnstartedCorridors(t *testing.T) {
	f := gameapi.Frame{Corridors: []gameapi.Corridor{
		{ID: "new", FromCity: "a", ToCity: "b", Alignments: []gameapi.Alignment{{ID: "valley"}}},
		{ID: "public", Existing: true, Alignments: []gameapi.Alignment{{ID: "public"}}},
		{ID: "started", Alignments: []gameapi.Alignment{{ID: "tunnel"}}},
	}, Projects: []gameapi.Project{{CorridorID: "started"}}}
	s := State{View: "services", ServiceCorridor: "existing-service", Fare: 12345, Frequency: 7, Expanded: true, LoanAmount: "123", Locale: "zh-Hans"}
	for _, id := range []string{"public", "started", "missing"} {
		before := s
		if SelectProspect(&s, f, id) || !reflect.DeepEqual(s, before) {
			t.Fatal("invalid prospect changed draft", id)
		}
	}
	if !SelectProspect(&s, f, "new") || s.FromCity != "a" || s.ToCity != "b" || s.AlignmentID != "valley" || s.View != "build" || s.BuildPane != "plan" {
		t.Fatal(s)
	}
	if !s.Expanded || s.ServiceCorridor != "existing-service" || s.Fare != 12345 || s.Frequency != 7 || s.LoanAmount != "123" {
		t.Fatal("prospect lost unrelated drafts")
	}
	Select(&s, "arbitrary-city")
	if s.FromCity != "a" || s.ToCity != "b" || s.AlignmentID != "valley" {
		t.Fatal("city inspection changed the construction pair")
	}
}

func TestDedicatedProspectSelectionPreservesExistingService(t *testing.T) {
	f := gameapi.Frame{Corridors: []gameapi.Corridor{{ID: "conventional", FromCity: "lisbon", ToCity: "porto", Existing: true}, {ID: "dedicated", FromCity: "lisbon", ToCity: "porto", ConstructionKind: "dedicated_hsr", Alignments: []gameapi.Alignment{{ID: "launch", Cost: 1000, Grant: 900}}}}}
	s := State{ServiceCorridor: "conventional"}
	if !SelectProspect(&s, f, "dedicated") {
		t.Fatal("cannot select new physical project")
	}
	co, ok := Pair(f, s.FromCity, s.ToCity)
	if !ok || co.ID != "dedicated" {
		t.Fatal("selection fell through to conventional track")
	}
	q, ok := FundingBreakdown(f, s)
	if !ok || q.CorridorID != "dedicated" || q.Contribution != 100 {
		t.Fatal("wrong construction quote")
	}
	if !Open(f, "conventional") || s.ServiceCorridor != "conventional" {
		t.Fatal("existing service was hidden or draft changed")
	}
}

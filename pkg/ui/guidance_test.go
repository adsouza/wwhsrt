package ui

import (
	"encoding/json"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestContextGuidanceAcrossBusinessLifecycle(t *testing.T) {
	f := gameapi.Frame{CampaignMonths: 360, Services: []gameapi.Service{{CorridorID: "a-b", Frequency: 6, Fare: 9000}}, Fleet: []gameapi.FleetLease{{ID: 1, CorridorID: "a-b"}}, Corridors: []gameapi.Corridor{{ID: "new", FromCity: "a", ToCity: "c", Alignments: []gameapi.Alignment{{ID: "route", Cost: 100000, Grant: 90000}}}}}
	s := State{View: "map", ServiceCorridor: "a-b", Frequency: 6, Fare: 9000, FromCity: "a", ToCity: "c", AlignmentID: "route"}
	check := func(key, action string) {
		t.Helper()
		before, _ := json.Marshal(f)
		cue := HelpFor(f, s)
		after, _ := json.Marshal(f)
		if cue.Key != key || cue.Action != action || string(before) != string(after) {
			t.Fatal("wrong or mutating guidance", cue, key, action)
		}
	}
	check("helpFirstMonth", "advance_month")
	s.View = "build"
	check("helpFunding", "company")
	f.Player.Cash = 10000
	check("helpBuildReady", "")
	s.View = "company"
	check("helpFunded", "build")
	f.Player.Cash = 0
	check("helpFinance", "build")
	f.Player.Cash = 10000
	s.View = "build"
	f.Projects = []gameapi.Project{{CorridorID: "new", Ready: 36}}
	check("helpConstruction", "advance_year")
	f.MonthsResolved = 36
	check("helpOpen", "services")
	f.Services = append(f.Services, gameapi.Service{CorridorID: "new", Frequency: 4, Fare: 9000})
	check("helpBuiltOperating", "services")
	f.Services = f.Services[:1]
	s.View = "services"
	s.ServiceCorridor = "new"
	check("helpFleet", "")
	s.Frequency = 0
	check("helpDraft", "")
	s.Frequency = 6
	s.ServiceCorridor = "a-b"
	s.Frequency = 4
	check("helpDraft", "")
	s.Frequency = 6
	check("helpOperating", "advance_month")
	s.Frequency = 0
	f.Services[0].Frequency = 0
	check("helpSuspended", "")
	s.View = "report"
	f.Reports = []gameapi.Report{{Costs: 100, Revenue: 50}}
	check("helpLoss", "services")
	f.IdleLines = []gameapi.IdleLine{{CorridorID: "new"}}
	check("helpIdle", "services")
	s.HintDismissed = true
	check("", "")
	s.HintDismissed = false
	f.MonthsResolved = 360
	check("helpFinished", "company")
}

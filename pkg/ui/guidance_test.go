package ui

import (
	"encoding/json"
	"reflect"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func helpFixture() (gameapi.Frame, State) {
	return gameapi.Frame{Corridors: []gameapi.Corridor{{ID: "a-b", FromCity: "a", ToCity: "b", Alignments: []gameapi.Alignment{{ID: "route", Cost: 100000, Grant: 10000}}}}}, State{Locale: "en", View: "build", FromCity: "a", ToCity: "b", AlignmentID: "route"}
}
func TestContextGuidancePerConceptAndPersistence(t *testing.T) {
	f, s := helpFixture()
	before, _ := json.Marshal(f)
	if g := HelpFor(f, s); g.Concept != "terrain" || g.Anchor != "alignments" {
		t.Fatal(g)
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "dismiss", Concept: "terrain"})
	if g := HelpFor(f, s); g.Concept != "funding" {
		t.Fatal(g)
	}
	b, _ := json.Marshal(s)
	_ = json.Unmarshal(b, &s)
	s.Portrait = true
	f.MonthsResolved++
	if !s.Help.Concepts["terrain"].Dismissed || HelpFor(f, s).Concept != "funding" {
		t.Fatal(s.Help)
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "mute"})
	if HelpFor(f, s).Key != "" {
		t.Fatal("mute")
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "expand", Concept: "terrain", Anchor: "alignments"})
	if len(BuildHelpModel(f, s, nil, "").Slots) == 0 || !s.Help.Concepts["terrain"].Explained {
		t.Fatal("on demand unavailable")
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "unmute"})
	if HelpFor(f, s).Concept != "funding" {
		t.Fatal("unmute forgot dismissal")
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "reset"})
	if HelpFor(f, s).Concept != "terrain" {
		t.Fatal("explicit reset failed")
	}
	f.MonthsResolved-- // Undo only the month change made explicitly by this test.
	after, _ := json.Marshal(f)
	if string(before) != string(after) {
		t.Fatal("guidance mutated world")
	}

}
func TestHelpLegacyMigrationAndBoundedState(t *testing.T) {
	s := State{HintDismissed: true}
	NormalizeHelp(&s)
	if !s.Help.Muted || s.HintDismissed || len(s.Help.Concepts) != 0 {
		t.Fatal(s)
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "dismiss", Concept: "unknown"})
	if len(s.Help.Concepts) != 0 {
		t.Fatal("unknown concept persisted")
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "query", Value: string(make([]rune, 200))})
	if len([]rune(s.Help.Query)) > 128 {
		t.Fatal("unbounded search")
	}
}
func TestHelpReferencePreservesTaskAndNewGameLearning(t *testing.T) {
	f, s := helpFixture()
	s.Expanded = true
	s.Fare = 12345
	s.Frequency = 9
	s.DraftName = "草案"
	s.ReplaceServiceID = "tokyo-nagoya"
	ApplyHelpEvent(&s, HelpEvent{Kind: "dismiss", Concept: "terrain"})
	draft := s
	ApplyHelpEvent(&s, HelpEvent{Kind: "open", Concept: "funding", Anchor: "expanded-endpoint", Scroll: .6})
	if !s.Help.Open || s.Expanded {
		t.Fatal(s)
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "query", Value: "fare"})
	ApplyHelpEvent(&s, HelpEvent{Kind: "close"})
	if !s.Expanded || s.View != draft.View || s.FromCity != draft.FromCity || s.Fare != draft.Fare || s.DraftName != draft.DraftName || s.ReplaceServiceID != draft.ReplaceServiceID || s.Help.ReturnScroll != .6 {
		t.Fatal("lost task", s)
	}
	RecordHelpAction(&s, f, gameapi.BuildLine)
	if !s.Help.Concepts["connections"].Completed {
		t.Fatal("accepted action not tracked")
	}
	next := NewGameState(s, f)
	if !next.Help.Concepts["terrain"].Dismissed || next.Help.Concepts["connections"].Completed || next.Help.Open {
		t.Fatal(next.Help)
	}
}
func TestHelpLayeringSearchAndErrorPriority(t *testing.T) {
	f, s := helpFixture()
	before := f
	m := BuildHelpModel(f, s, nil, "insufficient_cash")
	if m.Cue.Concept != "" || len(m.Slots) == 0 {
		t.Fatal("error obscured or help hidden")
	}
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, topic := range helpTopics {
			if Text(locale, topic.TitleKey) == topic.TitleKey || Text(locale, topic.DetailKey) == topic.DetailKey {
				t.Fatal("missing translation", locale, topic)
			}
		}
	}
	if len(HelpTopics("zh-Hans", "票价")) == 0 || len(HelpTopics("en", "tunnel")) == 0 {
		t.Fatal("search missing")
	}
	s.Expanded = true
	m = BuildHelpModel(f, s, nil, "")
	if len(m.Slots) != 1 || m.Slots[0].Anchor != "expanded-endpoint" {
		t.Fatal(m)
	}
	if !reflect.DeepEqual(f, before) {
		t.Fatal("world mutated")
	}
}

func TestHelpDismissedPriorityFallsThroughToOtherConcepts(t *testing.T) {
	f, s := helpFixture()
	s.View = "services"
	s.ServiceCorridor = "a-b"
	s.Frequency = 6
	f.IdleLines = []gameapi.IdleLine{{CorridorID: "a-b"}}
	ApplyHelpEvent(&s, HelpEvent{Kind: "dismiss", Concept: "idle_assets"})
	if HelpFor(f, s).Concept != "fleet" {
		t.Fatal("idle dismissal hid fleet learning", HelpFor(f, s))
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "dismiss", Concept: "fleet"})
	if HelpFor(f, s).Concept != "service_plans" {
		t.Fatal("fleet dismissal hid service learning", HelpFor(f, s))
	}
	s.View = "map"
	s.Selected = "a"
	s.Watching = true
	if HelpFor(f, s).Concept != "operating_day" {
		t.Fatal("replay action not taught")
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "dismiss", Concept: "operating_day"})
	if HelpFor(f, s).Concept != "city_demand" {
		t.Fatal("replay dismissal hid city learning")
	}
	s.View = "services"
	s.Help.Concepts = map[string]ConceptProgress{}
	f.IdleLines = nil
	f.Services = []gameapi.Service{{CorridorID: "a-b", Frequency: 6}}
	f.Fleet = []gameapi.FleetLease{{ID: 1, CorridorID: "a-b"}}
	forecast := gameapi.Forecast{FleetAvailable: 1, FleetNeeded: 3}
	if BuildHelpModel(f, s, &forecast, "").Cue.Concept != "fleet" {
		t.Fatal("partial fleet shortage not explained")
	}
}

func TestHelpFactsExposeAuthoritativeQuotesAndForecasts(t *testing.T) {
	f, s := helpFixture()
	f.Player.Cash = -200
	facts := HelpFacts(f, s, "funding", nil)
	values := map[string]string{}
	for _, row := range facts {
		values[row.Label] = row.Value
	}
	if values[Text("en", "fundingGap")] != "= "+Cash(90200) || values[Text("en", "fundingDeficit")] != "+ "+Cash(200) {
		t.Fatal("funding explanation does not reconcile", facts)
	}
	forecast := gameapi.Forecast{FleetAvailable: 1, FleetNeeded: 3, Passengers: 77, Revenue: 43210, Contribution: -1234, CompanyChange: 9876, FeederRevenue: 321, Minutes: 156}
	facts = HelpFacts(f, s, "fares_demand", &forecast)
	values = map[string]string{}
	for _, row := range facts {
		values[row.Label] = row.Value
	}
	if values[Text("en", "companyChange")] != Cash(9876) || values[Text("en", "operatingContribution")] != Cash(-1234) {
		t.Fatal("help recomputed forecast", facts)
	}
	b, _ := json.Marshal(facts)
	var wire []map[string]string
	_ = json.Unmarshal(b, &wire)
	if len(wire) == 0 || wire[0]["label"] == "" || wire[0]["value"] == "" {
		t.Fatal("browser detail contract lost", string(b))
	}
}

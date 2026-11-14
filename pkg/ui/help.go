package ui

import (
	"math"
	"strings"
	"unicode/utf8"
	"wwhsrt/pkg/gameapi"
)

type ConceptProgress struct {
	Seen      bool `json:"seen"`
	Dismissed bool `json:"dismissed"`
	Explained bool `json:"explained"`
	Completed bool `json:"completed"`
}
type HelpState struct {
	Version        int                        `json:"version"`
	Revision       uint64                     `json:"revision"`
	Muted          bool                       `json:"muted"`
	Concepts       map[string]ConceptProgress `json:"concepts"`
	Expanded       string                     `json:"expanded"`
	Open           bool                       `json:"open"`
	Topic          string                     `json:"topic"`
	Query          string                     `json:"query"`
	ReturnAnchor   string                     `json:"return_anchor"`
	ReturnExpanded bool                       `json:"return_expanded"`
	ReturnScroll   float64                    `json:"return_scroll"`
}
type HelpTopic struct {
	ID         string `json:"id"`
	TitleKey   string `json:"title_key"`
	SummaryKey string `json:"summary_key"`
	DetailKey  string `json:"detail_key"`
	View       string `json:"view"`
	Anchor     string `json:"anchor"`
}
type HelpSlot struct {
	Anchor string    `json:"anchor"`
	Topic  HelpTopic `json:"topic"`
	Facts  []Detail  `json:"facts"`
}
type HelpModel struct {
	Cue          Guidance    `json:"cue"`
	Slots        []HelpSlot  `json:"slots"`
	Results      []HelpTopic `json:"results"`
	Article      HelpTopic   `json:"article"`
	ArticleFacts []Detail    `json:"article_facts"`
}
type HelpEvent struct {
	Kind    string  `json:"kind"`
	Concept string  `json:"concept"`
	Anchor  string  `json:"anchor"`
	Value   string  `json:"value"`
	Scroll  float64 `json:"scroll"`
}

var helpTopics = func() []HelpTopic {
	rows := [][3]string{
		{"getting_started", "map", "start"},
		{"city_demand", "map", "city"},
		{"connections", "build", "endpoints"},
		{"terrain", "build", "alignments"},
		{"funding", "company", "finance"},
		{"construction", "build", "project"},
		{"fleet", "services", "fleet"},
		{"service_plans", "services", "service-plan"},
		{"fares_demand", "services", "fares"},
		{"through_services", "services", "through"},
		{"capacity", "services", "capacity"},
		{"monthly_report", "report", "accounts"},
		{"idle_assets", "services", "idle"},
		{"operating_day", "map", "replay"},
	}
	out := []HelpTopic{}
	for _, r := range rows {
		out = append(
			out,
			HelpTopic{
				ID:         r[0],
				TitleKey:   "learnTitle_" + r[0],
				SummaryKey: "learnSummary_" + r[0],
				DetailKey:  "learnDetail_" + r[0],
				View:       r[1],
				Anchor:     r[2],
			},
		)
	}
	return out
}()

func Topic(id string) (HelpTopic, bool) {
	for _, t := range helpTopics {
		if t.ID == id {
			return t, true
		}
	}
	return HelpTopic{}, false
}
func cloneHelp(h HelpState) HelpState { h.Concepts = cloneConcepts(h.Concepts); return h }
func cloneConcepts(in map[string]ConceptProgress) map[string]ConceptProgress {
	out := map[string]ConceptProgress{}
	for id, p := range in {
		if _, ok := Topic(id); ok {
			out[id] = p
		}
	}
	return out
}
func NormalizeHelp(s *State) {
	if s.Help.Version == 0 {
		s.Help = HelpState{Version: 1, Muted: s.HintDismissed}
	}
	if s.Help.Version != 1 {
		s.Help = HelpState{Version: 1, Muted: true}
	}
	s.HintDismissed = false
	s.Help.Concepts = cloneConcepts(s.Help.Concepts)
	if _, ok := Topic(s.Help.Topic); !ok {
		s.Help.Topic = ""
	}
	if !validHelpAnchor(s.Help.Expanded) {
		s.Help.Expanded = ""
	}
	if !validHelpAnchor(s.Help.ReturnAnchor) {
		s.Help.ReturnAnchor = ""
	}
	if math.IsNaN(s.Help.ReturnScroll) || math.IsInf(s.Help.ReturnScroll, 0) {
		s.Help.ReturnScroll = 0
	}
	s.Help.ReturnScroll = max(0, min(1, s.Help.ReturnScroll))
	s.Help.Query = strings.ToValidUTF8(s.Help.Query, "")
	if utf8.RuneCountInString(s.Help.Query) > 128 {
		s.Help.Query = string([]rune(s.Help.Query)[:128])
	}
	if s.Help.Open {
		s.Expanded = false
	}
}
func validHelpAnchor(id string) bool {
	if id == "" || id == "funding" || id == "expanded-endpoint" || id == "engineering" || id == "fleet-plan" {
		return true
	}
	for _, t := range helpTopics {
		if t.Anchor == id {
			return true
		}
	}
	return false
}

// Help events affect local presentation only. No event dispatches a game command.
func ApplyHelpEvent(s *State, e HelpEvent) {
	NormalizeHelp(s)
	p := s.Help.Concepts[e.Concept]
	_, known := Topic(e.Concept)
	switch e.Kind {
	case "seen":
		if known {
			p.Seen = true
		}
	case "dismiss":
		if known {
			p.Seen = true
			p.Dismissed = true
		}
	case "expand":
		if known && validHelpAnchor(e.Anchor) {
			p.Seen = true
			p.Explained = true
			s.Help.Expanded = e.Anchor
		}
	case "collapse":
		if s.Help.Expanded == e.Anchor {
			s.Help.Expanded = ""
		}
	case "mute":
		s.Help.Muted = true
	case "unmute":
		s.Help.Muted = false
	case "reset":
		s.Help.Concepts = map[string]ConceptProgress{}
		s.Help.Expanded = "" // Deliberately retain the separate mute preference.
	case "open":
		if !s.Help.Open {
			s.Help.ReturnExpanded = s.Expanded
			s.Help.ReturnScroll = max(0, min(1, e.Scroll))
			s.Help.ReturnAnchor = e.Anchor
		}
		s.Help.Open = true
		s.Expanded = false
		if known {
			s.Help.Topic = e.Concept
			s.Help.Query = ""
			p.Seen = true
			p.Explained = true
		}
	case "close":
		s.Help.Open = false
		s.Expanded = s.Help.ReturnExpanded
		s.Help.ReturnExpanded = false
	case "topic":
		if known {
			s.Help.Topic = e.Concept
			s.Help.Query = ""
			p.Seen = true
			p.Explained = true
		}
	case "query":
		s.Help.Query = e.Value
		s.Help.Topic = ""
	case "task":
		if t, ok := Topic(e.Concept); ok {
			s.View = t.View
			s.Help.Open = false
			s.Expanded = false
			s.Help.ReturnExpanded = false
			s.Help.ReturnAnchor = t.Anchor
			HelpPane(s, t.Anchor)
			s.Help.Expanded = t.Anchor
		}
	case "leave":
		if e.Value == "map" || e.Value == "build" || e.Value == "services" || e.Value == "company" ||
			e.Value == "report" {
			s.View = e.Value
			if e.Value != "services" {
				s.ReviewMap = false
			}
			s.Help.Open = false
			s.Expanded = false
			s.Help.ReturnExpanded = false
			s.Help.ReturnAnchor = ""
		}
	}
	if known && e.Kind != "reset" {
		s.Help.Concepts[e.Concept] = p
	}
	s.Help.Revision++
	NormalizeHelp(s)
}
func RecordHelpAction(s *State, f gameapi.Frame, kind string) {
	NormalizeHelp(s)
	complete := func(id string) { p := s.Help.Concepts[id]; p.Completed = true; s.Help.Concepts[id] = p }
	switch kind {
	case gameapi.AdvanceMonth, gameapi.AdvanceYear:
		complete("getting_started")
	case gameapi.BuildLine:
		complete("connections")
		complete("funding")
	case gameapi.TakeLoan, gameapi.RepayLoan:
		complete("funding")
	case gameapi.LeaseTrain, gameapi.ReturnTrain:
		complete("fleet")
	case gameapi.SetService:
		complete("service_plans")
	case gameapi.ExtendService:
		complete("service_plans")
		complete("fleet")
		complete("through_services")
	}
	for _, p := range f.Projects {
		if p.Ready <= f.MonthsResolved {
			complete("construction")
		}
	}
	s.Help.Revision++
}
func HelpTopics(locale, query string) []HelpTopic {
	q := strings.ToLower(strings.TrimSpace(query))
	out := []HelpTopic{}
	for _, t := range helpTopics {
		text := t.ID + " " + Text(
			locale,
			t.TitleKey,
		) + " " + Text(
			locale,
			t.SummaryKey,
		) + " " + Text(
			locale,
			t.DetailKey,
		) + " " + Text(
			"en",
			t.TitleKey,
		) + " " + Text(
			"zh-Hans",
			t.TitleKey,
		)
		if q == "" || strings.Contains(strings.ToLower(text), q) {
			out = append(out, t)
		}
	}
	return out
}
func BlockingHelpIssue(issue string) bool {
	for _, prefix := range []string{
		"invalid_",
		"storage_",
		"stale_",
		"insufficient_",
		"fleet_",
		"loan_",
		"track_",
		"station_",
		"line_not_",
		"cash_",
		"action_limit",
		"no_route",
	} {
		if strings.HasPrefix(issue, prefix) {
			return true
		}
	}
	return false
}
func helpSlots(s State) [][2]string {
	if s.Help.Open {
		return nil
	}
	if s.Expanded {
		if s.View == "build" {
			return [][2]string{{"expanded-endpoint", "connections"}}
		}
		return nil
	}
	var slots [][2]string
	switch s.View {
	case "build":
		if BuildPane(s) == "costs" {
			return [][2]string{{"engineering", "terrain"}}
		}
		if BuildPane(s) == "projects" {
			return nil
		}
		slots = [][2]string{
			{"endpoints", "connections"},
			{"alignments", "terrain"},
			{"funding", "funding"},
			{"project", "construction"},
		}
	case "services":
		if ServicePane(s) == "fleet" {
			return [][2]string{{"fleet", "fleet"}}
		}
		if ServicePane(s) == "analysis" {
			return [][2]string{{"capacity", "capacity"}, {"fares", "fares_demand"}}
		}
		slots = [][2]string{
			{"service-plan", "service_plans"},
			{"through", "through_services"},
			{"fares", "fares_demand"},
			{"fleet-plan", "fleet"},
			{"capacity", "capacity"},
			{"idle", "idle_assets"},
		}
	case "company":
		if CompanyPane(s) != "finance" {
			return nil
		}
		slots = [][2]string{{"finance", "funding"}}
	case "report":
		slots = [][2]string{{"accounts", "monthly_report"}, {"capacity", "capacity"}, {"idle", "idle_assets"}}
	default:
		slots = [][2]string{
			{"start", "getting_started"},
			{"city", "city_demand"},
			{"replay", "operating_day"},
			{"idle", "idle_assets"},
		}
	}
	return slots
}
func BuildHelpModel(f gameapi.Frame, s State, forecast *gameapi.Forecast, issue string) HelpModel {
	NormalizeHelp(&s)
	m := HelpModel{
		Cue:          HelpFor(f, s),
		Slots:        []HelpSlot{},
		Results:      HelpTopics(s.Locale, s.Help.Query),
		ArticleFacts: []Detail{},
	}
	// An under-provisioned draft needs fleet help beside its lease controls,
	// even if one train is already assigned. Extension quotes can lease atomically.
	fleet := s.Help.Concepts["fleet"]
	if forecast != nil && s.View == "services" && ServicePane(s) == "plan" && !s.Help.Open && !s.Expanded &&
		!s.Help.Muted &&
		s.ReplaceServiceID == "" &&
		forecast.FleetNeeded > forecast.FleetAvailable &&
		!fleet.Dismissed &&
		!fleet.Completed &&
		m.Cue.Concept != "idle_assets" {
		m.Cue = Guidance{Concept: "fleet", Anchor: "fleet-plan", Key: "learnSummary_fleet"}
	}
	if BlockingHelpIssue(issue) {
		m.Cue = Guidance{}
	}
	for _, slot := range helpSlots(s) {
		id := slot[1]
		if m.Cue.Anchor == slot[0] {
			id = m.Cue.Concept
		}
		t, _ := Topic(id)
		m.Slots = append(m.Slots, HelpSlot{Anchor: slot[0], Topic: t, Facts: HelpFacts(f, s, t.ID, forecast)})
	}
	for _, t := range m.Results {
		if t.ID == s.Help.Topic {
			m.Article = t
			m.ArticleFacts = HelpFacts(f, s, t.ID, forecast)
		}
	}
	return m
}

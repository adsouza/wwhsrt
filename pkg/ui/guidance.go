package ui

import "wwhsrt/pkg/gameapi"

type Guidance struct {
	Concept    string `json:"concept"`
	Anchor     string `json:"anchor"`
	Key        string `json:"key"`
	Action     string `json:"action"`
	CorridorID string `json:"corridor_id"`
}

// Guidance chooses one relevant concept, never a compulsory sequence. Explicit
// help stays available even after suggestions are muted, dismissed or completed.
func HelpFor(f gameapi.Frame, s State) Guidance {
	NormalizeHelp(&s)
	if s.Help.Muted || s.Help.Open {
		return Guidance{}
	}
	cue := func(concept, anchor string) Guidance {
		p := s.Help.Concepts[concept]
		if p.Dismissed || p.Completed {
			return Guidance{}
		}
		t, ok := Topic(concept)
		if !ok {
			return Guidance{}
		}
		return Guidance{Concept: concept, Anchor: anchor, Key: t.SummaryKey}
	}
	if s.Expanded {
		if s.View == "build" {
			return cue("connections", "expanded-endpoint")
		}
		return Guidance{}
	}
	if len(f.IdleLines) > 0 &&
		(s.View == "map" || s.View == "report" && ReportPane(
			s,
		) == "overview" || s.View == "services" && ServicePane(s) == "plan") {
		if g := cue("idle_assets", "idle"); g.Concept != "" {
			return g
		}
	}
	switch s.View {
	case "build":
		if BuildPane(s) == "costs" {
			return cue("terrain", "engineering")
		}
		if BuildPane(s) == "projects" {
			return Guidance{}
		}
		c, ok := Pair(f, s.FromCity, s.ToCity)
		if !ok || s.FromCity == s.ToCity {
			return cue("connections", "endpoints")
		}
		for _, p := range f.Projects {
			if p.CorridorID == c.ID {
				if p.Ready <= f.MonthsResolved {
					g := cue("fleet", "project")
					g.CorridorID = c.ID
					g.Action = "services"
					return g
				}
				return cue("construction", "project")
			}
		}
		if c.Existing {
			return cue("service_plans", "endpoints")
		}
		if g := cue("terrain", "alignments"); g.Concept != "" {
			return g
		}
		if q, ok := FundingBreakdown(f, s); ok && q.Shortfall > 0 {
			g := cue("funding", "funding")
			g.Action = "company"
			return g
		}
		return cue("connections", "endpoints")
	case "services":
		if ServicePane(s) == "fleet" {
			return cue("fleet", "fleet")
		}
		if ServicePane(s) == "analysis" {
			return cue("capacity", "capacity")
		}
		if s.ReplaceServiceID != "" {
			if g := cue("through_services", "through"); g.Concept != "" {
				return g
			}
		}
		count := 0
		for _, train := range f.Fleet {
			if train.CorridorID == s.ServiceCorridor {
				count++
			}
		}
		if count == 0 && s.Frequency > 0 {
			if g := cue("fleet", "fleet-plan"); g.Concept != "" {
				return g
			}
		}
		if ServiceDirty(s, f) {
			if g := cue("service_plans", "service-plan"); g.Concept != "" {
				return g
			}
		}
		for _, section := range f.Network.Sections {
			if section.Delivered < section.Requested {
				if g := cue("capacity", "capacity"); g.Concept != "" {
					return g
				}
			}
		}
		return cue("fares_demand", "fares")
	case "company":
		if CompanyPane(s) != "finance" {
			return Guidance{}
		}
		return cue("funding", "finance")
	case "report":
		if ReportPane(s) == "network" {
			return cue("capacity", "capacity")
		}
		if ReportPane(s) == "markets" {
			return Guidance{}
		}
		return cue("monthly_report", "accounts")
	default:
		if s.Watching {
			if g := cue("operating_day", "replay"); g.Concept != "" {
				return g
			}
		}
		if s.Selected != "" {
			if g := cue("city_demand", "city"); g.Concept != "" {
				return g
			}
		}
		if f.MonthsResolved == 0 {
			return cue("getting_started", "start")
		}
		return Guidance{}
	}
}

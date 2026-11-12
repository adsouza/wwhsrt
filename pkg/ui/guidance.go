package ui

import "wwhsrt/pkg/gameapi"

type Guidance struct {
	Key        string `json:"key"`
	Action     string `json:"action"`
	CorridorID string `json:"corridor_id"`
}

// Guidance is a read-only presentation projection. It never gates a command.
func HelpFor(f gameapi.Frame, s State) Guidance {
	if s.HintDismissed {
		return Guidance{}
	}
	if f.MonthsResolved >= f.CampaignMonths && f.CampaignMonths > 0 {
		return Guidance{Key: "helpFinished", Action: "company"}
	}
	if len(f.IdleLines) > 0 && (s.View == "map" || s.View == "report") {
		return Guidance{Key: "helpIdle", Action: "services", CorridorID: f.IdleLines[0].CorridorID}
	}
	switch s.View {
	case "build":
		c, ok := Pair(f, s.FromCity, s.ToCity)
		if !ok {
			return Guidance{Key: "helpPair"}
		}
		for _, p := range f.Projects {
			if p.CorridorID == c.ID {
				if p.Ready > f.MonthsResolved {
					return Guidance{Key: "helpConstruction", Action: "advance_year"}
				}
				for _, service := range f.Services {
					if service.CorridorID == c.ID && service.Frequency > 0 {
						return Guidance{Key: "helpBuiltOperating", Action: "services", CorridorID: c.ID}
					}
				}
				return Guidance{Key: "helpOpen", Action: "services", CorridorID: c.ID}
			}
		}
		if c.Existing {
			return Guidance{Key: "helpPublic", Action: "services", CorridorID: c.ID}
		}
		if q, ok := FundingBreakdown(f, s); ok && q.Shortfall > 0 {
			return Guidance{Key: "helpFunding", Action: "company"}
		}
		return Guidance{Key: "helpBuildReady"}
	case "services":
		leased := 0
		for _, l := range f.Fleet {
			if l.CorridorID == s.ServiceCorridor {
				leased++
			}
		}
		if leased == 0 && s.Frequency > 0 {
			return Guidance{Key: "helpFleet"}
		}
		if ServiceDirty(s, f) {
			return Guidance{Key: "helpDraft"}
		}
		if s.Frequency == 0 {
			return Guidance{Key: "helpSuspended"}
		}
		return Guidance{Key: "helpOperating", Action: "advance_month"}
	case "report":
		if len(f.Reports) == 0 {
			return Guidance{Key: "helpFirstMonth", Action: "advance_month"}
		}
		r := f.Reports[len(f.Reports)-1]
		if r.Costs > r.Revenue {
			return Guidance{Key: "helpLoss", Action: "services", CorridorID: s.ServiceCorridor}
		}
		return Guidance{Key: "helpReport", Action: "build"}
	case "company":
		if q, ok := FundingBreakdown(f, s); ok {
			if q.Shortfall > 0 {
				return Guidance{Key: "helpFinance", Action: "build"}
			}
			return Guidance{Key: "helpFunded", Action: "build"}
		}
		return Guidance{Key: "helpCompany", Action: "services", CorridorID: s.ServiceCorridor}
	default:
		if f.MonthsResolved == 0 {
			return Guidance{Key: "helpFirstMonth", Action: "advance_month"}
		}
		return Guidance{Key: "helpExplore", Action: "build"}
	}
}

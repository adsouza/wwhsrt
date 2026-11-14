package ui

import "image"

const DefaultPanelWidth = 440

type WorkspaceLayout struct {
	Header, Dock, Map, Inspector, Divider, Footer image.Rectangle
	DockLeft, Compact                             bool
	PanelWidth                                    int
}

func PanelWidth(preferred int) int {
	if preferred == 0 {
		return DefaultPanelWidth
	}
	return max(320, min(800, preferred))
}
func Workspace(w, h int, s State) WorkspaceLayout {
	w = max(1, w)
	h = max(1, h)
	l := WorkspaceLayout{Compact: w < 800}
	if s.Expanded {
		bottom := 0
		if s.View == "build" {
			bottom = min(270, max(0, h-100))
		}
		l.Map = image.Rect(0, 0, w, h-bottom)
		return l
	}
	header, footer, dock := 48, 64, 56
	if l.Compact {
		footer = 92
		dock = 96
		if w >= 390 {
			dock = 52
		}
	}
	l.Header = image.Rect(0, 0, w, header)
	l.Footer = image.Rect(0, max(header, h-footer), w, h)
	l.DockLeft = s.CommandDock == "left" && w >= 1000 && h >= 600
	top, left := header, 0
	if l.DockLeft {
		left = 164
		l.Dock = image.Rect(0, header, left, l.Footer.Min.Y)
	} else {
		l.Dock = image.Rect(0, header, w, header+dock)
		top += dock
	}
	if l.Compact {
		mh := 0
		if s.View == "map" && !s.Help.Open {
			mh = min(max(150, h/3), max(0, l.Footer.Min.Y-top-120))
		}
		if s.View == "services" && s.ReviewMap && !s.Help.Open {
			mh = min(240, max(0, l.Footer.Min.Y-top-120))
		}
		l.Map = image.Rect(0, top, w, top+mh)
		l.Inspector = image.Rect(0, top+mh, w, l.Footer.Min.Y)
		l.PanelWidth = w
	} else {
		pw := min(PanelWidth(s.PanelWidth), max(320, w-left-328))
		l.PanelWidth = pw
		l.Inspector = image.Rect(w-pw, top, w, l.Footer.Min.Y)
		l.Divider = image.Rect(w-pw-8, top, w-pw, l.Footer.Min.Y)
		l.Map = image.Rect(left, top, w-pw-8, l.Footer.Min.Y)
	}
	return l
}
func ServicePane(s State) string {
	if s.ServicePane == "fleet" || s.ServicePane == "analysis" {
		return s.ServicePane
	}
	return "plan"
}
func CompanyPane(s State) string {
	if s.CompanyPane == "identity" || s.CompanyPane == "settings" {
		return s.CompanyPane
	}
	return "finance"
}
func BuildPane(s State) string {
	if s.BuildPane == "costs" || s.BuildPane == "projects" {
		return s.BuildPane
	}
	return "plan"
}
func ReportPane(s State) string {
	if s.ReportPane == "accounts" || s.ReportPane == "network" || s.ReportPane == "markets" {
		return s.ReportPane
	}
	return "overview"
}

// Disclosure state is presentation only and is bounded to known sections.
func Disclosure(s State, id string) bool { return s.Disclosures[id] }
func SetDisclosure(s *State, id string, open bool) {
	allowed := map[string]bool{
		"idle":            true,
		"capacity":        true,
		"service-path":    true,
		"opportunities":   true,
		"funding":         true,
		"company-funding": true,
		"report-costs":    true,
		"city-list":       true,
		"station":         true,
		"build-details":   true,
		"route-search":    true,
	}
	if !allowed[id] {
		return
	}
	next := map[string]bool{}
	for k, v := range s.Disclosures {
		if allowed[k] && v {
			next[k] = true
		}
	}
	if open {
		next[id] = true
	} else {
		delete(next, id)
	}
	s.Disclosures = next
}
func HelpPane(s *State, anchor string) {
	switch anchor {
	case "fleet":
		s.ServicePane = "fleet"
	case "capacity", "fares", "service-plan", "through", "idle":
		s.ServicePane = "plan"
	case "alignments", "endpoints", "project", "funding":
		s.BuildPane = "plan"
	case "finance":
		s.CompanyPane = "finance"
	case "accounts":
		s.ReportPane = "accounts"
	}
}

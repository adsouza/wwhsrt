package ui

import "image"

const DefaultPanelWidth = 440

type WorkspaceLayout struct {
	Header, Dock, Map, Prospects, Inspector, Divider, Footer image.Rectangle
	DockLeft, Compact                                        bool
	PanelWidth                                               int
	// Step is the applied text step in percent. Rectangles and PanelWidth are in window pixels.
	Step int
}

func PanelWidth(preferred int) int {
	if preferred == 0 {
		return DefaultPanelWidth
	}
	return max(320, min(800, preferred))
}

// Workspace lays out a window. A text step of k is the 100% layout of a window 1/k the size with every
// coordinate multiplied by k; the reduced window's far edges land exactly on the real window's.
func Workspace(w, h int, s State) WorkspaceLayout {
	w = max(1, w)
	h = max(1, h)
	step := AppliedTextStep(w, h, s.TextSize)
	if step == 100 {
		return workspaceAt100(w, h, s)
	}
	rw, rh := w*100/step, h*100/step
	l := workspaceAt100(rw, rh, s)
	l.Step = step
	scale := func(n, reduced, full int) int {
		if n == reduced {
			return full
		}
		return (n*step + 50) / 100
	}
	for _, r := range []*image.Rectangle{&l.Header, &l.Dock, &l.Map, &l.Prospects, &l.Inspector, &l.Divider, &l.Footer} {
		if r.Empty() {
			*r = image.Rectangle{}
			continue
		}
		*r = image.Rect(scale(r.Min.X, rw, w), scale(r.Min.Y, rh, h), scale(r.Max.X, rw, w), scale(r.Max.Y, rh, h))
	}
	l.PanelWidth = scale(l.PanelWidth, rw, w)
	return l
}

func workspaceAt100(w, h int, s State) WorkspaceLayout {
	l := WorkspaceLayout{Compact: w < 800, Step: 100}
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
	// Give the prospect list its own viewport so reviewing a plan never
	// scrolls the player's next construction goal out of sight.
	if !s.Help.Open {
		if l.Compact {
			height := min(119, max(98, (l.Footer.Min.Y-top)/6))
			if !l.Map.Empty() {
				l.Map.Max.Y = min(l.Map.Max.Y, max(l.Map.Min.Y+80, l.Footer.Min.Y-height-120))
			}
			l.Prospects = image.Rect(0, l.Map.Max.Y, w, l.Map.Max.Y+height)
			l.Inspector.Min.Y = l.Prospects.Max.Y
		} else {
			height := min(202, max(108, l.Map.Dy()/4))
			l.Prospects = image.Rect(l.Map.Min.X, l.Map.Max.Y-height, l.Map.Max.X, l.Map.Max.Y)
			l.Map.Max.Y = l.Prospects.Min.Y
		}
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

package ui

import (
	"image"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestWorkspaceLayoutAdaptsWithoutLosingPreferences(t *testing.T) {
	for _, size := range [][2]int{{320, 480}, {390, 844}, {800, 600}, {1000, 700}, {1280, 850}, {1920, 1080}} {
		for _, dock := range []string{"top", "left"} {
			for _, width := range []int{0, 320, 440, 800, 9999} {
				s := State{View: "services", CommandDock: dock, PanelWidth: width}
				before := s
				l := Workspace(size[0], size[1], s)
				bounds := image.Rect(0, 0, size[0], size[1])
				for _, r := range []image.Rectangle{l.Header, l.Dock, l.Map, l.Inspector, l.Footer, l.Divider} {
					if !r.Empty() && !r.In(bounds) {
						t.Fatal("workspace escapes window", size, dock, width, l)
					}
				}
				if l.Inspector.Overlaps(l.Dock) || l.Inspector.Overlaps(l.Footer) || l.Map.Overlaps(l.Inspector) || l.Map.Overlaps(l.Dock) {
					t.Fatal("overlapping panels", l)
				}
				if !l.Compact && l.Map.Dx() < 320 {
					t.Fatal("inspector starves map", l)
				}
				if s.PanelWidth != before.PanelWidth || s.CommandDock != before.CommandDock {
					t.Fatal("responsive fallback overwrote preference")
				}
			}
		}
	}
	s := State{CommandDock: "left", PanelWidth: 720, View: "services"}
	if Workspace(390, 844, s).DockLeft || Workspace(390, 844, s).PanelWidth != 390 {
		t.Fatal("phone fallback")
	}
	if !Workspace(1920, 1080, s).DockLeft || Workspace(1920, 1080, s).PanelWidth != 720 {
		t.Fatal("desktop preference not restored")
	}
	next := NewGameState(s, gameapi.Frame{})
	if next.PanelWidth != 720 || next.CommandDock != "left" {
		t.Fatal("new campaign lost workspace")
	}
}
func TestWorkspaceDisclosuresAreBoundedAndTaskLinksChoosePane(t *testing.T) {
	s := State{}
	SetDisclosure(&s, "capacity", true)
	SetDisclosure(&s, "arbitrary", true)
	if len(s.Disclosures) != 1 || !Disclosure(s, "capacity") {
		t.Fatal(s.Disclosures)
	}
	ApplyHelpEvent(&s, HelpEvent{Kind: "task", Concept: "fleet"})
	if ServicePane(s) != "fleet" || s.Help.Expanded != "fleet" {
		t.Fatal("help cannot reach fleet controls", s)
	}
	s.Expanded = true
	l := Workspace(1280, 850, s)
	if !l.Dock.Empty() || !l.Inspector.Empty() || l.Map != image.Rect(0, 0, 1280, 850) {
		t.Fatal("expanded map still has panels", l)
	}
}

package ui

import (
	"fmt"
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
				if l.Inspector.Overlaps(l.Dock) || l.Inspector.Overlaps(l.Footer) || l.Map.Overlaps(l.Inspector) ||
					l.Map.Overlaps(l.Dock) {
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

// The native Services panel gates route search and the region filter behind
// this disclosure, so the player must be able to open and close it.
func TestRouteSearchDisclosureCanOpenAndClose(t *testing.T) {
	s := State{}
	SetDisclosure(&s, "route-search", true)
	if !Disclosure(s, "route-search") {
		t.Fatal("route search disclosure did not open")
	}
	SetDisclosure(&s, "capacity", true)
	if !Disclosure(s, "route-search") || !Disclosure(s, "capacity") {
		t.Fatal("opening another disclosure must not close route search")
	}
	SetDisclosure(&s, "route-search", false)
	if Disclosure(s, "route-search") {
		t.Fatal("route search disclosure did not close")
	}
}

// scaledRect is the rule a step follows: the 100% rectangle of the reduced window with each coordinate
// multiplied by the step, where the reduced window's far edges land exactly on the real window's.
func scaledRect(r image.Rectangle, reduced, window image.Point, step int) image.Rectangle {
	x := func(n int) int {
		if n == reduced.X {
			return window.X
		}
		return (n*step + 50) / 100
	}
	y := func(n int) int {
		if n == reduced.Y {
			return window.Y
		}
		return (n*step + 50) / 100
	}
	if r.Empty() {
		return image.Rectangle{}
	}
	return image.Rect(x(r.Min.X), y(r.Min.Y), x(r.Max.X), y(r.Max.Y))
}

func TestWorkspaceStepEqualsSmallerWindow(t *testing.T) {
	for _, c := range [][3]int{
		{1920, 1080, 200}, {3840, 2160, 200}, {3840, 2160, 150}, {1280, 850, 150}, {1600, 1000, 125},
	} {
		window, step := image.Pt(c[0], c[1]), c[2]
		reduced := image.Pt(c[0]*100/step, c[1]*100/step)
		for _, view := range []string{"map", "build", "services"} {
			for _, dock := range []string{"", "left"} {
				for _, expanded := range []bool{false, true} {
					s := State{View: view, CommandDock: dock, Expanded: expanded}
					base := Workspace(reduced.X, reduced.Y, s)
					s.TextSize = step
					got := Workspace(window.X, window.Y, s)
					name := fmt.Sprint(c, view, dock, expanded)
					if got.Step != step || got.Compact != base.Compact || got.DockLeft != base.DockLeft {
						t.Fatal(name, "step or layout kind differs", got.Step, got.Compact, got.DockLeft)
					}
					for i, pair := range [][2]image.Rectangle{
						{got.Header, base.Header}, {got.Dock, base.Dock}, {got.Map, base.Map},
						{got.Inspector, base.Inspector}, {got.Divider, base.Divider}, {got.Footer, base.Footer},
					} {
						if want := scaledRect(pair[1], reduced, window, step); pair[0] != want {
							t.Errorf("%s rectangle %d: got %v, want %v (100%%: %v)", name, i, pair[0], want, pair[1])
						}
					}
				}
			}
		}
	}
}

func TestWorkspaceUnchangedAt100(t *testing.T) {
	for _, size := range [][2]int{{320, 480}, {390, 844}, {800, 600}, {1280, 850}, {1920, 1080}} {
		for _, view := range []string{"map", "build", "services"} {
			unset := Workspace(size[0], size[1], State{View: view})
			if set := Workspace(size[0], size[1], State{View: view, TextSize: 100}); set != unset || set.Step != 100 {
				t.Fatal("100% must be the layout of an unset text size", size, view, set, unset)
			}
		}
	}
}

func TestWorkspaceStepFallsBackWhenTooSmall(t *testing.T) {
	if got := Workspace(1280, 850, State{View: "map", TextSize: 200}).Step; got != 150 {
		t.Fatal("1280x850 has room for 150% at most, got", got)
	}
	phone := Workspace(390, 844, State{View: "map", TextSize: 200})
	if phone.Step != 100 || phone != Workspace(390, 844, State{View: "map"}) {
		t.Fatal("a phone-sized window stays at 100%", phone)
	}
}

func TestWorkspacePanelWidthIsStoredAt100(t *testing.T) {
	if got := Workspace(3840, 2160, State{View: "map", PanelWidth: 500, TextSize: 100}).PanelWidth; got != 500 {
		t.Fatal("want 500 px at 100%, got", got)
	}
	if got := Workspace(3840, 2160, State{View: "map", PanelWidth: 500, TextSize: 200}).PanelWidth; got != 1000 {
		t.Fatal("the stored width is in 100% units, so 200% shows 1000 px, got", got)
	}
}

//go:build !js

package app

import (
	"fmt"
	"image"
	"path/filepath"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

// chromeWorld is one campaign and one map shared by a layout test's windows, which only build panels:
// opening them for each of several hundred windows would dominate the suite's time. It belongs to one
// test and is garbage afterwards; kept for the whole run it enlarges every later collection.
type chromeWorld struct {
	port  *application.Game
	frame gameapi.Frame
	m     *render.Map
}

func newChromeWorld(t *testing.T) *chromeWorld {
	t.Helper()
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, err := render.New()
	if err != nil {
		t.Fatal(err)
	}
	return &chromeWorld{port: port, frame: port.Snapshot(), m: m}
}

// game builds a window the way the app does: laid out, then built.
func (w *chromeWorld) game(width, height int, state ui.State) *Game {
	if state.Locale == "" {
		state.Locale = "en"
	}
	state.FromCity, state.ToCity, state.ServiceCorridor = "tsuruga", "osaka", "tokyo-nagoya"
	state.Frequency, state.Fare, state.LoanAmount, state.DraftName = 6, 9000, "1000000", "Rail"
	g := &Game{Port: w.port, Frame: w.frame, Map: w.m, width: width, height: height, State: state}
	g.host.layout(g)
	g.host.build(g)
	return g
}

// chromeGame builds a single window with a campaign of its own.
func chromeGame(t *testing.T, width, height int, state ui.State) *Game {
	t.Helper()
	return newChromeWorld(t).game(width, height, state)
}

// placedWidget is one hand-placed widget: its rectangle within its parent and what it holds.
type placedWidget struct {
	Path  string
	Rect  image.Rectangle
	Label string
	Need  image.Point
	Leaf  bool
}

// placedWidgets flattens the widget tree to every child whose position is a hand-written rectangle.
func placedWidgets(g *Game) []placedWidget {
	root := g.host.ui.Container.(*widget.Container)
	out := []placedWidget{}
	var walk func(c *widget.Container, path string)
	walk = func(c *widget.Container, path string) {
		for i, w := range c.Children() {
			name := fmt.Sprintf("%s/%d", path, i)
			inner, isContainer := w.(*widget.Container)
			if r, ok := w.GetWidget().LayoutData.(image.Rectangle); ok {
				p := placedWidget{Path: name, Rect: r, Leaf: !isContainer}
				switch v := w.(type) {
				case *widget.Text:
					p.Label = v.Label
				case *widget.Button:
					if v.Text() != nil {
						p.Label = v.Text().Label
					}
				case *widget.ScrollContainer, *widget.Slider:
					p.Leaf = false // sized by their own scrolling, not by text
				}
				if p.Leaf {
					// Validate only what is measured: validating the whole tree builds every dropdown's
					// entries, which the image registry then keeps for the rest of the run.
					w.Validate()
					if b, ok := w.(*widget.Button); ok && b.Text() != nil {
						p.Label = b.Text().Label
					}
					width, height := w.PreferredSize()
					p.Need = image.Pt(width, height)
				}
				out = append(out, p)
			}
			if isContainer {
				walk(inner, name)
			}
		}
	}
	walk(root, "")
	return out
}

// chromeOverflows lists hand-placed text and buttons that need more room than their rectangle gives.
func chromeOverflows(g *Game) []string {
	out := []string{}
	for _, p := range placedWidgets(g) {
		if p.Leaf && (p.Need.X > p.Rect.Dx() || p.Need.Y > p.Rect.Dy()) {
			out = append(out, fmt.Sprintf("%s %q needs %v in %v", p.Path, p.Label, p.Need, p.Rect.Size()))
		}
	}
	return out
}

func TestNativeChromeTextFitsItsBoxes(t *testing.T) {
	world := newChromeWorld(t)
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, size := range [][2]int{{320, 480}, {390, 844}, {800, 600}, {1280, 850}, {1920, 1080}} {
			for _, dock := range []string{"", "left"} {
				for _, view := range []string{"map", "build", "services", "report", "company"} {
					if dock == "left" && view != "map" {
						continue // header, dock and footer do not depend on the view
					}
					g := world.game(size[0], size[1], ui.State{Locale: locale, View: view, CommandDock: dock})
					name := fmt.Sprintf("%s %dx%d dock=%q %s", locale, size[0], size[1], dock, view)
					for _, overflow := range chromeOverflows(g) {
						t.Errorf("%s: %s", name, overflow)
					}
					buttons := map[image.Rectangle]bool{}
					for _, p := range placedWidgets(g) {
						buttons[p.Rect] = true
					}
					for _, r := range g.host.mapControls {
						if !buttons[r] {
							t.Errorf("%s: map control %v is not a button's rectangle", name, r)
						}
					}
				}
			}
		}
	}
}

// stepPair is one window at a text step beside the window 1/step the size at 100%, which the step is
// defined to match.
type stepPair struct {
	Name          string
	Step          int
	Window, Small image.Point
	At, Base      *Game
}

// One window per step, plus a second at 150% whose reduced size is not a whole number of pixels.
var stepWindows = [][3]int{{1920, 1080, 200}, {3840, 2160, 150}, {1280, 850, 150}, {1600, 1000, 125}}

// eachStepPair builds one pair at a time, so several hundred windows are never alive together. Header,
// dock and footer are the same in every view, so the left dock is paired with the map view only.
func eachStepPair(t *testing.T, check func(p stepPair)) {
	t.Helper()
	world := newChromeWorld(t)
	for _, c := range stepWindows {
		window, step := image.Pt(c[0], c[1]), c[2]
		small := image.Pt(c[0]*100/step, c[1]*100/step)
		for _, locale := range []string{"en", "zh-Hans"} {
			for _, dock := range []string{"", "left"} {
				for _, view := range []string{"map", "build", "services", "report", "company"} {
					if dock == "left" && view != "map" {
						continue
					}
					state := ui.State{Locale: locale, View: view, CommandDock: dock}
					base := world.game(small.X, small.Y, state)
					state.TextSize = step
					check(stepPair{
						Name:   fmt.Sprintf("%s %dx%d at %d%% dock=%q %s", locale, c[0], c[1], step, dock, view),
						Step:   step,
						Window: window,
						Small:  small,
						At:     world.game(window.X, window.Y, state),
						Base:   base,
					})
				}
			}
		}
	}
}

// A text step is defined as the 100% layout of a window 1/step the size, scaled up. The three checks
// share each pair of windows: rectangles match, no hand-placed text is clipped, and panels fit.
func TestNativeStepEqualsSmallerWindow(t *testing.T) {
	// Rows of cells share a width by integer division, so a cell can drift a pixel from its neighbour and
	// the last of six by several once scaled. An unscaled 44 px control is 22 px out at 150%.
	near := func(got, base, step int) bool { d := got - (base*step+50)/100; return d >= -6 && d <= 6 }
	eachStepPair(t, func(p stepPair) {
		if p.At.host.workspace.Step != p.Step {
			t.Fatalf("%s: applied step %d", p.Name, p.At.host.workspace.Step)
		}
		at, base := placedWidgets(p.At), placedWidgets(p.Base)
		if len(at) != len(base) || len(p.At.host.mapControls) != len(p.Base.host.mapControls) {
			t.Errorf("%s: %d placed widgets and %d map controls, 100%% has %d and %d", p.Name,
				len(at), len(p.At.host.mapControls), len(base), len(p.Base.host.mapControls))
			return
		}
		same := func(what string, got, want image.Rectangle) bool {
			if near(got.Min.X, want.Min.X, p.Step) && near(got.Min.Y, want.Min.Y, p.Step) &&
				near(got.Max.X, want.Max.X, p.Step) && near(got.Max.Y, want.Max.Y, p.Step) {
				return true
			}
			t.Errorf("%s: %s is %v, but 100%% has %v", p.Name, what, got, want)
			return false
		}
		for i := range at {
			if !same(fmt.Sprintf("%s %q", at[i].Path, at[i].Label), at[i].Rect, base[i].Rect) {
				break // later rectangles usually follow from the first that differs
			}
		}
		for i, r := range p.At.host.mapControls {
			if !same(fmt.Sprint("map control ", i), r, p.Base.host.mapControls[i]) {
				break
			}
		}
		for _, overflow := range chromeOverflows(p.At) {
			t.Errorf("%s: clipped text: %s", p.Name, overflow)
		}
		h := &p.At.host
		if w, _ := h.scroll.PreferredSize(); w > h.scroll.ViewRect().Dx() {
			t.Errorf("%s: panel content needs %d px but the viewport has %d", p.Name, w, h.scroll.ViewRect().Dx())
		}
		footer := h.ui.Container.Children()[1]
		if w, _ := footer.PreferredSize(); w > h.workspace.Footer.Dx() {
			t.Errorf("%s: footer needs %d px but has %d", p.Name, w, h.workspace.Footer.Dx())
		}
	})
}

func TestNativeResizeDropsTheStep(t *testing.T) {
	g := chromeGame(t, 1920, 1080, ui.State{View: "map", TextSize: 200})
	if g.host.workspace.Step != 200 || g.host.text.Step != 200 {
		t.Fatal("1920x1080 has room for 200%", g.host.workspace.Step, g.host.text.Step)
	}
	g.dirty = false
	g.Layout(1280, 850)
	if g.host.workspace.Step != 150 || g.host.text.Step != 150 || !g.dirty {
		t.Fatal("shrinking to 1280x850 must drop to 150% and rebuild", g.host.workspace.Step, g.host.text, g.dirty)
	}
	if int(g.Camera.Width) != g.host.workspace.Map.Dx() || int(g.Camera.Height) != g.host.workspace.Map.Dy() {
		t.Fatal("the camera must follow the new map rectangle", g.Camera.Width, g.host.workspace.Map)
	}
}

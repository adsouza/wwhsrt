//go:build !js

package app

import (
	"fmt"
	"image"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

// chromeWorld is one campaign and one map shared by the layout tests, which only build panels: opening
// them for each of several hundred windows would dominate the suite's time.
var chromeWorld = sync.OnceValue(func() (w struct {
	port *application.Game
	m    *render.Map
	err  error
}) {
	if w.port, w.err = application.Open(nil, nil); w.err != nil {
		return w
	}
	w.m, w.err = render.New()
	return w
})

// chromeGame builds a game the way the window does: laid out, then built.
func chromeGame(t *testing.T, width, height int, state ui.State) *Game {
	t.Helper()
	old := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = old })
	w := chromeWorld()
	if w.err != nil {
		t.Fatal(w.err)
	}
	p, m := w.port, w.m
	if state.Locale == "" {
		state.Locale = "en"
	}
	state.FromCity, state.ToCity, state.ServiceCorridor = "tsuruga", "osaka", "tokyo-nagoya"
	state.Frequency, state.Fare, state.LoanAmount, state.DraftName = 6, 9000, "1000000", "Rail"
	g := &Game{Port: p, Frame: p.Snapshot(), Map: m, width: width, height: height, State: state}
	g.host.layout(g)
	g.host.build(g)
	return g
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
	root.Validate()
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
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, size := range [][2]int{{320, 480}, {390, 844}, {800, 600}, {1280, 850}, {1920, 1080}} {
			for _, dock := range []string{"", "left"} {
				for _, view := range []string{"map", "build", "services", "report", "company"} {
					g := chromeGame(t, size[0], size[1], ui.State{Locale: locale, View: view, CommandDock: dock})
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

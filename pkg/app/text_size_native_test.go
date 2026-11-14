//go:build !js

package app

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/pkg/ui"
)

// Every font is made in one place per package so that no call site can skip the ladder or the step.
func TestNativeFontsComeFromTheScale(t *testing.T) {
	for _, dir := range []string{".", "../render"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			src, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, name, src, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || fn.Name.Name == "face" && fn.Recv != nil {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					lit, ok := n.(*ast.CompositeLit)
					if !ok {
						return true
					}
					if sel, ok := lit.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "GoTextFace" {
						t.Errorf("%s: font made outside the face helper, in %s", set.Position(lit.Pos()), fn.Name.Name)
					}
					return true
				})
			}
		}
	}
}

func faceSize(t *testing.T, f text.Face) float64 {
	t.Helper()
	face, ok := f.(*text.GoTextFace)
	if !ok {
		t.Fatalf("want a *text.GoTextFace, got %T", f)
	}
	return face.Size
}

func TestNativePanelTextUsesTheDesktopLadder(t *testing.T) {
	g := servicesGame(t, 1280, 850, "japan")
	if want := (ui.TextScale{Ladder: "desktop", Step: 100}); g.host.text != want {
		t.Fatalf("a 1280 px window uses the desktop ladder, got %+v", g.host.text)
	}
	if small, body := faceSize(t, g.host.face(g, 13)), faceSize(t, g.host.face(g, 14)); small != 14 || body != 16 {
		t.Fatal("desktop: 13 px becomes 14 and 14 px becomes 16, got", small, body)
	}
	g = servicesGame(t, 390, 844, "japan")
	if g.host.text.Ladder != "phone" || faceSize(t, g.host.face(g, 13)) != 13 {
		t.Fatalf("a phone-width window keeps its panel sizes, got %+v", g.host.text)
	}
}

func TestNativeTextFieldGrowsWithItsFont(t *testing.T) {
	const name = "晨曦国际高速铁路"
	for _, chosen := range []int{0, 200} {
		g := servicesGame(t, 3840, 2160, "japan")
		g.State.View, g.State.CompanyPane, g.State.DraftName, g.State.TextSize = "company", "identity", name, chosen
		g.host.layout(g)
		g.host.build(g)
		field := g.host.nameInput
		if field == nil {
			t.Fatal("the identity pane must show the company name field")
		}
		want := &text.GoTextFace{Source: g.Map.Font, Size: ui.NativeText(3840, 2160, g.State).Size(16)}
		if got := faceSize(t, field.face); got != want.Size {
			t.Fatalf("chosen %d%%: the field's font is %v px, want %v", chosen, got, want.Size)
		}
		metrics := want.Metrics()
		line := int(metrics.HAscent + metrics.HDescent + 0.5)
		if _, height := field.PreferredSize(); height < line+16 {
			t.Fatalf("chosen %d%%: the field is %d px tall for a %d px line", chosen, height, line)
		}
		boundary := len("晨曦国")
		x := 10 + int(text.Advance(name[:boundary], want)+text.Advance(name[:boundary+3], want))/2 - 2
		field.pointerDown(x, false, time.Now())
		if field.caret != boundary {
			t.Fatalf("chosen %d%%: a click just left of rune 4's middle puts the caret at %d, want %d (%s)",
				chosen, field.caret, boundary, fmt.Sprint(x))
		}
	}
}

// panelButton finds a button in the panel by the start of its label, the way a player would.
func panelButton(t *testing.T, g *Game, prefix string) *widget.Button {
	t.Helper()
	for _, child := range g.host.content.Children() {
		if b, ok := child.(*widget.Button); ok {
			b.Validate()
			if b.Text() != nil && strings.HasPrefix(b.Text().Label, prefix) {
				return b
			}
		}
	}
	t.Fatalf("no button starting %q in the panel", prefix)
	return nil
}

func TestNativeTextSizeButtonCyclesAndRebuilds(t *testing.T) {
	g := chromeGame(t, 3840, 2160, ui.State{View: "company", CompanyPane: "settings"})
	if b := panelButton(t, g, "Text size"); b.Text().Label != "Text size · 100%" {
		t.Fatal("the setting starts at 100%, got", b.Text().Label)
	}
	for _, want := range []int{125, 150, 200, 100} {
		b := panelButton(t, g, "Text size")
		g.dirty = false
		b.ClickedEvent.Fire(&widget.ButtonClickedEventArgs{Button: b})
		event.ExecuteDeferred()
		if g.State.TextSize != want || !g.dirty {
			t.Fatalf("a click must choose %d%% and ask for a rebuild, got %d dirty=%v", want, g.State.TextSize, g.dirty)
		}
		if g.host.workspace.Step != want || g.Map.Text.Step != want {
			t.Fatalf("layout and map must follow at once: %d and %d, want %d", g.host.workspace.Step, g.Map.Text.Step, want)
		}
		g.host.build(g)
		if label := panelButton(t, g, "Text size").Text().Label; label != fmt.Sprintf("Text size · %d%%", want) {
			t.Fatal("the button must show the chosen step, got", label)
		}
	}
	g.State.TextSize = 200
	g.host.layout(g)
	g.host.build(g)
	if panelShows(g, "200% chosen · 200% fits this window") || panelShows(g, "200% chosen · 150% fits this window") {
		t.Fatal("no fallback note belongs in a window with room")
	}
}

func TestNativeTextSizeExplainsAFallback(t *testing.T) {
	g := chromeGame(t, 1280, 850, ui.State{View: "company", CompanyPane: "settings", TextSize: 200})
	if g.host.workspace.Step != 150 || !panelShows(g, "200% chosen · 150% fits this window") {
		t.Fatal("a window too small for the chosen step must say which step it shows", g.host.workspace.Step)
	}
	if label := panelButton(t, g, "Text size").Text().Label; label != "Text size · 200%" {
		t.Fatal("the button still shows what was chosen, got", label)
	}
	g = chromeGame(t, 1280, 850, ui.State{Locale: "zh-Hans", View: "company", CompanyPane: "settings", TextSize: 200})
	if !panelShows(g, "已选 200% · 此窗口可用 150%") {
		t.Fatal("the note must be translated")
	}
}

func TestNativeTextSizeSurvivesRestartAndBadValues(t *testing.T) {
	world := newChromeWorld(t)
	// A file from before the setting existed has no key; it loads unset, which means 100%.
	for stored, want := range map[int]int{150: 150, 200: 200, -5: 100, 1: 100, 137: 100, 100000: 100, 0: 0} {
		state := ui.NewGameState(ui.State{Locale: "en", TextSize: stored}, world.frame)
		raw, _ := json.Marshal(nativePreferences{State: state})
		if stored == 0 {
			raw = []byte(strings.Replace(string(raw), `"text_size":0,`, "", 1))
			if strings.Contains(string(raw), "text_size") {
				t.Fatal("the fixture must lack the key")
			}
		}
		if err := os.WriteFile(preferencesPath, raw, 0600); err != nil {
			t.Fatal(err)
		}
		g := &Game{Port: world.port, Frame: world.frame}
		g.host.init(g)
		if g.State.TextSize != want || ui.TextStep(g.State.TextSize) != max(want, 100) {
			t.Fatalf("stored %d must load as %d, got %d", stored, want, g.State.TextSize)
		}
		g.host.persist(g)
		saved, err := os.ReadFile(preferencesPath)
		if err != nil || !strings.Contains(string(saved), fmt.Sprintf(`"text_size":%d`, want)) {
			t.Fatalf("stored %d must be saved as %d: %v", stored, want, err)
		}
	}
}

func TestTextSizeStringsAreTranslated(t *testing.T) {
	for _, key := range []string{"textSize", "textSizeApplied"} {
		en, zh := ui.Text("en", key), ui.Text("zh-Hans", key)
		if en == key || zh == key || en == zh {
			t.Errorf("%s needs English and Chinese text, got %q and %q", key, en, zh)
		}
	}
}

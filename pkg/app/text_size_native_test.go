//go:build !js

package app

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

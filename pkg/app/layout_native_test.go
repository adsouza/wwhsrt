//go:build !js

package app

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/assets"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativePanelsFitAvailableWidth(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	port, err := application.Open(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"en", "zh-Hans"} {
		for _, dark := range []bool{false, true} {
			for _, width := range []int{320, 390, 800, 1200, 1920} {
				for _, view := range []string{"map", "build", "services", "report", "company"} {
					t.Run(fmt.Sprintf("%s/dark=%v/%d/%s", locale, dark, width, view), func(t *testing.T) {
						panel, mapWidth := 360, width-360
						if width < 800 {
							panel, mapWidth = width, width
						}
						g := &Game{
							Port:   port,
							Frame:  port.Snapshot(),
							width:  width,
							height: 844,
							Map:    &render.Map{Font: font},
							Camera: mapview.Camera{Width: float64(mapWidth), Height: 300},
							State: ui.State{
								Locale:          locale,
								View:            view,
								Dark:            dark,
								FromCity:        "shin_hakodate_hokuto",
								ToCity:          "sapporo",
								AlignmentID:     "mountain",
								ServiceCorridor: "tokyo-nagoya",
								Frequency:       6,
								Fare:            9000,
								LoanAmount:      "1000000",
								DraftName:       "晨曦国际高速铁路运输与基础设施开发公司",
							},
						}
						g.host.build(g)
						if w, _ := g.host.scroll.PreferredSize(); w > g.host.scroll.ViewRect().Dx() {
							t.Errorf(
								"sidebar content needs %d px but viewport has %d px",
								w,
								g.host.scroll.ViewRect().Dx(),
							)
						}
						footer := g.host.ui.Container.Children()[1]
						if w, _ := footer.PreferredSize(); w > panel {
							t.Errorf("footer needs %d px but panel has %d px", w, panel)
						}
					})
				}
			}
		}
	}
}

func TestNativeWrappedLabelsFitWithoutLosingText(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	compact := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
	}
	for _, locale := range []string{"en", "zh-Hans"} {
		for key, value := range ui.Catalog(locale) {
			for _, size := range []float64{15, 24} {
				face := &text.GoTextFace{Source: font, Size: size}
				measure := func(s string) float64 { return text.Advance(s, face) }
				wrapped := ui.WrapText(value, 260, measure)
				if compact(wrapped) != compact(value) {
					t.Fatalf("lost text in %s/%s", locale, key)
				}
				for _, line := range strings.Split(wrapped, "\n") {
					if w := measure(line); w > 260 {
						t.Errorf("%s/%s: %.1f px line exceeds 260 px: %q", locale, key, w, line)
					}
				}
			}
		}
	}
}

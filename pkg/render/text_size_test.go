package render

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/ui"
)

// The browser client never sets a text scale, so the zero value must draw what it draws today.
func TestMapTextZeroValueDrawsTodaysSizes(t *testing.T) {
	m := &Map{}
	for _, size := range []float64{9, 11, 12, 13, 15, 20} {
		if got := m.face(size).Size; got != size {
			t.Errorf("an unset scale changed %v px to %v", size, got)
		}
		if got := m.by(size, 22); got != 22 {
			t.Errorf("an unset scale changed a 22 px length beside %v px text to %v", size, got)
		}
	}
}

// The positions the legends had before text could be scaled, which an unset scale must reproduce.
func TestMapLegendsUnchangedWithoutAScale(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	m := &Map{Font: font}
	c := mapview.New(1400, 900)
	if top := m.legendTop(); top != 56 {
		t.Fatal("the legend rows start 56 px down, got", top)
	}
	rows := m.legendRows(c, ui.State{Locale: "en", View: "map"}, true, true)
	prospect := text.Advance(ui.Text("en", "prospectLegend"), &text.GoTextFace{Source: font, Size: 11}) + 48
	want := []labelRect{{12, 56, prospect, 22}, {12, 78, 285, 24}, {12, 102, 285, 24}}
	for i, row := range rows {
		if row.box != want[i] || row.textX != []float64{54, 52, 52}[i] {
			t.Errorf("legend row %d is %+v with text at %v, want %+v", i, row.box, row.textX, want[i])
		}
	}
	narrow := m.legendRows(mapview.New(200, 300), ui.State{Locale: "en", View: "services"}, true, false)
	if len(narrow) != 1 || narrow[0].box != (labelRect{12, 56, 176, 24}) {
		t.Fatalf("a narrow map clips the idle legend to its width less 24 px, got %+v", narrow)
	}
	legend := m.terrainLegend(c, "en")
	if legend.box != (labelRect{12, 0, 245, 52}) || legend.swatchY != 24 || legend.swatchW != 24 || legend.labelY != 33 {
		t.Fatalf("terrain legend is %+v", legend)
	}
	for i, x := range []float64{8, 93, 138} {
		if legend.labels[i].x != x {
			t.Errorf("terrain caption %d is at %v, want %v", i, legend.labels[i].x, x)
		}
	}
}

func TestMapTextUsesTheLadderAndStep(t *testing.T) {
	m := &Map{Text: ui.TextScale{Ladder: "desktop", Step: 150}}
	if small, city := m.face(11).Size, m.face(15).Size; small != 21 || city != 25.5 {
		t.Fatal("desktop at 150%: 11 px becomes 21 and 15 px becomes 25.5, got", small, city)
	}
	if got := m.by(11, 22); got != 42 {
		t.Fatal("a 22 px row for 11 px text grows with its text to 42, got", got)
	}
	m.Text = ui.TextScale{Ladder: "phone", Step: 100}
	if got := m.face(9).Size; got != 12 {
		t.Fatal("phone: 9 px becomes 12, got", got)
	}
}

func TestMapSceneRedrawsWhenTextChanges(t *testing.T) {
	c := mapview.New(800, 600)
	f := gameapi.Frame{Checkpoint: "accepted-world"}
	s := ui.State{Locale: "en", View: "map"}
	m := &Map{Text: ui.TextScale{Ladder: "desktop", Step: 100}}
	base := m.sceneKeyFor(c, f, s)
	for _, next := range []ui.TextScale{{Ladder: "desktop", Step: 150}, {Ladder: "phone", Step: 100}, {}} {
		m.Text = next
		if m.sceneKeyFor(c, f, s) == base {
			t.Errorf("the cached scene must be redrawn when the text scale becomes %+v", next)
		}
	}
}

func TestMapLegendBoxesHoldTheirText(t *testing.T) {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		t.Fatal(err)
	}
	c := mapview.New(1400, 900)
	for _, scale := range []ui.TextScale{
		{Ladder: "desktop", Step: 100}, {Ladder: "desktop", Step: 200}, {Ladder: "phone", Step: 125},
	} {
		for _, locale := range []string{"en", "zh-Hans"} {
			m := &Map{Font: font, Text: scale}
			s := ui.State{Locale: locale, View: "map"}
			rows := m.legendRows(c, s, true, true)
			if len(rows) != 3 {
				t.Fatalf("want the prospect row and both idle rows, got %d", len(rows))
			}
			for i, row := range rows {
				end := row.textX + text.Advance(row.label, m.face(row.size))
				if end > row.box.x+row.box.w+0.5 {
					t.Errorf("%+v %s: legend %q ends at %.1f, outside its box %+v", scale, locale, row.label, end, row.box)
				}
				if i > 0 && rows[i-1].box.y+rows[i-1].box.h > row.box.y+0.5 {
					t.Errorf("%+v %s: legend row %d overlaps the row above", scale, locale, i)
				}
			}
			legend := m.terrainLegend(c, locale)
			for i, label := range legend.labels {
				end := label.x + text.Advance(label.text, m.face(9))
				if i+1 < len(legend.labels) && end > legend.labels[i+1].x {
					t.Errorf("%+v %s: terrain legend %q runs into %q", scale, locale, label.text, legend.labels[i+1].text)
				}
				if end > legend.box.w+0.5 {
					t.Errorf("%+v %s: terrain legend %q ends at %.1f in a %.1f px box", scale, locale, label.text, end, legend.box.w)
				}
			}
			title := text.Advance(ui.Text(locale, "terrainLegend"), m.face(11))
			if m.by(11, 8)+title > legend.box.w+0.5 {
				t.Errorf("%+v %s: the terrain legend's title does not fit its box", scale, locale)
			}
		}
	}
}

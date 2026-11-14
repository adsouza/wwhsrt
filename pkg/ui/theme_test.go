package ui

import (
	"image/color"
	"math"
	"testing"
)

func TestAppearanceOverridesAndSystemFallback(t *testing.T) {
	for _, row := range []struct {
		mode             string
		systemDark, want bool
	}{
		{"", true, true}, {"system", false, false}, {"system", true, true},
		{"dark", false, true}, {"light", true, false}, {"old-unknown-value", false, false},
	} {
		if got := IsDark(State{Appearance: row.mode, Dark: row.systemDark}); got != row.want {
			t.Fatal(row, got)
		}
	}
}

func TestThemeTextContrast(t *testing.T) {
	luminance := func(c color.NRGBA) float64 {
		channel := func(v uint8) float64 {
			n := float64(v) / 255
			if n <= .04045 {
				return n / 12.92
			}
			return math.Pow((n+.055)/1.055, 2.4)
		}
		return .2126*channel(c.R) + .7152*channel(c.G) + .0722*channel(c.B)
	}
	for _, mode := range []string{"light", "dark"} {
		p := PaletteFor(State{Appearance: mode})
		pairs := map[string][2]color.NRGBA{
			"body":          {p.Text, p.Background},
			"panel":         {p.Text, p.Surface},
			"control":       {p.Text, p.Control},
			"hover":         {p.Text, p.Hover},
			"muted":         {p.Muted, p.Surface},
			"selected":      {p.Accent, p.Selected},
			"primary":       {p.OnAction, p.Action},
			"primary-hover": {p.OnAction, p.ActionHover},
			"warning":       {p.WarningText, p.Warning},
			"warning-hover": {p.WarningText, p.WarningHover},
			"terrain-label": {p.TerrainText, p.TerrainFill},
			"map-label":     {p.Text, p.Land},
		}
		for name, pair := range pairs {
			a, b := luminance(pair[0]), luminance(pair[1])
			if a < b {
				a, b = b, a
			}
			if ratio := (a + .05) / (b + .05); ratio < 4.5 {
				t.Errorf("%s %s: %.2f contrast, need 4.5", mode, name, ratio)
			}
		}
	}
}

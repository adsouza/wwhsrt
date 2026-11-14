package ui

import "slices"

// TextSteps are the text sizes a player can choose, in percent.
var TextSteps = []int{100, 125, 150, 200}

// TextStep normalises a stored text size; anything that is not a step means 100%.
func TextStep(percent int) int {
	if slices.Contains(TextSteps, percent) {
		return percent
	}
	return 100
}

func NextTextStep(percent int) int {
	i := slices.Index(TextSteps, TextStep(percent))
	return TextSteps[(i+1)%len(TextSteps)]
}

// AppliedTextStep is the largest step up to the chosen one that still leaves the smallest window the
// app allows (320×480) once the window is divided by it.
func AppliedTextStep(width, height, chosen int) int {
	applied := 100
	for _, step := range TextSteps {
		if step <= TextStep(chosen) && width*100/step >= 320 && height*100/step >= 480 {
			applied = step
		}
	}
	return applied
}

// TextScale turns the sizes written at call sites into the sizes drawn. Its zero value changes nothing,
// which is what the browser client uses.
type TextScale struct {
	Ladder string // "" sizes as written, "desktop", "phone"
	Step   int    // percent; 0 means 100
}

func (t TextScale) step() int {
	if t.Step == 0 {
		return 100
	}
	return t.Step
}

// Size applies the ladder's floor and bump, then the step.
func (t TextScale) Size(base float64) float64 {
	switch t.Ladder {
	case "desktop":
		if base < 14 {
			base = 14
		} else {
			base += 2
		}
	case "phone":
		base = max(base, 12)
	}
	return base * float64(t.step()) / 100
}

// Px scales a layout length with the step, rounding half up.
func (t TextScale) Px(n int) int { return (n*t.step() + 50) / 100 }

// NativeText is the scale for a native window: a step behaves as a window 1/step the size, and the
// ladder follows that window's layout (compact below 800 px).
func NativeText(width, height int, s State) TextScale {
	step := AppliedTextStep(width, height, s.TextSize)
	if width*100/step < 800 {
		return TextScale{Ladder: "phone", Step: step}
	}
	return TextScale{Ladder: "desktop", Step: step}
}

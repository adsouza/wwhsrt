package ui

import (
	"testing"

	"wwhsrt/pkg/gameapi"
)

func TestTextLadders(t *testing.T) {
	bases := []float64{9, 11, 12, 13, 14, 15, 16, 17, 18, 20}
	desktop := []float64{14, 14, 14, 14, 16, 17, 18, 19, 20, 22}
	phone := []float64{12, 12, 12, 13, 14, 15, 16, 17, 18, 20}
	for _, step := range []int{0, 100} {
		for i, base := range bases {
			if got := (TextScale{Ladder: "desktop", Step: step}).Size(base); got != desktop[i] {
				t.Errorf("desktop step %d: %v px became %v, want %v", step, base, got, desktop[i])
			}
			if got := (TextScale{Ladder: "phone", Step: step}).Size(base); got != phone[i] {
				t.Errorf("phone step %d: %v px became %v, want %v", step, base, got, phone[i])
			}
		}
	}
}

func TestTextScaleZeroValueLeavesSizesAlone(t *testing.T) {
	for _, size := range []float64{9, 11, 13.5, 20} {
		if got := (TextScale{}).Size(size); got != size {
			t.Errorf("the zero scale changed %v px to %v", size, got)
		}
	}
	for _, n := range []int{0, 1, 44, 164} {
		if got := (TextScale{}).Px(n); got != n {
			t.Errorf("the zero scale changed %d px to %d", n, got)
		}
	}
}

func TestTextStepMultiplies(t *testing.T) {
	for i, step := range []int{125, 150, 200} {
		scale := TextScale{Ladder: "desktop", Step: step}
		if got, want := scale.Size(14), []float64{20, 24, 32}[i]; got != want {
			t.Errorf("step %d: body text is %v px, want %v", step, got, want)
		}
		if got, want := scale.Px(44), []int{55, 66, 88}[i]; got != want {
			t.Errorf("step %d: 44 px became %d, want %d", step, got, want)
		}
	}
	if got := (TextScale{Step: 150}).Px(3); got != 5 {
		t.Errorf("3 px at 150%% must round half up to 5, got %d", got)
	}
}

func TestTextStepNormalisesAndCycles(t *testing.T) {
	for _, bad := range []int{-5, 0, 1, 137, 100000} {
		if got := TextStep(bad); got != 100 {
			t.Errorf("TextStep(%d) = %d, want 100", bad, got)
		}
	}
	for _, step := range []int{125, 150, 200} {
		if got := TextStep(step); got != step {
			t.Errorf("TextStep(%d) = %d", step, got)
		}
	}
	step := 100
	for _, want := range []int{125, 150, 200, 100} {
		if step = NextTextStep(step); step != want {
			t.Fatalf("the steps must cycle 100, 125, 150, 200: got %d, want %d", step, want)
		}
	}
	if got := NextTextStep(137); got != 125 {
		t.Errorf("an unknown step counts as 100, so the next is 125: got %d", got)
	}
}

func TestAppliedTextStepNeedsRoom(t *testing.T) {
	for _, c := range [][4]int{
		{3840, 2160, 200, 200}, {1920, 1080, 200, 200}, {1280, 850, 200, 150}, {1280, 850, 150, 150},
		{800, 600, 125, 125}, {800, 600, 150, 125}, {390, 844, 200, 100}, {320, 480, 125, 100},
	} {
		if got := AppliedTextStep(c[0], c[1], c[2]); got != c[3] {
			t.Errorf("%dx%d choosing %d%%: applied %d%%, want %d%%", c[0], c[1], c[2], got, c[3])
		}
	}
}

func TestNativeTextFollowsTheLayout(t *testing.T) {
	for _, c := range []struct {
		w, h, chosen int
		want         TextScale
	}{
		{1280, 850, 0, TextScale{"desktop", 100}},
		{799, 850, 0, TextScale{"phone", 100}},
		{1500, 1100, 200, TextScale{"phone", 200}},
		{1920, 1080, 200, TextScale{"desktop", 200}},
	} {
		if got := NativeText(c.w, c.h, State{TextSize: c.chosen}); got != c.want {
			t.Errorf("%dx%d choosing %d%%: %+v, want %+v", c.w, c.h, c.chosen, got, c.want)
		}
	}
}

func TestNewGameKeepsTextSize(t *testing.T) {
	if got := NewGameState(State{TextSize: 150}, gameapi.Frame{}).TextSize; got != 150 {
		t.Fatalf("text size is a device preference and must survive New game, got %d", got)
	}
}

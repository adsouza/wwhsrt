package render

import (
	"testing"

	"wwhsrt/pkg/mapview"
)

// A Map built without scenario anchors (as widget tests do) must still frame
// the default region instead of panicking.
func TestCameraWithoutAnchorsFallsBackToDefaultFrame(t *testing.T) {
	for _, portrait := range []bool{false, true} {
		got := (&Map{}).Camera(640, 480, portrait)
		if want := mapview.New(640, 480); got != want {
			t.Fatal("want the default frame", portrait, got, want)
		}
	}
}

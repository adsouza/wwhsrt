//go:build !js

package app

import (
	"bytes"
	"image"
	"path/filepath"
	"testing"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

func TestNativeReplayPlaySurvivesSliderRenderAndRebuild(t *testing.T) {
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
	before, _ := port.Export()
	for _, locale := range []string{"en", "zh-Hans"} {
		g := &Game{Port: port, Frame: port.Snapshot(), Map: m, width: 1280, height: 900,
			State: ui.State{Locale: locale, View: "map", Motion: true, Watching: true, ReplayMinute: 113}}
		build := func() (*widget.Slider, map[string]func()) {
			content := widget.NewContainer()
			buttons := map[string]func(){}
			g.host.replayControls(g, func(string, float64) {}, func(s string, fn func()) *widget.Button {
				buttons[s] = fn
				return widget.NewButton()
			}, content)
			return content.Children()[0].(*widget.Slider), buttons
		}
		slider, buttons := build()
		oldSlider := slider
		buttons[ui.Text(locale, "playDay")]()
		if !g.State.ReplayPlaying {
			t.Fatal("Play did not start")
		}
		// Native panel rebuild after clicking Play: the first real slider render
		// must not emit a seek that pauses the newly started replay.
		slider, buttons = build()
		slider.Validate()
		slider.SetLocation(image.Rect(0, 0, 300, 44))
		slider.Render(ebiten.NewImage(320, 60))
		event.ExecuteDeferred()
		if !g.State.ReplayPlaying {
			t.Fatal("initial slider render paused playback")
		}
		oldSlider.ChangedEvent.Fire(&widget.SliderChangedEventArgs{Slider: oldSlider, Current: 900})
		event.ExecuteDeferred()
		if !g.State.ReplayPlaying || g.State.ReplayMinute != 113 {
			t.Fatal("deferred event from a replaced slider interrupted playback")
		}
		g.replayTick = 120
		if g.replayTime() != 133 {
			t.Fatal("replay clock did not advance at ten minutes per second")
		}
		// Unrelated native panel rebuilds must also retain elapsed replay time.
		slider, buttons = build()
		slider.Validate()
		slider.SetLocation(image.Rect(0, 0, 300, 44))
		slider.Render(ebiten.NewImage(320, 60))
		event.ExecuteDeferred()
		if !g.State.ReplayPlaying || g.replayTime() != 133 {
			t.Fatal("panel rebuild reset elapsed playback")
		}
		buttons[ui.Text(locale, "pauseDay")]()
		if g.State.ReplayPlaying || g.State.ReplayMinute != 133 {
			t.Fatal("pause lost displayed time")
		}
		slider.Current = 500
		slider.Render(ebiten.NewImage(320, 60))
		event.ExecuteDeferred()
		if g.State.ReplayPlaying || g.State.ReplayMinute != 500 {
			t.Fatal("scrub did not seek and pause")
		}
		_, buttons = build()
		buttons[ui.Text(locale, "playDay")]()
		if !g.State.ReplayPlaying || g.replayTime() != 500 {
			t.Fatal("resume did not start at the scrubbed time")
		}
	}
	after, _ := port.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("replay changed campaign")
	}
}

//go:build !js

package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"wwhsrt/internal/application"
	"wwhsrt/pkg/ui"
)

func TestNativeWindowRestoreFitsAvailableMonitor(t *testing.T) {
	monitors := []monitorInfo{{"primary", 1920, 1080}, {"external", 2560, 1440}}
	w := windowPlacement{Width: 900, Height: 700, X: 150, Y: 110, MonitorName: "external", MonitorIndex: 1}
	if windowMonitor(&w, monitors) != 1 || fitWindow(w, 2560, 1440) != w {
		t.Fatal("valid placement changed")
	}
	if windowMonitor(&w, []monitorInfo{monitors[1], monitors[0]}) != 0 {
		t.Fatal("reordered monitor not found by name")
	}
	if windowMonitor(&w, monitors[:1]) != 0 {
		t.Fatal("missing monitor did not fall back")
	}
	w.Width, w.Height, w.X, w.Y = 5000, 3000, 4000, -800
	got := fitWindow(w, 1280, 800)
	if got.Width != 1280 || got.Height != 736 || got.X != 0 || got.Y != 0 {
		t.Fatal("oversized/offscreen placement not recovered", got)
	}
	for _, bounds := range [][2]int{{0, 0}, {-1, 1}, {1000000, 800}, {800, 1000000}} {
		w.Width, w.Height = bounds[0], bounds[1]
		got = fitWindow(w, 1920, 1080)
		if got.Width != 1200 || got.Height != 800 {
			t.Fatal("invalid size did not use defaults", got)
		}
	}
	duplicates := []monitorInfo{{"Generic PnP Monitor", 1920, 1080}, {"Generic PnP Monitor", 1920, 1080}}
	w.MonitorName, w.MonitorIndex = duplicates[0].name, 1
	if windowMonitor(&w, duplicates) != 1 {
		t.Fatal("duplicate monitor names lost index")
	}
}

func TestNativeWindowPreferencesPreserveLegacyUIAndFailedWrites(t *testing.T) {
	oldPath := preferencesPath
	preferencesPath = filepath.Join(t.TempDir(), "ui.json")
	t.Cleanup(func() { preferencesPath = oldPath })
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	frame := port.Snapshot()
	state := ui.NewGameState(ui.State{Locale: "zh-Hans", Motion: false, HideTerrain: true, Appearance: "dark", Dark: true}, frame)
	state.DraftName, state.View, state.HintDismissed = "Unsaved company draft", "company", true
	legacy, _ := json.Marshal(state)
	if err := os.WriteFile(preferencesPath, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	g := &Game{Port: port, Frame: frame}
	g.host.init(g)
	ui.NormalizeHelp(&state)
	if !reflect.DeepEqual(g.State, state) || g.host.window != nil {
		t.Fatal("legacy UI preferences changed")
	}
	w := windowPlacement{Width: 940, Height: 710, X: 180, Y: 120, MonitorName: "external", MonitorIndex: 1}
	g.host.observeWindow(w, time.Now())
	g.FlushWindowPreferences()
	durable, err := os.ReadFile(preferencesPath)
	if err != nil {
		t.Fatal(err)
	}
	reloaded := &Game{Port: port, Frame: frame}
	reloaded.host.init(reloaded)
	if !reflect.DeepEqual(reloaded.State, state) || reloaded.host.window == nil || *reloaded.host.window != w || g.host.windowDirty {
		t.Fatal("placement/UI did not round trip")
	}
	// A new campaign resets local drafts while preserving native placement.
	reloaded.State = ui.NewGameState(reloaded.State, frame)
	reloaded.host.persist(reloaded)
	b, _ := os.ReadFile(preferencesPath)
	var prefs nativePreferences
	if err := json.Unmarshal(b, &prefs); err != nil || prefs.Window == nil || *prefs.Window != w || prefs.DraftName != frame.Player.Name {
		t.Fatal("new game lost placement", err)
	}
	// A failed write remains pending and cannot affect the original file/game.
	path := preferencesPath
	preferencesPath = filepath.Join(path, "cannot-create-under-file.json")
	w.X = 220
	g.host.observeWindow(w, time.Now())
	g.FlushWindowPreferences()
	if !g.host.windowDirty || g.host.message != "storage_failed" || port.Snapshot().Checkpoint != frame.Checkpoint {
		t.Fatal("failed preferences write changed campaign or lost pending placement")
	}
	still, _ := os.ReadFile(path)
	if !bytes.Equal(still, b) {
		t.Fatal("failed preferences write damaged previous preferences")
	}
	preferencesPath = path
	g.FlushWindowPreferences()
	if g.host.windowDirty {
		t.Fatal("failed placement save could not be retried")
	}
	if bytes.Equal(durable, still) {
		t.Fatal("fixture did not reset drafts")
	}
}

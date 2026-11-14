//go:build !js

package app

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"wwhsrt/pkg/ui"
)

// Native-only fields extend the existing flat ui.json; older preferences still
// decode, and window placement never enters a campaign snapshot or browser UI.
type nativePreferences struct {
	ui.State
	Window *windowPlacement `json:"window,omitempty"`
}

type windowPlacement struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	MonitorName  string `json:"monitor_name"`
	MonitorIndex int    `json:"monitor_index"`
	Maximized    bool   `json:"maximized"`
}

type monitorInfo struct {
	name          string
	width, height int
}

func windowMonitor(w *windowPlacement, monitors []monitorInfo) int {
	if w == nil {
		return 0
	}
	if i := w.MonitorIndex; i >= 0 && i < len(monitors) && monitors[i].name == w.MonitorName {
		return i
	}
	for i, m := range monitors {
		if m.name == w.MonitorName {
			return i
		}
	}
	return 0 // Disconnected display: recover on the primary monitor.
}

func fitWindow(w windowPlacement, screenWidth, screenHeight int) windowPlacement {
	if w.Width < 320 || w.Height < 480 || w.Width > 32768 || w.Height > 32768 {
		w.Width, w.Height = 1200, 800
	}
	if screenWidth > 0 && screenHeight > 0 {
		// Leave vertical space for native decorations, menu bars and taskbars.
		w.Width = min(w.Width, max(320, screenWidth))
		w.Height = min(w.Height, max(480, screenHeight-64))
		w.X = max(0, min(w.X, max(0, screenWidth-w.Width)))
		w.Y = max(0, min(w.Y, max(0, screenHeight-w.Height-32)))
	}
	return w
}

// ConfigureWindow runs on the main thread before RunGame, as required by the
// monitor APIs. Positions and sizes are monitor-relative device-independent px.
func (g *Game) ConfigureWindow() {
	g.host.refreshAppearance(g)
	monitors := ebiten.AppendMonitors(nil)
	infos := make([]monitorInfo, len(monitors))
	for i, m := range monitors {
		w, h := m.Size()
		infos[i] = monitorInfo{m.Name(), w, h}
	}
	w := windowPlacement{Width: 1200, Height: 800}
	if g.host.window != nil {
		w = *g.host.window
	}
	i := windowMonitor(g.host.window, infos)
	if len(monitors) > 0 {
		ebiten.SetMonitor(monitors[i])
		w = fitWindow(w, infos[i].width, infos[i].height)
		w.MonitorName, w.MonitorIndex = infos[i].name, i
	} else {
		w = fitWindow(w, 0, 0)
	}
	ebiten.SetWindowSize(w.Width, w.Height)
	if g.host.window != nil {
		ebiten.SetWindowPosition(w.X, w.Y)
	}
	if w.Maximized {
		ebiten.MaximizeWindow()
	}
	// Seed normal bounds before a restored maximized window produces its first
	// frame; maximized dimensions must never replace the normal restore rectangle.
	g.host.observeWindow(w, time.Now())
	ebiten.SetWindowClosingHandled(true)
	g.host.windowConfigured = true
}

func (h *host) observeWindow(w windowPlacement, now time.Time) {
	if h.window == nil || *h.window != w {
		copy := w
		h.window = &copy
		h.windowDirty = true
		h.windowChanged = now
	}
}

func (h *host) windowUpdate(g *Game) bool {
	if !h.windowConfigured {
		return false
	}
	h.windowClosing = h.windowClosing || ebiten.IsWindowBeingClosed()
	now := time.Now()
	if h.windowClosing || now.Sub(h.windowPoll) >= 100*time.Millisecond {
		h.windowPoll = now
		// Keep the last normal bounds during minimization/fullscreen. Maximized
		// windows retain those bounds too, plus an explicit maximize preference.
		if !ebiten.IsWindowMinimized() && !ebiten.IsFullscreen() {
			m := ebiten.Monitor()
			w, height := ebiten.WindowSize()
			if w >= 320 && height >= 480 && m != nil {
				x, y := ebiten.WindowPosition()
				placement := windowPlacement{
					Width:       w,
					Height:      height,
					X:           x,
					Y:           y,
					MonitorName: m.Name(),
					Maximized:   ebiten.IsWindowMaximized(),
				}
				for i, monitor := range ebiten.AppendMonitors(nil) {
					if monitor == m {
						placement.MonitorIndex = i
						break
					}
				}
				if placement.Maximized && h.window != nil {
					placement.Width, placement.Height = h.window.Width, h.window.Height
					placement.X, placement.Y = h.window.X, h.window.Y
				}
				h.observeWindow(placement, now)
			}
		}
	}
	if h.windowDirty &&
		(h.windowClosing || now.Sub(h.windowChanged) >= 400*time.Millisecond && now.Sub(h.windowAttempt) >= time.Second) {
		h.windowAttempt = now
		message := h.message
		h.persist(g)
		if h.message != message {
			g.dirty = true
		}
	}
	// Let an already accepted business command finish before releasing its lock.
	return h.windowClosing && !h.busy
}

// FlushWindowPreferences saves the last observed bounds after RunGame returns;
// it deliberately avoids querying a window that Ebitengine has already destroyed.
func (g *Game) FlushWindowPreferences() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.host.windowDirty {
		g.host.persist(g)
	}
}

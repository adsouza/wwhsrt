//go:build js && wasm

package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/mapview"
	"wwhsrt/pkg/render"
	"wwhsrt/pkg/ui"
)

type browserStore struct{}

func await(p js.Value) (js.Value, error) {
	type result struct {
		value js.Value
		err   error
	}
	ch := make(chan result, 1)
	ok := js.FuncOf(func(_ js.Value, a []js.Value) any { ch <- result{value: a[0]}; return nil })
	bad := js.FuncOf(func(_ js.Value, a []js.Value) any {
		ch <- result{err: fmt.Errorf("browser storage unavailable")}
		return nil
	})
	p.Call("then", ok).Call("catch", bad)
	r := <-ch
	ok.Release()
	bad.Release()
	return r.value, r.err
}
func (browserStore) Load() ([]byte, error) {
	v, e := await(js.Global().Get("parent").Call("railStorageLoad"))
	if e != nil {
		return nil, e
	}
	return []byte(v.String()), nil
}
func (browserStore) Commit(b []byte) error {
	_, e := await(js.Global().Get("parent").Call("railStorageCommit", string(b)))
	return e
}
func OpenHost() (gameapi.Game, func(), error) {
	g, e := application.Open(browserStore{}, slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	return g, func() {}, e
}
func StartupError(e error) { js.Global().Get("parent").Call("railStartupError", e.Error()) }

type request struct {
	kind, body string
	resolve    js.Value
}
type host struct {
	queue  chan request
	bridge js.Func
}

func (h *host) init(g *Game) {
	h.queue = make(chan request, 32)
	h.bridge = js.FuncOf(func(_ js.Value, args []js.Value) any {
		kind, body := args[0].String(), args[1].String()
		var executor js.Func
		executor = js.FuncOf(func(_ js.Value, a []js.Value) any {
			select {
			case h.queue <- request{kind, body, a[0]}:
			default:
				a[0].Invoke(`{"error":"busy"}`)
			}
			return nil
		})
		p := js.Global().Get("Promise").New(executor)
		executor.Release()
		return p
	})
	js.Global().Set("rail", h.bridge)
	go h.run(g)
	js.Global().Get("parent").Call("railReady")
}
func (h *host) snapshot(g *Game) map[string]any {
	points := map[string]mapview.Point{}
	for _, city := range g.Frame.Cities {
		points[string(city.ID)] = render.CityPoint(g.Camera, city)
	}
	return map[string]any{"frame": g.Frame, "state": g.State, "texts": ui.Catalog(g.State.Locale), "camera": g.Camera, "points": points, "palette": ui.PaletteFor(g.State)}
}
func (h *host) reply(g *Game, r request, err error) {
	v := h.snapshot(g)
	if err != nil {
		v["error"] = err.Error()
	}
	b, _ := json.Marshal(v)
	r.resolve.Invoke(string(b))
}

// Commands keep working while the map iframe is hidden, backgrounded, or
// suspended. The renderer loop is never the application's scheduler.
func (h *host) run(g *Game) {
	for r := range h.queue {
		if r.kind == "command" {
			var c gameapi.Command
			var f gameapi.Frame
			var e error
			if json.Unmarshal([]byte(r.body), &c) != nil {
				f = g.Port.Snapshot()
				e = &gameapi.Error{Code: "unknown_command"}
			} else {
				f, e = g.Port.Apply(c)
			}
			g.mu.Lock()
			g.Frame = f
			if e == nil && c.Kind == gameapi.NewGame {
				g.newGamePresentation()
			}
			h.reply(g, r, e)
			g.mu.Unlock()
			continue
		}
		g.mu.Lock()
		switch r.kind {
		case "forecast":
			var q gameapi.ForecastRequest
			var forecast gameapi.Forecast
			var e error
			if json.Unmarshal([]byte(r.body), &q) != nil {
				e = &gameapi.Error{Code: "invalid_service"}
			} else {
				forecast, e = g.Port.Forecast(q)
			}
			v := map[string]any{"forecast": forecast}
			if e != nil {
				v["error"] = e.Error()
			}
			b, _ := json.Marshal(v)
			r.resolve.Invoke(string(b))
		case "guidance":
			var state ui.State
			_ = json.Unmarshal([]byte(r.body), &state)
			b, _ := json.Marshal(ui.HelpFor(g.Frame, state))
			r.resolve.Invoke(string(b))
		case "ui":
			var s ui.State
			if json.Unmarshal([]byte(r.body), &s) == nil {
				s.Locale = ui.Locale(s.Locale)
				s.Appearance = ui.AppearanceMode(s.Appearance)
				s.Dark = ui.IsDark(s)
				if s.Portrait != g.State.Portrait {
					g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, s.Portrait)
				}
				g.State = s
			}
			h.reply(g, r, nil)
		case "fit":
			g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, g.State.Portrait)
			h.reply(g, r, nil)
		case "focus_route":
			g.focusSelected()
			h.reply(g, r, nil)
		case "zoom_in":
			g.Camera.ZoomAt(1.3, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2})
			h.reply(g, r, nil)
		case "zoom_out":
			g.Camera.ZoomAt(1/1.3, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2})
			h.reply(g, r, nil)
		case "export":
			b, e := g.Port.Export()
			v := h.snapshot(g)
			v["save"] = string(b)
			if e != nil {
				v["error"] = e.Error()
			}
			out, _ := json.Marshal(v)
			r.resolve.Invoke(string(out))
		case "snapshot":
			h.reply(g, r, nil)
		default:
			h.reply(g, r, &gameapi.Error{Code: "unknown_command"})
		}
		g.mu.Unlock()
	}
}
func (h *host) update(g *Game)  {}
func (h *host) blocksMap() bool { return false }
func (h *host) changed(g *Game) {
	b, _ := json.Marshal(h.snapshot(g))
	js.Global().Get("parent").Call("railMapChanged", string(b))
}
func (h *host) persist(g *Game) {} // Campaign commits use IndexedDB; camera is presentation only.
func (h *host) layout(g *Game) {
	g.Camera.Width = float64(g.width)
	g.Camera.Height = float64(g.height)
	if g.Camera.Zoom == 0 {
		g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, g.State.Portrait)
	}
}
func (h *host) draw(g *Game, screen *ebiten.Image) {
	g.Map.Draw(screen, g.Camera, g.Frame, g.State, g.tick)
}

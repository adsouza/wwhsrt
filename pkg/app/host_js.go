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
	queue                         chan request
	bridge                        js.Func
	helpLoaded                    bool
	viewportWidth, viewportHeight int
	sentFrame                     bool
	sentRevision                  uint64
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

// The frame carries the whole static catalog (megabytes of geometry and
// routes). It is sent when its revision changes or when full is set; the page
// keeps the last frame it received and reuses it for camera and UI replies.
func (h *host) snapshot(g *Game, full bool) map[string]any {
	points := map[string]mapview.Point{}
	for _, city := range g.Frame.Cities {
		points[string(city.ID)] = render.CityPoint(g.Camera, city)
	}
	v := map[string]any{
		"source_station": g.Map.PublicStop(g.State.SourceStation),
		"regions":        mapview.VisibleRegions(g.State.Portrait),
		"workspace":      ui.Workspace(h.viewportWidth, h.viewportHeight, g.State),
		"state":          g.State,
		"texts":          ui.Catalog(g.State.Locale),
		"camera":         g.Camera,
		"points":         points,
		"palette":        ui.PaletteFor(g.State),
	}
	if full || !h.sentFrame || h.sentRevision != g.Frame.Revision {
		v["frame"] = g.Frame
		v["prospects"] = ui.RankedProspects(g.Frame)
		v["next_service"] = ui.NextService(g.Frame)
		h.sentFrame, h.sentRevision = true, g.Frame.Revision
	}
	return v
}
func (h *host) reply(g *Game, r request, err error, full bool) {
	v := h.snapshot(g, full)
	if err != nil {
		v["error"] = err.Error()
		if capacity, ok := err.(*gameapi.Error); ok {
			v["capacity_usage"] = capacity.CapacityUsage
		}
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
			if e == nil && c.Kind != gameapi.NewGame {
				ui.RecordHelpAction(&g.State, f, c.Kind)
			}
			if e == nil && c.Kind == gameapi.NewGame {
				g.newGamePresentation()
			}
			h.reply(g, r, e, true)
			g.mu.Unlock()
			continue
		}
		g.mu.Lock()
		switch r.kind {
		case "best_frequency":
			var q gameapi.ForecastRequest
			_ = json.Unmarshal([]byte(r.body), &q)
			n, ok := ui.BestFrequency(g.Frame, q, g.Port.Forecast)
			b, _ := json.Marshal(map[string]any{"frequency": n, "available": ok})
			r.resolve.Invoke(string(b))
		case "forecast":
			var q gameapi.ForecastRequest
			var forecast gameapi.Forecast
			var e error
			if json.Unmarshal([]byte(r.body), &q) != nil {
				e = &gameapi.Error{Code: "invalid_service"}
			} else {
				forecast, e = g.Port.Forecast(q)
			}
			v := map[string]any{
				"forecast":         forecast,
				"capacity_details": ui.CapacityDetails(g.Frame, g.State.Locale, forecast.CapacityUsage),
				"capacity_summary": ui.CapacitySummary(g.Frame, g.State.Locale, forecast.CapacityUsage),
			}
			if e != nil {
				v["error"] = e.Error()
			}
			b, _ := json.Marshal(v)
			r.resolve.Invoke(string(b))
		case "help", "help_event":
			var q struct {
				State ui.State     `json:"state"`
				Event ui.HelpEvent `json:"event"`
				Issue string       `json:"issue"`
			}
			_ = json.Unmarshal([]byte(r.body), &q)
			q.State.Help = g.State.Help
			if r.kind == "help_event" {
				ui.ApplyHelpEvent(&q.State, q.Event)
				g.State.Help = q.State.Help
			}
			var forecast *gameapi.Forecast
			if q.State.ServiceCorridor != "" && (q.State.View == "services" || q.State.Help.Open) {
				f, e := g.Port.Forecast(
					gameapi.ForecastRequest{
						CorridorID:       q.State.ServiceCorridor,
						Frequency:        q.State.Frequency,
						Fare:             q.State.Fare,
						ReplaceServiceID: q.State.ReplaceServiceID,
					},
				)
				if e == nil || q.State.Help.Topic == "capacity" {
					forecast = &f
				}
				if e != nil {
					q.Issue = e.Error()
				}
			}
			b, _ := json.Marshal(
				map[string]any{"model": ui.BuildHelpModel(g.Frame, q.State, forecast, q.Issue), "state": q.State},
			)
			r.resolve.Invoke(string(b))
		case "guidance":
			var state ui.State
			_ = json.Unmarshal([]byte(r.body), &state)
			b, _ := json.Marshal(ui.HelpFor(g.Frame, state))
			r.resolve.Invoke(string(b))
		case "ui":
			var s ui.State
			var viewport struct {
				Width  int `json:"viewport_width"`
				Height int `json:"viewport_height"`
			}
			_ = json.Unmarshal([]byte(r.body), &viewport)
			if viewport.Width > 0 && viewport.Height > 0 {
				h.viewportWidth = viewport.Width
				h.viewportHeight = viewport.Height
			}
			if json.Unmarshal([]byte(r.body), &s) == nil {
				if s.ReplayMinute != g.State.ReplayMinute || s.Watching != g.State.Watching ||
					s.ReplayPlaying != g.State.ReplayPlaying {
					g.replayTick = 0
				}
				s.ReplayMinute = max(0, min(1439, s.ReplayMinute))
				s.Locale = ui.Locale(s.Locale)
				s.Appearance = ui.AppearanceMode(s.Appearance)
				s.Dark = ui.IsDark(s)
				if !mapview.KnownRegion(s.Region) {
					s.Region = "" // the default, East Asia
				}
				if !h.helpLoaded || s.Portrait != g.State.Portrait {
					previous := g.State.Region
					g.State.Region = s.Region
					g.Camera = g.regionCamera(s.Portrait)
					g.State.Region = previous
				}
				if h.helpLoaded {
					s.Help = g.State.Help
				} else {
					ui.NormalizeHelp(&s)
					h.helpLoaded = true
				}
				g.State = s
			}
			h.reply(g, r, nil, false)
		case "market":
			var q struct {
				From, To gameapi.CityID
				Class    string
			}
			_ = json.Unmarshal([]byte(r.body), &q)
			var value gameapi.MarketRun
			var err error
			if port, ok := g.Port.(gameapi.MarketInspector); ok {
				value, err = port.InspectMarket(q.From, q.To, q.Class)
			} else {
				err = &gameapi.Error{Code: "no_route"}
			}
			out := map[string]any{"market": value}
			if err != nil {
				out["error"] = err.Error()
			}
			bytes, _ := json.Marshal(out)
			r.resolve.Invoke(string(bytes))
		case "region":
			var q struct{ ID string }
			_ = json.Unmarshal([]byte(r.body), &q)
			g.State.Region = q.ID
			if q.ID == "east-asia" || q.ID == "" {
				g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, g.State.Portrait)
			} else {
				g.Camera = mapview.RegionCamera(g.Camera.Width, g.Camera.Height, q.ID, g.State.Portrait)
			}
			h.reply(g, r, nil, false)
		case "replay":
			b, _ := json.Marshal(
				map[string]any{
					"minute":      g.replayTime(),
					"description": render.TrainDescription(g.Camera, g.Frame, g.State, g.replayTime()),
				},
			)
			r.resolve.Invoke(string(b))
		case "fit":
			if g.State.Region != "" && g.State.Region != "east-asia" {
				g.Camera = mapview.RegionCamera(g.Camera.Width, g.Camera.Height, g.State.Region, g.State.Portrait)
			} else {
				g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, g.State.Portrait)
			}
			h.reply(g, r, nil, false)
		case "focus_idle":
			var q struct {
				CorridorID string `json:"corridor_id"`
			}
			_ = json.Unmarshal([]byte(r.body), &q)
			g.focusIdle(q.CorridorID)
			h.reply(g, r, nil, false)
		case "focus_route":
			g.focusSelected()
			h.reply(g, r, nil, false)
		case "zoom_in":
			g.Camera.ZoomAt(1.3, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2})
			h.reply(g, r, nil, false)
		case "zoom_out":
			g.Camera.ZoomAt(1/1.3, mapview.Point{X: g.Camera.Width / 2, Y: g.Camera.Height / 2})
			h.reply(g, r, nil, false)
		case "export":
			b, e := g.Port.Export()
			v := h.snapshot(g, true)
			v["save"] = string(b)
			if e != nil {
				v["error"] = e.Error()
			}
			out, _ := json.Marshal(v)
			r.resolve.Invoke(string(out))
		case "snapshot":
			h.reply(g, r, nil, true)
		default:
			h.reply(g, r, &gameapi.Error{Code: "unknown_command"}, false)
		}
		g.mu.Unlock()
	}
}
func (h *host) update(g *Game)  {}
func (h *host) blocksMap() bool { return false }
func (h *host) changed(g *Game) {
	b, _ := json.Marshal(h.snapshot(g, false))
	js.Global().Get("parent").Call("railMapChanged", string(b))
}
func (h *host) persist(g *Game) {} // Campaign commits use IndexedDB; camera is presentation only.
func (h *host) layout(g *Game) {
	g.Camera.Resize(float64(g.width), float64(g.height))
	if g.Camera.Zoom == 0 {
		g.Camera = g.Map.Camera(g.Camera.Width, g.Camera.Height, g.State.Portrait)
	}
}
func (h *host) draw(g *Game, screen *ebiten.Image) {
	g.Map.Draw(screen, g.Camera, g.Frame, g.State, g.replayTime())
}

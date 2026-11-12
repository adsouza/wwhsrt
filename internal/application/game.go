// Package application owns command coordination, projections, and save mapping.
package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"unicode/utf8"

	"wwhsrt/assets"
	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
)

// Store must atomically replace the prior committed snapshot or leave it intact.
type Store interface {
	Load() ([]byte, error)
	Commit([]byte) error
}

// This bounded development snapshot is deliberately distinct from production
// wwhsrt-save v1. The full production save contract remains separate from this business slice.
type payload struct {
	ScenarioSHA string               `json:"scenario_sha256"`
	BusinessSHA string               `json:"business_sha256"`
	Rules       string               `json:"rules"`
	Revision    uint64               `json:"revision"`
	Months      int                  `json:"months"`
	Company     gameapi.Company      `json:"company"`
	Reports     []gameapi.Report     `json:"reports"`
	Projects    []gameapi.Project    `json:"projects"`
	Services    []gameapi.Service    `json:"services"`
	Fleet       []gameapi.FleetLease `json:"fleet"`
	Debt        gameapi.Money        `json:"debt"`
	Pending     gameapi.CashFlows    `json:"pending"`
	NextTrain   uint64               `json:"next_train"`
}
type envelope struct {
	Format  string          `json:"format"`
	Version int             `json:"version"`
	SHA     string          `json:"payload_sha256"`
	Payload json.RawMessage `json:"payload"`
}
type Game struct {
	mu      sync.Mutex
	w       domain.World
	store   Store
	log     *slog.Logger
	cities  []gameapi.City
	catalog domain.Catalog
}

func complete(b []byte, t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct:
		var obj map[string]json.RawMessage
		if json.Unmarshal(b, &obj) != nil || len(obj) != t.NumField() {
			return false
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			raw, ok := obj[f.Tag.Get("json")]
			if !ok || !complete(raw, f.Type) {
				return false
			}
		}
	case reflect.Slice:
		var values []json.RawMessage
		if json.Unmarshal(b, &values) != nil || values == nil {
			return false
		}
		for _, v := range values {
			if !complete(v, t.Elem()) {
				return false
			}
		}
	}
	return true
}
func catalog() (domain.Catalog, error) {
	var rows []gameapi.Corridor
	if err := json.Unmarshal(assets.Business, &rows); err != nil {
		return domain.Catalog{}, err
	}
	return domain.NewCatalog(rows)
}

func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func strict(b []byte, v any) error {
	if !utf8.Valid(b) {
		return &gameapi.Error{Code: "invalid_save"}
	}
	// Detect duplicate keys at every object depth before typed decoding.
	d := json.NewDecoder(bytes.NewReader(b))
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 16 {
			return &gameapi.Error{Code: "invalid_save"}
		}
		t, e := d.Token()
		if e != nil {
			return e
		}
		if t == nil {
			return &gameapi.Error{Code: "invalid_save"}
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, e := d.Token()
					if e != nil {
						return e
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return &gameapi.Error{Code: "invalid_save"}
					}
					seen[key] = true
					if e = walk(depth + 1); e != nil {
						return e
					}
				}
			case '[':
				for d.More() {
					if e := walk(depth + 1); e != nil {
						return e
					}
				}
			default:
				return &gameapi.Error{Code: "invalid_save"}
			}
			_, e = d.Token()
			return e
		}
		return nil
	}
	if e := walk(0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return &gameapi.Error{Code: "invalid_save"}
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func Open(store Store, log *slog.Logger) (*Game, error) {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	c, e := catalog()
	if e != nil {
		return nil, e
	}
	g := &Game{w: domain.New(c), catalog: c, store: store, log: log}
	var scenario struct {
		ID     string         `json:"id"`
		Cities []gameapi.City `json:"cities"`
	}
	if e := json.Unmarshal(assets.Scenario, &scenario); e != nil {
		return nil, e
	}
	g.cities = scenario.Cities
	if store != nil {
		b, e := store.Load()
		if e != nil {
			log.Error("storage.load_failed")
			return nil, &gameapi.Error{Code: "storage_failed"}
		}
		if len(b) > 0 {
			w, e := decode(b)
			if e != nil {
				log.Error("storage.invalid_save")
				return nil, e
			}
			g.w = w
		} else {
			b, _ = encode(g.w)
			if e := store.Commit(b); e != nil {
				return nil, &gameapi.Error{Code: "storage_failed"}
			}
		}
	}
	log.Info("world.opened", "world_id", "first-business", "revision", g.w.Revision(), "scenario_sha256", hash(assets.Scenario))
	return g, nil
}
func data(w domain.World) payload {
	return payload{ScenarioSHA: hash(assets.Scenario), BusinessSHA: hash(assets.Business), Rules: "first-business-v2", Revision: w.Revision(), Months: w.Months(), Company: w.Company(), Reports: w.Reports(), Projects: w.Projects(), Services: w.Services(), Fleet: w.Fleet(), Debt: w.Debt(), Pending: w.Pending(), NextTrain: w.NextTrain()}
}
func encode(w domain.World) ([]byte, error) {
	b, e := json.Marshal(data(w))
	if e != nil {
		return nil, e
	}
	return json.Marshal(envelope{"wwhsrt-first-business", 2, hash(b), b})
}
func decode(b []byte) (domain.World, error) {
	bad := func() (domain.World, error) { return domain.World{}, &gameapi.Error{Code: "invalid_save"} }
	if len(b) > 1024*1024 {
		return bad()
	}
	var env envelope
	if strict(b, &env) != nil || env.Format != "wwhsrt-first-business" || env.Version != 2 || hash(env.Payload) != env.SHA {
		return bad()
	}
	var p payload
	if strict(env.Payload, &p) != nil || !complete(env.Payload, reflect.TypeOf(p)) || p.ScenarioSHA != hash(assets.Scenario) || p.BusinessSHA != hash(assets.Business) || p.Rules != "first-business-v2" {
		return bad()
	}
	c, e := catalog()
	if e != nil {
		return bad()
	}
	return domain.Restore(c, p.Revision, p.Months, p.Company, p.Reports, p.Projects, p.Services, p.Fleet, p.Debt, p.Pending, p.NextTrain)
}
func (g *Game) frame() gameapi.Frame {
	p, _ := json.Marshal(data(g.w))
	return gameapi.Frame{WorldID: "first-business", Revision: g.w.Revision(), Month: domain.Calendar(g.w.Months()), MonthsResolved: g.w.Months(), CampaignMonths: domain.CampaignMonths, Player: g.w.Company(), Cities: append([]gameapi.City{}, g.cities...), Stations: g.w.Stations(), TrainFamily: domain.TrainFamily(), Reports: g.w.Reports(), Checkpoint: hash(p), Corridors: g.catalog.Corridors(), Projects: g.w.Projects(), Services: g.w.Services(), Fleet: g.w.Fleet(), IdleLines: g.w.IdleLines(), Debt: g.w.Debt(), DebtLimit: domain.DebtLimit, Pending: g.w.Pending()}
}
func (g *Game) Forecast(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.w.Forecast(q)
}
func (g *Game) Snapshot() gameapi.Frame { g.mu.Lock(); defer g.mu.Unlock(); return g.frame() }
func (g *Game) Export() ([]byte, error) { g.mu.Lock(); defer g.mu.Unlock(); return encode(g.w) }
func (g *Game) Apply(c gameapi.Command) (gameapi.Frame, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.log.Info("command.received", "kind", c.Kind, "revision", c.Revision)
	w, e := g.w.Change(c)
	if e != nil {
		g.log.Info("command.rejected", "code", e.Error())
		return g.frame(), e
	}
	if g.store != nil {
		b, err := encode(w)
		if err != nil {
			return g.frame(), err
		}
		if err = g.store.Commit(b); err != nil {
			g.log.Error("storage.commit_failed", "revision", g.w.Revision())
			return g.frame(), &gameapi.Error{Code: "storage_failed"}
		}
	}
	g.w = w
	f := g.frame()
	g.log.Info("command.committed", "kind", c.Kind, "revision", f.Revision, "checkpoint", f.Checkpoint)
	return f, nil
}

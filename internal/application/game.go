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
	"strings"
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
type legacyPayload struct {
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
type payload struct {
	NetworkSHA   string               `json:"network_sha256"`
	NetworkSince int                  `json:"network_since"`
	ScenarioSHA  string               `json:"scenario_sha256"`
	BusinessSHA  string               `json:"business_sha256"`
	Rules        string               `json:"rules"`
	Revision     uint64               `json:"revision"`
	Months       int                  `json:"months"`
	Company      gameapi.Company      `json:"company"`
	Reports      []gameapi.Report     `json:"reports"`
	Projects     []gameapi.Project    `json:"projects"`
	Services     []gameapi.Service    `json:"services"`
	Fleet        []gameapi.FleetLease `json:"fleet"`
	Debt         gameapi.Money        `json:"debt"`
	Pending      gameapi.CashFlows    `json:"pending"`
	NextTrain    uint64               `json:"next_train"`
}
type expandedPayload struct {
	payload
	PublicSHA    string `json:"public_sha256"`
	CatalogSince int    `json:"catalog_since"`
}
type balancedPayload struct {
	expandedPayload
	EconomySHA   string `json:"economy_sha256"`
	EconomySince int    `json:"economy_since"`
}
type topologyPayload struct {
	balancedPayload
	TopologySince  int    `json:"topology_since"`
	PriorPublicSHA string `json:"prior_public_sha256"`
}
type connectivityPayload struct {
	topologyPayload
	ConnectivitySince int    `json:"connectivity_since"`
	PreviousRepairSHA string `json:"previous_repair_sha256"`
}
type constructionPayload struct {
	connectivityPayload
	ConstructionSince int    `json:"construction_since"`
	LaunchPriorSHA    string `json:"launch_prior_sha256"`
}
type regionalPayload struct {
	constructionPayload
	RegionalSince    int    `json:"regional_since"`
	RegionalPriorSHA string `json:"regional_prior_sha256"`
}

type connectorPayload struct {
	regionalPayload
	ConnectorSince    int    `json:"connector_since"`
	ConnectorPriorSHA string `json:"connector_prior_sha256"`
}

type envelope struct {
	Format  string          `json:"format"`
	Version int             `json:"version"`
	SHA     string          `json:"payload_sha256"`
	Payload json.RawMessage `json:"payload"`
}
type Game struct {
	mu       sync.Mutex
	w        domain.World
	store    Store
	log      *slog.Logger
	cities   []gameapi.City
	catalog  domain.Catalog
	expanded bool
}

func complete(b []byte, t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct:
		var obj map[string]json.RawMessage
		if json.Unmarshal(b, &obj) != nil {
			return false
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous {
				if !complete(b, f.Type) {
					return false
				}
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			raw, ok := obj[tag[0]]
			if !ok && len(tag) > 1 && tag[1] == "omitempty" {
				continue
			}
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
	c, err := domain.NewCatalog(rows)
	if err != nil {
		return c, err
	}
	var rules gameapi.NetworkRules
	if err = json.Unmarshal(assets.Network, &rules); err != nil {
		return c, err
	}
	return c.WithNetwork(rules)
}

func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// Embedded assets never change at run time, so each digest is computed once
// rather than on every encode and frame.
var (
	scenarioSHA = hash(assets.Scenario)
	businessSHA = hash(assets.Business)
	networkSHA  = hash(assets.Network)
	economySHA  = hash(assets.Economy)
	publicV1SHA = hash(assets.PublicNetworkV1)
)

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
	return openMode(store, log, true)
}
func openMode(store Store, log *slog.Logger, expanded bool) (*Game, error) {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	c, e := catalog()
	if expanded {
		c, e = publicCatalog()
	}
	if e != nil {
		return nil, e
	}
	g := &Game{w: domain.New(c), catalog: c, store: store, log: log, expanded: expanded}
	var scenario struct {
		ID     string         `json:"id"`
		Cities []gameapi.City `json:"cities"`
	}
	if e := json.Unmarshal(assets.Scenario, &scenario); e != nil {
		return nil, e
	}
	g.cities = scenario.Cities
	if expanded {
		source, err := publicData()
		if err != nil {
			return nil, err
		}
		g.cities = source.Cities
	}
	if store != nil {
		b, e := store.Load()
		if e != nil {
			log.Error("storage.load_failed")
			return nil, &gameapi.Error{Code: "storage_failed"}
		}
		if len(b) > 0 {
			w, e := decode(b)
			if expanded {
				w, e = decodeExpanded(b, c)
			}
			if e != nil {
				log.Error("storage.invalid_save")
				return nil, e
			}
			g.w = w
			var header envelope
			_ = json.Unmarshal(b, &header)
			targetVersion := 3
			if expanded {
				targetVersion = 10
			}
			if header.Version < targetVersion {
				log.Info(
					"storage.upgrade_read",
					"from_version",
					header.Version,
					"to_version",
					targetVersion,
					"preserved_months",
					w.Months(),
				)
			}
		} else {
			b, _ = encode(g.w)
			if e := store.Commit(b); e != nil {
				return nil, &gameapi.Error{Code: "storage_failed"}
			}
		}
	}
	rules := "network-v3"
	if expanded {
		rules = "network-v10"
	}
	log.Info(
		"world.opened",
		"world_id",
		"first-business",
		"revision",
		g.w.Revision(),
		"scenario_sha256",
		scenarioSHA,
		"network_sha256",
		networkSHA,
		"rules_version",
		rules,
		"public_network_sha256",
		publicSHA,
		"network_since",
		g.w.NetworkSince(),
	)
	return g, nil
}
func data(w domain.World) payload {
	return payload{
		NetworkSHA:   networkSHA,
		NetworkSince: w.NetworkSince(),
		ScenarioSHA:  scenarioSHA,
		BusinessSHA:  businessSHA,
		Rules:        "network-v3",
		Revision:     w.Revision(),
		Months:       w.Months(),
		Company:      w.Company(),
		Reports:      w.Reports(),
		Projects:     w.Projects(),
		Services:     w.Services(),
		Fleet:        w.Fleet(),
		Debt:         w.Debt(),
		Pending:      w.Pending(),
		NextTrain:    w.NextTrain(),
	}
}
func stateData(w domain.World) any {
	p := data(w)
	if w.Expanded() {
		p.Rules = "network-v10"
		return connectorPayload{
			regionalPayload{
				constructionPayload{
					connectivityPayload{
						topologyPayload{
							balancedPayload{
								expandedPayload{p, publicSHA, w.CatalogSince()},
								economySHA,
								w.EconomySince(),
							},
							w.TopologySince(),
							publicV1SHA,
						},
						w.ConnectivitySince(),
						previousRepairSHA,
					},
					w.ConstructionSince(),
					launchPriorSHA,
				},
				w.RegionalSince(),
				regionalPriorSHA,
			},
			w.ConnectorSince(),
			connectorPriorSHA,
		}
	}
	return p
}
func encode(w domain.World) ([]byte, error) {
	b, e := json.Marshal(stateData(w))
	if e != nil {
		return nil, e
	}
	version := 3
	if w.Expanded() {
		version = 10
	}
	return json.Marshal(envelope{"wwhsrt-first-business", version, hash(b), b})
}
func decode(b []byte) (domain.World, error) {
	bad := func() (domain.World, error) { return domain.World{}, &gameapi.Error{Code: "invalid_save"} }
	if len(b) > 1024*1024 {
		return bad()
	}
	var env envelope
	if strict(b, &env) != nil || env.Format != "wwhsrt-first-business" || (env.Version != 2 && env.Version != 3) ||
		hash(env.Payload) != env.SHA {
		return bad()
	}
	if env.Version == 2 {
		var old legacyPayload
		if strict(env.Payload, &old) != nil || !complete(env.Payload, reflect.TypeOf(old)) ||
			old.ScenarioSHA != scenarioSHA ||
			old.BusinessSHA != businessSHA ||
			old.Rules != "first-business-v2" {
			return bad()
		}
		c, e := catalog()
		if e != nil {
			return bad()
		}
		return domain.Restore(
			c,
			old.Revision,
			old.Months,
			old.Company,
			old.Reports,
			old.Projects,
			old.Services,
			old.Fleet,
			old.Debt,
			old.Pending,
			old.NextTrain,
			domain.Cutovers{Network: old.Months, Economy: old.Months},
		)
	}
	var p payload
	if strict(env.Payload, &p) != nil || !complete(env.Payload, reflect.TypeOf(p)) || p.ScenarioSHA != scenarioSHA ||
		p.BusinessSHA != businessSHA ||
		p.Rules != "network-v3" ||
		p.NetworkSHA != networkSHA {
		return bad()
	}
	c, e := catalog()
	if e != nil {
		return bad()
	}
	return domain.Restore(
		c,
		p.Revision,
		p.Months,
		p.Company,
		p.Reports,
		p.Projects,
		p.Services,
		p.Fleet,
		p.Debt,
		p.Pending,
		p.NextTrain,
		domain.Cutovers{Network: p.NetworkSince, Economy: p.Months},
	)
}
func (g *Game) frame() gameapi.Frame {
	p, _ := json.Marshal(stateData(g.w))
	network := g.w.NetworkReport()
	routes := g.w.Routes()
	available := map[string]bool{}
	for _, r := range routes {
		available[r.ID] = r.Open
	}
	day := []gameapi.DayTrip{}
	for _, trip := range g.w.RepresentativeDay(network) {
		if available[trip.ServiceID] {
			day = append(day, trip)
		}
	}
	return gameapi.Frame{
		PublicNetwork:  g.expanded,
		Routes:         routes,
		Network:        network,
		Day:            day,
		WorldID:        "first-business",
		Revision:       g.w.Revision(),
		Month:          domain.Calendar(g.w.Months()),
		MonthsResolved: g.w.Months(),
		CampaignMonths: domain.CampaignMonths,
		Player:         g.w.Company(),
		Cities:         append([]gameapi.City{}, g.cities...),
		Stations:       g.w.Stations(),
		TrainFamily:    g.w.TrainFamily(),
		Reports:        g.w.Reports(),
		Checkpoint:     hash(p),
		Corridors:      g.displayCorridors(),
		Projects:       g.w.Projects(),
		Services:       g.w.Services(),
		Fleet:          g.w.Fleet(),
		IdleLines:      g.w.IdleLines(),
		Debt:           g.w.Debt(),
		DebtLimit:      domain.DebtLimit,
		Pending:        g.w.Pending(),
	}
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
	g.log.Info(
		"command.received",
		"kind",
		c.Kind,
		"revision",
		c.Revision,
		"service_or_corridor_id",
		c.CorridorID,
		"alignment_id",
		c.AlignmentID,
		"replace_service_id",
		c.ReplaceServiceID,
		"frequency",
		c.Frequency,
		"fare_cents",
		c.Fare,
		"lease_count",
		c.LeaseCount,
		"amount_cents",
		c.Amount,
	)
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
	previousMonths := g.w.Months()
	g.w = w
	f := g.frame()
	if f.MonthsResolved > previousMonths {
		for _, r := range f.Reports[previousMonths:] {
			g.log.Info(
				"month.resolved",
				"year",
				r.Month.Year,
				"month",
				r.Month.Month,
				"journeys",
				r.Passengers,
				"revenue_cents",
				r.Revenue,
				"costs_cents",
				r.Costs,
				"closing_cash_cents",
				r.ClosingCash,
			)
		}
		g.log.Info(
			"network.resolved",
			"requested",
			f.Network.Requested,
			"carried",
			f.Network.Carried,
			"transfers",
			f.Network.Transfers,
			"rejected",
			f.Network.Rejected,
			"alternate_served",
			f.Network.AlternateServed,
			"checkpoint",
			f.Checkpoint,
		)
	}
	g.log.Info("command.committed", "kind", c.Kind, "revision", f.Revision, "checkpoint", f.Checkpoint)
	return f, nil
}

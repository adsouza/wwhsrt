package application

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"reflect"
	"sync"
	"wwhsrt/assets"
	"wwhsrt/internal/domain"
	"wwhsrt/pkg/gameapi"
)

type importedNetwork struct {
	ID        string               `json:"id"`
	Cities    []gameapi.City       `json:"cities"`
	Corridors []gameapi.Corridor   `json:"corridors"`
	Network   gameapi.NetworkRules `json:"network"`
}

var publicSHA = assets.PublicNetworkSHA256()
var publicOnce sync.Once
var imported importedNetwork
var importedCatalog domain.Catalog
var previousPublicCatalog domain.Catalog
var previousRepairCatalog domain.Catalog
var historyCatalog domain.Catalog
var priorLaunchCatalog domain.Catalog
var priorRegionalCatalog domain.Catalog
var priorConnectorCatalog domain.Catalog

var priorSerbiaCatalog domain.Catalog

const serbiaPriorSHA = "50409815e0ed33fd9636969dfde1cbc4e3bdf3dd71b94b1efab3d8885c39b44b"

var priorOccitanieCatalog domain.Catalog

const occitaniePriorSHA = "6e1aa71d35426789f42dfaa2f72c6f887a64f66c202412ccda6e6aa03b141dd1"

var priorGermanCatalog domain.Catalog

const germanPriorSHA = "4bce8d7884cea0c93ae23243f4b22ba0fec4be0db591b08b0cdc9ca85dc14cca"

var priorRhineCatalog domain.Catalog

const rhinePriorSHA = "df6df65b2f9a9be5f26cb416d174025652b3bd76c5655f3b49e44a2b4e302a1a"

const connectorPriorSHA = "f4d94ee8afa0239270975e78033cad2458de90fa012f3037e5adc39b07acdad1"

const regionalPriorSHA = "0d74a64753d4ec70d353decfd80584cc17c6fc49adfb7c4bd640864d331ba5e9"

const launchPriorSHA = "b5940f7c0c130ec255f0783da9c09e12355b6c61131396a1b5f590c4c72ce2f8"

var historyOnce sync.Once
var historyError error

const previousRepairSHA = "9fcd93c3356bd5545896684e98e102cf7d22144884bf984abd80998cffe942f5"

var importError error
var railDisplayPaths map[string]map[string][]gameapi.GeoPoint

func publicData() (*importedNetwork, error) {
	publicOnce.Do(func() {
		if importError = json.Unmarshal(assets.PublicNetwork, &imported); importError != nil {
			return
		}
		if importError = json.Unmarshal(assets.RailDisplayPaths, &railDisplayPaths); importError != nil {
			return
		}
		var c domain.Catalog
		c, importError = domain.NewCatalog(imported.Corridors)
		if importError != nil {
			return
		}
		c, importError = c.WithNetwork(imported.Network)
		if importError != nil {
			return
		}
		var economy gameapi.EconomyRules
		if importError = json.Unmarshal(assets.Economy, &economy); importError != nil {
			return
		}
		c, importError = c.WithEconomy(economy)
		if importError != nil {
			return
		}
		prior, err := catalog()
		if err != nil {
			importError = err
			return
		}
		importedCatalog = c.WithPrior(prior)
	})
	return &imported, importError
}
func publicCatalog() (domain.Catalog, error) { _, e := publicData(); return importedCatalog, e }

// Historical catalogs are immutable and loaded only when a save references them.
func publicHistoryCatalog() (domain.Catalog, error) {
	if _, e := publicData(); e != nil {
		return domain.Catalog{}, e
	}
	historyOnce.Do(func() {
		prior, e := catalog()
		if e != nil {
			historyError = e
			return
		}
		var economy gameapi.EconomyRules
		if e = json.Unmarshal(assets.Economy, &economy); e != nil {
			historyError = e
			return
		}
		makeCatalog := func(data []byte) (domain.Catalog, error) {
			var raw importedNetwork
			if e := json.Unmarshal(data, &raw); e != nil {
				return domain.Catalog{}, e
			}
			c, e := domain.NewCatalog(raw.Corridors)
			if e != nil {
				return c, e
			}
			c, e = c.WithNetwork(raw.Network)
			if e != nil {
				return c, e
			}
			c, e = c.WithEconomy(economy)
			return c.WithPrior(prior), e
		}
		// Archived catalogs are pinned by digest before they are parsed.
		archived := func(compressed []byte, sha string) (domain.Catalog, error) {
			reader, e := gzip.NewReader(bytes.NewReader(compressed))
			if e != nil {
				return domain.Catalog{}, e
			}
			data, e := io.ReadAll(io.LimitReader(reader, 16*1024*1024+1))
			reader.Close()
			if e != nil || hash(data) != sha {
				return domain.Catalog{}, &gameapi.Error{Code: "invalid_scenario"}
			}
			return makeCatalog(data)
		}
		if previousPublicCatalog, e = makeCatalog(assets.PublicNetworkV1); e != nil {
			historyError = e
			return
		}
		if previousRepairCatalog, e = archived(assets.PublicNetworkV2Compressed, previousRepairSHA); e != nil {
			historyError = e
			return
		}
		previousRepairCatalog = previousRepairCatalog.WithPrevious(previousPublicCatalog)
		if priorLaunchCatalog, e = archived(assets.PublicNetworkV3Compressed, launchPriorSHA); e != nil {
			historyError = e
			return
		}
		priorLaunchCatalog = priorLaunchCatalog.WithPrevious(previousPublicCatalog).
			WithConnectivityPrior(previousRepairCatalog)
		if priorRegionalCatalog, e = archived(assets.PublicNetworkV4Compressed, regionalPriorSHA); e != nil {
			historyError = e
			return
		}
		priorRegionalCatalog = priorRegionalCatalog.WithPrevious(previousPublicCatalog).
			WithConnectivityPrior(previousRepairCatalog).
			WithConstructionPrior(priorLaunchCatalog)
		if priorConnectorCatalog, e = archived(assets.PublicNetworkV5Compressed, connectorPriorSHA); e != nil {
			historyError = e
			return
		}
		priorConnectorCatalog = priorConnectorCatalog.WithPrevious(previousPublicCatalog).
			WithConnectivityPrior(previousRepairCatalog).
			WithConstructionPrior(priorLaunchCatalog).
			WithRegionalPrior(priorRegionalCatalog)
		if priorRhineCatalog, e = archived(assets.PublicNetworkV6Compressed, rhinePriorSHA); e != nil {
			historyError = e
			return
		}
		withHistory := func(c domain.Catalog) domain.Catalog {
			return c.WithPrevious(previousPublicCatalog).
				WithConnectivityPrior(previousRepairCatalog).
				WithConstructionPrior(priorLaunchCatalog).
				WithRegionalPrior(priorRegionalCatalog).
				WithConnectorPrior(priorConnectorCatalog)
		}
		priorRhineCatalog = withHistory(priorRhineCatalog)
		if priorGermanCatalog, e = archived(assets.PublicNetworkV7Compressed, germanPriorSHA); e != nil {
			historyError = e
			return
		}
		priorGermanCatalog = withHistory(priorGermanCatalog).WithRhinePrior(priorRhineCatalog)
		if priorOccitanieCatalog, e = archived(assets.PublicNetworkV8Compressed, occitaniePriorSHA); e != nil {
			historyError = e
			return
		}
		priorOccitanieCatalog = withHistory(priorOccitanieCatalog).WithRhinePrior(priorRhineCatalog).WithGermanPrior(priorGermanCatalog)
		if priorSerbiaCatalog, e = archived(assets.PublicNetworkV9Compressed, serbiaPriorSHA); e != nil {
			historyError = e
			return
		}
		withOccitanie := func(c domain.Catalog) domain.Catalog {
			return withHistory(c).WithRhinePrior(priorRhineCatalog).WithGermanPrior(priorGermanCatalog).WithOccitaniePrior(priorOccitanieCatalog)
		}
		priorSerbiaCatalog = withOccitanie(priorSerbiaCatalog)
		historyCatalog = withOccitanie(importedCatalog).WithSerbiaPrior(priorSerbiaCatalog)
	})
	return historyCatalog, historyError
}
func decodeExpanded(b []byte, c domain.Catalog) (domain.World, error) {
	bad := func() (domain.World, error) { return domain.World{}, &gameapi.Error{Code: "invalid_save"} }
	if len(b) > 16*1024*1024 {
		return bad()
	}
	var env envelope
	if strict(b, &env) != nil || env.Format != "wwhsrt-first-business" || hash(env.Payload) != env.SHA {
		return bad()
	}
	if env.Version == 14 {
		return decodeNetworkVersion(b, env, c)
	}
	if env.Version < 2 || env.Version > 13 {
		return bad()
	}
	current, e := publicHistoryCatalog()
	if e != nil {
		return bad()
	}
	var previous domain.World
	switch {
	case env.Version == 13:
		previous, e = decodeNetworkVersion(b, env, priorSerbiaCatalog)
	case env.Version == 12:
		previous, e = decodeNetworkVersion(b, env, priorOccitanieCatalog)
	case env.Version == 11:
		previous, e = decodeNetworkVersion(b, env, priorGermanCatalog)
	default:
		previous, e = decodeNetworkVersion(b, env, priorRhineCatalog)
		if e == nil {
			previous, e = previous.UpgradeRhine(priorGermanCatalog)
		}
	}
	if e == nil && env.Version < 12 {
		previous, e = previous.UpgradeGerman(priorOccitanieCatalog)
	}
	if e == nil && env.Version < 13 {
		previous, e = previous.UpgradeOccitanie(priorSerbiaCatalog)
	}
	if e != nil {
		return bad()
	}
	return previous.UpgradeSerbia(current)
}

func decodeNetworkVersion(b []byte, env envelope, c domain.Catalog) (domain.World, error) {
	bad := func() (domain.World, error) { return domain.World{}, &gameapi.Error{Code: "invalid_save"} }
	if env.Version == 2 || env.Version == 3 {
		old, e := decode(b)
		if e != nil {
			return bad()
		}
		return old.UpgradeCatalog(c)
	}

	var p expandedPayload
	economySince := -1
	topologySince := 0
	connectivitySince := 0
	constructionSince := 0
	regionalSince := 0
	connectorSince := 0
	rhineSince, germanSince, occitanieSince, serbiaSince := 0, 0, 0, 0
	if env.Version >= 10 {
		var modern connectorPayload
		if env.Version >= 13 {
			var latest occitaniePayload
			if env.Version == 14 {
				var next serbiaPayload
				if strict(env.Payload, &next) != nil || !complete(env.Payload, reflect.TypeOf(next)) || next.SerbiaPriorSHA != serbiaPriorSHA || next.Rules != "network-v14" {
					return bad()
				}
				latest, serbiaSince = next.occitaniePayload, next.SerbiaSince
			} else if strict(env.Payload, &latest) != nil || !complete(env.Payload, reflect.TypeOf(latest)) || latest.Rules != "network-v13" {
				return bad()
			}
			if latest.OccitaniePriorSHA != occitaniePriorSHA || latest.GermanPriorSHA != germanPriorSHA || latest.RhinePriorSHA != rhinePriorSHA {
				return bad()
			}
			modern, rhineSince, germanSince, occitanieSince = latest.connectorPayload, latest.RhineSince, latest.GermanSince, latest.OccitanieSince
		} else if env.Version == 12 {
			var latest germanPayload
			if strict(env.Payload, &latest) != nil || !complete(env.Payload, reflect.TypeOf(latest)) ||
				latest.GermanPriorSHA != germanPriorSHA || latest.RhinePriorSHA != rhinePriorSHA || latest.Rules != "network-v12" {
				return bad()
			}
			modern, rhineSince, germanSince = latest.connectorPayload, latest.RhineSince, latest.GermanSince
		} else if env.Version == 11 {
			var latest rhinePayload
			if strict(env.Payload, &latest) != nil || !complete(env.Payload, reflect.TypeOf(latest)) ||
				latest.RhinePriorSHA != rhinePriorSHA || latest.Rules != "network-v11" {
				return bad()
			}
			modern, rhineSince = latest.connectorPayload, latest.RhineSince
		} else if strict(env.Payload, &modern) != nil || !complete(env.Payload, reflect.TypeOf(modern)) || modern.Rules != "network-v10" {
			return bad()
		}
		if modern.ConnectorPriorSHA != connectorPriorSHA ||
			modern.RegionalPriorSHA != regionalPriorSHA ||
			modern.LaunchPriorSHA != launchPriorSHA ||
			modern.EconomySHA != economySHA ||
			modern.PriorPublicSHA != publicV1SHA ||
			modern.PreviousRepairSHA != previousRepairSHA {
			return bad()
		}
		p = modern.expandedPayload
		economySince, topologySince, connectivitySince, constructionSince, regionalSince, connectorSince =
			modern.EconomySince, modern.TopologySince, modern.ConnectivitySince,
			modern.ConstructionSince, modern.RegionalSince, modern.ConnectorSince
	} else if env.Version == 9 {
		var modern regionalPayload
		if strict(env.Payload, &modern) != nil || !complete(env.Payload, reflect.TypeOf(modern)) ||
			modern.RegionalPriorSHA != regionalPriorSHA ||
			modern.LaunchPriorSHA != launchPriorSHA ||
			modern.EconomySHA != economySHA ||
			modern.PriorPublicSHA != publicV1SHA ||
			modern.PreviousRepairSHA != previousRepairSHA ||
			modern.Rules != "network-v9" {
			return bad()
		}
		p = modern.expandedPayload
		economySince, topologySince, connectivitySince, constructionSince, regionalSince =
			modern.EconomySince, modern.TopologySince, modern.ConnectivitySince,
			modern.ConstructionSince, modern.RegionalSince
	} else if env.Version == 8 {
		var modern constructionPayload
		if strict(env.Payload, &modern) != nil || !complete(env.Payload, reflect.TypeOf(modern)) ||
			modern.LaunchPriorSHA != launchPriorSHA ||
			modern.EconomySHA != economySHA ||
			modern.PriorPublicSHA != publicV1SHA ||
			modern.PreviousRepairSHA != previousRepairSHA ||
			modern.Rules != "network-v8" {
			return bad()
		}
		p = modern.expandedPayload
		economySince = modern.EconomySince
		topologySince = modern.TopologySince
		connectivitySince = modern.ConnectivitySince
		constructionSince = modern.ConstructionSince
	} else if env.Version == 7 {
		var modern connectivityPayload
		if strict(env.Payload, &modern) != nil || !complete(env.Payload, reflect.TypeOf(modern)) ||
			modern.EconomySHA != economySHA ||
			modern.PriorPublicSHA != publicV1SHA ||
			modern.PreviousRepairSHA != previousRepairSHA ||
			modern.Rules != "network-v7" {
			return bad()
		}
		p = modern.expandedPayload
		economySince = modern.EconomySince
		topologySince = modern.TopologySince
		connectivitySince = modern.ConnectivitySince
	} else if env.Version == 6 {
		var modern topologyPayload
		if strict(env.Payload, &modern) != nil || !complete(env.Payload, reflect.TypeOf(modern)) ||
			modern.EconomySHA != economySHA ||
			modern.PriorPublicSHA != publicV1SHA ||
			modern.Rules != "network-v6" {
			return bad()
		}
		p = modern.expandedPayload
		economySince = modern.EconomySince
		topologySince = modern.TopologySince
	} else if env.Version == 5 {
		var modern balancedPayload
		if strict(env.Payload, &modern) != nil || !complete(env.Payload, reflect.TypeOf(modern)) ||
			modern.EconomySHA != economySHA ||
			modern.Rules != "network-v5" {
			return bad()
		}
		p = modern.expandedPayload
		economySince = modern.EconomySince
	} else {
		if strict(env.Payload, &p) != nil || !complete(env.Payload, reflect.TypeOf(p)) || p.Rules != "network-v4" {
			return bad()
		}
		economySince = p.Months
	}
	expectedPublic := publicSHA
	if env.Version == 13 {
		expectedPublic = serbiaPriorSHA
	}
	if env.Version == 12 {
		expectedPublic = occitaniePriorSHA
	}
	if env.Version == 11 {
		expectedPublic = germanPriorSHA
	}
	if env.Version == 10 {
		expectedPublic = rhinePriorSHA
	}
	if env.Version == 9 {
		expectedPublic = connectorPriorSHA
	}
	if env.Version == 8 {
		expectedPublic = regionalPriorSHA
	}
	if env.Version == 7 {
		expectedPublic = launchPriorSHA
	}
	if env.Version == 6 {
		expectedPublic = previousRepairSHA
	}
	if env.Version < 6 {
		expectedPublic = publicV1SHA
	}
	if p.PublicSHA != expectedPublic || p.ScenarioSHA != scenarioSHA || p.BusinessSHA != businessSHA ||
		p.NetworkSHA != networkSHA {
		return bad()
	}
	if env.Version == 14 && (serbiaSince != 0 || occitanieSince != 0 || germanSince != 0 || rhineSince != 0 || topologySince != 0 || connectivitySince != 0 || constructionSince != 0 || regionalSince != 0 || connectorSince != 0) {
		var e error
		c, e = publicHistoryCatalog()
		if e != nil {
			return bad()
		}
	}
	// Epochs a save predates stay zero, which Restore reads as "not yet cut over".
	at := domain.Cutovers{
		Network:      p.NetworkSince,
		Catalog:      p.CatalogSince,
		Economy:      economySince,
		Topology:     topologySince,
		Connectivity: connectivitySince,
		Construction: constructionSince,
		Regional:     regionalSince,
		Connector:    connectorSince,
		Rhine:        rhineSince,
		German:       germanSince,
		Occitanie:    occitanieSince,
		Serbia:       serbiaSince,
	}
	if env.Version == 9 {
		w, e := domain.Restore(
			priorConnectorCatalog,
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
			at,
		)
		if e != nil {
			return bad()
		}
		return w.UpgradeConnectors(c)
	}
	if env.Version == 8 {
		w, e := domain.Restore(
			priorRegionalCatalog,
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
			at,
		)
		if e != nil {
			return bad()
		}
		return w.UpgradeRegional(c)
	}
	if env.Version == 7 {
		w, e := domain.Restore(
			priorLaunchCatalog,
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
			at,
		)
		if e != nil {
			return bad()
		}
		return w.UpgradeConstruction(c)
	}
	if env.Version == 6 {
		w, e := domain.Restore(
			previousRepairCatalog,
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
			at,
		)
		if e != nil {
			return bad()
		}
		return w.UpgradeConnectivity(c)
	}
	if env.Version < 6 {
		// Reuse the immutable cached catalog: reparsing its geometry for every
		// restore needlessly doubles the browser's migration scratch space.
		if _, e := publicData(); e != nil {
			return bad()
		}
		pc := previousPublicCatalog

		w, e := domain.Restore(
			pc,
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
			at,
		)
		if e != nil {
			return bad()
		}
		return w.UpgradeTopology(c)
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
		at,
	)
}

func (g *Game) InspectMarket(from, to gameapi.CityID, class string) (gameapi.MarketRun, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := g.w.NetworkReport()
	for _, m := range n.Markets {
		if m.From == from && m.To == to && m.Class == class {
			return m, nil
		}
	}
	return g.w.InspectMarket(from, to, class)
}

// The authoritative catalog keeps historical prices/times and save identities.
// Its frame projection uses source track for legacy drawings and train movement.
func (g *Game) displayCorridors() []gameapi.Corridor {
	rows := g.w.DisplayCorridors()
	if g.expanded {
		for i, co := range rows {
			for j, a := range co.Alignments {
				if path := railDisplayPaths[co.ID][a.ID]; len(path) > 1 {
					rows[i].Alignments[j].Path = append([]gameapi.GeoPoint{}, path...)
				}
			}
		}
	}
	return rows
}

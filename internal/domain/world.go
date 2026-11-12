// Package domain owns deterministic railway-business state and performs no I/O.
package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
	"wwhsrt/pkg/gameapi"
)

const CampaignMonths = 360
const MaxRevision = 1_000_000
const StartingCash gameapi.Money = 10_000_000_000
const RepaymentStep gameapi.Money = 50_000_000_000 // USD 500M per quick repayment.
const DebtLimit gameapi.Money = 500_000_000_000    // USD 5B.
const LeaseMonthly gameapi.Money = 40_000_000      // USD 400,000, one 400-seat train.
const maintenanceDivisor gameapi.Money = 6000
const moneyLimit gameapi.Money = 1_000_000_000_000_000

type Catalog struct{ corridors []gameapi.Corridor }

func NewCatalog(rows []gameapi.Corridor) (Catalog, error) {
	c := Catalog{corridors: []gameapi.Corridor{}}
	seen := map[string]bool{}
	pairs := map[string]bool{}
	for _, r := range rows {
		a, b := string(r.FromCity), string(r.ToCity)
		if a > b {
			a, b = b, a
		}
		pair := a + "/" + b
		if r.ID == "" || seen[r.ID] || pairs[pair] || r.FromCity == r.ToCity || r.DailyMarket < 1 || r.DailyMarket > 100000 || r.OutsideCost < 1000 || r.OutsideCost > 100000 || len(r.Alignments) < 1 {
			return c, errCode("invalid_scenario")
		}
		seen[r.ID] = true
		pairs[pair] = true
		ids := map[string]bool{}
		r.Alignments = append([]gameapi.Alignment{}, r.Alignments...)
		for i, a := range r.Alignments {
			if a.ID == "" || ids[a.ID] || a.KM < 1 || a.KM > 2000 || a.Minutes < 1 || a.Minutes > 1000 || a.SurfaceKM < 0 || a.HillKM < 0 || a.ViaductKM < 0 || a.TunnelKM < 0 || a.SurfaceKM+a.HillKM+a.ViaductKM+a.TunnelKM != a.KM || len(a.Path) < 2 || len(a.Path) > 64 || (!r.Existing && (a.Months < 1 || a.Months > 120)) {
				return c, errCode("invalid_scenario")
			}
			ids[a.ID] = true
			for _, p := range a.Path {
				if !(p.Lat >= -90 && p.Lat <= 90 && p.Lon >= -180 && p.Lon <= 180) {
					return c, errCode("invalid_scenario")
				}
			}
			a.Path = append([]gameapi.GeoPoint{}, a.Path...)
			if !r.Existing {
				// Japan profile: B=20M/km, systems=3M/km. Terrain factors 1/1.5/2.5/5.
				// Management 15%, then contingency 20%; no engineering interval is free.
				a.Estimate = gameapi.ConstructionEstimate{Surface: gameapi.Money(a.SurfaceKM) * 2_000_000_000, Hills: gameapi.Money(a.HillKM) * 2_850_000_000, Viaducts: gameapi.Money(a.ViaductKM) * 4_550_000_000, Tunnels: gameapi.Money(a.TunnelKM) * 8_800_000_000}
				direct := a.Estimate.Surface + a.Estimate.Hills + a.Estimate.Viaducts + a.Estimate.Tunnels
				a.Estimate.Management = direct * 15 / 100
				// Preserve the scenario's existing cent rounding exactly.
				a.Cost = (direct + a.Estimate.Management) * 120 / 100
				a.Estimate.Contingency = a.Cost - direct - a.Estimate.Management
				a.Grant = a.Cost * 90 / 100
			} else {
				a.Estimate = gameapi.ConstructionEstimate{}
				a.Cost = 0
				a.Grant = 0
			}
			r.Alignments[i] = a
		}
		c.corridors = append(c.corridors, r)
	}
	return c, nil
}
func (c Catalog) Corridors() []gameapi.Corridor {
	out := append([]gameapi.Corridor{}, c.corridors...)
	for i := range out {
		out[i].Alignments = append([]gameapi.Alignment{}, out[i].Alignments...)
		for j := range out[i].Alignments {
			out[i].Alignments[j].Path = append([]gameapi.GeoPoint{}, out[i].Alignments[j].Path...)
		}
	}
	return out
}
func (c Catalog) find(id string) (gameapi.Corridor, bool) {
	for _, r := range c.corridors {
		if r.ID == id {
			return r, true
		}
	}
	return gameapi.Corridor{}, false
}
func alternative(c gameapi.Corridor, id string) (gameapi.Alignment, bool) {
	for _, a := range c.Alignments {
		if a.ID == id {
			return a, true
		}
	}
	return gameapi.Alignment{}, false
}
func errCode(s string) error { return &gameapi.Error{Code: s} }

type World struct {
	revision  uint64
	months    int
	company   gameapi.Company
	reports   []gameapi.Report
	catalog   Catalog
	projects  []gameapi.Project
	services  []gameapi.Service
	fleet     []gameapi.FleetLease
	debt      gameapi.Money
	pending   gameapi.CashFlows
	nextTrain uint64
}

func New(c Catalog) World {
	return World{catalog: c, company: gameapi.Company{ID: "company-1", Name: "Sunrise Rail", Cash: StartingCash}, reports: []gameapi.Report{}, projects: []gameapi.Project{}, services: []gameapi.Service{{CorridorID: "tokyo-nagoya", Frequency: 6, Fare: 9000}}, fleet: []gameapi.FleetLease{{ID: 1, CorridorID: "tokyo-nagoya"}, {ID: 2, CorridorID: "tokyo-nagoya"}}, nextTrain: 3}
}
func Calendar(offset int) gameapi.Month {
	return gameapi.Month{Year: 2029 + offset/12, Month: 1 + offset%12}
}
func (w World) Revision() uint64           { return w.revision }
func (w World) Months() int                { return w.months }
func (w World) Company() gameapi.Company   { return w.company }
func (w World) Debt() gameapi.Money        { return w.debt }
func (w World) Pending() gameapi.CashFlows { return w.pending }
func (w World) NextTrain() uint64          { return w.nextTrain }
func (w World) Reports() []gameapi.Report {
	out := append([]gameapi.Report{}, w.reports...)
	for i := range out {
		out[i].Services = append([]gameapi.ServiceRun{}, out[i].Services...)
	}
	return out
}
func (w World) Projects() []gameapi.Project { return append([]gameapi.Project{}, w.projects...) }
func (w World) Services() []gameapi.Service { return append([]gameapi.Service{}, w.services...) }
func (w World) Fleet() []gameapi.FleetLease { return append([]gameapi.FleetLease{}, w.fleet...) }
func (w World) clone() World {
	w.reports = w.Reports()
	w.projects = w.Projects()
	w.services = w.Services()
	w.fleet = w.Fleet()
	return w
}
func (w World) project(id string) (gameapi.Project, bool) {
	for _, p := range w.projects {
		if p.CorridorID == id {
			return p, true
		}
	}
	return gameapi.Project{}, false
}
func (w World) operational(c gameapi.Corridor) bool {
	if c.Existing {
		return true
	}
	p, ok := w.project(c.ID)
	return ok && p.Ready <= w.months
}
func (w World) alignment(c gameapi.Corridor, id string) (gameapi.Alignment, bool) {
	if c.Existing {
		return c.Alignments[0], true
	}
	if p, ok := w.project(c.ID); ok {
		return alternative(c, p.AlignmentID)
	}
	return alternative(c, id)
}
func (w World) FleetCount(id string) int {
	n := 0
	for _, f := range w.fleet {
		if f.CorridorID == id {
			n++
		}
	}
	return n
}
func days(m gameapi.Month) int {
	d := [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if m.Month == 2 && (m.Year%400 == 0 || m.Year%4 == 0 && m.Year%100 != 0) {
		return 29
	}
	return d[m.Month-1]
}
func required(frequency, minutes int) int {
	return (frequency*(minutes*2+roundTripTurnaround) + trainDayMinutes - 1) / trainDayMinutes
}
func (w World) Forecast(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
	c, ok := w.catalog.find(q.CorridorID)
	if !ok {
		return gameapi.Forecast{}, errCode("no_route")
	}
	a, ok := w.alignment(c, q.AlignmentID)
	if !ok {
		return gameapi.Forecast{}, errCode("no_route")
	}
	if q.Frequency < 0 || q.Frequency > 16 || q.Fare < 1000 || q.Fare > 30000 {
		return gameapi.Forecast{}, errCode("invalid_service")
	}
	result := gameapi.Forecast{FleetNeeded: required(q.Frequency, a.Minutes), FleetAvailable: w.FleetCount(c.ID), Open: w.operational(c)}
	d := int64(days(Calendar(w.months)))
	f := int64(q.Frequency)
	if f > 0 {
		generalized := int64(q.Fare) + int64(a.Minutes)*20 + 9600/f
		outside := int64(c.OutsideCost)
		demand := int64(c.DailyMarket) * outside * outside / (outside*outside + generalized*generalized)
		capacity := f * 2 * trainSeats
		carried := min(demand, capacity)
		result.Passengers = int(carried * d)
		result.Capacity = int(capacity * d)
		result.Revenue = gameapi.Money(result.Passengers) * q.Fare
		km := gameapi.Money(f * 2 * int64(a.KM) * d)
		running := km * runningPerKM
		access := gameapi.Money(0)
		if c.Existing {
			access = km * publicAccessPerKM
		}
		stations := gameapi.Money(result.Passengers) * stationPerJourney
		result.Costs = running + access + stations
	}
	result.Costs += gameapi.Money(result.FleetNeeded) * LeaseMonthly
	if !c.Existing {
		result.Costs += a.Cost / maintenanceDivisor
	}
	result.Contribution = result.Revenue - result.Costs
	return result, nil
}
func (w World) Change(c gameapi.Command) (World, error) {
	if c.Revision != w.revision {
		return w, errCode("stale_revision")
	}
	if w.revision >= MaxRevision {
		return w, errCode("action_limit")
	}
	if w.months >= CampaignMonths && c.Kind != gameapi.RenameCompany && c.Kind != gameapi.NewGame {
		return w, errCode("campaign_complete")
	}
	next := w.clone()
	if e := next.change(c); e != nil {
		return w, e
	}
	next.revision++
	if next.company.Cash > moneyLimit || next.company.Cash < -moneyLimit || next.pending.Borrowed > moneyLimit || next.pending.Repaid > moneyLimit {
		return w, errCode("cash_limit")
	}
	return next, nil
}
func (w *World) change(c gameapi.Command) error {
	switch c.Kind {
	case gameapi.NewGame:
		// Keep the command revision monotonic across campaigns so queued commands
		// from the previous company cannot become valid again after a restart.
		revision := w.revision
		*w = New(w.catalog)
		w.revision = revision
	case gameapi.RenameCompany:
		name := strings.TrimSpace(c.Name)
		if !ValidName(name) {
			return errCode("invalid_name")
		}
		w.company.Name = name
	case gameapi.TakeLoan:
		if c.Amount <= 0 {
			return errCode("invalid_loan_amount")
		}
		if c.Amount > DebtLimit-w.debt {
			return errCode("loan_limit")
		}
		w.debt += c.Amount
		w.company.Cash += c.Amount
		w.pending.Borrowed += c.Amount
	case gameapi.RepayLoan:
		amount := min(RepaymentStep, w.debt)
		if amount <= 0 || w.company.Cash < amount {
			return errCode("insufficient_cash")
		}
		w.debt -= amount
		w.company.Cash -= amount
		w.pending.Repaid += amount
	case gameapi.BuildLine:
		corridor, ok := w.catalog.find(c.CorridorID)
		if !ok {
			return errCode("no_route")
		}
		if corridor.Existing {
			return errCode("already_built")
		}
		if _, ok = w.project(c.CorridorID); ok {
			return errCode("already_built")
		}
		a, ok := alternative(corridor, c.AlignmentID)
		if !ok {
			return errCode("no_route")
		}
		net := a.Cost - a.Grant
		if w.company.Cash < net {
			return errCode("insufficient_cash")
		}
		w.company.Cash -= net
		w.pending.Construction += a.Cost
		w.pending.Grants += a.Grant
		w.projects = append(w.projects, gameapi.Project{CorridorID: c.CorridorID, AlignmentID: a.ID, Owner: w.company.ID, Started: w.months, Ready: w.months + a.Months})
	case gameapi.LeaseTrain:
		corridor, ok := w.catalog.find(c.CorridorID)
		if !ok {
			return errCode("no_route")
		}
		if !w.operational(corridor) {
			return errCode("line_not_open")
		}
		if len(w.fleet) >= 32 || w.company.Cash < 0 {
			return errCode("fleet_limit")
		}
		w.fleet = append(w.fleet, gameapi.FleetLease{ID: w.nextTrain, CorridorID: c.CorridorID})
		w.nextTrain++
	case gameapi.ReturnTrain:
		count := w.FleetCount(c.CorridorID)
		if count < 1 {
			return errCode("fleet_shortage")
		}
		for _, s := range w.services {
			if s.CorridorID == c.CorridorID {
				co, _ := w.catalog.find(s.CorridorID)
				a, _ := w.alignment(co, "")
				if required(s.Frequency, a.Minutes) > count-1 {
					return errCode("fleet_in_use")
				}
			}
		}
		for i := len(w.fleet) - 1; i >= 0; i-- {
			if w.fleet[i].CorridorID == c.CorridorID {
				w.fleet = append(w.fleet[:i], w.fleet[i+1:]...)
				break
			}
		}
	case gameapi.SetService:
		q := gameapi.ForecastRequest{CorridorID: c.CorridorID, Frequency: c.Frequency, Fare: c.Fare}
		forecast, e := w.Forecast(q)
		if e != nil {
			return e
		}
		if !forecast.Open {
			return errCode("line_not_open")
		}
		if forecast.FleetNeeded > forecast.FleetAvailable {
			return errCode("fleet_shortage")
		}
		replaced := false
		for i := range w.services {
			if w.services[i].CorridorID == c.CorridorID {
				w.services[i] = gameapi.Service{CorridorID: c.CorridorID, Frequency: c.Frequency, Fare: c.Fare}
				replaced = true
			}
		}
		if !replaced {
			w.services = append(w.services, gameapi.Service{CorridorID: c.CorridorID, Frequency: c.Frequency, Fare: c.Fare})
		}
	case gameapi.AdvanceMonth, gameapi.AdvanceYear:
		n := 1
		if c.Kind == gameapi.AdvanceYear {
			n = 12
		}
		n = min(n, CampaignMonths-w.months)
		if n <= 0 {
			return errCode("campaign_complete")
		}
		for i := 0; i < n; i++ {
			if e := w.resolve(); e != nil {
				return e
			}
		}
	default:
		return errCode("unknown_command")
	}
	return nil
}
func (w *World) resolve() error {
	r := gameapi.Report{Month: Calendar(w.months), CompanyID: w.company.ID, OpeningCash: StartingCash, Construction: w.pending.Construction, Grants: w.pending.Grants, Borrowed: w.pending.Borrowed, Repaid: w.pending.Repaid, Services: []gameapi.ServiceRun{}}
	if len(w.reports) > 0 {
		r.OpeningCash = w.reports[len(w.reports)-1].ClosingCash
	}
	for _, s := range w.services {
		c, _ := w.catalog.find(s.CorridorID)
		if !w.operational(c) {
			continue
		}
		a, _ := w.alignment(c, "")
		forecast, e := w.Forecast(gameapi.ForecastRequest{CorridorID: s.CorridorID, Frequency: s.Frequency, Fare: s.Fare})
		if e != nil {
			return e
		}
		r.Passengers += forecast.Passengers
		r.Revenue += forecast.Revenue
		r.Services = append(r.Services, gameapi.ServiceRun{CorridorID: s.CorridorID, AlignmentID: a.ID, Frequency: s.Frequency, Passengers: forecast.Passengers, Revenue: forecast.Revenue, Fare: s.Fare})
		km := gameapi.Money(s.Frequency * 2 * a.KM * days(r.Month))
		r.RunningCosts += km * runningPerKM
		if c.Existing {
			r.AccessCosts += km * publicAccessPerKM
		}
		r.StationCosts += gameapi.Money(forecast.Passengers) * stationPerJourney
	}
	r.LeaseCosts = gameapi.Money(len(w.fleet)) * LeaseMonthly
	for _, p := range w.projects {
		if p.Ready <= w.months {
			c, _ := w.catalog.find(p.CorridorID)
			a, _ := alternative(c, p.AlignmentID)
			r.Maintenance += a.Cost / maintenanceDivisor
		}
	}
	r.Interest = (w.debt + 599) / 600 // 2% annual, rounded up to a cent each month.
	if w.company.Cash < 0 {
		r.Interest += (-w.company.Cash + 99) / 100
	} // Disclosed 1% monthly cash-overdraft charge.
	r.Costs = r.RunningCosts + r.AccessCosts + r.StationCosts + r.LeaseCosts + r.Maintenance + r.Interest
	r.ClosingCash = w.company.Cash + r.Revenue - r.Costs
	if r.ClosingCash > moneyLimit || r.ClosingCash < -moneyLimit {
		return errCode("cash_limit")
	}
	w.company.Cash = r.ClosingCash
	w.reports = append(w.reports, r)
	w.pending = gameapi.CashFlows{}
	w.months++
	return nil
}
func ValidName(name string) bool {
	if !utf8.ValidString(name) || strings.TrimSpace(name) != name || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 40 {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}

// Restore validates the development business snapshot without executing commands.
func Restore(c Catalog, revision uint64, months int, company gameapi.Company, reports []gameapi.Report, projects []gameapi.Project, services []gameapi.Service, fleet []gameapi.FleetLease, debt gameapi.Money, pending gameapi.CashFlows, nextTrain uint64) (World, error) {
	fail := func() (World, error) { return World{}, errCode("invalid_save") }
	if months < 0 || months > CampaignMonths || revision > MaxRevision || company.ID != "company-1" || !ValidName(company.Name) || company.Cash > moneyLimit || company.Cash < -moneyLimit || reports == nil || len(reports) != months || projects == nil || services == nil || fleet == nil || len(projects) > len(c.corridors) || len(services) > len(c.corridors) || len(fleet) > 32 || debt < 0 || debt > DebtLimit || nextTrain < 3 || nextTrain > MaxRevision+3 {
		return fail()
	}
	w := World{catalog: c, revision: revision, months: months, company: company, reports: append([]gameapi.Report{}, reports...), projects: append([]gameapi.Project{}, projects...), services: append([]gameapi.Service{}, services...), fleet: append([]gameapi.FleetLease{}, fleet...), debt: debt, pending: pending, nextTrain: nextTrain}
	cash := StartingCash
	var borrowed, repaid, construction, grants gameapi.Money
	validMoney := func(n gameapi.Money) bool { return n >= 0 && n <= moneyLimit }
	for i, r := range reports {
		if r.Month != Calendar(i) || r.CompanyID != company.ID || r.OpeningCash != cash || r.Passengers < 0 || r.Passengers > 100000000 {
			return fail()
		}
		for _, n := range []gameapi.Money{r.Revenue, r.Costs, r.RunningCosts, r.AccessCosts, r.StationCosts, r.LeaseCosts, r.Maintenance, r.Interest, r.Construction, r.Grants, r.Borrowed, r.Repaid} {
			if !validMoney(n) {
				return fail()
			}
		}
		if r.Costs != r.RunningCosts+r.AccessCosts+r.StationCosts+r.LeaseCosts+r.Maintenance+r.Interest {
			return fail()
		}
		var revenue gameapi.Money
		passengers := 0
		runs := map[string]bool{}
		for _, run := range r.Services {
			co, ok := c.find(run.CorridorID)
			_, exists := alternative(co, run.AlignmentID)
			if !ok || !exists || runs[run.CorridorID] || run.Frequency < 0 || run.Frequency > 16 || run.Fare < 1000 || run.Fare > 30000 || run.Passengers < 0 || run.Passengers > run.Frequency*trainSeats*2*days(r.Month) || run.Revenue != gameapi.Money(run.Passengers)*run.Fare {
				return fail()
			}
			runs[run.CorridorID] = true
			revenue += run.Revenue
			passengers += run.Passengers
		}
		if revenue != r.Revenue || passengers != r.Passengers {
			return fail()
		}
		if r.ClosingCash != cash+r.Revenue-r.Costs+r.Grants+r.Borrowed-r.Repaid-r.Construction || r.ClosingCash > moneyLimit || r.ClosingCash < -moneyLimit {
			return fail()
		}
		cash = r.ClosingCash
		borrowed += r.Borrowed
		repaid += r.Repaid
		construction += r.Construction
		grants += r.Grants
	}
	for _, n := range []gameapi.Money{pending.Construction, pending.Grants, pending.Borrowed, pending.Repaid} {
		if !validMoney(n) {
			return fail()
		}
	}
	if company.Cash != cash+pending.Grants+pending.Borrowed-pending.Repaid-pending.Construction || debt != borrowed-repaid+pending.Borrowed-pending.Repaid {
		return fail()
	}
	seen := map[string]bool{}
	var committed, awarded gameapi.Money
	for _, p := range projects {
		co, ok := c.find(p.CorridorID)
		a, found := alternative(co, p.AlignmentID)
		if !ok || !found || co.Existing || seen[p.CorridorID] || p.Owner != company.ID || p.Started < 0 || p.Started > months || p.Ready != p.Started+a.Months {
			return fail()
		}
		seen[p.CorridorID] = true
		committed += a.Cost
		awarded += a.Grant
	}
	if committed != construction+pending.Construction || awarded != grants+pending.Grants {
		return fail()
	}
	for i, r := range reports {
		for _, run := range r.Services {
			co, _ := c.find(run.CorridorID)
			if !co.Existing {
				p, ok := w.project(co.ID)
				if !ok || p.Ready > i {
					return fail()
				}
			}
		}
	}
	trainIDs := map[uint64]bool{}
	for _, f := range fleet {
		co, ok := c.find(f.CorridorID)
		if !ok || !w.operational(co) || f.ID == 0 || f.ID >= nextTrain || trainIDs[f.ID] {
			return fail()
		}
		trainIDs[f.ID] = true
	}
	seen = map[string]bool{}
	for _, s := range services {
		forecast, e := w.Forecast(gameapi.ForecastRequest{CorridorID: s.CorridorID, Frequency: s.Frequency, Fare: s.Fare})
		if e != nil || seen[s.CorridorID] || !forecast.Open || forecast.FleetAvailable < forecast.FleetNeeded {
			return fail()
		}
		seen[s.CorridorID] = true
	}
	return w, nil
}

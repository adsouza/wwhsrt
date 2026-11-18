// Package ui contains local presentation state and translations, never rules.
package ui

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
)

//go:embed en.json
var english []byte

//go:embed zh-Hans.json
var chinese []byte

// Built on first use so importing the package costs nothing at start-up.
var catalogs = sync.OnceValue(func() map[string]map[string]string {
	result := map[string]map[string]string{}
	for locale, b := range map[string][]byte{"en": english, "zh-Hans": chinese} {
		var c map[string]string
		if err := json.Unmarshal(b, &c); err != nil {
			panic(err)
		}
		result[locale] = c
	}
	{
		for _, city := range publicCities() {
			for _, locale := range []string{"en", "zh-Hans"} {
				if result[locale][string(city.ID)] == "" {
					name := city.Name
					name = strings.NewReplacer(
						"Ł",
						"L",
						"ł",
						"l",
						"š",
						"s",
						"ą",
						"a",
						"ł",
						"l",
						"ś",
						"s",
						"ę",
						"e",
						"ź",
						"z",
						"ż",
						"z",
						"ğ",
						"g",
						"ş",
						"s",
						"İ",
						"I",
						"ı",
						"i",
						"қ",
						"к",
						"ө",
						"о",
					).
						Replace(name)
					result[locale][string(city.ID)] = name
				}
			}
		}
	}
	return result
})

// publicCities decodes only the "cities" member of the public catalog and
// stops there, instead of scanning megabytes of corridor geometry again.
func publicCities() []gameapi.City {
	d := json.NewDecoder(bytes.NewReader(assets.PublicNetwork))
	if t, err := d.Token(); err != nil || t != json.Delim('{') {
		return nil
	}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil
		}
		if key == "cities" {
			var cities []gameapi.City
			if d.Decode(&cities) != nil {
				return nil
			}
			return cities
		}
		var skipped json.RawMessage
		if d.Decode(&skipped) != nil {
			return nil
		}
	}
	return nil
}

func Locale(s string) string {
	s = strings.ToLower(s)
	if s == "zh" || strings.HasPrefix(s, "zh-hans") || s == "zh-cn" || s == "zh-sg" {
		return "zh-Hans"
	}
	return "en"
}
func Text(locale, key string) string {
	if s := catalogs()[Locale(locale)][key]; s != "" {
		return s
	}
	if s := catalogs()["en"][key]; s != "" {
		return s
	}
	return key
}
func Catalog(locale string) map[string]string {
	out := map[string]string{}
	for k := range catalogs()["en"] {
		out[k] = Text(locale, k)
	}
	return out
}
func Matches(c gameapi.City, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	return strings.Contains(strings.ToLower(c.Name+" "+Text("zh-Hans", string(c.ID))), q)
}

type State struct {
	SourceStation string          `json:"source_station"`
	ServiceQuery  string          `json:"service_query"`
	Region        string          `json:"region"`
	PanelWidth    int             `json:"panel_width"`
	CommandDock   string          `json:"command_dock"`
	ServicePane   string          `json:"service_pane"`
	CompanyPane   string          `json:"company_pane"`
	BuildPane     string          `json:"build_pane"`
	ReportPane    string          `json:"report_pane"`
	Disclosures   map[string]bool `json:"disclosures"`
	ReviewMap     bool            `json:"review_map"`

	Portrait           bool           `json:"portrait"`
	Locale             string         `json:"locale"`
	Selected           gameapi.CityID `json:"selected"`
	View               string         `json:"view"`
	Query              string         `json:"query"`
	DraftName          string         `json:"draft_name"`
	HintDismissed      bool           `json:"hint_dismissed"`
	Help               HelpState      `json:"help"`
	Motion             bool           `json:"motion"`
	FromCity           gameapi.CityID `json:"from_city"`
	ToCity             gameapi.CityID `json:"to_city"`
	Choosing           string         `json:"choosing"`
	AlignmentID        string         `json:"alignment_id"`
	ServiceMapSections []string       `json:"service_map_sections,omitempty"`
	ServiceCorridor    string         `json:"service_corridor"`
	Frequency          int            `json:"frequency"`
	Fare               gameapi.Money  `json:"fare"`
	Expanded           bool           `json:"expanded"`
	LoanAmount         string         `json:"loan_amount"`
	HideTerrain        bool           `json:"hide_terrain"`
	Appearance         string         `json:"appearance"`
	TextSize           int            `json:"text_size"`
	Dark               bool           `json:"dark"`
	FleetDetails       bool           `json:"fleet_details"`
	ReplaceServiceID   string         `json:"replace_service_id"`
	NetworkDetails     bool           `json:"network_details"`
	Watching           bool           `json:"watching"`
	ReplayPlaying      bool           `json:"replay_playing"`
	ReplayMinute       int            `json:"replay_minute"`
	SelectedTrain      string         `json:"selected_train"`
	MarketFrom         gameapi.CityID `json:"market_from"`
	MarketTo           gameapi.CityID `json:"market_to"`
	MarketClass        string         `json:"market_class"`
}

// NewGameState clears company-specific drafts and completed actions, retaining
// device preferences and explicitly dismissed learning concepts. It runs only after the new save commits.
func NewGameState(s State, f gameapi.Frame) State {
	next := State{
		Region:      s.Region,
		PanelWidth:  s.PanelWidth,
		CommandDock: s.CommandDock,
		Locale:      s.Locale,
		Portrait:    s.Portrait,
		Motion:      s.Motion,
		HideTerrain: s.HideTerrain,
		Appearance:  s.Appearance,
		TextSize:    s.TextSize,
		Dark:        s.Dark,
		View:        "map",
		DraftName:   f.Player.Name,
		FromCity:    "tsuruga",
		ToCity:      "osaka",
		Choosing:    "from",
		LoanAmount:  "1000000.00",
	}
	DraftService(&next, f, "tokyo-nagoya")
	if s.Help.Version > 0 {
		next.Help = cloneHelp(s.Help)
		next.Help.Open = false
		next.Help.Expanded = ""
		next.Help.Topic = ""
		next.Help.Query = ""
		next.Help.ReturnAnchor = ""
		next.Help.ReturnExpanded = false
		next.Help.ReturnScroll = 0
		next.Help.Revision++
		for id, p := range next.Help.Concepts {
			p.Completed = false
			next.Help.Concepts[id] = p
		}
	}
	return next
}

func Select(s *State, id gameapi.CityID) {
	s.Selected = id
	s.SourceStation = ""

}
func Pair(f gameapi.Frame, a, b gameapi.CityID) (gameapi.Corridor, bool) {
	var public gameapi.Corridor
	for _, c := range f.Corridors {
		if c.FromCity == a && c.ToCity == b || c.FromCity == b && c.ToCity == a {
			if !c.Existing {
				return c, true
			}
			public = c
		}
	}
	return public, public.ID != ""
}
func Open(f gameapi.Frame, id string) bool {
	for _, r := range f.Routes {
		if r.ID == id {
			return r.Open
		}
	}
	for _, c := range f.Corridors {
		if c.ID == id && c.Existing {
			return true
		}
	}
	for _, p := range f.Projects {
		if p.CorridorID == id && p.Ready <= f.MonthsResolved {
			return true
		}
	}
	return false
}

// ConstructionOpportunities excludes committed projects, including completed
// lines. Choosing another alignment does not make a committed pair buildable again.
func ConstructionOpportunities(f gameapi.Frame) []gameapi.Corridor {
	committed := make(map[string]bool, len(f.Projects))
	for _, p := range f.Projects {
		committed[p.CorridorID] = true
	}
	var result []gameapi.Corridor
	for _, c := range f.Corridors {
		if !c.Existing && !committed[c.ID] {
			result = append(result, c)
		}
	}
	return result
}

// SelectProspect changes only the local construction draft. Availability is
// checked against the current frame, so stale list/map choices cannot resurrect
// public or already committed lines.
func SelectProspect(s *State, f gameapi.Frame, id string) bool {
	for _, co := range ConstructionOpportunities(f) {
		if co.ID != id || len(co.Alignments) == 0 {
			continue
		}
		s.FromCity, s.ToCity = co.FromCity, co.ToCity
		s.AlignmentID = cheapestAlignment(co).ID
		s.Selected, s.SourceStation = "", ""
		s.View, s.BuildPane = "build", "plan"
		return true
	}
	return false
}

func DraftService(s *State, f gameapi.Frame, id string) {
	if r, ok := ServiceRoute(f, id); !ok || !routeUsesSections(r, s.ServiceMapSections) {
		s.ServiceMapSections = nil
	}
	s.ServiceCorridor = id
	s.ReplaceServiceID = ""
	s.Frequency = 4
	s.Fare = 9000
	if route, ok := ServiceRoute(f, id); ok && len(route.Sections) > 1 {
		s.Fare = gameapi.Money(min(30000, max(1000, route.KM*25)))
	}
	if route, ok := ServiceRoute(f, id); ok && route.SuggestedFrequency > 0 {
		s.Frequency = route.SuggestedFrequency
		s.Fare = route.SuggestedFare
	}
	if id == "tokyo-osaka" {
		for _, p := range f.Services {
			if p.CorridorID == "tokyo-nagoya" && p.Frequency > 0 {
				s.ReplaceServiceID = p.CorridorID
				s.Frequency = p.Frequency
			}
		}
	}
	for _, p := range f.Services {
		if p.CorridorID == id {
			s.ReplaceServiceID = ""
			s.Frequency = p.Frequency
			s.Fare = p.Fare
		}
	}
}
func ServiceDirty(s State, f gameapi.Frame) bool {
	if s.ReplaceServiceID != "" {
		return true
	}
	for _, p := range f.Services {
		if p.CorridorID == s.ServiceCorridor {
			return p.Frequency != s.Frequency || p.Fare != s.Fare
		}
	}
	return true
}
func IdleCosts(f gameapi.Frame) gameapi.Money {
	var total gameapi.Money
	for _, line := range f.IdleLines {
		total += line.Maintenance + line.Leases
	}
	return total
}
func Cash(n gameapi.Money) string {
	sign := ""
	if n < 0 {
		sign = "−"
		n = -n
	}
	whole := strconv.FormatInt(int64(n/100), 10)
	result := ""
	for len(whole) > 3 {
		result = "," + whole[len(whole)-3:] + result
		whole = whole[:len(whole)-3]
	}
	cents := strconv.Itoa(int(n % 100))
	if len(cents) < 2 {
		cents = "0" + cents
	}
	return sign + "USD " + whole + result + "." + cents
}

// ParseDollars accepts decimal USD, with optional correctly grouped thousands.
// It never rounds or uses floating-point arithmetic.
func ParseDollars(value string) (gameapi.Money, bool) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, false
	}
	whole := parts[0]
	if strings.Contains(whole, ",") {
		groups := strings.Split(whole, ",")
		if len(groups[0]) < 1 || len(groups[0]) > 3 {
			return 0, false
		}
		for _, g := range groups[1:] {
			if len(g) != 3 {
				return 0, false
			}
		}
		whole = strings.Join(groups, "")
	}
	for _, c := range whole {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	dollars, e := strconv.ParseInt(whole, 10, 64)
	if e != nil {
		return 0, false
	}
	cents := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, false
		}
		for _, c := range parts[1] {
			if c < '0' || c > '9' {
				return 0, false
			}
		}
		fraction := parts[1] + "00"
		cents, _ = strconv.ParseInt(fraction[:2], 10, 64)
	}
	const max int64 = 1<<63 - 1
	if dollars > (max-cents)/100 {
		return 0, false
	}
	return gameapi.Money(dollars*100 + cents), true
}
func DollarInput(n gameapi.Money) string {
	return strings.ReplaceAll(strings.TrimPrefix(Cash(n), "USD "), ",", "")
}
func FundingShortfall(f gameapi.Frame, s State) gameapi.Money {
	q, _ := FundingBreakdown(f, s)
	return q.Shortfall
}

// ConstructionFunding explains a preview; it never commits a project or loan.
type ConstructionFunding struct {
	CorridorID                                                       string
	FromCity, ToCity                                                 gameapi.CityID
	AlignmentID                                                      string
	Cost, Grant, Contribution, CashAvailable, CashDeficit, Shortfall gameapi.Money
}

func FundingBreakdown(f gameapi.Frame, s State) (ConstructionFunding, bool) {
	co, ok := Pair(f, s.FromCity, s.ToCity)
	if !ok || co.Existing || len(co.Alignments) == 0 {
		return ConstructionFunding{}, false
	}
	for _, p := range f.Projects {
		if p.CorridorID == co.ID {
			return ConstructionFunding{}, false
		}
	}
	id := s.AlignmentID
	if id == "" {
		id = co.Alignments[0].ID
	}
	for _, a := range co.Alignments {
		if a.ID == id {
			q := ConstructionFunding{
				CorridorID:    co.ID,
				FromCity:      co.FromCity,
				ToCity:        co.ToCity,
				AlignmentID:   a.ID,
				Cost:          a.Cost,
				Grant:         a.Grant,
				Contribution:  a.Cost - a.Grant,
				CashAvailable: max(0, f.Player.Cash),
				CashDeficit:   max(0, -f.Player.Cash),
			}
			q.Shortfall = max(0, q.Contribution-q.CashAvailable+q.CashDeficit)
			return q, true
		}
	}
	return ConstructionFunding{}, false
}

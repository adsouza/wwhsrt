// Package gameapi is the renderer-independent driving contract.
package gameapi

type CityID string
type CompanyID string
type Money int64 // Game USD cents.

type City struct {
	ID          CityID  `json:"id"`
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	StationLat  float64 `json:"station_lat,omitempty"`
	StationLon  float64 `json:"station_lon,omitempty"`
	StationName string  `json:"station_name,omitempty"`
}
type Company struct {
	ID   CompanyID `json:"id"`
	Name string    `json:"name"`
	Cash Money     `json:"cash"`
}
type Month struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}
type Report struct {
	Month        Month        `json:"month"`
	CompanyID    CompanyID    `json:"company_id"`
	OpeningCash  Money        `json:"opening_cash"`
	Revenue      Money        `json:"revenue"`
	Costs        Money        `json:"costs"`
	ClosingCash  Money        `json:"closing_cash"`
	Passengers   int          `json:"passengers"`
	RunningCosts Money        `json:"running_costs"`
	AccessCosts  Money        `json:"access_costs"`
	StationCosts Money        `json:"station_costs"`
	LeaseCosts   Money        `json:"lease_costs"`
	Maintenance  Money        `json:"maintenance"`
	Interest     Money        `json:"interest"`
	Construction Money        `json:"construction"`
	Grants       Money        `json:"grants"`
	Borrowed     Money        `json:"borrowed"`
	Repaid       Money        `json:"repaid"`
	Services     []ServiceRun `json:"services"`
}
type ServiceRun struct {
	RequestedFrequency int    `json:"requested_frequency,omitempty"`
	CorridorID         string `json:"corridor_id"`
	AlignmentID        string `json:"alignment_id"`
	Frequency          int    `json:"frequency"`
	Passengers         int    `json:"passengers"`
	Revenue            Money  `json:"revenue"`
	Fare               Money  `json:"fare"`
}
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}
type Alignment struct {
	RetainedKM    int                    `json:"retained_km,omitempty"`
	CostIntervals []ConstructionInterval `json:"cost_intervals,omitempty"`
	ConnectorKM   int                    `json:"connector_km,omitempty"`
	ID            string                 `json:"id"`
	KM            int                    `json:"km"`
	Minutes       int                    `json:"minutes"`
	Months        int                    `json:"months"`
	SurfaceKM     int                    `json:"surface_km"`
	HillKM        int                    `json:"hill_km"`
	ViaductKM     int                    `json:"viaduct_km"`
	TunnelKM      int                    `json:"tunnel_km"`
	Path          []GeoPoint             `json:"path"`
	Cost          Money                  `json:"cost"`
	Grant         Money                  `json:"grant"`
	Estimate      ConstructionEstimate   `json:"estimate"`
}

// ConstructionInterval assigns physical engineering quantities to one regional
// cost profile. All amounts use game USD cents; no endpoint-wide cost shortcut.
type ConstructionInterval struct {
	Country           string `json:"country"`
	ProfileID         string `json:"profile_id"`
	BasePerKM         Money  `json:"base_per_km"`
	SurfaceKM         int    `json:"surface_km"`
	HillKM            int    `json:"hill_km"`
	ViaductKM         int    `json:"viaduct_km"`
	TunnelKM          int    `json:"tunnel_km"`
	HillFactorPercent int    `json:"hill_factor_percent"`
	TunnelFactor      int    `json:"tunnel_factor"`
}
type ConstructionEstimate struct {
	Surface     Money `json:"surface"`
	Hills       Money `json:"hills"`
	Viaducts    Money `json:"viaducts"`
	Tunnels     Money `json:"tunnels"`
	Management  Money `json:"management"`
	Contingency Money `json:"contingency"`
}

// Stations are delivered with their line. Ready is a resolved-month offset;
// public starter stations have Ready zero. Open is authoritative eligibility.
type Station struct {
	CityID CityID    `json:"city_id"`
	Owner  CompanyID `json:"owner"`
	Public bool      `json:"public"`
	Ready  int       `json:"ready"`
	Open   bool      `json:"open"`
}
type TrainFamily struct {
	ID                  string `json:"id"`
	NameKey             string `json:"name_key"`
	Seats               int    `json:"seats"`
	LeaseMonthly        Money  `json:"lease_monthly"`
	RunningPerKM        Money  `json:"running_per_km"`
	PublicAccessPerKM   Money  `json:"public_access_per_km"`
	StationPerJourney   Money  `json:"station_per_journey"`
	DailyMinutes        int    `json:"daily_minutes"`
	RoundTripTurnaround int    `json:"round_trip_turnaround"`
}
type Corridor struct {
	ReplacesCorridor   string      `json:"replaces_corridor,omitempty"`
	StationNames       []string    `json:"station_names,omitempty"`
	Via                []string    `json:"via,omitempty"`
	ThroughOnly        bool        `json:"through_only,omitempty"`
	Retired            bool        `json:"retired,omitempty"`
	PartOf             string      `json:"part_of,omitempty"`
	ParallelTo         []string    `json:"parallel_to,omitempty"`
	ConstructionKind   string      `json:"construction_kind,omitempty"`
	OpeningService     string      `json:"opening_service,omitempty"`
	ConstructionNote   string      `json:"construction_note,omitempty"`
	InfrastructureKind string      `json:"infrastructure_kind,omitempty"`
	SharedSections     []string    `json:"shared_sections,omitempty"`
	ID                 string      `json:"id"`
	FromCity           CityID      `json:"from_city"`
	ToCity             CityID      `json:"to_city"`
	Existing           bool        `json:"existing"`
	DailyMarket        int         `json:"daily_market"`
	OutsideCost        Money       `json:"outside_cost"`
	Alignments         []Alignment `json:"alignments"`
}

// ProjectID resolves a service section to its construction contract.
func (c Corridor) ProjectID() string {
	if c.PartOf != "" {
		return c.PartOf
	}
	return c.ID
}

type Project struct {
	CorridorID  string    `json:"corridor_id"`
	AlignmentID string    `json:"alignment_id"`
	Owner       CompanyID `json:"owner"`
	Started     int       `json:"started"`
	Ready       int       `json:"ready"`
}
type Service struct {
	CorridorID string `json:"corridor_id"`
	Frequency  int    `json:"frequency"`
	Fare       Money  `json:"fare"`
}
type FleetLease struct {
	ID         uint64 `json:"id"`
	CorridorID string `json:"corridor_id"`
}
type IdleLine struct {
	CorridorID  string `json:"corridor_id"`
	FromCity    CityID `json:"from_city"`
	ToCity      CityID `json:"to_city"`
	Maintenance Money  `json:"maintenance"`
	Leases      Money  `json:"leases"`
	NeedsTrains bool   `json:"needs_trains"`
}
type CashFlows struct {
	Construction Money `json:"construction"`
	Grants       Money `json:"grants"`
	Borrowed     Money `json:"borrowed"`
	Repaid       Money `json:"repaid"`
}

const MaxFrequency = 16

type ForecastRequest struct {
	ReplaceServiceID string `json:"replace_service_id"`
	CorridorID       string `json:"corridor_id"`
	AlignmentID      string `json:"alignment_id"`
	Frequency        int    `json:"frequency"`
	Fare             Money  `json:"fare"`
}
type Forecast struct {
	RunningCosts             Money `json:"running_costs"`
	AccessCosts              Money `json:"access_costs"`
	HandlingCosts            Money `json:"handling_costs"`
	LeaseCosts               Money `json:"lease_costs"`
	MaintenanceCosts         Money `json:"maintenance_costs"`
	BreakEvenAvailable       bool  `json:"break_even_available"`
	BreakEvenUsesObservedMix bool  `json:"break_even_uses_observed_mix"`
	BreakEvenPassengers      int   `json:"break_even_passengers"`

	CapacityUsage      []CapacityUsage `json:"capacity_usage"`
	AdditionalLeases   int             `json:"additional_leases"`
	Passengers         int             `json:"passengers"`
	Capacity           int             `json:"capacity"`
	FleetNeeded        int             `json:"fleet_needed"`
	FleetAvailable     int             `json:"fleet_available"`
	Revenue            Money           `json:"revenue"`
	Costs              Money           `json:"costs"`
	Contribution       Money           `json:"contribution"`
	Open               bool            `json:"open"`
	DeliveredFrequency int             `json:"delivered_frequency"`
	CompanyPassengers  int             `json:"company_passengers"`
	CompanyRevenue     Money           `json:"company_revenue"`
	CompanyChange      Money           `json:"company_change"`
	FeederRevenue      Money           `json:"feeder_revenue"`
	KM                 int             `json:"km"`
	Minutes            int             `json:"minutes"`
}
type Frame struct {
	PublicNetwork  bool          `json:"public_network,omitempty"`
	Routes         []Route       `json:"routes"`
	Network        NetworkReport `json:"network"`
	Day            []DayTrip     `json:"day"`
	WorldID        string        `json:"world_id"`
	Revision       uint64        `json:"revision"`
	Month          Month         `json:"month"`
	MonthsResolved int           `json:"months_resolved"`
	CampaignMonths int           `json:"campaign_months"`
	Player         Company       `json:"player"`
	Cities         []City        `json:"cities"`
	Stations       []Station     `json:"stations"`
	TrainFamily    TrainFamily   `json:"train_family"`
	Reports        []Report      `json:"reports"`
	Checkpoint     string        `json:"checkpoint"`
	Corridors      []Corridor    `json:"corridors"`
	Projects       []Project     `json:"projects"`
	Services       []Service     `json:"services"`
	Fleet          []FleetLease  `json:"fleet"`
	IdleLines      []IdleLine    `json:"idle_lines"`
	Debt           Money         `json:"debt"`
	DebtLimit      Money         `json:"debt_limit"`
	Pending        CashFlows     `json:"pending"`
}
type Command struct {
	ReplaceServiceID string `json:"replace_service_id"`
	LeaseCount       int    `json:"lease_count"`
	Kind             string `json:"kind"`
	Revision         uint64 `json:"revision"`
	Name             string `json:"name"`
	CorridorID       string `json:"corridor_id"`
	AlignmentID      string `json:"alignment_id"`
	Frequency        int    `json:"frequency"`
	Fare             Money  `json:"fare"`
	Amount           Money  `json:"amount"`
}

const (
	NewGame       = "new_game"
	AdvanceMonth  = "advance_month"
	RenameCompany = "rename_company"
	BuildLine     = "build_line"
	SetService    = "set_service"
	LeaseTrain    = "lease_train"
	ReturnTrain   = "return_train"
	TakeLoan      = "take_loan"
	RepayLoan     = "repay_loan"
	AdvanceYear   = "advance_year"
	ExtendService = "extend_service"
)

type Error struct {
	Code          string
	CapacityUsage []CapacityUsage
}

func (e *Error) Error() string { return e.Code }

// Game serializes commands, persists before acknowledgment, and returns copies.
type Game interface {
	Snapshot() Frame
	Apply(Command) (Frame, error)
	Export() ([]byte, error)
	Forecast(ForecastRequest) (Forecast, error)
}

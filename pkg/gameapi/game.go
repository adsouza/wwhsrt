// Package gameapi is the renderer-independent driving contract.
package gameapi

type CityID string
type CompanyID string
type Money int64 // Game USD cents.

type City struct {
	ID   CityID  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
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
	CorridorID  string `json:"corridor_id"`
	AlignmentID string `json:"alignment_id"`
	Frequency   int    `json:"frequency"`
	Passengers  int    `json:"passengers"`
	Revenue     Money  `json:"revenue"`
	Fare        Money  `json:"fare"`
}
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}
type Alignment struct {
	ID        string     `json:"id"`
	KM        int        `json:"km"`
	Minutes   int        `json:"minutes"`
	Months    int        `json:"months"`
	SurfaceKM int        `json:"surface_km"`
	HillKM    int        `json:"hill_km"`
	ViaductKM int        `json:"viaduct_km"`
	TunnelKM  int        `json:"tunnel_km"`
	Path      []GeoPoint `json:"path"`
	Cost      Money      `json:"cost"`
	Grant     Money      `json:"grant"`
}
type Corridor struct {
	ID          string      `json:"id"`
	FromCity    CityID      `json:"from_city"`
	ToCity      CityID      `json:"to_city"`
	Existing    bool        `json:"existing"`
	DailyMarket int         `json:"daily_market"`
	OutsideCost Money       `json:"outside_cost"`
	Alignments  []Alignment `json:"alignments"`
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
type ForecastRequest struct {
	CorridorID  string `json:"corridor_id"`
	AlignmentID string `json:"alignment_id"`
	Frequency   int    `json:"frequency"`
	Fare        Money  `json:"fare"`
}
type Forecast struct {
	Passengers     int   `json:"passengers"`
	Capacity       int   `json:"capacity"`
	FleetNeeded    int   `json:"fleet_needed"`
	FleetAvailable int   `json:"fleet_available"`
	Revenue        Money `json:"revenue"`
	Costs          Money `json:"costs"`
	Contribution   Money `json:"contribution"`
	Open           bool  `json:"open"`
}
type Frame struct {
	WorldID        string       `json:"world_id"`
	Revision       uint64       `json:"revision"`
	Month          Month        `json:"month"`
	MonthsResolved int          `json:"months_resolved"`
	CampaignMonths int          `json:"campaign_months"`
	Player         Company      `json:"player"`
	Cities         []City       `json:"cities"`
	Reports        []Report     `json:"reports"`
	Checkpoint     string       `json:"checkpoint"`
	Corridors      []Corridor   `json:"corridors"`
	Projects       []Project    `json:"projects"`
	Services       []Service    `json:"services"`
	Fleet          []FleetLease `json:"fleet"`
	IdleLines      []IdleLine   `json:"idle_lines"`
	Debt           Money        `json:"debt"`
	DebtLimit      Money        `json:"debt_limit"`
	Pending        CashFlows    `json:"pending"`
}
type Command struct {
	Kind        string `json:"kind"`
	Revision    uint64 `json:"revision"`
	Name        string `json:"name"`
	CorridorID  string `json:"corridor_id"`
	AlignmentID string `json:"alignment_id"`
	Frequency   int    `json:"frequency"`
	Fare        Money  `json:"fare"`
	Amount      Money  `json:"amount"`
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
)

type Error struct{ Code string }

func (e *Error) Error() string { return e.Code }

// Game serializes commands, persists before acknowledgment, and returns copies.
type Game interface {
	Snapshot() Frame
	Apply(Command) (Frame, error)
	Export() ([]byte, error)
	Forecast(ForecastRequest) (Forecast, error)
}

// Package contracts contains compile-checked design references, not the game engine.
// Domain shapes and save DTOs are kept here until their production packages exist.
package contracts

type (
	WorldID       string
	CompanyID     string
	CityID        string
	NodeID        string
	StationID     string
	SectionID     string
	ProjectID     string
	ServiceID     string
	TrainsetID    string
	OrderID       string
	LoanID        string
	GrantID       string
	AuctionID     string
	BidID         string
	ReservationID string
	LeaseID       string
	Money         int64 // Game USD cents; arithmetic must check overflow.
	FareRate      int64 // Hundredths of a game cent (0.0001 USD) per passenger-km.
	Metres        int64
	Seconds       int64
	Passengers    int64
	BasisPoints   uint32
)

// World illustrates aggregate ownership. Production domain entities will have
// private fields and invariant-preserving methods, not JSON persistence tags.
// Map traversal must use sorted IDs for consequential operations.
type World struct {
	id                        WorldID
	month                     GameMonth
	startMonth                GameMonth
	campaignMonths            uint16
	restructuringWindowMonths uint16
	seed                      uint64
	rules                     RulesRef
	networkRevision           uint64
	nextID                    uint64
	companies                 map[CompanyID]Company
	nodes                     map[NodeID]Node
	stations                  map[StationID]Station
	stationAccess             []StationAccess
	trackAccess               []TrackAccessAgreement
	sections                  map[SectionID]TrackSection
	projects                  map[ProjectID]Project
	trainsets                 map[TrainsetID]Trainset
	orders                    map[OrderID]TrainOrder
	services                  map[ServiceID]ServiceSpec
	plans                     map[CompanyID]CompanyPlan
	loans                     map[LoanID]Loan
	leases                    map[LeaseID]Lease
	grants                    map[GrantID]Grant
	concessions               []Concession
	auctions                  map[AuctionID]Auction
	bids                      map[BidID]Bid
	reservations              map[ReservationID]Reservation
	cashJournal               []CashEntry
	reports                   []MonthReport
	latestReplay              *OperatingDay
	rngState                  *RNGState
}

// GameMonth is the calendar month still to be resolved; no wall-clock timestamp.
type GameMonth struct {
	Year  uint16 `json:"year"`
	Month uint8  `json:"month"`
}

type DataRef struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type RulesRef struct {
	Scenario   DataRef `json:"scenario"`
	Balance    DataRef `json:"balance"`
	Simulation string  `json:"simulation"`
	Demand     string  `json:"demand"`
	Allocation string  `json:"allocation"`
	RNG        string  `json:"rng"`
}

// PayloadV1 is an application-owned wire type, never a player-facing online frame.
// See codec.go for its raw-JSON envelope and partial reference encoder/decoder.
type PayloadV1 struct {
	StateRevision uint64   `json:"state_revision,string"`
	Rules         RulesRef `json:"rules"`
	World         WorldV1  `json:"world"`
}

type WorldV1 struct {
	ID             WorldID   `json:"id"`
	Month          GameMonth `json:"month"`
	CampaignMonths uint16    `json:"campaign_months"`
	// Pinned at world creation: max(baseline window, real-time minimum at the interval).
	RestructuringWindowMonths uint16                 `json:"restructuring_window_months"`
	StartMonth                GameMonth              `json:"start_month"`
	Seed                      uint64                 `json:"seed,string"`
	NextID                    uint64                 `json:"next_id,string"`
	NetworkRevision           uint64                 `json:"network_revision,string"`
	Companies                 []Company              `json:"companies"`
	Nodes                     []Node                 `json:"nodes"`
	Stations                  []Station              `json:"stations"`
	StationAccess             []StationAccess        `json:"station_access"`
	TrackAccess               []TrackAccessAgreement `json:"track_access"`
	Sections                  []TrackSection         `json:"sections"`
	Projects                  []Project              `json:"projects"`
	Trainsets                 []Trainset             `json:"trainsets"`
	Orders                    []TrainOrder           `json:"orders"`
	Services                  []ServiceSpec          `json:"services"`
	Plans                     []CompanyPlan          `json:"plans"`
	Loans                     []Loan                 `json:"loans"`
	Leases                    []Lease                `json:"leases"`
	Grants                    []Grant                `json:"grants"`
	Concessions               []Concession           `json:"concessions"`
	Auctions                  []Auction              `json:"auctions"`
	Bids                      []Bid                  `json:"bids"`
	Reservations              []Reservation          `json:"reservations"`
	CashJournal               []CashEntry            `json:"cash_journal"`
	Reports                   []MonthReport          `json:"reports"`
	LatestReplay              *OperatingDay          `json:"latest_replay"`
	RNGState                  *RNGState              `json:"rng_state"`
}

type Company struct {
	ID                CompanyID  `json:"id"`
	Name              string     `json:"name"`
	Status            string     `json:"status"` // active, restructuring, bankrupt
	HomeCity          CityID     `json:"home_city"`
	OpeningCash       Money      `json:"opening_cash,string"`
	Cash              Money      `json:"cash,string"`
	RestructuringEnds *GameMonth `json:"restructuring_ends"`
}

type Ownership struct {
	Kind      string     `json:"kind"`       // public, company, estate
	CompanyID *CompanyID `json:"company_id"` // nil only for public
}

type Node struct {
	ID          NodeID  `json:"id"`
	Kind        string  `json:"kind"` // station, junction
	CityID      *CityID `json:"city_id"`
	GeometryRef string  `json:"geometry_ref"` // scenario-bundle geometry, not pixels
}

type Station struct {
	ID               StationID `json:"id"`
	NodeID           NodeID    `json:"node_id"`
	Status           string    `json:"status"` // construction, operational, closed
	Owner            Ownership `json:"owner"`
	TrainCallsPerDay uint32    `json:"train_calls_per_day"`
	BookValue        Money     `json:"book_value,string"`
}

type StationAccess struct {
	StationID StationID `json:"station_id"`
	CompanyID CompanyID `json:"company_id"`
	TermsRef  string    `json:"terms_ref"`
}

type TrackAccessOffer struct {
	Revision      uint64 `json:"revision,string"`
	FeePerTrainKM Money  `json:"fee_per_train_km,string"`
}

type TrackAccessAgreement struct {
	SectionID     SectionID  `json:"section_id"`
	OwnerID       CompanyID  `json:"owner_id"`
	OperatorID    CompanyID  `json:"operator_id"`
	FeePerTrainKM Money      `json:"fee_per_train_km,string"`
	Effective     GameMonth  `json:"effective"`
	Ends          *GameMonth `json:"ends"` // First month without access.
}

type TrackSection struct {
	ID                SectionID         `json:"id"`
	From              NodeID            `json:"from"`
	To                NodeID            `json:"to"`
	AlignmentRef      string            `json:"alignment_ref"`
	EngineeringRef    string            `json:"engineering_ref"`
	TrackClassRef     string            `json:"track_class_ref"`
	Owner             Ownership         `json:"owner"`
	Access            string            `json:"access"` // regulated_open, owner_only
	PaidAccessOffer   *TrackAccessOffer `json:"paid_access_offer"`
	Status            string            `json:"status"` // construction, operational, closed
	SpeedLimitKPH     uint16            `json:"speed_limit_kph"`
	ReturnPairsPerDay uint32            `json:"return_pairs_per_day"`
	BookValue         Money             `json:"book_value,string"`
}

type Installment struct {
	Due    GameMonth `json:"due"`
	Amount Money     `json:"amount,string"`
	Paid   bool      `json:"paid"`
}

type Project struct {
	ID             ProjectID     `json:"id"`
	CompanyID      CompanyID     `json:"company_id"`
	Kind           string        `json:"kind"` // new_line, alignment_upgrade, capacity_upgrade, station_upgrade
	AlignmentRef   string        `json:"alignment_ref"`
	SectionIDs     []SectionID   `json:"section_ids"`
	StationIDs     []StationID   `json:"station_ids"`
	Started        GameMonth     `json:"started"`
	Ready          GameMonth     `json:"ready"`
	Status         string        `json:"status"`          // pending, committed, building, completed, cancelled
	CompetitionRef *string       `json:"competition_ref"` // Reviewed competition for a pending request.
	TotalCost      Money         `json:"total_cost,string"`
	Installments   []Installment `json:"installments"`
}

type Trainset struct {
	ID          TrainsetID `json:"id"`
	CompanyID   CompanyID  `json:"company_id"`
	ModelRef    string     `json:"model_ref"`
	HomeStation StationID  `json:"home_station"`
	Status      string     `json:"status"` // delivery, operational, relocation, retired (sold or withdrawn)
	Ready       GameMonth  `json:"ready"`
	BookValue   Money      `json:"book_value,string"`
	LeaseID     *LeaseID   `json:"lease_id"`
}

type TrainOrder struct {
	ID           OrderID       `json:"id"`
	CompanyID    CompanyID     `json:"company_id"`
	TrainsetIDs  []TrainsetID  `json:"trainset_ids"`
	Status       string        `json:"status"` // ordered, delivered, cancelled
	Ready        GameMonth     `json:"ready"`
	TotalCost    Money         `json:"total_cost,string"`
	Installments []Installment `json:"installments"`
}

type Traversal struct {
	SectionID SectionID `json:"section_id"`
	Forward   bool      `json:"forward"`
}

type ServiceSpec struct {
	ID                 ServiceID    `json:"id"`
	CompanyID          CompanyID    `json:"company_id"`
	Path               []Traversal  `json:"path"`  // ordered; return journey reverses it
	Stops              []StationID  `json:"stops"` // ordered calls, not every passing node
	TrainsetIDs        []TrainsetID `json:"trainset_ids"`
	ReturnTripsPerDay  uint16       `json:"return_trips_per_day"`
	FarePerPassengerKM FareRate     `json:"fare_per_passenger_km,string"` // charged on distance ridden
	Suspended          bool         `json:"suspended"`
}

type StandingPolicy struct {
	MinimumFreeCash      Money   `json:"minimum_free_cash,string"`
	WithdrawEligibleBids bool    `json:"withdraw_eligible_bids"`
	ApprovedCreditRef    *LoanID `json:"approved_credit_ref"`
}

type CompanyPlan struct {
	CompanyID      CompanyID      `json:"company_id"`
	Revision       uint64         `json:"revision,string"`
	EffectiveMonth GameMonth      `json:"effective_month"`
	Services       []ServiceSpec  `json:"services"` // complete desired set, not a patch
	StandingPolicy StandingPolicy `json:"standing_policy"`
}

type Loan struct {
	ID               LoanID      `json:"id"`
	CompanyID        CompanyID   `json:"company_id"`
	ContractRef      string      `json:"contract_ref"`
	Outstanding      Money       `json:"outstanding,string"`
	CreditLimit      Money       `json:"credit_limit,string"`
	AnnualRateBPS    BasisPoints `json:"annual_rate_bps"`
	ScheduledPayment Money       `json:"scheduled_payment,string"`
	PrincipalArrears Money       `json:"principal_arrears,string"` // subset of Outstanding
	InterestArrears  Money       `json:"interest_arrears,string"`  // additional liability
	NextDue          *GameMonth  `json:"next_due"`
	RemainingMonths  uint16      `json:"remaining_months"`
}

type Lease struct {
	ID              LeaseID      `json:"id"`
	CompanyID       CompanyID    `json:"company_id"`
	ContractRef     string       `json:"contract_ref"`
	TrainsetIDs     []TrainsetID `json:"trainset_ids"`
	MonthlyPayment  Money        `json:"monthly_payment,string"`
	Arrears         Money        `json:"arrears,string"`
	NextDue         GameMonth    `json:"next_due"`
	RemainingMonths uint16       `json:"remaining_months"`
}

type Grant struct {
	ID        GrantID    `json:"id"`
	CompanyID CompanyID  `json:"company_id"`
	OfferRef  string     `json:"offer_ref"`
	ProjectID *ProjectID `json:"project_id"`
	Awarded   Money      `json:"awarded,string"`
	Disbursed Money      `json:"disbursed,string"`
}

type Concession struct {
	CompanyID       CompanyID  `json:"company_id"`
	JurisdictionRef string     `json:"jurisdiction_ref"`
	TermsRef        string     `json:"terms_ref"`
	Effective       GameMonth  `json:"effective"` // First usable month; a later month means the lead time is still running.
	Expires         *GameMonth `json:"expires"`
}

type Auction struct {
	ID           AuctionID   `json:"id"`
	SectionIDs   []SectionID `json:"section_ids"`
	ReservePrice Money       `json:"reserve_price,string"`
	Closes       GameMonth   `json:"closes"` // Offline cutoff or expected online settlement.
	Status       string      `json:"status"` // listed, sold, unsold, withdrawn
	Winner       *CompanyID  `json:"winner"`
	SalePrice    *Money      `json:"sale_price,string"`
}

type Bid struct {
	ID            BidID         `json:"id"`
	AuctionID     AuctionID     `json:"auction_id"`
	CompanyID     CompanyID     `json:"company_id"`
	Maximum       Money         `json:"maximum,string"`
	ReservationID ReservationID `json:"reservation_id"`
}

type Reservation struct {
	ID        ReservationID `json:"id"`
	CompanyID CompanyID     `json:"company_id"`
	Kind      string        `json:"kind"` // construction, auction
	TargetID  string        `json:"target_id"`
	Amount    Money         `json:"amount,string"`
}

type CashEntry struct {
	Sequence    uint64    `json:"sequence,string"`
	CompanyID   CompanyID `json:"company_id"`
	Month       GameMonth `json:"month"`
	Category    string    `json:"category"` // versioned category in the balance bundle
	Delta       Money     `json:"delta,string"`
	ReferenceID string    `json:"reference_id"`
	TransferID  *string   `json:"transfer_id"` // links equal/opposite intercompany payments
}

type MonthReport struct {
	Month     GameMonth       `json:"month"`
	Companies []CompanyReport `json:"companies"`
	Services  []ServiceReport `json:"services"`
	Sections  []SectionReport `json:"sections"`
}

type CompanyReport struct {
	CompanyID         CompanyID `json:"company_id"`
	OpeningCash       Money     `json:"opening_cash,string"`
	ClosingCash       Money     `json:"closing_cash,string"`
	OperatingResult   Money     `json:"operating_result,string"`
	NonCashCost       Money     `json:"non_cash_cost,string"`
	ClosingAssetValue Money     `json:"closing_asset_value,string"`
	OutstandingDebt   Money     `json:"outstanding_debt,string"`
	ClosingReserved   Money     `json:"closing_reserved,string"`
	FirstCashSequence uint64    `json:"first_cash_sequence,string"`
	LastCashSequence  uint64    `json:"last_cash_sequence,string"`
	// Rail journeys for this company's offers: requested = carried + rejected;
	// alternate_served is the carried subset moved to a second itinerary.
	JourneysRequested       Passengers `json:"journeys_requested,string"`
	JourneysCarried         Passengers `json:"journeys_carried,string"`
	JourneysAlternateServed Passengers `json:"journeys_alternate_served,string"`
	JourneysRejected        Passengers `json:"journeys_rejected,string"`
}

type ServiceReport struct {
	ServiceID                  ServiceID   `json:"service_id"`
	Spec                       ServiceSpec `json:"spec"` // immutable resolved config
	RequestedReturnTripsPerDay uint16      `json:"requested_return_trips_per_day"`
	GrantedReturnTripsPerDay   uint16      `json:"granted_return_trips_per_day"`
	Passengers                 Passengers  `json:"passengers,string"` // boardings on this service
	PassengerMetres            Metres      `json:"passenger_metres,string"`
	TrainMetres                Metres      `json:"train_metres,string"`
	Revenue                    Money       `json:"revenue,string"`
	OperatingCost              Money       `json:"operating_cost,string"`
}

// SectionReport explains shared-capacity outcomes for one track section.
type SectionReport struct {
	SectionID                  SectionID   `json:"section_id"`
	CapacityReturnPairsPerDay  uint32      `json:"capacity_return_pairs_per_day"`
	RequestedReturnPairsPerDay uint32      `json:"requested_return_pairs_per_day"`
	GrantedReturnPairsPerDay   uint32      `json:"granted_return_pairs_per_day"`
	ServiceIDs                 []ServiceID `json:"service_ids"` // contributing services, sorted
}

type OperatingDay struct {
	Month    GameMonth       `json:"month"`
	Services []ReplayService `json:"services"`
	Stations []ReplayStation `json:"stations"`
}

type ReplayService struct {
	ServiceID        ServiceID   `json:"service_id"`
	Path             []Traversal `json:"path"`
	LegSeconds       []Seconds   `json:"leg_seconds"` // each value must fit a JSON-safe int
	DepartureMinutes []uint16    `json:"departure_minutes"`
	// Average occupancy per leg, aligned with LegSeconds; inbound legs in travel order.
	OutboundLegLoadBPS []BasisPoints `json:"outbound_leg_load_bps"`
	InboundLegLoadBPS  []BasisPoints `json:"inbound_leg_load_bps"`
}

// ReplayStation is aggregate representative-day activity, not individual passengers.
type ReplayStation struct {
	StationID  StationID  `json:"station_id"`
	Boardings  Passengers `json:"boardings,string"`
	Alightings Passengers `json:"alightings,string"`
	Transfers  Passengers `json:"transfers,string"`
}

type RNGState struct {
	Algorithm string   `json:"algorithm"`
	Words     []string `json:"words"` // canonical unsigned decimal strings
}

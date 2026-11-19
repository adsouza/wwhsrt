package gameapi

// Route is a service pattern, distinct from physical construction corridors.
// Existing corridor_id fields on Service/FleetLease are stable service-pattern
// IDs for wire/save compatibility. Direct patterns retain their corridor IDs.
type Route struct {
	StationNames       []string `json:"station_names,omitempty"`
	Via                []string `json:"via,omitempty"`
	InfrastructureKind string   `json:"infrastructure_kind,omitempty"`
	ConnectorKM        int      `json:"connector_km,omitempty"`
	SuggestedFare      Money    `json:"suggested_fare,omitempty"`
	SuggestedFrequency int      `json:"suggested_frequency,omitempty"`
	ID                 string   `json:"id"`
	Sections           []string `json:"sections"`
	Cities             []CityID `json:"cities"`
	Stops              []CityID `json:"stops"`
	KM                 int      `json:"km"`
	Minutes            int      `json:"minutes"`
	Open               bool     `json:"open"`
}
type MarketCity struct {
	City
	DailyTrips int    `json:"daily_trips"`
	Attraction int    `json:"attraction"`
	RoadRegion string `json:"road_region"`
}
type NetworkRules struct {
	SupersededRoutes  []string         `json:"superseded_routes,omitempty"`
	ID                string           `json:"id"`
	Cities            []MarketCity     `json:"cities"`
	Routes            []Route          `json:"routes"`
	ConventionalPairs [][]CityID       `json:"conventional_pairs"`
	PublicPaths       int              `json:"public_paths"`
	PrivatePaths      int              `json:"private_paths"`
	StationCalls      int              `json:"station_calls"`
	DwellMinutes      int              `json:"dwell_minutes"`
	TransferMinutes   int              `json:"transfer_minutes"`
	Distances         []MarketDistance `json:"distances"`
	Sparse            bool             `json:"sparse,omitempty"`
	DistanceKM        []uint16         `json:"distance_km,omitempty"`
}
type MarketDistance struct {
	From CityID `json:"from"`
	To   CityID `json:"to"`
	KM   int    `json:"km"`
}
type NetworkReport struct {
	UnservedDemand  int               `json:"unserved_demand,omitempty"`
	Month           Month             `json:"month"`
	Available       bool              `json:"available"`
	Demand          int               `json:"demand"`
	Requested       int               `json:"requested"`
	Carried         int               `json:"carried"`
	Transfers       int               `json:"transfers"`
	AlternateServed int               `json:"alternate_served"`
	Rejected        int               `json:"rejected"`
	Revenue         Money             `json:"revenue"`
	PassengerKM     int64             `json:"passenger_km"`
	SeatKM          int64             `json:"seat_km"`
	Markets         []MarketRun       `json:"markets"`
	Journeys        []JourneyFlow     `json:"journeys"`
	Services        []NetworkService  `json:"services"`
	Sections        []SectionUsage    `json:"sections"`
	Stations        []StationActivity `json:"stations"`
}
type MarketRun struct {
	From            CityID      `json:"from"`
	To              CityID      `json:"to"`
	Class           string      `json:"class"`
	Demand          int         `json:"demand"`
	Requested       int         `json:"requested"`
	Carried         int         `json:"carried"`
	AlternateServed int         `json:"alternate_served"`
	Rejected        int         `json:"rejected"`
	Road            int         `json:"road"`
	Air             int         `json:"air"`
	Conventional    int         `json:"conventional"`
	NoTrip          int         `json:"no_trip"`
	RailFare        Money       `json:"rail_fare"`
	RailMinutes     int         `json:"rail_minutes"`
	Offers          []ModeOffer `json:"offers"`
}
type ModeOffer struct {
	Mode        string `json:"mode"`
	Fare        Money  `json:"fare"`
	Minutes     int    `json:"minutes"`
	Generalized Money  `json:"generalized"`
}
type JourneyFlow struct {
	From       CityID   `json:"from"`
	To         CityID   `json:"to"`
	Class      string   `json:"class"`
	Passengers int      `json:"passengers"`
	Fare       Money    `json:"fare"`
	Minutes    int      `json:"minutes"`
	Services   []string `json:"services"`
	Transfers  []CityID `json:"transfers"`
	Alternate  bool     `json:"alternate"`
}
type NetworkService struct {
	ID        string    `json:"id"`
	Requested int       `json:"requested"`
	Delivered int       `json:"delivered"`
	Boardings int       `json:"boardings"`
	Revenue   Money     `json:"revenue"`
	Running   Money     `json:"running"`
	Access    Money     `json:"access"`
	Handling  Money     `json:"handling"`
	Legs      []LegLoad `json:"legs"`
}
type LegLoad struct {
	CorridorID string `json:"corridor_id"`
	From       CityID `json:"from"`
	To         CityID `json:"to"`
	KM         int    `json:"km"`
	Minutes    int    `json:"minutes"`
	Passengers int    `json:"passengers"`
	Capacity   int    `json:"capacity"`
}
type SectionUsage struct {
	CorridorID string `json:"corridor_id"`
	Requested  int    `json:"requested"`
	Delivered  int    `json:"delivered"`
	Capacity   int    `json:"capacity"`
}
type StationActivity struct {
	CityID     CityID `json:"city_id"`
	Calls      int    `json:"calls"`
	Capacity   int    `json:"capacity"`
	Boardings  int    `json:"boardings"`
	Alightings int    `json:"alightings"`
	Transfers  int    `json:"transfers"`
}
type DayTrip struct {
	ID        string    `json:"id"`
	ServiceID string    `json:"service_id"`
	Departure int       `json:"departure"` // Minutes since 06:00; repeat every 1440.
	Arrival   int       `json:"arrival"`
	Legs      []DayLeg  `json:"legs"`
	Calls     []DayCall `json:"calls"`
}
type DayLeg struct {
	CorridorID   string `json:"corridor_id"`
	AlignmentID  string `json:"alignment_id"`
	From         CityID `json:"from"`
	To           CityID `json:"to"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	LoadPermille int    `json:"load_permille"`
}
type DayCall struct {
	CityID    CityID `json:"city_id"`
	Arrival   int    `json:"arrival"`
	Departure int    `json:"departure"`
}

// CapacityUsage explains the hypothetical accepted plan, including replacement
// semantics. Track units are daily return trips; station units are daily calls.
// This projection is not part of the canonical save or settlement report.
type CapacityUsage struct {
	Kind           string            `json:"kind"`
	CorridorID     string            `json:"corridor_id,omitempty"`
	CityID         CityID            `json:"city_id,omitempty"`
	Requested      int               `json:"requested"`
	Capacity       int               `json:"capacity"`
	Excess         int               `json:"excess"`
	Draft          bool              `json:"draft"`
	DraftUse       int               `json:"draft_use"`
	OtherRequested int               `json:"other_requested"`
	MaxFrequency   int               `json:"max_frequency"`
	Services       []CapacityService `json:"services"`
}
type CapacityService struct {
	ServiceID string `json:"service_id"`
	Frequency int    `json:"frequency"`
	Usage     int    `json:"usage"`
	Draft     bool   `json:"draft"`
}

// MarketInspector supplies outside-mode detail for an OD omitted from the sparse
// company network projection. Queries never change the world or advance time.
type MarketInspector interface {
	InspectMarket(CityID, CityID, string) (MarketRun, error)
}

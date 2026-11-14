package gameapi

// EconomyRules is a versioned gameplay calibration. Currency values are cents.
type EconomyRules struct {
	ID                string `json:"id"`
	LeaseMonthly      Money  `json:"lease_monthly"`
	RunningPerKM      Money  `json:"running_per_km"`
	PublicAccessPerKM Money  `json:"public_access_per_km"`
	TripScale         int    `json:"trip_scale"`
	CatchmentBonusCap int    `json:"catchment_bonus_cap"`
	DailyTripCap      int    `json:"daily_trip_cap"`
	GravityPaddingKM  int    `json:"gravity_padding_km"`
	DomesticWeight    int    `json:"domestic_weight"`
	WaitNumerator     int    `json:"wait_numerator"`
	FareBase          Money  `json:"fare_base"`
	FarePerKM         Money  `json:"fare_per_km"`
}

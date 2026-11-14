package ui

import (
	"fmt"
	"strconv"
	"wwhsrt/pkg/gameapi"
)

type Detail struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func StationAt(f gameapi.Frame, id gameapi.CityID) (gameapi.Station, bool) {
	for _, station := range f.Stations {
		if station.CityID == id {
			return station, true
		}
	}
	return gameapi.Station{}, false
}
func StationDetails(f gameapi.Frame, locale string, id gameapi.CityID) []Detail {
	t := func(k string) string { return Text(locale, k) }
	station, ok := StationAt(f, id)
	if !ok {
		return []Detail{{t("station"), t("stationAbsent")}}
	}
	status := t("stationOpen")
	if !station.Open {
		status = t("stationBuilding")
	}
	owner := f.Player.Name
	if station.Public {
		owner = t("stationPublic")
	}
	rows := []Detail{{t("station"), status}, {t("stationOwner"), owner}}
	for _, city := range f.Cities {
		if city.ID == id && city.StationName != "" {
			rows = append(rows, Detail{t("mappedStation"), city.StationName})
			break
		}
	}
	if !station.Open {
		rows = append(rows, Detail{t("opens"), fmt.Sprintf("%04d-%02d", 2029+station.Ready/12, 1+station.Ready%12)})
	}
	return append(rows, Detail{t("stationHandling"), Cash(f.TrainFamily.StationPerJourney)})
}
func TrainDetails(f gameapi.Frame, locale string) []Detail {
	t := func(k string) string { return Text(locale, k) }
	a := f.TrainFamily
	return []Detail{{t("trainFamily"), t(a.NameKey)}, {t("seatsPerTrain"), strconv.Itoa(a.Seats)}, {t("trainLease"), Cash(a.LeaseMonthly)}, {t("trainRunning"), Cash(a.RunningPerKM)}, {t("trackAccess"), Cash(a.PublicAccessPerKM)}, {t("stationHandling"), Cash(a.StationPerJourney)}, {t("trainDay"), strconv.Itoa(a.DailyMinutes / 60)}, {t("trainTurnaround"), strconv.Itoa(a.RoundTripTurnaround)}}
}
func EstimateDetails(a gameapi.Alignment, locale string) []Detail {
	t := func(k string) string { return Text(locale, k) }
	e := a.Estimate
	rows := []Detail{}
	for _, p := range a.CostIntervals {
		rows = append(rows, Detail{t("regionalCostBasis") + " · " + p.Country, Cash(p.BasePerKM) + " / km"})
	}
	for _, r := range []struct {
		key   string
		money gameapi.Money
		km    int
	}{{"surfaceWorks", e.Surface, a.SurfaceKM}, {"hillWorks", e.Hills, a.HillKM}, {"viaductWorks", e.Viaducts, a.ViaductKM}, {"tunnelWorks", e.Tunnels, a.TunnelKM}} {
		if r.km > 0 {
			rows = append(rows, Detail{t(r.key) + " · " + strconv.Itoa(r.km) + " km", Cash(r.money)})
		}
	}
	return append(rows, Detail{t("managementAllowance"), Cash(e.Management)}, Detail{t("contingencyAllowance"), Cash(e.Contingency)}, Detail{t("totalCost"), Cash(a.Cost)})
}

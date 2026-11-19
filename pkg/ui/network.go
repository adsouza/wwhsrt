package ui

import (
	"fmt"
	"strings"
	"wwhsrt/pkg/gameapi"
)

func ServiceRoute(f gameapi.Frame, id string) (gameapi.Route, bool) {
	for _, r := range f.Routes {
		if r.ID == id {
			return r, true
		}
	}
	// Older fixture frames can still inspect a direct service.
	for _, c := range f.Corridors {
		if c.ID == id && !c.Retired && !c.ThroughOnly {
			return gameapi.Route{
				ID:       id,
				Sections: []string{id},
				Cities:   []gameapi.CityID{c.FromCity, c.ToCity},
				Stops:    []gameapi.CityID{c.FromCity, c.ToCity},
				Open:     Open(f, id),
			}, true
		}
	}
	return gameapi.Route{}, false
}
func RouteName(r gameapi.Route, locale string) string {
	if len(r.Cities) < 2 {
		return Text(locale, "retiredService")
	}
	suffix := ""
	if r.InfrastructureKind != "" {
		suffix = " · " + Text(locale, "trackKind_"+r.InfrastructureKind)
	}
	stops := r.Stops
	if len(stops) < 2 {
		stops = r.Cities
	}
	if len(stops) > 4 {
		return Text(
			locale,
			string(r.Cities[0]),
		) + " → " + Text(
			locale,
			string(r.Cities[len(r.Cities)-1]),
		) + " · " + fmt.Sprintf(
			Text(locale, "routeStopCount"),
			len(r.Stops),
		) + suffix
	}
	parts := []string{}
	for _, id := range stops {
		parts = append(parts, Text(locale, string(id)))
	}
	name := strings.Join(parts, " → ")
	if len(r.Stops) < len(r.Cities) {
		name += " · " + Text(locale, "express")
	}
	return name + suffix
}
func StopNames(stops []gameapi.CityID, locale string) string {
	out := []string{}
	for _, id := range stops {
		out = append(out, Text(locale, string(id)))
	}
	return strings.Join(out, " → ")
}
func ReplayClock(minute float64) string {
	m := (int(minute) + 360) % 1440
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}
func NetworkDetails(f gameapi.Frame, locale string) []Detail {
	n := f.Network
	if !n.Available {
		return []Detail{{Text(locale, "networkReport"), Text(locale, "noNetworkReport")}}
	}
	rows := []Detail{
		{Text(locale, "companyJourneys"), fmt.Sprint(n.Carried)},
		{Text(locale, "transferCount"), fmt.Sprint(n.Transfers)},
		{Text(locale, "railRequested"), fmt.Sprint(n.Requested)},
		{Text(locale, "alternateServed"), fmt.Sprint(n.AlternateServed)},
		{Text(locale, "railRejected"), fmt.Sprint(n.Rejected)},
	}
	load := int64(0)
	if n.SeatKM > 0 {
		load = n.PassengerKM * 1000 / n.SeatKM
	}
	rows = append(rows, Detail{Text(locale, "networkLoad"), fmt.Sprintf("%.1f%%", float64(load)/10)})
	for _, section := range n.Sections {
		coName := section.CorridorID
		for _, co := range f.Corridors {
			if co.ID == section.CorridorID {
				coName = Text(locale, string(co.FromCity)) + " – " + Text(locale, string(co.ToCity))
			}
		}
		rows = append(
			rows,
			Detail{
				coName,
				fmt.Sprintf(
					"%s %d / %d · %s %d",
					Text(locale, "pathsDelivered"),
					section.Delivered,
					section.Capacity,
					Text(locale, "pathsRequested"),
					section.Requested,
				),
			},
		)
	}
	for _, station := range n.Stations {
		rows = append(
			rows,
			Detail{
				Text(locale, string(station.CityID)),
				fmt.Sprintf(
					"%s %d/%d · %s %d · %s %d",
					Text(locale, "stationCalls"),
					station.Calls,
					station.Capacity,
					Text(locale, "boardings"),
					station.Boardings,
					Text(locale, "transferCount"),
					station.Transfers,
				),
			},
		)
	}
	return rows
}

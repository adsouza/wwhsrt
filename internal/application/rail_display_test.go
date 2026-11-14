package application

import (
	"reflect"
	"testing"
	"wwhsrt/assets"
	"wwhsrt/pkg/gameapi"
)

func TestPublicRailDrawingsConnectMappedStationsWithoutChangingEconomy(t *testing.T) {
	g := openPublic(t, nil)
	before, _ := g.Export()
	f := g.Snapshot()
	byCity := map[gameapi.CityID]gameapi.GeoPoint{}
	for _, c := range f.Cities {
		lat, lon := c.Lat, c.Lon
		if c.StationName != "" {
			lat, lon = c.StationLat, c.StationLon
		}
		byCity[c.ID] = gameapi.GeoPoint{Lat: lat, Lon: lon}
	}
	for _, co := range f.Corridors {
		for _, a := range co.Alignments {
			if a.Path[0] != byCity[co.FromCity] || a.Path[len(a.Path)-1] != byCity[co.ToCity] {
				t.Fatalf("%s/%s drawing detached from station", co.ID, a.ID)
			}
		}
	}
	osaka := gameapi.GeoPoint{Lat: 34.73354, Lon: 135.50019}
	kyoto := gameapi.GeoPoint{Lat: 34.98535, Lon: 135.75877}
	var tail []gameapi.GeoPoint
	for _, co := range f.Corridors {
		if co.ID == "public_kyoto__osaka" {
			tail = co.Alignments[0].Path
		}
	}
	if len(tail) < 3 || tail[0] != kyoto || tail[len(tail)-1] != osaka {
		t.Fatal("source approach missing")
	}
	for _, co := range f.Corridors {
		if co.ID != "nagoya-osaka" && co.ID != "tsuruga-osaka" {
			continue
		}
		for _, a := range co.Alignments {
			if len(a.Path) < len(tail) || !reflect.DeepEqual(a.Path[len(a.Path)-len(tail):], tail) {
				t.Fatalf("%s/%s kept schematic city-centre tail", co.ID, a.ID)
			}
		}
	}
	original := g.catalog.Corridors()
	for i, co := range f.Corridors {
		for j, a := range co.Alignments {
			a.Path = nil
			b := original[i].Alignments[j]
			b.Path = nil
			if !reflect.DeepEqual(a, b) {
				t.Fatal("render correction changed rules")
			}
		}
	}
	f.Corridors[0].Alignments[0].Path[0].Lat = 0
	if g.Snapshot().Corridors[0].Alignments[0].Path[0].Lat == 0 {
		t.Fatal("frame leaked display asset")
	}
	after, _ := g.Export()
	if string(before) != string(after) {
		t.Fatal("drawing/inspection changed save")
	}
	if hash(assets.PublicNetworkV1) != "2a8d0522b5a678579d17d5f257d0f6c1fc1c4f4d79879414d720ee6249486e59" {
		t.Fatal("drawing fix changed pinned save catalog")
	}
}

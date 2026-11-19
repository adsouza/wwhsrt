package application

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"reflect"
	"slices"
	"testing"
	"wwhsrt/pkg/gameapi"
)

const (
	frankfurtLeg = "public_frankfurt_am_main__stop_node_2453992382"
	cologneLeg   = "public_koln__stop_node_2453992382"
)

func TestGermanTrackAndMontabaurGeometry(t *testing.T) {
	frame := openPublic(t, nil).Snapshot()
	corridors := make(map[string]gameapi.Corridor)
	for _, co := range frame.Corridors {
		corridors[co.ID] = co
	}
	trunk := corridors["public_frankfurt_am_main__koln"]
	a, b := corridors[frankfurtLeg].Alignments[0], corridors[cologneLeg].Alignments[0]
	reverse := slices.Clone(b.Path)
	slices.Reverse(reverse)
	if a.Path[len(a.Path)-1] != reverse[0] || !slices.Equal(append(slices.Clone(a.Path), reverse[1:]...), trunk.Alignments[0].Path) ||
		!slices.Equal(trunk.SharedSections, []string{frankfurtLeg, cologneLeg}) || a.KM+b.KM != trunk.Alignments[0].KM ||
		a.Minutes+b.Minutes != trunk.Alignments[0].Minutes {
		t.Fatal("Montabaur does not divide the express alignment")
	}
	hamm := corridors["public_hamburg__stop_node_2146217850"]
	if !hamm.Existing || hamm.InfrastructureKind != "hsr" || hamm.Alignments[0].KM != 322 || !slices.Equal(hamm.Via, []string{"Bremen Hbf", "Osnabrück Hbf", "Münster (Westf) Hbf"}) {
		t.Fatal("Hamburg–Hamm must use the qualifying Osnabrück alignment")
	}
	last := -1
	for _, stop := range [][2]float64{{8.81455, 53.08309}, {8.0616, 52.2728}, {7.6353, 51.9568}} {
		path := hamm.Alignments[0].Path
		line := slices.IndexFunc(path[last+1:], func(p gameapi.GeoPoint) bool { return math.Hypot((p.Lon-stop[0])*.62, p.Lat-stop[1]) < .005 })
		if line < 0 {
			t.Fatal("Hamburg–Hamm bypasses junction", stop)
		}
		last += line + 1
	}
	if _, ok := corridors["public_koln__stop_node_2146217850"]; ok {
		t.Fatal("conventional Cologne–Hamm still offered")
	}
	de := corridors["upgrade_de_berlin__dresden"]
	if de.ID == "" || de.Existing || de.Alignments[0].RetainedKM < 50 {
		t.Fatal("missing partially upgraded prospect", de.ID)
	}
	if _, ok := corridors["public_berlin__dresden"]; ok {
		t.Fatal("old Berlin–Dresden line still on starting map")
	}

}

func TestGermanTrackMigrationWithOperatingMontabaurServices(t *testing.T) {
	raw, err := os.ReadFile("testdata/german-v11.json")
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	var old rhinePayload
	if json.Unmarshal(raw, &env) != nil || env.Version != 11 || json.Unmarshal(env.Payload, &old) != nil || old.PublicSHA != germanPriorSHA {
		t.Fatal("invalid fixture")
	}
	store := &memoryStore{b: raw}
	g := openPublic(t, store)
	f := g.Snapshot()
	if !bytes.Equal(raw, store.b) || g.w.GermanSince() != old.Months || f.Player != old.Company || !reflect.DeepEqual(f.Reports, old.Reports) || !reflect.DeepEqual(f.Fleet, old.Fleet) || !reflect.DeepEqual(f.Services, old.Services) {
		t.Fatal("migration altered the operating company")
	}
	f = publicCommand(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if !reflect.DeepEqual(f.Reports[:old.Months], old.Reports) || openPublic(t, store).Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("settlement lost historical accounts")
	}
	if json.Unmarshal(store.b, &env) != nil || env.Version != 18 {
		t.Fatal("migration did not persist")
	}

	var next viennaPayload
	if err := json.Unmarshal(env.Payload, &next); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*viennaPayload){
		func(p *viennaPayload) { p.GermanPriorSHA = "unknown" },
		func(p *viennaPayload) { p.GermanSince = -1 },
		func(p *viennaPayload) { p.GermanSince = p.Months + 1 },
	} {
		bad := next
		mutate(&bad)
		env.Payload, _ = json.Marshal(bad)
		env.SHA = hash(env.Payload)
		encoded, _ := json.Marshal(env)
		if _, err := Open(&memoryStore{b: encoded}, nil); err == nil {
			t.Fatal("invalid German catalog history accepted")
		}
	}
}

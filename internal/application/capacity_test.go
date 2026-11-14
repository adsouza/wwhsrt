package application

import (
	"bytes"
	"errors"
	"testing"
	"wwhsrt/pkg/gameapi"
)

func capacityFixture(t *testing.T) *Game {
	t.Helper()
	g := openTest(t, nil)
	act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 300_000_000_000})
	act(t, g, gameapi.Command{Kind: gameapi.BuildLine, CorridorID: "tsuruga-osaka", AlignmentID: "mountain"})
	for i := 0; i < 3; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	for _, id := range []string{"tsuruga-osaka", "nagoya-tsuruga-express", "tokyo-tsuruga"} {
		f, e := g.Forecast(gameapi.ForecastRequest{CorridorID: id, Frequency: 4, Fare: 9000})
		if e != nil {
			t.Fatal(e)
		}
		for i := 0; i < f.FleetNeeded; i++ {
			act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: id})
		}
		act(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: id, Frequency: 4, Fare: 9000})
	}
	return g
}
func TestCapacityExplanationMatchesRejectionAndFreedSpace(t *testing.T) {
	g := capacityFixture(t)
	before, _ := g.Export()
	q := gameapi.ForecastRequest{CorridorID: "tokyo-tsuruga-express", Frequency: 1, Fare: 18925}
	f, err := g.Forecast(q)
	var e *gameapi.Error
	if !errors.As(err, &e) || e.Code != "track_capacity" || len(e.CapacityUsage) != 1 {
		t.Fatal(f, err)
	}
	c := e.CapacityUsage[0]
	if c.CorridorID != "tsuruga-osaka" || c.Requested != 13 || c.Capacity != 12 || c.DraftUse != 1 ||
		c.OtherRequested != 12 ||
		c.MaxFrequency != 0 ||
		c.Excess != 1 ||
		len(c.Services) != 4 {
		t.Fatal(c)
	}
	total := 0
	draft := 0
	for _, s := range c.Services {
		total += s.Usage
		if s.Draft {
			draft++
			if s.ServiceID != q.CorridorID || s.Usage != 1 {
				t.Fatal(s)
			}
		}
	}
	if total != 13 || draft != 1 {
		t.Fatal(c)
	}
	_, err = g.Apply(
		gameapi.Command{
			Kind:       gameapi.SetService,
			Revision:   revision(g),
			CorridorID: q.CorridorID,
			Frequency:  q.Frequency,
			Fare:       q.Fare,
		},
	)
	if !errors.As(err, &e) || e.Code != "track_capacity" || e.CapacityUsage[0].Requested != 13 {
		t.Fatal(err)
	}
	after, _ := g.Export()
	if !bytes.Equal(before, after) {
		t.Fatal("preview/rejection changed save")
	}
	// Editing the same service replaces its accepted frequency, rather than
	// charging both the old and new draft against capacity.
	f, err = g.Forecast(gameapi.ForecastRequest{CorridorID: "tsuruga-osaka", Frequency: 3, Fare: 9000})
	if err != nil || f.CapacityUsage[0].Requested != 11 || f.CapacityUsage[0].MaxFrequency != 4 {
		t.Fatal(f, err)
	}
	act(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: "tsuruga-osaka", Frequency: 3, Fare: 9000})
	f, err = g.Forecast(q)
	if err != nil || f.CapacityUsage[0].Requested != 12 || f.CapacityUsage[0].MaxFrequency != 1 {
		t.Fatal(f, err)
	}
	// Mutating a returned explanation cannot corrupt future validation.
	f.CapacityUsage[0].Services[0].Usage = 999
	fresh, err := g.Forecast(q)
	if err != nil || fresh.CapacityUsage[0].Requested != 12 {
		t.Fatal(fresh, err)
	}
}

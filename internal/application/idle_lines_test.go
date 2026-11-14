package application

import (
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestIdleLineWarningTracksOpeningFleetAndAcceptedServices(t *testing.T) {
	store := &memoryStore{}
	g := openTest(t, store)
	if len(g.Snapshot().IdleLines) != 0 {
		t.Fatal("untouched public network must not warn")
	}
	financedBuild(t, g, "hokkaido-extension", "mountain")
	for i := 0; i < 47; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	}
	if len(g.Snapshot().IdleLines) != 0 {
		t.Fatal("construction must not be flagged as an idle open line")
	}
	f := act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if len(f.IdleLines) != 1 {
		t.Fatal("newly opened line must warn immediately", f.IdleLines)
	}
	warning := f.IdleLines[0]
	if warning.CorridorID != "hokkaido-extension" || !warning.NeedsTrains || warning.Maintenance != 383927500 ||
		warning.Leases != 0 {
		t.Fatal("wrong line or costs", warning)
	}
	before := f.Checkpoint
	f.IdleLines[0].Maintenance = 0
	if g.Snapshot().IdleLines[0].Maintenance != 383927500 || g.Snapshot().Checkpoint != before {
		t.Fatal("warning aliases/mutates world")
	}
	g = openTest(t, store)
	if g.Snapshot().IdleLines[0] != warning {
		t.Fatal("reload lost warning")
	}
	f = act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: warning.CorridorID})
	if len(f.IdleLines) != 1 || f.IdleLines[0].NeedsTrains || f.IdleLines[0].Leases != 40000000 {
		t.Fatal("leasing alone must retain warning with idle lease cost", f.IdleLines)
	}
	before = f.Checkpoint
	if _, err := g.Forecast(
		gameapi.ForecastRequest{CorridorID: warning.CorridorID, Frequency: 4, Fare: 9000},
	); err != nil {
		t.Fatal(err)
	}
	if len(g.Snapshot().IdleLines) != 1 || g.Snapshot().Checkpoint != before {
		t.Fatal("preview must not clear warning or change state")
	}
	f = act(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: warning.CorridorID, Frequency: 4, Fare: 9000})
	if len(f.IdleLines) != 0 {
		t.Fatal("accepted running service should clear warning")
	}
	f = act(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: warning.CorridorID, Frequency: 0, Fare: 9000})
	if len(f.IdleLines) != 1 {
		t.Fatal("suspension must warn again")
	}
	act(t, g, gameapi.Command{Kind: gameapi.ReturnTrain, CorridorID: warning.CorridorID})
	f = act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if f.Reports[len(f.Reports)-1].Maintenance != warning.Maintenance {
		t.Fatal("warning costs disagree with actual settlement")
	}
}

func TestIdlePublicFleetWarnsWithoutFlaggingUnusedPublicLines(t *testing.T) {
	g := openTest(t, nil)
	f := act(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: "tokyo-nagoya", Frequency: 0, Fare: 9000})
	if len(f.IdleLines) != 1 || f.IdleLines[0].CorridorID != "tokyo-nagoya" || f.IdleLines[0].Maintenance != 0 ||
		f.IdleLines[0].Leases != 80000000 {
		t.Fatal("suspended public service should disclose idle fleet rent", f.IdleLines)
	}
	act(t, g, gameapi.Command{Kind: gameapi.ReturnTrain, CorridorID: "tokyo-nagoya"})
	f = act(t, g, gameapi.Command{Kind: gameapi.ReturnTrain, CorridorID: "tokyo-nagoya"})
	if len(f.IdleLines) != 0 {
		t.Fatal("unused public track with no continuing company costs must not warn")
	}
}

package application

import (
	"testing"
	"wwhsrt/pkg/gameapi"
)

// The starter company holds two trains, so forty more leases pass the old
// company-wide limit of thirty-two.
func leaseFortyMore(t *testing.T, g *Game) gameapi.Frame {
	t.Helper()
	for i := 0; i < 40; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "tokyo-nagoya"})
	}
	return act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
}

func TestLeasingHasNoCompanyFleetCap(t *testing.T) {
	f := leaseFortyMore(t, openTest(t, nil))
	if len(f.Fleet) != 42 || f.Reports[0].LeaseCosts != 42*f.TrainFamily.LeaseMonthly {
		t.Fatal("every lease must be held and billed", len(f.Fleet), f.Reports[0].LeaseCosts)
	}
}

func TestSaveWithLargeFleetReloads(t *testing.T) {
	s := &memoryStore{}
	f := leaseFortyMore(t, openTest(t, s))
	reopened, err := openMode(s, nil, false)
	if err != nil {
		t.Fatal("a committed large-fleet save must reopen:", err)
	}
	if got := reopened.Snapshot(); got.Checkpoint != f.Checkpoint || len(got.Fleet) != 42 {
		t.Fatal("reloaded campaign differs", len(got.Fleet))
	}
}

func TestExtensionLeasesPastThirtyTwoTrains(t *testing.T) {
	g := openTest(t, nil)
	for i := 0; i < 30; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "nagoya-osaka"})
	}
	f := act(
		t,
		g,
		gameapi.Command{
			Kind:             gameapi.ExtendService,
			CorridorID:       "tokyo-osaka",
			ReplaceServiceID: "tokyo-nagoya",
			Frequency:        6,
			Fare:             13800,
			LeaseCount:       1,
		},
	)
	if len(f.Fleet) != 33 {
		t.Fatal("extension did not order its additional lease", len(f.Fleet))
	}
}

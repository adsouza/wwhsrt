package application

import (
	"bytes"
	"reflect"
	"testing"

	"wwhsrt/pkg/gameapi"
)

func TestNewGameReplacesCampaignAtomicallyAndResumes(t *testing.T) {
	s := &memoryStore{}
	g := openTest(t, s)
	act(t, g, gameapi.Command{Kind: gameapi.RenameCompany, Name: "Old Railway"})
	financedBuild(t, g, "hokkaido-extension", "mountain")
	for i := 0; i < 4; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	act(t, g, gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "hokkaido-extension"})
	act(t, g, gameapi.Command{Kind: gameapi.SetService, CorridorID: "hokkaido-extension", Frequency: 1, Fare: 11000})
	act(t, g, gameapi.Command{Kind: gameapi.TakeLoan, Amount: 1234567})
	before, durable := g.Snapshot(), append([]byte{}, s.b...)
	reset := gameapi.Command{Kind: gameapi.NewGame, Revision: before.Revision}
	s.fail = true
	if _, err := g.Apply(reset); err == nil || err.Error() != "storage_failed" {
		t.Fatal("failed reset acknowledged", err)
	}
	if g.Snapshot().Checkpoint != before.Checkpoint || !bytes.Equal(s.b, durable) {
		t.Fatal("failed reset lost company")
	}
	s.fail = false
	f, err := g.Apply(reset)
	if err != nil {
		t.Fatal(err)
	}
	if f.Month != (gameapi.Month{Year: 2029, Month: 1}) || f.MonthsResolved != 0 || f.Player.Name != "Sunrise Rail" || f.Player.Cash != 10_000_000_000 || f.Debt != 0 || len(f.Reports) != 0 || len(f.Projects) != 0 || len(f.IdleLines) != 0 || f.Pending != (gameapi.CashFlows{}) || len(f.Services) != 1 || len(f.Fleet) != 2 || f.Services[0] != (gameapi.Service{CorridorID: "tokyo-nagoya", Frequency: 6, Fare: 9000}) {
		t.Fatal("restart did not restore starting company", f)
	}
	if f.Revision != before.Revision+1 {
		t.Fatal("revision reused across restart")
	}
	if _, err := g.Apply(reset); err == nil || err.Error() != "stale_revision" {
		t.Fatal("duplicate restart accepted", err)
	}
	if _, err := g.Apply(gameapi.Command{Kind: gameapi.AdvanceMonth, Revision: 0}); err == nil || err.Error() != "stale_revision" {
		t.Fatal("old campaign command accepted", err)
	}
	if g.Snapshot().Checkpoint != f.Checkpoint {
		t.Fatal("stale command changed fresh game")
	}
	reloaded := openTest(t, s)
	if !reflect.DeepEqual(reloaded.Snapshot(), f) {
		t.Fatal("new game did not survive reload")
	}
	firstMonth := act(t, reloaded, gameapi.Command{Kind: gameapi.AdvanceMonth})
	if firstMonth.Reports[0].Revenue != 714_132_000 || firstMonth.Player.Cash != 10_398_706_200 {
		t.Fatal("starter business failed after reset")
	}
	// Compare the whole persisted state, including future fleet identity.
	control := openTest(t, nil)
	act(t, control, gameapi.Command{Kind: gameapi.AdvanceMonth})
	want, got := data(control.w), data(reloaded.w)
	want.Revision = got.Revision
	if !reflect.DeepEqual(got, want) {
		t.Fatal("old campaign state survived restart")
	}
}

func TestNewGameAfterCampaignEnd(t *testing.T) {
	g := openTest(t, nil)
	for i := 0; i < 30; i++ {
		act(t, g, gameapi.Command{Kind: gameapi.AdvanceYear})
	}
	if g.Snapshot().MonthsResolved != 360 {
		t.Fatal("fixture did not finish campaign")
	}
	f := act(t, g, gameapi.Command{Kind: gameapi.NewGame})
	if f.MonthsResolved != 0 {
		t.Fatal("completed campaign could not restart")
	}
	act(t, g, gameapi.Command{Kind: gameapi.AdvanceMonth})
}

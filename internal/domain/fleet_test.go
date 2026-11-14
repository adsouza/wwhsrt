package domain

import (
	"testing"
	"wwhsrt/pkg/gameapi"
)

func TestLeaseNeedsNonnegativeCash(t *testing.T) {
	w := networkWorld(t)
	w.company.Cash = -1
	_, err := w.Change(gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: "tokyo-nagoya", Revision: w.revision})
	if err == nil || err.Error() != "insufficient_cash" {
		t.Fatal("wanted insufficient_cash, got", err)
	}
}

func TestExtensionLeaseNeedsNonnegativeCash(t *testing.T) {
	w := networkWorld(t)
	w.company.Cash = -1
	_, err := w.Change(
		gameapi.Command{
			Kind:             gameapi.ExtendService,
			CorridorID:       "tokyo-osaka",
			ReplaceServiceID: "tokyo-nagoya",
			Frequency:        6,
			Fare:             13800,
			LeaseCount:       1,
			Revision:         w.revision,
		},
	)
	if err == nil || err.Error() != "insufficient_cash" {
		t.Fatal("wanted insufficient_cash, got", err)
	}
}

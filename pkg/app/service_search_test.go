package app

import (
	"context"
	"testing"

	"wwhsrt/internal/application"
	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

type countingServicePort struct {
	gameapi.Game
	forecasts int
}

func (p *countingServicePort) CompareServices() gameapi.ServiceComparison {
	batch := p.Game.CompareServices()
	forecast := batch.Forecast
	batch.Forecast = func(q gameapi.ForecastRequest) (gameapi.Forecast, error) { p.forecasts++; return forecast(q) }
	return batch
}

func TestServiceSearchReusesRankingAfterStartingService(t *testing.T) {
	port, err := application.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	counter := &countingServicePort{Game: port}
	g := &Game{Port: counter, Frame: port.Snapshot()}
	search := func() (gameapi.ForecastRequest, int) {
		t.Helper()
		counter.forecasts = 0
		job := g.searchServices()
		<-job.done
		return job.plan, counter.forecasts
	}
	apply := func(c gameapi.Command) {
		t.Helper()
		c.Revision = g.Frame.Revision
		f, err := port.Apply(c)
		if err != nil {
			t.Fatal(err)
		}
		g.discardServiceSearch()
		g.Frame = f
	}
	first, coldCalls := search()
	apply(gameapi.Command{Kind: gameapi.RenameCompany, Name: "New Name"})
	if got, calls := search(); got != first || calls != 0 {
		t.Fatalf("renaming changed the ranking or ran %d forecasts: %+v", calls, got)
	}
	preview, err := port.Forecast(first)
	if err != nil {
		t.Fatal(err)
	}
	for range preview.FleetNeeded {
		apply(gameapi.Command{Kind: gameapi.LeaseTrain, CorridorID: first.CorridorID})
	}
	apply(gameapi.Command{Kind: gameapi.SetService, CorridorID: first.CorridorID, Frequency: first.Frequency, Fare: first.Fare})
	next, warmCalls := search()
	if next.CorridorID == first.CorridorID || warmCalls == 0 || warmCalls >= coldCalls/4 {
		t.Fatalf("expected a new service with localized repricing: %s, %d vs %d forecasts", next.CorridorID, warmCalls, coldCalls)
	}
	want := ui.RecommendService(context.Background(), g.Frame, port.ServiceForecasts())
	if next != want {
		t.Fatalf("cached ranking gave %+v; exhaustive comparison gave %+v", next, want)
	}
	t.Logf("first %s: %d forecasts; after starting it, %s: %d forecasts", first.CorridorID, coldCalls, next.CorridorID, warmCalls)
}

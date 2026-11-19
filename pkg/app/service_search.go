package app

import (
	"context"
	"runtime"
	"time"

	"wwhsrt/pkg/gameapi"
	"wwhsrt/pkg/ui"
)

type serviceSearch struct {
	checkpoint string
	done       chan struct{}
	cancel     context.CancelFunc
	plan       gameapi.ForecastRequest
	ranking    ui.ServiceRanking
}

func (g *Game) canSuggestService() bool {
	if job := g.serviceSearch; job != nil && job.checkpoint == g.Frame.Checkpoint {
		select {
		case <-job.done:
			return job.plan.CorridorID != ""
		default:
		}
	}
	return ui.CanSuggestService(g.Frame)
}

// Called under the presentation lock. A completed plan can be reused until
// the company changes. The next worker inherits the last completed ranking.
func (g *Game) searchServices() *serviceSearch {
	if job := g.serviceSearch; job != nil && job.checkpoint == g.Frame.Checkpoint {
		return job
	}
	g.discardServiceSearch()
	ctx, cancel := context.WithCancel(context.Background())
	f := g.Frame
	job := &serviceSearch{checkpoint: f.Checkpoint, done: make(chan struct{}), cancel: cancel}
	g.serviceSearch = job
	previous := g.serviceRanking
	batch := g.Port.CompareServices()
	go func() {
		defer close(job.done)
		if runtime.GOOS == "js" {
			// Let browser input and rendering run between slices of WASM work.
			last := time.Now()
			forecast := batch.Forecast
			batch.Forecast = func(q gameapi.ForecastRequest) (gameapi.Forecast, error) {
				if time.Since(last) >= 12*time.Millisecond {
					time.Sleep(time.Millisecond)
					last = time.Now()
				}
				return forecast(q)
			}
		}
		job.plan, job.ranking = previous.Recommend(ctx, f, batch)
	}()
	return job
}

func (g *Game) discardServiceSearch() {
	if g.serviceSearch != nil {
		select {
		case <-g.serviceSearch.done:
			g.serviceRanking = g.serviceSearch.ranking
		default:
		}
		g.serviceSearch.cancel()
		g.serviceSearch = nil
	}
}

package datasource

import (
	"sort"
	"time"
	"github.com/nstandage/f1-go-cli-app/model"
)

type ReplayEngine struct {
	EventData *model.EventData
}

func (eng *ReplayEngine) Start(out chan<- *model.Event, sessionStart time.Time) {
	defer close(out)
	eng.sortEventData()

	for i, e := range eng.EventData.EventModels {
		skipForward := 26 * time.Minute + 30 * time.Second
		if e.GetDateStart().Before(sessionStart.Add(skipForward)) {
		// if e.GetDateStart().Before(sessionStart) {
			continue
		}
		if i > 0 { // prevents sleep for first event only. i-1 crashes if i = 0
			previousTime := laterOf(sessionStart, eng.EventData.EventModels[i-1].GetDateStart())
			duration := e.GetDateStart().Sub(previousTime)
			time.Sleep(duration)
			// time.Sleep(100 * time.Millisecond)
		}
		out <- &model.Event{Model: e}
	}
}

func laterOf(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func (eng *ReplayEngine) sortEventData() {
	sort.Slice(eng.EventData.EventModels, func(i, j int) bool {
		return eng.EventData.EventModels[i].GetDateStart().Before(eng.EventData.EventModels[j].GetDateStart())
	})
}

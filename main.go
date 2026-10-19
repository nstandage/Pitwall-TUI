package main

import (
	"context"
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/nstandage/f1-go-cli-app/aggregator"
	"github.com/nstandage/f1-go-cli-app/datasource"
	"github.com/nstandage/f1-go-cli-app/service"
	"github.com/nstandage/f1-go-cli-app/tui"
)

func main() {
	var sessionKey = "9939"
	var meetingKey = "1265"
	service := service.NewOpenF1HTTP()

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	hs := datasource.NewHistoricalSource(service)

	err = hs.Fetch(context.Background(), sessionKey, meetingKey)
	if err != nil {
		log.Fatal(err)
	}

	receiver, snapShotter := aggregator.NewAggregator(hs)

	p := tea.NewProgram(tui.Model{
		Window: tui.Window{},
		Snapshotter: snapShotter,
	})

	go receiver.Start()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

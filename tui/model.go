package tui

import (
	"log"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/nstandage/f1-go-cli-app/aggregator"
	"github.com/nstandage/f1-go-cli-app/tui/view"
)

type Model struct {
	Window   Window
	Snapshotter   *aggregator.Snapshotter
	offset   uint
	isPaused bool
}

type Window struct {
	width  int
	height int
}

type TickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Window.width = msg.Width
		m.Window.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			log.Println("-----------------------")
			return m, tea.Quit
		}
	case TickMsg:
		return m, tick()
	default:
		// log.Printf("Unable to read msg of type: %T", msg)
	}
	return m, nil
}

func (m Model) View() tea.View {
	snapshot := m.Snapshotter.GetSnapshot(0)
	sessionBar := view.SessionBar(snapshot.SessionBar)
	legendBar := view.LegendBar()
	positionColumn := view.PositionsColumn()
	topBar := view.Topbar(snapshot.SectorCounts[:])

	drivers := snapshot.Drivers

	intervals := snapshot.Intervals

	gapToLeaders := snapshot.GapsToLeaders

	lastLap := snapshot.LastLap

	pits := snapshot.PitCounts

	tires := snapshot.TireCompounds

	tireAge := snapshot.TireAges

	raceControlMessages := snapshot.RaceControlMsgs

	driverColumn := view.DriverColumn(drivers)
	intervalColumn := view.DefaultColumn(intervals)
	gapToLeaderColumn := view.DefaultColumn(gapToLeaders)
	lastLapColumn := view.LastLapColumn(lastLap, snapshot.LastLapIsPitOut)
	pitColumn := view.PitColumn(pits)
	tiresColumn := view.TireColumn(tires)
	tireAgeColumn := view.TireAgeColumn(tireAge)
	laps := view.Laps(snapshot.Sectors)
	raceControl := view.RaceControl(raceControlMessages)
	pitStopView := view.PitStops(snapshot.RecentPitStops)

	driverView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		positionColumn,
		driverColumn,
		intervalColumn,
		gapToLeaderColumn,
		lastLapColumn,
		pitColumn,
		tiresColumn,
		tireAgeColumn,
		laps,
	)

	topView := lipgloss.JoinVertical(
		lipgloss.Top,
		sessionBar,
		topBar,
	)

	componentHeight := lipgloss.Height(driverView) +
		lipgloss.Height(topView) +
		lipgloss.Height(legendBar)

	var spacerSize uint = uint(m.Window.height - componentHeight)

	combined := lipgloss.JoinVertical(
		lipgloss.Top,
		topView,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(
				lipgloss.Center,
				driverView,
				view.Spacer(spacerSize),
				legendBar,
			),
			lipgloss.JoinVertical(
				lipgloss.Top,
				raceControl,
				pitStopView,
			),
			// legendBar,
		),
	)
	v := tea.NewView(combined)
	v.AltScreen = true
	return v
}

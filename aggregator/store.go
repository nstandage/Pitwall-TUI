package aggregator

import (
	"context"
	"sync"
	"time"

	"github.com/nstandage/f1-go-cli-app/model"
)

type Store struct {
	history       []model.Snapshot
	Drivers       map[uint]*Driver // mapped to DriverNumber
	RaceControl   []model.RaceControl
	TotalLaps     uint
	CurrentLap    uint
	IsReplay      bool
	Session       *model.Session
	Meeting       *model.Meeting
	StartingGrid  []model.StartingGrid
	FastestLap    *FastestLap
	Stints        map[uint][]model.Stint // mapped to DriverNumber
	RecentPits    []model.PitStopEntry
	SectorCounts  [3]int
	RaceStartTime *time.Time
}

type Driver struct {
	Number           uint
	Info             *model.Driver
	Position         uint
	StartingPosition uint
	IsOut            bool
	Interval         string
	ToLeader         string
	LastLap          float64
	LastLapIsPitOut  bool
	LapsOnTire       uint
	PitCount         uint
	CurrentLap       uint
	TireCompound	string
	mu               sync.RWMutex
	Sectors          [3][]uint
	cancelSectors    context.CancelFunc
}

type FastestLap struct {
	LapTime      float64
	DriverNumber uint
	LapNumber    uint
}


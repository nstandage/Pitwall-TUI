package aggregator

// MARK: Snapshot Functions
import (
	"github.com/nstandage/f1-go-cli-app/model"
	"time"
	"strconv"
	"math"
	"log"
	"fmt"
)

type Snapshotter struct {
	store *Store
}

func (s *Snapshotter) GetSnapshot(offset uint) *model.Snapshot {
	var raceElapsed time.Duration
	if s.store.RaceStartTime != nil {
		raceElapsed = time.Since(*s.store.RaceStartTime)
	}
	sessionBar := model.SessionBarSnapShot{
		EventName:   s.store.Meeting.MeetingOfficialName,
		EventType:   s.store.Session.SessionType,
		CurrentLap:  s.store.CurrentLap,
		FastestLap:  s.getFastestLap(),
		TotalLaps:   s.store.TotalLaps,
		IsReplay:    s.store.IsReplay,
		EventDate:   s.store.Session.DateStart,
		RaceElapsed: raceElapsed,
	}
	snapshot := model.Snapshot{
		SessionBar:      &sessionBar,
		RaceControlMsgs: s.getRaceControlMessages(),
		Drivers:         s.getDriverNames(),
		LastLap:         s.getLastLaps(),
		LastLapIsPitOut: s.getOutLaps(),
		Intervals:       s.getIntervals(),
		GapsToLeaders:   s.getGapToLeader(),
		PitCounts:       s.getPitCounts(),
		TireCompounds:   s.getTireCompounds(),
		TireAges:        s.getTireAges(),
		RecentPitStops:  s.store.RecentPits,
		Sectors:         s.getSectors(),
		SectorCounts:    s.store.SectorCounts,
	}

	s.updateHistory(&snapshot)

	return &snapshot
}

// MARK: Get Functions

func (s *Snapshotter) getFastestLap() *model.FastestLapSnapshot {
	if s.store.FastestLap == nil {
		return &model.FastestLapSnapshot{
			LapTime: formatLapTime(0.0),
			Driver: "---",
			LapNumber: "---",
		}
	}
	fastestLap := s.store.FastestLap

	driverNumber := fastestLap.DriverNumber
	driverName := s.store.Drivers[driverNumber].Info.NameAcronym
	lapNumber := fastestLap.LapNumber
	return &model.FastestLapSnapshot{
		LapTime:   formatLapTime(fastestLap.LapTime),
		Driver:    driverName,
		LapNumber: strconv.FormatUint(uint64(lapNumber), 10),
	}
}

func (s *Snapshotter) getRaceControlMessages() []string {
	strs := []string{}
	for _, rc := range s.store.RaceControl {
		strs = append(strs, rc.Message)
	}

	return strs
}

func (s *Snapshotter) getDriverNames() []model.DriverSnapshot {
	drivers := make([]model.DriverSnapshot, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		name := d.Info.NameAcronym
		isFastest := false
		if s.store.FastestLap != nil {
			isFastest = s.store.FastestLap.DriverNumber == d.Number
		}
		drivers[d.Position-1] = model.DriverSnapshot{Name: name, IsFastestLap: isFastest}
	}

	return drivers
}

func (s *Snapshotter) getLastLaps() []string {
	strs := make([]string, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		strs[d.Position-1] = formatLapTime(d.LastLap)
	}
	return strs
}

func (s *Snapshotter) getOutLaps() []bool {
	isPitOut := make([]bool, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		isPitOut[d.Position-1] = d.LastLapIsPitOut
	}
	return isPitOut
}

func (s *Snapshotter) getIntervals() []string {
	strs := make([]string, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		strs[d.Position-1] = getTimingText(d.Interval)
	}
	return strs
}

func (s *Snapshotter) getGapToLeader() []string {
	strs := make([]string, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		strs[d.Position-1] = getTimingText(d.ToLeader)
	}
	return strs
}

func (s *Snapshotter) getPitCounts() []string {
	counts := make([]string, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		counts[d.Position-1] = strconv.Itoa(int(d.PitCount))
	}
	return counts
}

func (s *Snapshotter) getTireCompounds() []string {
	compounds := make([]string, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		compounds[d.Position-1] = normalizeCompound(d.TireCompound)
	}
	return compounds
}

func (s *Snapshotter) getTireAges() []string {
	ages := make([]string, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		ages[d.Position-1] = strconv.FormatUint(uint64(d.LapsOnTire), 10)
		log.Printf("Laps On Tire: %v", d.LapsOnTire)
	}
	return ages
}

func (s *Snapshotter) getSectors() [][][]uint { // driver, sector, minisector
	result := make([][][]uint, len(s.store.Drivers))
	for _, d := range s.store.Drivers {
		if d.Position == 0 {
			continue
		}
		sectors := make([][]uint, 3)
		d.mu.RLock()
		for i := range len(sectors)-1 {
			seg := make([]uint, len(d.Sectors[i]))
			copy(seg, d.Sectors[i])
			sectors[i] = seg
		}
		d.mu.RUnlock()
		result[d.Position-1] = sectors
	}
	return result
}

// MARK: Helper Functions

func formatLapTime(seconds float64) string {
	if seconds <= 0 {
		return "--:--.---"
	}

	totalMs := int(math.Round(seconds * 1000))
	totalSecs := totalMs / 1000
	ms := totalMs % 1000
	secs := totalSecs % 60
	mins := totalSecs / 60

	return fmt.Sprintf("%d:%02d.%03d", mins, secs, ms)
}

func (s *Snapshotter) updateHistory(h *model.Snapshot) {
	s.store.history = appendCapped(s.store.history, *h, historyMaxLength)
}

func getTimingText(i string) string {
	if i == "" {
		return "----"
	}
	return i
}

func normalizeCompound(compound string) string {
	switch compound {
		case "MEDIUM": 
		return "MED"
	case "INTERMEDIATE":
		return "INT"
	case "":
		return "---"
	default:
		return compound
	}
}

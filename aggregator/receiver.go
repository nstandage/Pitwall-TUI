package aggregator

import (
	"context"
	"log"
	"time"
	"github.com/nstandage/f1-go-cli-app/datasource"
	"github.com/nstandage/f1-go-cli-app/model"
)

func NewAggregator(ds datasource.DataSource) (*Receiver, *Snapshotter) {
	store := Store{}
	r := Receiver{
		store: &store,
		datasource: ds,
	}
	s := Snapshotter{
		store: &store,
	}

	return &r, &s
}

var (
	historyMaxLength     = 180
	raceControlMaxLength = 6
	recentPitsMaxLength  = 8
)

type Receiver struct {
	store      *Store
	datasource datasource.DataSource
}

// func NewReceiver(ds datasource.DataSource) *Receiver {
// 	return &Receiver{
// 		store:      &Store{},
// 		datasource: ds,
// 	}
// }

func (r *Receiver) Start() {
	raceData, c := r.datasource.Start()
	r.setUpInitialStore(raceData)
	r.listen(c)
}

func (r *Receiver) setUpInitialStore(rd *model.RaceData) {
	r.store.Meeting = rd.Meeting
	r.store.Session = rd.Session
	r.store.TotalLaps = rd.TotalLaps
	r.store.StartingGrid = rd.StartingGrid
	r.store.IsReplay = r.datasource.IsReplay()
	r.store.Drivers = convertDrivers(rd.Drivers)
	r.store.Stints = make(map[uint][]model.Stint)
	for _, s := range rd.Stints {
		r.store.Stints[s.DriverNumber] = append(r.store.Stints[s.DriverNumber], s)
	}

	for _, sg := range r.store.StartingGrid {
		driver, ok := r.store.Drivers[sg.DriverNumber]
		if ok {
			driver.StartingPosition = sg.Position
			driver.Position = sg.Position
			driver.LastLap = sg.LapDuration
			driver.LapsOnTire = 0
		}
	}
}

func (r *Receiver) listen(c <-chan *model.Event) {
	for event := range c {
		if r.store.RaceStartTime == nil {
			now := time.Now()
			r.store.RaceStartTime = &now
		}
		r.handle(event)
	}
}

func (r *Receiver) handle(event *model.Event) {
	switch m := event.Model.(type) {
	case *model.Interval:
		r.updateInterval(m)
	case *model.Lap:
		log.Printf("Lap: %v", m.LapNumber)
		r.updateGlobalLapNumber(m)
		r.setGlobalLapSectors(m)
		r.updateDriverLapNumber(m)
		r.updateLap(m)
	case *model.Position:
		r.updatePosition(m)
	case *model.RaceControl:
		r.updateRaceControl(m)
	case *model.Pit:
		r.updatePit(m)
		r.setTiresAtPit(m)
		r.addPitEntry(m)
	}
}


// MARK: update methods

func (r *Receiver) updateInterval(i *model.Interval) {
	driver := r.store.Drivers[i.DriverNumber]
	driver.Interval = string(i.Interval)
	driver.ToLeader = string(i.GapToLeader)
}

func (r *Receiver) updateLap(l *model.Lap) {
	if l.LapDuration <= 0 {
		return
	}

	if driver, ok := r.store.Drivers[l.DriverNumber]; ok {
		if driver.cancelSectors != nil {
			driver.cancelSectors()
		}
		driver.mu.Lock()
		driver.Sectors = makeFutureSectors(r.store.SectorCounts)
		driver.mu.Unlock()
		ctx, cancel := context.WithCancel(context.Background())
		driver.cancelSectors = cancel
		go r.animateSectors(ctx, driver, l)
	}

	go r.sleepForLapDuration(l)
}

func (r *Receiver) updateTireAge(l *model.Lap) {
	if l.IsPitOutLap {
		return
	}
	if driver, ok := r.store.Drivers[l.DriverNumber]; ok {
		driver.LapsOnTire++
	}
}

func (r *Receiver) updateGlobalLapNumber(l *model.Lap) {
	if l.LapNumber > r.store.CurrentLap {
		r.store.CurrentLap = l.LapNumber
	}
}

func (r *Receiver) updateDriverLapNumber(l *model.Lap) {
	if driver, ok := r.store.Drivers[l.DriverNumber]; ok {
		if l.LapNumber > driver.CurrentLap {
			driver.CurrentLap = l.LapNumber
		}
	}
}

func (r *Receiver) setGlobalLapSectors(l *model.Lap) {
	if r.store.SectorCounts[0] == 0 && len(l.SegmentsSector1) > 0 {
		r.store.SectorCounts = [3]int{
			len(l.SegmentsSector1),
			len(l.SegmentsSector2),
			len(l.SegmentsSector3),
		}
	}
}

func (r *Receiver) updatePosition(p *model.Position) {
	if driver, ok := r.store.Drivers[p.DriverNumber]; ok {
		driver.Position = p.Position
	}
}

func (r *Receiver) updatePit(p *model.Pit) {
	driver, ok := r.store.Drivers[p.DriverNumber]
	if !ok {
		return
	}
	driver.PitCount++
}
func (r *Receiver) setTiresAtPit(p *model.Pit) {
	stint := r.getActiveStint(p)
	driver, ok := r.store.Drivers[p.DriverNumber]
	if !ok {
		return
	}

	driver.LapsOnTire = stint.TyreAgeAtStart
	driver.TireCompound = stint.Compound
}

func (r *Receiver) getActiveStint(p *model.Pit) *model.Stint {
	var active *model.Stint
	stints := r.store.Stints[p.DriverNumber]
	for i := range len(stints) {
		if stints[i].LapStart <= p.LapNumber { //Stint.LapStart is one higher than the current driver lap
			if active == nil || stints[i].LapStart > active.LapStart {
				active = &stints[i]
			}
		}
	}
	return active
}

func (r *Receiver) addPitEntry(p *model.Pit) {
	if driver, ok := r.store.Drivers[p.DriverNumber]; ok {
		entry := model.PitStopEntry{
			DriverAcronym: driver.Info.NameAcronym,
			StopDuration:  p.StopDuration,
		}
		r.store.RecentPits = appendCapped(r.store.RecentPits, entry, recentPitsMaxLength)
	}
}

func (r *Receiver) updateRaceControl(rc *model.RaceControl) {
	r.store.RaceControl = appendCapped(r.store.RaceControl, *rc, raceControlMaxLength)
}

// MARK: Helper Functions
func makeFutureSectors(counts [3]int) [3][]uint {
	var sectors [3][]uint
	for i, count := range counts {
		sectors[i] = make([]uint, count)
	}
	return sectors
}

func (r *Receiver) animateSectors(ctx context.Context, driver *Driver, data *model.Lap) {
	sectors := [3][]uint{data.SegmentsSector1, data.SegmentsSector2, data.SegmentsSector3}
	SectorDurations := [3]float64{data.DurationSector1, data.DurationSector2, data.DurationSector3}

	for i, sector := range sectors {
		if len(sector) == 0 {
			continue
		}
		delay := time.Duration(SectorDurations[i] / float64(len(sector)) * float64(time.Second))
		for idx, miniSec := range sector {
			time.Sleep(delay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}

			driver.mu.Lock()
			if ctx.Err() != nil {
				driver.mu.Unlock()
				return
			}

			if idx < len(driver.Sectors[i]) {
				driver.Sectors[i][idx] = miniSec
			}
			driver.mu.Unlock()
		}
	}
}

func (r *Receiver) sleepForLapDuration(l *model.Lap) {
	duration := time.Duration(l.LapDuration * float64(time.Second))
	time.Sleep(duration)

	driver, ok := r.store.Drivers[l.DriverNumber]
	if !ok {
		return
	}
	driver.mu.Lock()
	driver.LastLap = l.LapDuration
	driver.LastLapIsPitOut = l.IsPitOutLap
	r.updateTireAge(l)
	r.checkForFastestLap(l)
	driver.mu.Unlock()
}


func (r *Receiver) checkForFastestLap(l *model.Lap) {
	if r.store.FastestLap == nil || l.LapDuration < r.store.FastestLap.LapTime {
		r.store.FastestLap = &FastestLap{
			LapTime:      l.LapDuration,
			DriverNumber: l.DriverNumber,
			LapNumber:    l.LapNumber,
		}
	}
}

func appendCapped[T any](s []T, val T, max int) []T {
	s = append(s, val)
	if len(s) > max {
		extra := len(s) - max
		s = s[extra:]
	}
	return s
}

func convertDrivers(ds []model.Driver) map[uint]*Driver {
	drivers := make(map[uint]*Driver, len(ds))
	for _, d := range ds {
		driver := Driver{
			Number: d.DriverNumber,
			Info:   &d,
		}
		drivers[d.DriverNumber] = &driver
	}

	return drivers
}

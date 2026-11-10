package greenwave

import (
	"math"
	"sort"

	"github.com/LdDl/greenwave/greeninterval"
	"github.com/LdDl/greenwave/junction"
)

// GreenWave represents a green wave between two junctions.
type GreenWave struct {
	// Green interval on the first junction.
	intervalJunOne *greeninterval.GreenInterval
	// Green interval on the second junction.
	intervalJunTwo *greeninterval.GreenInterval
	// Distance in meters between the two junctions.
	distance float64
	// Travel time in seconds between the two junctions.
	travelTime float64
	// Bandwidth of the green wave in seconds.
	bandwidth float64
}

// NewGreenWave creates a new GreenWave instance with the specified parameters.
func NewGreenWave(intervalJunOne, intervalJunTwo *greeninterval.GreenInterval, distanceMeters, travelTimeSeconds float64) *GreenWave {
	return &GreenWave{
		intervalJunOne: greeninterval.New(intervalJunOne.PhaseIdx, intervalJunOne.Start, intervalJunOne.End),
		intervalJunTwo: greeninterval.New(intervalJunTwo.PhaseIdx, intervalJunTwo.Start, intervalJunTwo.End),
		distance:       distanceMeters,
		travelTime:     travelTimeSeconds,
		bandwidth:      float64(intervalJunOne.End - intervalJunOne.Start),
	}
}

// Clone creates a deep copy of the GreenWave instance.
func (gw *GreenWave) Clone() *GreenWave {
	return &GreenWave{
		intervalJunOne: greeninterval.New(gw.intervalJunOne.PhaseIdx, gw.intervalJunOne.Start, gw.intervalJunOne.End),
		intervalJunTwo: greeninterval.New(gw.intervalJunTwo.PhaseIdx, gw.intervalJunTwo.Start, gw.intervalJunTwo.End),
		distance:       gw.distance,
		travelTime:     gw.travelTime,
		bandwidth:      gw.bandwidth,
	}
}

// IntervalJunOne returns the green interval on the first junction.
func (gw *GreenWave) IntervalJunOne() *greeninterval.GreenInterval {
	return gw.intervalJunOne
}

// IntervalJunTwo returns the green interval on the second junction.
func (gw *GreenWave) IntervalJunTwo() *greeninterval.GreenInterval {
	return gw.intervalJunTwo
}

// Distance returns the distance between the two junctions in meters.
func (gw *GreenWave) Distance() float64 {
	return gw.distance
}

// TravelTime returns the travel time between the two junctions in seconds.
func (gw *GreenWave) TravelTime() float64 {
	return gw.travelTime
}

// Bandwidth returns the bandwidth of the green wave in seconds.
func (gw *GreenWave) Bandwidth() float64 {
	return gw.bandwidth
}

// FindGreenWavesBetweenIntervals finds green waves between two sets of green intervals.
func FindGreenWavesBetweenIntervals(greenIntervalsOne, greenIntervalsTwo []*greeninterval.GreenInterval, distanceMeters, travelTimeSeconds float64) []*GreenWave {
	var greenWaves []*GreenWave
	for _, greenIntervalOne := range greenIntervalsOne {
		startOne, endOne := float64(greenIntervalOne.Start), float64(greenIntervalOne.End)
		firstArrivalJunTwo := startOne + travelTimeSeconds
		lastArrivalJunTwo := endOne + travelTimeSeconds
		for _, greenIntervalTwo := range greenIntervalsTwo {
			startTwo, endTwo := float64(greenIntervalTwo.Start), float64(greenIntervalTwo.End)
			overlapStart := math.Max(firstArrivalJunTwo, startTwo)
			overlapEnd := math.Min(lastArrivalJunTwo, endTwo)
			if overlapStart >= overlapEnd {
				// No overlap, continue to the next interval
				continue
			}
			adjustedStartJunOne := overlapStart - travelTimeSeconds
			adjustedEndJunOne := overlapEnd - travelTimeSeconds
			// adjustedStartJunOne >= startOne - departure not before start of first interval
			// adjustedEndJunOne <= endOne - arrival not after end of first interval
			// adjustedStartJunOne < adjustedEndJunOne - ensure valid interval
			if adjustedStartJunOne >= startOne && adjustedEndJunOne <= endOne && adjustedStartJunOne < adjustedEndJunOne {
				greenWave := NewGreenWave(
					greeninterval.New(greenIntervalOne.PhaseIdx, adjustedStartJunOne, adjustedEndJunOne),
					greeninterval.New(greenIntervalTwo.PhaseIdx, overlapStart, overlapEnd),
					distanceMeters,
					travelTimeSeconds,
				)
				greenWaves = append(greenWaves, greenWave)
			}
		}
	}
	return greenWaves
}

// FindGreenWaves evaluates departures in [0, first junction cycle).
// At junction i, the evaluation window is shifted by cumulative travel time.
// Programs repeat periodically, while returned intervals use absolute seconds.
// With unequal cycles this is one departure window, not their common period.
// groupIDs selects the coordinated signal group at each junction.
func FindGreenWaves(junctions []*junction.Junction, groupIDs map[int]junction.GroupID, desiredSpeedKmh float64) [][]*GreenWave {
	if len(junctions) < 2 {
		return [][]*GreenWave{}
	}
	waves := make([][]*GreenWave, len(junctions)-1)
	if desiredSpeedKmh <= 0 || math.IsNaN(desiredSpeedKmh) || math.IsInf(desiredSpeedKmh, 0) {
		return waves
	}
	window := float64(junctions[0].GetTotalDuration())
	arrival := 0.0
	previous := periodicGreenIntervals(junctions[0], groupIDs[junctions[0].ID], arrival, window)
	for i := range waves {
		from, to := junctions[i].GetPoint(), junctions[i+1].GetPoint()
		distance := math.Hypot(to.X-from.X, to.Y-from.Y)
		travel := distance / (desiredSpeedKmh / 3.6)
		nextArrival := arrival + travel
		if math.IsInf(nextArrival, 0) || math.IsNaN(nextArrival) {
			return waves
		}
		next := periodicGreenIntervals(junctions[i+1], groupIDs[junctions[i+1].ID], nextArrival, window)
		// Both interval sets use departure time at the first junction as their origin.
		for _, overlap := range FindGreenWavesBetweenIntervals(previous, next, distance, 0) {
			one, two := overlap.IntervalJunOne(), overlap.IntervalJunTwo()
			waves[i] = append(waves[i], NewGreenWave(
				greeninterval.New(one.PhaseIdx, one.Start+arrival, one.End+arrival),
				greeninterval.New(two.PhaseIdx, two.Start+nextArrival, two.End+nextArrival),
				distance, travel,
			))
		}
		previous = next
		arrival = nextArrival
	}
	return waves
}

// periodicGreenIntervals projects a junction's green onto the departure window.
// Modulo skips elapsed cycles directly. Work depends on the window and program,
// not on how many cycles pass while vehicles travel to this junction.
func periodicGreenIntervals(jun *junction.Junction, groupID junction.GroupID, arrival, window float64) []*greeninterval.GreenInterval {
	cycle := float64(jun.GetTotalDuration())
	intervals := make([]*greeninterval.GreenInterval, 0)
	if cycle <= 0 || window <= 0 {
		return intervals
	}
	for _, green := range jun.GetGreenIntervals(groupID) {
		duration := green.End - green.Start
		if duration <= 0 {
			continue
		}
		if duration >= cycle {
			return []*greeninterval.GreenInterval{greeninterval.New(green.PhaseIdx, 0, window)}
		}
		start := math.Mod(green.Start+float64(jun.GetOffset())-math.Mod(arrival, cycle), cycle)
		if start < 0 {
			start += cycle
		}
		// The preceding repetition can overlap the beginning of the window.
		for position := start - cycle; position < window; position += cycle {
			left, right := math.Max(0, position), math.Min(window, position+duration)
			if right > left {
				intervals = append(intervals, greeninterval.New(green.PhaseIdx, left, right))
			}
		}
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i].Start < intervals[j].Start })
	return intervals
}

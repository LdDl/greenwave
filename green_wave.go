package greenwave

import (
	"math"

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

// FindGreenWaves finds green waves between a sequence of junctions based on their green intervals and desired speed.
// groupIDs maps each junction ID to the signal group to use for green wave coordination in this corridor.
// A junction may have multiple signal groups (e.g. northbound, eastbound, pedestrian); only one group represents
// the through-movement for a given corridor. The caller is responsible for providing the correct group per junction.
// It returns a slice of slices, where each inner slice contains green waves for the segment between two junctions.
func FindGreenWaves(junctions []*junction.Junction, groupIDs map[int]junction.GroupID, desiredSpeedKmh float64) [][]*GreenWave {
	speedMs := desiredSpeedKmh / 3.6
	waves := make([][]*GreenWave, 0, len(junctions)-1)
	for i := 0; i < len(junctions)-1; i++ {
		junctionOne := junctions[i]
		junctionTwo := junctions[i+1]
		greenIntervalsOne := junctionOne.GetGreenIntervals(groupIDs[junctionOne.ID])
		greenIntervalsTwo := junctionTwo.GetGreenIntervals(groupIDs[junctionTwo.ID])

		offsetJunctionOne := junctionOne.GetOffset()
		offsetJunctionTwo := junctionTwo.GetOffset()

		totalDurationOne := junctionOne.GetTotalDuration()
		totalDurationTwo := junctionTwo.GetTotalDuration()

		adjustedIntervalsOne := make([]*greeninterval.GreenInterval, 0, len(greenIntervalsOne))
		for _, interval := range greenIntervalsOne {
			start := (int(interval.Start) + offsetJunctionOne) % totalDurationOne
			end := (int(interval.End) + offsetJunctionOne) % totalDurationOne
			if end < start {
				// Interval split due cycle wrap
				adjustedIntervalsOne = append(adjustedIntervalsOne, greeninterval.New(interval.PhaseIdx, float64(start), float64(totalDurationOne)))
				adjustedIntervalsOne = append(adjustedIntervalsOne, greeninterval.New(interval.PhaseIdx, 0, float64(end)))
			} else {
				// Common case
				adjustedIntervalsOne = append(adjustedIntervalsOne, greeninterval.New(interval.PhaseIdx, float64(start), float64(end)))
			}
		}

		adjustedIntervalsTwo := make([]*greeninterval.GreenInterval, 0, len(greenIntervalsTwo))
		for _, interval := range greenIntervalsTwo {
			start := (int(interval.Start) + offsetJunctionTwo) % totalDurationTwo
			end := (int(interval.End) + offsetJunctionTwo) % totalDurationTwo
			if end < start {
				// Interval split due cycle wrap
				adjustedIntervalsTwo = append(adjustedIntervalsTwo, greeninterval.New(interval.PhaseIdx, float64(start), float64(totalDurationTwo)))
				adjustedIntervalsTwo = append(adjustedIntervalsTwo, greeninterval.New(interval.PhaseIdx, 0, float64(end)))
			} else {
				// Common case
				adjustedIntervalsTwo = append(adjustedIntervalsTwo, greeninterval.New(interval.PhaseIdx, float64(start), float64(end)))
			}
		}

		pointOne := junctionOne.GetPoint()
		pointTwo := junctionTwo.GetPoint()
		distanceMeters := math.Sqrt(math.Pow(pointOne.X-pointTwo.X, 2) + math.Pow(pointOne.Y-pointTwo.Y, 2))
		travelTimeSeconds := distanceMeters / speedMs

		segmentWaves := FindGreenWavesBetweenIntervals(adjustedIntervalsOne, adjustedIntervalsTwo, distanceMeters, travelTimeSeconds)
		waves = append(waves, segmentWaves)
	}
	return waves
}

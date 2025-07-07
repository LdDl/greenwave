package greenwave

import "math"

// GreenWave represents a green wave between two junctions.
type GreenWave struct {
	// Green interval on the first junction.
	intervalJunOne *GreenInterval
	// Green interval on the second junction.
	intervalJunTwo *GreenInterval
	// Distance in meters between the two junctions.
	distance float64
	// Travel time in seconds between the two junctions.
	travelTime float64
	// Bandwidth of the green wave in seconds.
	bandWidth float64
}

// NewGreenWave creates a new GreenWave instance with the specified parameters.
func NewGreenWave(intervalJunOne, intervalJunTwo *GreenInterval, distanceMeters, travelTimeSeconds float64) *GreenWave {
	return &GreenWave{
		intervalJunOne: NewGreenInterval(intervalJunOne.PhaseIdx, intervalJunOne.Start, intervalJunOne.End),
		intervalJunTwo: NewGreenInterval(intervalJunTwo.PhaseIdx, intervalJunTwo.Start, intervalJunTwo.End),
		distance:       distanceMeters,
		travelTime:     travelTimeSeconds,
		bandWidth:      float64(intervalJunOne.End - intervalJunOne.Start),
	}
}

func FindGreenWavesBetweenIntervals(greenIntervalsOne, greenIntervalsTwo []*GreenInterval, distanceMeters, travelTimeSeconds float64) []*GreenWave {
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
					NewGreenInterval(greenIntervalOne.PhaseIdx, adjustedStartJunOne, adjustedEndJunOne),
					NewGreenInterval(greenIntervalTwo.PhaseIdx, overlapStart, overlapEnd),
					distanceMeters,
					travelTimeSeconds,
				)
				greenWaves = append(greenWaves, greenWave)
			}
		}
	}
	return greenWaves
}

func FindGreenWaves(junctions []*Junction, desiredSpeedKmh float64) [][]*GreenWave {
	speedMs := desiredSpeedKmh / 3.6
	waves := make([][]*GreenWave, 0, len(junctions)-1)
	for i := 0; i < len(junctions)-1; i++ {
		junctionOne := junctions[i]
		junctionTwo := junctions[i+1]
		greenIntervalsOne := junctionOne.GetGreenIntervals()
		greenIntervalsTwo := junctionTwo.GetGreenIntervals()

		offsetJunctionOne := junctionOne.GetOffset()
		offsetJunctionTwo := junctionTwo.GetOffset()

		adjustedIntervalsOne := make([]*GreenInterval, 0, len(greenIntervalsOne))
		for _, interval := range greenIntervalsOne {
			start := (int(interval.Start) + offsetJunctionOne) % junctionOne.totalDuration
			end := (int(interval.End) + offsetJunctionOne) % junctionOne.totalDuration
			if end < start {
				// Interval split due cycle wrap
				adjustedIntervalsOne = append(adjustedIntervalsOne, NewGreenInterval(interval.PhaseIdx, float64(start), float64(junctionOne.totalDuration)))
				adjustedIntervalsOne = append(adjustedIntervalsOne, NewGreenInterval(interval.PhaseIdx, 0, float64(end)))
			} else {
				// Common case
				adjustedIntervalsOne = append(adjustedIntervalsOne, NewGreenInterval(interval.PhaseIdx, float64(start), float64(end)))
			}
		}

		adjustedIntervalsTwo := make([]*GreenInterval, 0, len(greenIntervalsTwo))
		for _, interval := range greenIntervalsTwo {
			start := (int(interval.Start) + offsetJunctionTwo) % junctionTwo.totalDuration
			end := (int(interval.End) + offsetJunctionTwo) % junctionTwo.totalDuration
			if end < start {
				// Interval split due cycle wrap
				adjustedIntervalsTwo = append(adjustedIntervalsTwo, NewGreenInterval(interval.PhaseIdx, float64(start), float64(junctionTwo.totalDuration)))
				adjustedIntervalsTwo = append(adjustedIntervalsTwo, NewGreenInterval(interval.PhaseIdx, 0, float64(end)))
			} else {
				// Common case
				adjustedIntervalsTwo = append(adjustedIntervalsTwo, NewGreenInterval(interval.PhaseIdx, float64(start), float64(end)))
			}
		}

		distanceMeters := math.Sqrt(math.Pow(junctionOne.point.X-junctionTwo.point.X, 2) + math.Pow(junctionOne.point.Y-junctionTwo.point.Y, 2))
		travelTimeSeconds := distanceMeters / speedMs

		segmentWaves := FindGreenWavesBetweenIntervals(adjustedIntervalsOne, adjustedIntervalsTwo, distanceMeters, travelTimeSeconds)
		waves = append(waves, segmentWaves)
	}
	return waves
}

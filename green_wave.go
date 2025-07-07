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

package greenwave

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

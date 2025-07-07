package greenwave

// GreenInterval represents a time interval during which a traffic light is green (for specific phase).
type GreenInterval struct {
	// Phase index in which the green interval occurs
	PhaseIdx int
	// Start time of the green interval in seconds
	Start int
	// End time of the green interval in seconds
	End int
}

// NewGreenInterval creates a new GreenInterval instance with the specified phase index, start time, and end time.
func NewGreenInterval(phaseIdx, start, end int) *GreenInterval {
	return &GreenInterval{
		PhaseIdx: phaseIdx,
		Start:    start,
		End:      end,
	}
}

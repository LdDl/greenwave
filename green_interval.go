package greenwave

import "math"

const (
	eps = 0.01
)

// GreenInterval represents a time interval during which a traffic light is green (for specific phase).
type GreenInterval struct {
	// Phase index in which the green interval occurs
	PhaseIdx int
	// Start time of the green interval in seconds
	Start float64
	// End time of the green interval in seconds
	End float64
}

// NewGreenInterval creates a new GreenInterval instance with the specified phase index, start time, and end time.
func NewGreenInterval(phaseIdx int, start, end float64) *GreenInterval {
	return &GreenInterval{
		PhaseIdx: phaseIdx,
		Start:    start,
		End:      end,
	}
}

// CanConnect checks if two green intervals can be connected based on their start and end times and returns a new GreenInterval as overlap if they can be connected.
func (interval *GreenInterval) CanConnect(otherInterval *GreenInterval) *GreenInterval {
	overlapStart := math.Max(interval.Start, otherInterval.Start)
	overlapEnd := math.Min(interval.End, otherInterval.End)
	if overlapEnd-overlapStart > eps {
		return NewGreenInterval(interval.PhaseIdx, overlapStart, overlapEnd)
	}
	return nil
}

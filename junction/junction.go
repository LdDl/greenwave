package junction

import (
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/geom"
	"github.com/LdDl/greenwave/greeninterval"
)

// Junction represents a traffic light junction
type Junction struct {
	// Traffic light identifier
	ID int
	// User defined alias
	Label string
	// Cycle is a list of phases that define the traffic light cycle for this junction
	Cycle []*Phase
	// Total duration of the cycle in seconds
	totalDuration int
	// Offset of the cycle
	offset int
	// Location of the junction
	point geom.Point
}

// NewJunction creates a new Junction instance with the specified ID, label, cycle (list of phases)
func NewJunction(cycle []*Phase, options ...func(*Junction)) *Junction {
	totalDuration := 0
	for _, phase := range cycle {
		totalDuration += phase.GetTotalSeconds()
	}
	junction := &Junction{
		ID:            -1,
		Label:         "-1",
		Cycle:         cycle,
		totalDuration: totalDuration,
		offset:        0,
	}
	for _, option := range options {
		option(junction)
	}
	return junction
}

// GetOffset returns the offset for the junction.
func (jun *Junction) GetOffset() int {
	return jun.offset
}

// SetOffset sets the offset for the junction.
// Offset is normalized to [0, totalDuration) range (ring buffer semantics).
func (jun *Junction) SetOffset(offset int) {
	if jun.totalDuration <= 0 {
		jun.offset = 0
		return
	}
	jun.offset = ((offset % jun.totalDuration) + jun.totalDuration) % jun.totalDuration
}

// WithID is an option function that sets the ID for the junction.
func WithID(id int) func(*Junction) {
	return func(j *Junction) {
		j.ID = id
	}
}

// WithLabel is an option function that sets the label for the junction.
func WithLabel(label string) func(*Junction) {
	return func(j *Junction) {
		j.Label = label
	}
}

// WithPoint is an option function that sets the point (location) for the junction.
func WithPoint(point geom.Point) func(*Junction) {
	return func(j *Junction) {
		j.point = point
	}
}

// GetTotalDuration returns the total duration of the junction's cycle in seconds.
func (jun *Junction) GetTotalDuration() int {
	return jun.totalDuration
}

// GetPoint returns the point (location) of the junction.
func (jun *Junction) GetPoint() geom.Point {
	return jun.point
}

// GetGreenIntervals extracts green intervals from the junction's cycle.
func (jun *Junction) GetGreenIntervals() []*greeninterval.GreenInterval {
	intervals := make([]*greeninterval.GreenInterval, 0)

	cycleDuration := jun.totalDuration
	if cycleDuration <= 0 {
		return intervals
	}

	currentTime := 0
	for phaseIdx, phase := range jun.Cycle {
		phaseEnd := currentTime + phase.GetTotalSeconds()
		signalStart := currentTime
		for _, signal := range phase.Signals {
			if signal.Color == color.GREEN || signal.Color == color.GREENPRIORITY {
				start := signalStart
				end := signalStart + signal.Duration
				if end == cycleDuration {
					intervals = append(intervals, greeninterval.New(phaseIdx, float64(start%cycleDuration), float64(end)))
				} else {
					intervals = append(intervals, greeninterval.New(phaseIdx, float64(start%cycleDuration), float64(end%cycleDuration)))
				}
			}
			signalStart += signal.Duration
		}
		currentTime = phaseEnd
	}
	return intervals
}

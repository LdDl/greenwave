package greenwave

// Junction represents a traffic light junction
type Junction struct {
	// Traffic light identifier
	ID int
	// User defined alias
	Label string
	// Cycle is a list of phases that define the traffic light cycle for this junction
	Cycle []*Phase
	// totalDuration is the total duration of the cycle in seconds
	totalDuration int
	// offset of the cycle
	offset int
}

// NewJunction creates a new Junction instance with the specified ID, label, cycle (list of phases)
func NewJunction(cycle []*Phase, options ...func(*Junction)) *Junction {
	totalDuration := 0
	for _, phase := range cycle {
		totalDuration += phase.totalSeconds
	}
	junction := &Junction{
		ID:            -1,
		Label:         "-1",
		Cycle:         cycle,
		totalDuration: totalDuration,
		offset:        0, // Default offset is 0, can be set later if needed
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
func (jun *Junction) SetOffset(offset int) {
	jun.offset = offset
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

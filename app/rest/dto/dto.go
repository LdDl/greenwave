package dto

// JunctionDTO represents a junction for API communication.
// Represents a traffic light junction
// swagger:model
type JunctionDTO struct {
	// Traffic light identifier
	ID int `json:"id"`
	// User defined alias
	Label string `json:"label"`
	// Cycle is a list of phases that define the traffic light cycle for this junction
	Cycle []PhaseDTO `json:"cycle"`
	// Total duration of the cycle in seconds
	TotalDuration int `json:"total_duration"`
	// Offset of the cycle
	Offset int `json:"offset"`
	// Location of the junction
	Point PointDTO `json:"point"`
}

// PhaseDTO represents a phase for API communication.
// Represents a traffic light phase with an ID and a list of signals.
// swagger:model
type PhaseDTO struct {
	// Indentifier for the phase
	ID int `json:"id"`
	// List of signals that define the phase
	Signals []SignalDTO `json:"signals"`
	// Total duration of the phase in seconds, calculated from the signals
	TotalSeconds int `json:"total_seconds"`
}

// SignalDTO represents a signal for API communication.
// Represents a traffic light signal with its duration and color.
// Also it includes minimum and maximum duration threshold for the signal which could be usefull during optimizations of traffic light timings.
// swagger:model
type SignalDTO struct {
	// Duration is the duration of the signal in seconds.
	Duration int `json:"duration"`
	// MinDuration is the minimum duration for the signal in seconds. Could be used during optimizations in further researchs.
	MinDuration *int `json:"min_duration"`
	// MaxDuration is the maximum duration for the signal in seconds. Could be used during optimizations in further researchs.
	MaxDuration *int `json:"max_duration"`
	// Color is the color of the signal
	Color string `json:"color"`
}

// PointDTO represents a point for API communication.
// Point in 2D space.
// swagger:model
type PointDTO struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GreenWaveDTO represents a green wave for API communication.
// Represents a green wave between two junctions.
// swagger:model
type GreenWaveDTO struct {
	// Green interval on the first junction.
	IntervalJunOne *GreenIntervalDTO `json:"interval_jun_one"`
	// Green interval on the second junction.
	IntervalJunTwo *GreenIntervalDTO `json:"interval_jun_two"`
	// Distance in meters between the two junctions.
	Distance float64 `json:"distance"`
	// Travel time in seconds between the two junctions.
	TravelTime float64 `json:"travel_time"`
	// Bandwidth of the green wave in seconds.
	BandWidth float64 `json:"band_width"`
}

// GreenIntervalDTO represents a green interval for API communication.
// Represents a time interval during which a traffic light is green (for specific phase).
// swagger:model
type GreenIntervalDTO struct {
	// Phase index in which the green interval occurs
	PhaseIdx int `json:"phase_idx"`
	// Start time of the green interval in seconds
	Start float64 `json:"start"`
	// End time of the green interval in seconds
	End float64 `json:"end"`
}

// ThroughGreenWaveDTO represents a through green wave for API communication.
// Represents a green wave that passes through a series of green intervals between traffic lights.
// swagger:model
type ThroughGreenWaveDTO struct {
	// Intervals of green lights that form the green wave
	Intervals []GreenIntervalDTO `json:"intervals"`
	// Number of junctions which could be passed through
	Depth int `json:"depth"`
	// Bandwidth of the green wave, which is the minimum duration of the green intervals
	Bandwidth float64 `json:"bandwidth"`
}

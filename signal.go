package greenwave

import (
	"github.com/LdDl/greenwave/color"
)

// Signal represents a traffic light signal with its duration and color.
// Also it includes minimum and maximum duration threshold for the signal which could be usefull during optimizations of traffic light timings.
type Signal struct {
	// Duration is the duration of the signal in seconds.
	Duration int
	// MinDuration is the minimum duration for the signal in seconds. Could be used during optimizations in further researchs.
	MinDuration int
	// MaxDuration is the maximum duration for the signal in seconds. Could be used during optimizations in further researchs.
	MaxDuration int
	// Color is the color of the signal
	Color color.Color
}

// NewSignal creates a new Signal instance with the specified duration, minimum and maximum durations, and color.
func NewSignal(duration int, c color.Color, options ...func(*Signal)) *Signal {
	signal := &Signal{
		Duration:    duration,
		MinDuration: duration,
		MaxDuration: duration,
		Color:       c,
	}
	for _, option := range options {
		option(signal)
	}
	return signal
}

// WithMinDuration is an option function that sets the minimum duration for the signal.
func WithMinDuration(minDuration int) func(*Signal) {
	return func(s *Signal) {
		s.MinDuration = minDuration
	}
}

// WithMaxDuration is an option function that sets the maximum duration for the signal.
func WithMaxDuration(maxDuration int) func(*Signal) {
	return func(s *Signal) {
		s.MaxDuration = maxDuration
	}
}

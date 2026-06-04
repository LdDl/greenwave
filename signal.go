package greenwave

import (
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/signal"
)

// Signal is an alias for signal.Signal for backward compatibility.
type Signal = signal.Signal

// NewSignal creates a new Signal instance.
func NewSignal(duration int, c color.Color, options ...func(*signal.Signal)) *signal.Signal {
	return signal.NewSignal(duration, c, options...)
}

// WithMinDuration is an option function that sets the minimum duration for the signal.
func WithMinDuration(minDuration int) func(*signal.Signal) {
	return signal.WithMinDuration(minDuration)
}

// WithMaxDuration is an option function that sets the maximum duration for the signal.
func WithMaxDuration(maxDuration int) func(*signal.Signal) {
	return signal.WithMaxDuration(maxDuration)
}

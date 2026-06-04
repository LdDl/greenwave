package greenwave

import "github.com/LdDl/greenwave/signal"

// Phase is an alias for signal.Phase for backward compatibility.
type Phase = signal.Phase

// NewPhase creates a new Phase instance with the specified ID and signals.
func NewPhase(id int, signals []*signal.Signal) *signal.Phase {
	return signal.NewPhase(id, signals)
}

package greenwave

// Phase represents a traffic light phase with an ID and a list of signals.
type Phase struct {
	// Indentifier for the phase
	ID int
	// List of signals that define the phase
	Signals []*Signal
	// Total duration of the phase in seconds, calculated from the signals
	totalSeconds int
}

// NewPhase creates a new Phase instance with the specified ID and signals.
func NewPhase(id int, signals []*Signal) *Phase {
	totalSeconds := 0
	for _, signal := range signals {
		totalSeconds += signal.Duration
	}
	return &Phase{
		ID:           id,
		Signals:      signals,
		totalSeconds: totalSeconds,
	}
}

// GetTotalSeconds returns the total duration of the phase in seconds.
func (p *Phase) GetTotalSeconds() int {
	return p.totalSeconds
}

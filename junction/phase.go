package junction

type GroupID int

// SignalGroup associates a signal group ID with its signal sequence within a phase.
type SignalGroup struct {
	// ID identifies the signal group.
	ID GroupID
	// Signals is the ordered sequence of signal states for this group in this phase.
	Signals []*Signal
}

// Phase represents a traffic light phase with an ID and a list of signals.
type Phase struct {
	// Indentifier for the phase
	ID int
	// List of signals that define the phase for each signal group.
	SignalGroups []SignalGroup
	// Total duration of the phase (for given signal group) in seconds, calculated from the signals
	totalSeconds map[GroupID]int
}

// NewPhase creates a new Phase instance with the specified ID and signals for each signal group.
func NewPhase(id int, signalGroups []SignalGroup) *Phase {
	totalSeconds := make(map[GroupID]int)
	for _, sg := range signalGroups {
		dur := 0
		for _, signal := range sg.Signals {
			dur += signal.Duration
		}
		totalSeconds[sg.ID] = dur
	}
	return &Phase{
		ID:           id,
		SignalGroups: signalGroups,
		totalSeconds: totalSeconds,
	}
}

// GetTotalSeconds returns the total duration of the phase for given group in seconds.
// If the group does not exist, it returns -1 and false.
func (p *Phase) GetTotalSeconds(gid GroupID) (int, bool) {
	if duration, exists := p.totalSeconds[gid]; exists {
		return duration, true
	}
	return -1, false
}

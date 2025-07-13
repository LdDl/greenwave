package color

// Color is just for demonstration of traffic light signal
type Color uint16

const (
	// UNDEFINED is the default value for Color, used when no signal is set
	// All definitions have been taken from SUMO: https://sumo.dlr.de/docs/Simulation/Traffic_Lights.html#signal_state_definitions
	UNDEFINED Color = iota
	// 'red light' for a signal - vehicles must stop
	RED
	// 'amber (yellow) light' for a signal - vehicles will start to decelerate if far away from the junction, otherwise they pass
	YELLOW
	// 'green light' for a signal, no priority - vehicles may pass the junction if no vehicle uses a higher priorised foe stream, otherwise they decelerate for letting it pass. They always decelerate on approach until they are within the configured visibility distance
	GREEN
	// 'green light' for a signal, priority - vehicles may pass the junction
	GREENPRIORITY
	// 'green right-turn arrow' requires stopping - vehicles may pass the junction if no vehicle uses a higher priorised foe stream. They always stop before passing. This is only generated for junction type traffic_light_right_on_red.
	GREENRIGHT
	// 'red+yellow light' for a signal, may be used to indicate upcoming green phase but vehicles may not drive yet (shown as orange in the gui)
	REDYELLOW
	// 'off - blinking' signal is switched off, blinking light indicates vehicles have to yield
	BLINKING
	// 'off - no signal' signal is switched off, vehicles have the right of way
	NO
)

var colorToStr = [...]string{
	"UNDEFINED",
	"RED",
	"YELLOW",
	"GREEN",
	"GREENPRIORITY",
	"GREENRIGHT",
	"REDYELLOW",
	"BLINKING",
	"NO",
}

func (ioutIndex Color) String() string {
	return colorToStr[ioutIndex]
}

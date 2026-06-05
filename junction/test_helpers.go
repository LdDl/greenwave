package junction

import (
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/geom"
)

// BasicTestJunctions returns a set of 4 junctions used across tests.
// Each junction has a cycle of 85 seconds (two phases).
// Traffic lights IDs: 100500, 42, 78, 256. All junctions have the same single signal group with ID 0.
func BasicTestJunctions() []*Junction {
	return []*Junction{
		NewJunction(
			[]*Phase{
				NewPhase(0, []SignalGroup{{ID: 0, Signals: []*Signal{
					NewSignal(30, color.GREEN),
					NewSignal(20, color.RED),
				}}}),
				NewPhase(1, []SignalGroup{{ID: 0, Signals: []*Signal{
					NewSignal(20, color.GREEN),
					NewSignal(15, color.RED),
				}}}),
			},
			WithID(100500),
			WithPoint(geom.Point{X: 0, Y: 0}),
		),
		NewJunction(
			[]*Phase{
				NewPhase(10, []SignalGroup{{ID: 0, Signals: []*Signal{
					NewSignal(20, color.RED),
					NewSignal(35, color.GREEN),
					NewSignal(5, color.YELLOW),
				}}}),
				NewPhase(11, []SignalGroup{{ID: 0, Signals: []*Signal{
					NewSignal(10, color.RED),
					NewSignal(10, color.GREEN),
					NewSignal(5, color.YELLOW),
				}}}),
			},
			WithID(42),
			WithPoint(geom.Point{X: 0, Y: 200}),
		),
		NewJunction(
			[]*Phase{
				NewPhase(20, []SignalGroup{{ID: 0, Signals: []*Signal{
					NewSignal(45, color.RED),
					NewSignal(10, color.GREEN),
				}}}),
				NewPhase(21, []SignalGroup{{ID: 0, Signals: []*Signal{
					NewSignal(7, color.RED),
					NewSignal(18, color.GREEN),
					NewSignal(5, color.YELLOW),
				}}}),
			},
			WithID(78),
			WithPoint(geom.Point{X: 0, Y: 450}),
		),
		NewJunction(
			[]*Phase{
				NewPhase(20, []SignalGroup{{ID: 0, Signals: []*Signal{
					NewSignal(40, color.RED),
					NewSignal(15, color.GREEN),
				}}}),
				NewPhase(21, []SignalGroup{{ID: 0, Signals: []*Signal{
					NewSignal(10, color.RED),
					NewSignal(20, color.GREEN),
				}}}),
			},
			WithID(256),
			WithPoint(geom.Point{X: 0, Y: 600}),
		),
	}
}

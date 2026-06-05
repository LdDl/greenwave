package main

import (
	"fmt"

	"github.com/LdDl/greenwave"
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/geom"
	"github.com/LdDl/greenwave/junction"
)

func main() {
	junctions := []*junction.Junction{
		junction.NewJunction(
			[]*junction.Phase{
				junction.NewPhase(0, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
					junction.NewSignal(30, color.GREEN),
					junction.NewSignal(20, color.RED),
				}}}),
				junction.NewPhase(1, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
					junction.NewSignal(20, color.GREEN),
					junction.NewSignal(15, color.RED),
				}}}),
			},
			junction.WithID(1),
			junction.WithPoint(geom.Point{X: 0, Y: 0}),
		),
		junction.NewJunction(
			[]*junction.Phase{
				junction.NewPhase(10, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
					junction.NewSignal(20, color.RED),
					junction.NewSignal(35, color.GREEN),
					junction.NewSignal(5, color.YELLOW),
				}}}),
				junction.NewPhase(11, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
					junction.NewSignal(10, color.RED),
					junction.NewSignal(10, color.GREEN),
					junction.NewSignal(5, color.YELLOW),
				}}}),
			},
			junction.WithID(2),
			junction.WithPoint(geom.Point{X: 0, Y: 200}),
		),
		junction.NewJunction(
			[]*junction.Phase{
				junction.NewPhase(20, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
					junction.NewSignal(45, color.RED),
					junction.NewSignal(10, color.GREEN),
				}}}),
				junction.NewPhase(21, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
					junction.NewSignal(7, color.RED),
					junction.NewSignal(18, color.GREEN),
					junction.NewSignal(5, color.YELLOW),
				}}}),
			},
			junction.WithID(3),
			junction.WithPoint(geom.Point{X: 0, Y: 450}),
		),
		junction.NewJunction(
			[]*junction.Phase{
				junction.NewPhase(20, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
					junction.NewSignal(40, color.RED),
					junction.NewSignal(15, color.GREEN),
				}}}),
				junction.NewPhase(21, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
					junction.NewSignal(10, color.RED),
					junction.NewSignal(20, color.GREEN),
				}}}),
			},
			junction.WithID(4),
			junction.WithPoint(geom.Point{X: 0, Y: 600}),
		),
	}

	// groupIDs maps each junction ID to the signal group used for green wave coordination.
	// In this example all junctions have a single signal group with ID 0.
	groupIDs := map[int]junction.GroupID{
		1: 0,
		2: 0,
		3: 0,
		4: 0,
	}

	desiredSpeedKhm := 50.0
	optimizer := greenwave.NewOptimizerGenetic(
		junctions,
		groupIDs,
		desiredSpeedKhm,
		50,
		100,
		0.1,
		3,
		greenwave.CROSSOVER_BLEND,
		greenwave.OPTIMIZATION_FORWARD,
	)
	newOffsets := optimizer.Optimize()
	fmt.Println("Best fitness history:")
	fitnessHistory := optimizer.(*greenwave.OptimizerGenetic).BestFitnessHistory()
	for i, fitness := range fitnessHistory {
		fmt.Printf("Generation %d: %f\n", i, fitness)
	}
	fmt.Println("Optimized offsets:")
	for i := range junctions {
		fmt.Printf("Junction at position %d has new offset %f\n", i, newOffsets[i])
	}
}

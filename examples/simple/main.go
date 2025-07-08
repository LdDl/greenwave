package main

import (
	"fmt"

	. "github.com/LdDl/greenwave"
	"github.com/LdDl/greenwave/color"
)

func main() {
	junctions := []*Junction{
		NewJunction(
			[]*Phase{
				NewPhase(0, []*Signal{
					NewSignal(30, color.GREEN),
					NewSignal(20, color.RED),
				}),
				NewPhase(1, []*Signal{
					NewSignal(20, color.GREEN),
					NewSignal(15, color.RED),
				}),
			},
			WithPoint(Point{X: 0, Y: 0}),
		),
		NewJunction(
			[]*Phase{
				NewPhase(10, []*Signal{
					NewSignal(20, color.RED),
					NewSignal(35, color.GREEN),
					NewSignal(5, color.YELLOW),
				}),
				NewPhase(11, []*Signal{
					NewSignal(10, color.RED),
					NewSignal(10, color.GREEN),
					NewSignal(5, color.YELLOW),
				}),
			},
			WithPoint(Point{X: 0, Y: 200}),
		),
		NewJunction(
			[]*Phase{
				NewPhase(20, []*Signal{
					NewSignal(45, color.RED),
					NewSignal(10, color.GREEN),
				}),
				NewPhase(21, []*Signal{
					NewSignal(7, color.RED),
					NewSignal(18, color.GREEN),
					NewSignal(5, color.YELLOW),
				}),
			},
			WithPoint(Point{X: 0, Y: 450}),
		),
		NewJunction(
			[]*Phase{
				NewPhase(20, []*Signal{
					NewSignal(40, color.RED),
					NewSignal(15, color.GREEN),
				}),
				NewPhase(21, []*Signal{
					NewSignal(10, color.RED),
					NewSignal(20, color.GREEN),
				}),
			},
			WithPoint(Point{X: 0, Y: 600}),
		),
	}

	desiredSpeedKhm := 50.0
	optimizer := NewOptimizerGenetic(
		junctions,
		desiredSpeedKhm,
		50,
		100,
		0.1,
		3,
		CROSSOVER_BLEND,
	)
	newOffsets := optimizer.Optimize()
	fmt.Println("Best fitness history:")
	fitnessHistory := optimizer.(*OptimizerGenetic).BestFitnessHistory()
	for i, fitness := range fitnessHistory {
		fmt.Printf("Generation %d: %f\n", i, fitness)
	}
	fmt.Println("Optimized offsets:")
	for i := range junctions {
		fmt.Printf("Junction at position %d has new offset %f\n", i, newOffsets[i])
	}
}

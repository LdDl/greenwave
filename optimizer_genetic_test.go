package greenwave

import (
	"testing"

	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/geom"
	"github.com/LdDl/greenwave/junction"
	"github.com/stretchr/testify/require"
)

func TestBidirectionalFitnessUsesSelectedGroupsAndSharedOffsets(t *testing.T) {
	junctions := make([]*junction.Junction, 3)
	for i := range junctions {
		phase := junction.NewPhase(0, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{junction.NewSignal(45, color.GREEN), junction.NewSignal(15, color.RED)}},
			{ID: 1, Signals: []*junction.Signal{junction.NewSignal(10, color.RED), junction.NewSignal(30, color.GREEN), junction.NewSignal(20, color.RED)}},
		})
		junctions[i] = junction.NewJunction([]*junction.Phase{phase}, junction.WithID(i), junction.WithPoint(geom.Point{Y: float64(i * 50)}))
	}
	forward := map[int]junction.GroupID{0: 0, 1: 0, 2: 0}
	reverse := map[int]junction.GroupID{0: 1, 1: 1, 2: 1}
	optimizer := NewOptimizerGenetic(junctions, forward, 36, 10, 2, 0.1, 2, CROSSOVER_BLEND, OPTIMIZATION_BIDIRECTIONAL, WithReverseGroupIDs(reverse)).(*OptimizerGenetic)
	individual := &Individual{Offsets: []float64{0, 2, 4}}
	fitness := optimizer.evaluateFitness(individual)
	require.Equal(t, 0, junctions[0].GetOffset())
	require.Equal(t, 2, junctions[1].GetOffset())
	forwardFitness := calculateDirectionalFitness(junctions, forward, 36)
	reverseFitness := calculateDirectionalFitness(ReverseJunctions(junctions), reverse, 36)
	require.Greater(t, reverseFitness, 0.0)
	require.Equal(t, forwardFitness+reverseFitness, fitness)
	legacy := NewOptimizerGenetic(junctions, forward, 36, 10, 2, 0.1, 2, CROSSOVER_BLEND, OPTIMIZATION_BIDIRECTIONAL).(*OptimizerGenetic)
	require.NotEqual(t, legacy.evaluateFitness(individual), fitness)
	fallback := NewOptimizerGenetic(junctions, forward, 36, 10, 2, 0.1, 2, CROSSOVER_BLEND, OPTIMIZATION_BIDIRECTIONAL, WithReverseGroupIDs(nil)).(*OptimizerGenetic)
	require.Equal(t, legacy.evaluateFitness(individual), fallback.evaluateFitness(individual))
	partial := NewOptimizerGenetic(junctions, forward, 36, 10, 2, 0.1, 2, CROSSOVER_BLEND, OPTIMIZATION_BIDIRECTIONAL, WithReverseGroupIDs(map[int]junction.GroupID{0: 1})).(*OptimizerGenetic)
	require.Equal(t, map[int]junction.GroupID{0: 1, 1: 0, 2: 0}, partial.reverseGroupIDs)
	optimizer.optimizationMode = OPTIMIZATION_FORWARD
	require.Equal(t, forwardFitness, optimizer.evaluateFitness(individual))
}

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

func fractionalPeriodicCrossover(_ []float64, _, _ *Individual) *Individual {
	return &Individual{Offsets: []float64{0, -2.9, 194.8}}
}

func TestOptimizerReturnsTheAppliedIntegerOffsets(t *testing.T) {
	junctions := []*junction.Junction{
		periodicTestJunction(1, 100, 40, 0, 0),
		periodicTestJunction(2, 100, 40, 0, 98),
		periodicTestJunction(3, 100, 40, 0, 194),
	}
	optimizer := NewOptimizerGenetic(junctions, nil, 3.6, 2, 2, 0, 1, CROSSOVER_BLEND, OPTIMIZATION_FORWARD).(*OptimizerGenetic)
	optimizer.crossoverFunc = fractionalPeriodicCrossover
	offsets := optimizer.Optimize()
	require.Equal(t, []float64{0, 98, 94}, offsets)
	for i, jun := range junctions {
		require.Equal(t, int(offsets[i]), jun.GetOffset())
	}
	through := MergeGreenWaves(FindGreenWaves(junctions, nil, 3.6))
	require.Len(t, through, 1)
	require.Equal(t, 40.0, through[0].Bandwidth())
	history := optimizer.BestFitnessHistory()
	require.Equal(t, history[len(history)-1], calculateDirectionalFitness(junctions, nil, 3.6))
}

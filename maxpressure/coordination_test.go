package maxpressure

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoordinationOnlyFollowsConfiguredMovements(t *testing.T) {
	net := buildTwoIntersectionCorridor(t)
	assert.True(t, net.IsUpstreamServed(130))
	// Same incoming road, but the right turn is outside the selected corridor.
	assert.False(t, net.IsUpstreamServed(131))

	// A different green movement into road 5 must not activate corridor [1,5,25].
	net.Intersections[50].PreviousStage = 1
	assert.False(t, net.IsUpstreamServed(130))
	require.NoError(t, net.SetCoordinatedCorridors([][]gmns.LinkID{{1, 5, 25}, {4, 5, 25}}))
	assert.True(t, net.IsUpstreamServed(130))

	// Overlapping corridor declarations must not multiply a binary indicator.
	require.NoError(t, net.SetCoordinatedCorridors([][]gmns.LinkID{{4, 5, 25}, {4, 5, 25}}))
	assert.Equal(t, net.MovementWeight(130)+0.25, net.SmoothedMovementWeight(130, SmoothingConfig{Alpha: 1}))
	require.NoError(t, net.SetCoordinatedCorridors(nil))
	assert.False(t, net.IsUpstreamServed(130))
	assert.Equal(t, net.MovementWeight(130), net.SmoothedMovementWeight(130, SmoothingConfig{Alpha: 1}))
}

func TestCoordinationUsesLastCompletedActuation(t *testing.T) {
	net := buildTwoIntersectionCorridor(t)
	for _, inter := range net.Intersections {
		inter.HasPreviousStage = false
	}
	assert.False(t, net.IsUpstreamServed(130), "zero-value stage ID is not actuation history")
	assert.Empty(t, net.ServedLinks(net.Intersections[50]))
	opt := mustMPOptimizer(t, net, MPConfig{DeltaT: 1, Drain: NoDrain(), Smoothing: SmoothingConfig{Alpha: 1}})

	// Empty roads still have a selected green stage: Xu's indicator uses s, not y.
	clear(net.Queues)
	clear(net.MovementQueues)
	first := opt.Step()
	for _, result := range first {
		assert.False(t, result.Boosted, "current-step decisions cannot influence each other")
	}
	assert.Equal(t, StageID(0), net.Intersections[50].PreviousStage)
	assert.True(t, net.IsUpstreamServed(130), "actuation must reach the next step without extra lag")
	assert.Zero(t, net.TotalQueueLength())

	// Force A to switch away from the corridor while B still reads A's old signal.
	net.MovementQueues[114] = 40
	net.syncRoadQueues()
	second := opt.Step()
	for _, result := range second {
		if result.IntersectionID == 60 {
			assert.Equal(t, StageID(0), result.SelectedStage)
			assert.True(t, result.Boosted)
		}
	}
	assert.Equal(t, StageID(1), net.Intersections[50].PreviousStage)
	assert.False(t, net.IsUpstreamServed(130), "red on the corridor must remove the indicator immediately for the next step")
	for _, result := range opt.Step() {
		assert.False(t, result.Boosted)
	}
}

type invalidCoordinationTestCase struct {
	name string
	path []gmns.LinkID
}

func TestInvalidCorridorPreservesConfiguration(t *testing.T) {
	tests := []invalidCoordinationTestCase{
		{name: "too short", path: []gmns.LinkID{1, 5}},
		{name: "unknown road", path: []gmns.LinkID{1, 999, 25}},
		{name: "connector instead of road", path: []gmns.LinkID{1, 110, 25}},
		{name: "missing movement", path: []gmns.LinkID{1, 8, 25}},
		{name: "wrong direction", path: []gmns.LinkID{25, 5, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net := buildTwoIntersectionCorridor(t)
			require.Error(t, net.SetCoordinatedCorridors([][]gmns.LinkID{tt.path}))
			assert.True(t, net.IsUpstreamServed(130))
			assert.False(t, net.IsUpstreamServed(131))
		})
	}
}

func TestCorridorRequiresAssignedMovement(t *testing.T) {
	net := buildTwoIntersectionCorridor(t)
	net.Intersections[60].Stages[0].ConnectorIDs = []gmns.LinkID{131, 132, 133}
	require.ErrorContains(t, net.SetCoordinatedCorridors([][]gmns.LinkID{{1, 5, 25}}), "no signal-controlled movement")
}

func TestAlphaZeroIgnoresCoordination(t *testing.T) {
	net := buildTwoIntersectionCorridor(t)
	inter := net.Intersections[60]
	stage, pressure := net.SelectPhase(inter)
	smoothedStage, smoothedPressure := net.SmoothedSelectPhase(inter, SmoothingConfig{Alpha: 0})
	assert.Equal(t, stage, smoothedStage)
	assert.Equal(t, pressure, smoothedPressure)
}

func TestForwardAndReverseCorridorsAreIndependent(t *testing.T) {
	net := buildTwoIntersectionCorridor(t)
	net.Meso.Links[118] = connLink(118, 10, 10, 50, 26, 6, 1800)
	net.Intersections[50].Stages[1].ConnectorIDs = append(net.Intersections[50].Stages[1].ConnectorIDs, 118)
	require.NoError(t, net.SetCoordinatedCorridors([][]gmns.LinkID{{1, 5, 25}, {15, 26, 6}}))
	net.Intersections[60].PreviousStage = 0
	assert.True(t, net.IsUpstreamServed(130))
	assert.True(t, net.IsUpstreamServed(118))
	net.Intersections[50].PreviousStage = 1
	assert.False(t, net.IsUpstreamServed(130))
	assert.True(t, net.IsUpstreamServed(118))
	net.Intersections[60].PreviousStage = 1
	assert.False(t, net.IsUpstreamServed(118))
}

func TestCorridorRequiresSuccessiveIntersections(t *testing.T) {
	net := buildTwoIntersectionCorridor(t)
	net.Meso.Links[130] = connLink(130, 10, 10, 50, 5, 25, 1800)
	net.Intersections[50].Stages[0].ConnectorIDs = append(net.Intersections[50].Stages[0].ConnectorIDs, 130)
	require.ErrorContains(t, net.SetCoordinatedCorridors([][]gmns.LinkID{{1, 5, 25}}), "different intersections")
}

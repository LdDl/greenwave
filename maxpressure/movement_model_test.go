package maxpressure

import (
	"context"
	"math"
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/meso"
	"github.com/stretchr/testify/require"
)

func movementNetwork() *Network {
	net := NewNetwork(meso.NewNet())
	for _, id := range []gmns.LinkID{1, 2, 3, 4, 5} {
		net.Meso.Links[id] = segLink(id, 0, 1, 70, 1, 3600)
	}
	net.Meso.Links[100] = connLink(100, 1, 2, 10, 1, 2, 3600)
	net.Meso.Links[101] = connLink(101, 1, 2, 10, 1, 3, 3600)
	net.Meso.Links[200] = connLink(200, 2, 3, 20, 2, 4, 3600)
	net.Meso.Links[201] = connLink(201, 2, 3, 20, 2, 5, 3600)
	net.TurningRatios = map[gmns.LinkID]float64{100: .75, 101: .25, 200: .6, 201: .4}
	net.Intersections[10] = &IntersectionState{MacroNodeID: 10, Stages: []Stage{{ID: 0, ConnectorIDs: []gmns.LinkID{100}}, {ID: 1, ConnectorIDs: []gmns.LinkID{101}}}}
	net.Intersections[20] = &IntersectionState{MacroNodeID: 20, Stages: []Stage{{ID: 0, ConnectorIDs: []gmns.LinkID{200}}, {ID: 1, ConnectorIDs: []gmns.LinkID{201}}}}
	return net
}

func TestTurningDemandWaitsForItsOwnStage(t *testing.T) {
	net := movementNetwork()
	net.MovementQueues[100], net.MovementQueues[101] = 8, 2
	opt := mustMPOptimizer(t, net, MPConfig{DeltaT: 1, Drain: NoDrain()})
	opt.Step()
	require.Equal(t, 7.0, net.MovementQueues[100])
	require.Equal(t, 2.0, net.MovementQueues[101], "left-turn vehicles cannot leave during through green")
	require.InDelta(t, .6, net.MovementQueues[200], 1e-12)
	require.InDelta(t, .4, net.MovementQueues[201], 1e-12)
	require.Zero(t, net.Queues[4], "new arrivals cannot discharge again in this step")
	require.InDelta(t, 0, opt.Balance().Error, 1e-12)
}

func TestMovementPressureUsesWeightedDownstreamQueues(t *testing.T) {
	net := movementNetwork()
	net.MovementQueues = map[gmns.LinkID]float64{100: 10, 101: 2, 200: 3, 201: 7}
	mustMPOptimizer(t, net, MPConfig{DeltaT: 2})
	// Q=2 and downstream expectation .6*3+.4*7=4.6.
	require.InDelta(t, 10.8, net.MovementWeight(100), 1e-12)
	// An exit has no further turning queues.
	require.InDelta(t, 4, net.MovementWeight(101), 1e-12)
}

func TestDemandIsSplitBeforeAnyFutureGreenChoice(t *testing.T) {
	net := movementNetwork()
	opt := mustMPOptimizer(t, net, MPConfig{DeltaT: 1, Demand: ConstantDemand(map[gmns.LinkID]float64{1: 3600})})
	opt.Step()
	require.Equal(t, .75, net.MovementQueues[100])
	require.Equal(t, .25, net.MovementQueues[101])
	require.Zero(t, opt.Balance().Departed)
	require.Equal(t, 1.0, opt.Balance().Requested)
}

func TestFiniteStorageRetainsAllRejectedDemand(t *testing.T) {
	net := movementNetwork()
	net.FiniteStorage = true
	net.Queues[1], net.Queues[2], net.Queues[3] = 10, 10, 10
	opt := mustMPOptimizer(t, net, MPConfig{DeltaT: 1, SimTime: 10, Drain: NoDrain(), Demand: ConstantDemand(map[gmns.LinkID]float64{1: 72000})})
	evaluation, err := opt.Evaluate(context.Background())
	require.NoError(t, err)
	require.Equal(t, 200.0, evaluation.Balance.Requested)
	require.Positive(t, evaluation.Balance.BoundaryBacklog)
	require.Positive(t, evaluation.BoundaryWaitVehS)
	require.InDelta(t, 0, evaluation.Balance.Error, 1e-9)
	for _, queue := range net.Queues {
		require.LessOrEqual(t, queue, 10.0+1e-9)
	}
}

func TestPointQueuesAreNotClippedAndHorizonIsExact(t *testing.T) {
	net := movementNetwork()
	opt := mustMPOptimizer(t, net, MPConfig{DeltaT: 1, SimTime: 2.5, Drain: NoDrain(), Demand: ConstantDemand(map[gmns.LinkID]float64{1: 72000})})
	evaluation, err := opt.Evaluate(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2.5, opt.Time())
	require.Equal(t, 50.0, evaluation.Balance.Requested)
	require.Zero(t, evaluation.Balance.BoundaryBacklog)
	require.InDelta(t, 0, evaluation.Balance.Error, 1e-9)
	require.Greater(t, net.Queues[1], net.StorageCapacity(1))
}

func TestMinimumMaximumGreenAndClearance(t *testing.T) {
	net := movementNetwork()
	net.MovementQueues[100], net.MovementQueues[101] = 100, 1
	inter := net.Intersections[10]
	inter.MinGreenS, inter.MaxGreenS, inter.ClearanceS = 2, 3, 2
	opt := mustMPOptimizer(t, net, MPConfig{DeltaT: 1})
	actual := []StageID{}
	for i := 0; i < 8; i++ {
		for _, result := range opt.Step() {
			if result.IntersectionID == 10 {
				actual = append(actual, result.SelectedStage)
			}
		}
	}
	require.Equal(t, []StageID{0, 0, 0, NoStage, NoStage, 1, 1, NoStage}, actual)
}

func TestZeroRateDrainBlocksOnlyTheSpecifiedExit(t *testing.T) {
	drain := RateDrain(map[gmns.LinkID]float64{3: 0})
	require.Zero(t, drain(3, 10, 1))
	require.Equal(t, 10.0, drain(4, 10, 1))
}

func TestInvalidTrafficInputsFailBeforeSimulation(t *testing.T) {
	net := movementNetwork()
	delete(net.TurningRatios, 101)
	_, err := NewMPOptimizer(net, MPConfig{DeltaT: 1})
	require.ErrorContains(t, err, "explicit turning proportions")
	net = movementNetwork()
	net.MovementQueues[100] = -1
	_, err = NewMPOptimizer(net, MPConfig{DeltaT: 1})
	require.Error(t, err)
	net = movementNetwork()
	opt := mustMPOptimizer(t, net, MPConfig{DeltaT: 1, Demand: func(gmns.LinkID, float64) float64 { return math.NaN() }})
	require.Empty(t, opt.Step())
	require.Error(t, opt.Err())
	require.Zero(t, opt.Time())
}

func TestEvaluationHonorsCancellation(t *testing.T) {
	opt := mustMPOptimizer(t, movementNetwork(), MPConfig{DeltaT: 1, SimTime: 100})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := opt.Evaluate(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

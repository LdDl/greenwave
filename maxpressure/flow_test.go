package maxpressure

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/meso"
	"github.com/stretchr/testify/assert"
)

type flowTestMovement struct {
	id         gmns.LinkID
	upstream   gmns.LinkID
	downstream gmns.LinkID
	capacity   int
}

type flowStepTestCase struct {
	name       string
	queues     map[gmns.LinkID]float64
	movements  []flowTestMovement
	activeIDs  []gmns.LinkID
	wantQueues map[gmns.LinkID]float64
}

func TestStepConservesVehiclesThroughActiveMovements(t *testing.T) {
	tests := []flowStepTestCase{
		{
			name:       "diverging movements share one upstream queue",
			queues:     map[gmns.LinkID]float64{1: 1, 2: 0, 3: 0},
			movements:  []flowTestMovement{{100, 1, 2, 1800}, {101, 1, 3, 1800}},
			activeIDs:  []gmns.LinkID{100, 101},
			wantQueues: map[gmns.LinkID]float64{1: 0, 2: 0.5, 3: 0.5},
		},
		{
			name:       "initial queue follows explicitly supplied turning proportions",
			queues:     map[gmns.LinkID]float64{1: 1, 2: 0, 3: 0},
			movements:  []flowTestMovement{{100, 1, 2, 3600}, {101, 1, 3, 1800}},
			activeIDs:  []gmns.LinkID{100, 101},
			wantQueues: map[gmns.LinkID]float64{1: 0, 2: 2.0 / 3, 3: 1.0 / 3},
		},
		{
			name:       "unconstrained movements retain their saturation flow",
			queues:     map[gmns.LinkID]float64{1: 4, 2: 0, 3: 0},
			movements:  []flowTestMovement{{100, 1, 2, 3600}, {101, 1, 3, 1800}},
			activeIDs:  []gmns.LinkID{100, 101},
			wantQueues: map[gmns.LinkID]float64{1: 1, 2: 2, 3: 1},
		},
		{
			name:       "full destination keeps vehicles upstream",
			queues:     map[gmns.LinkID]float64{1: 4, 2: 100},
			movements:  []flowTestMovement{{100, 1, 2, 1800}},
			activeIDs:  []gmns.LinkID{100},
			wantQueues: map[gmns.LinkID]float64{1: 4, 2: 100},
		},
		{
			name:       "partially full destination accepts only available space",
			queues:     map[gmns.LinkID]float64{1: 1, 2: 99.75},
			movements:  []flowTestMovement{{100, 1, 2, 1800}},
			activeIDs:  []gmns.LinkID{100},
			wantQueues: map[gmns.LinkID]float64{1: 0.75, 2: 100},
		},
		{
			name:       "merging movements share receiving space",
			queues:     map[gmns.LinkID]float64{1: 3, 2: 3, 3: 99.25},
			movements:  []flowTestMovement{{100, 1, 3, 1800}, {101, 2, 3, 1800}},
			activeIDs:  []gmns.LinkID{100, 101},
			wantQueues: map[gmns.LinkID]float64{1: 2.625, 2: 2.625, 3: 100},
		},
		{
			name:       "merging allocation follows requested flow proportions",
			queues:     map[gmns.LinkID]float64{1: 3, 2: 3, 3: 99.25},
			movements:  []flowTestMovement{{100, 1, 3, 3600}, {101, 2, 3, 1800}},
			activeIDs:  []gmns.LinkID{100, 101},
			wantQueues: map[gmns.LinkID]float64{1: 2.5, 2: 2.75, 3: 100},
		},
		{
			name:       "blocked share stays upstream without being rerouted",
			queues:     map[gmns.LinkID]float64{1: 1, 2: 100, 3: 0},
			movements:  []flowTestMovement{{100, 1, 2, 1800}, {101, 1, 3, 1800}},
			activeIDs:  []gmns.LinkID{100, 101},
			wantQueues: map[gmns.LinkID]float64{1: 0.5, 2: 100, 3: 0.5},
		},
		{
			name:       "duplicate connector is served once",
			queues:     map[gmns.LinkID]float64{1: 4, 2: 0},
			movements:  []flowTestMovement{{100, 1, 2, 1800}},
			activeIDs:  []gmns.LinkID{100, 100},
			wantQueues: map[gmns.LinkID]float64{1: 3, 2: 1},
		},
		{
			name:       "arrivals cannot leave again in the same step",
			queues:     map[gmns.LinkID]float64{1: 2, 2: 0, 3: 0},
			movements:  []flowTestMovement{{100, 1, 2, 1800}, {101, 2, 3, 1800}},
			activeIDs:  []gmns.LinkID{100, 101},
			wantQueues: map[gmns.LinkID]float64{1: 1, 2: 1, 3: 0},
		},
		{
			name:       "receiving space uses the queue before transfers",
			queues:     map[gmns.LinkID]float64{1: 1, 2: 100, 3: 0},
			movements:  []flowTestMovement{{100, 1, 2, 1800}, {101, 2, 3, 1800}},
			activeIDs:  []gmns.LinkID{100, 101},
			wantQueues: map[gmns.LinkID]float64{1: 1, 2: 99, 3: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, alpha := range []float64{0, 1} {
				net := NewNetwork(meso.NewNet())
				net.FiniteStorage = true
				for id, queue := range tt.queues {
					net.Meso.Links[id] = segLink(id, 0, 1, 700, 1, 3600)
					net.Queues[id] = queue
				}
				for _, movement := range tt.movements {
					net.Meso.Links[movement.id] = connLink(movement.id, 1, 2, 50, movement.upstream, movement.downstream, movement.capacity)
				}
				net.Intersections[50] = &IntersectionState{
					MacroNodeID: 50,
					Stages:      []Stage{{ID: 0, ConnectorIDs: tt.activeIDs}},
				}
				totals := make(map[gmns.LinkID]float64)
				for _, movement := range tt.movements {
					totals[movement.upstream] += float64(movement.capacity)
				}
				for _, movement := range tt.movements {
					net.TurningRatios[movement.id] = float64(movement.capacity) / totals[movement.upstream]
				}
				opt := mustMPOptimizer(t, net, MPConfig{
					DeltaT:    2,
					Drain:     NoDrain(),
					Smoothing: SmoothingConfig{Alpha: alpha},
				})
				initialTotal := net.TotalQueueLength()

				opt.Step()

				assert.InDelta(t, initialTotal, net.TotalQueueLength(), 1e-9, "alpha=%v", alpha)
				for id, want := range tt.wantQueues {
					assert.InDelta(t, want, net.Queues[id], 1e-9, "link=%d, alpha=%v", id, alpha)
				}
				for step := 0; step < 100; step++ {
					opt.Step()
					assert.InDelta(t, initialTotal, net.TotalQueueLength(), 1e-9, "step=%d, alpha=%v", step, alpha)
					for id, queue := range net.Queues {
						assert.GreaterOrEqual(t, queue, -1e-9, "link=%d, step=%d", id, step)
						assert.LessOrEqual(t, queue, net.StorageCapacity(id)+1e-9, "link=%d, step=%d", id, step)
					}
				}
			}
		})
	}
}

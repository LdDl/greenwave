package maxpressure

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/meso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTwoIntersectionCorridor creates intersections A(50) and B(60)
// connected by link 5 (A=>B, EB direction).
func buildTwoIntersectionCorridor() *Network {
	mn := meso.NewNet()

	mn.Links[1] = segLink(1, 0, 10, 200, 2, 3600)  // WA=>A (EB)
	mn.Links[2] = segLink(2, 0, 10, 300, 2, 3600)  // B=>A (WB)
	mn.Links[3] = segLink(3, 0, 10, 200, 2, 3600)  // NA=>A (SB)
	mn.Links[4] = segLink(4, 0, 10, 250, 2, 3600)  // C=>A (NB)
	mn.Links[5] = segLink(5, 10, 20, 300, 2, 3600) // A=>B shared link
	mn.Links[6] = segLink(6, 10, 30, 200, 2, 3600) // A=>WA
	mn.Links[7] = segLink(7, 10, 40, 200, 2, 3600) // A=>NA
	mn.Links[8] = segLink(8, 10, 50, 250, 2, 3600) // A=>C

	mn.Links[110] = connLink(110, 10, 10, 50, 1, 5, 1800) // EBT => link 5
	mn.Links[111] = connLink(111, 10, 10, 50, 1, 7, 1600) // EBR
	mn.Links[112] = connLink(112, 10, 10, 50, 2, 6, 1800) // WBT
	mn.Links[113] = connLink(113, 10, 10, 50, 2, 8, 1400) // WBL
	mn.Links[114] = connLink(114, 10, 10, 50, 3, 8, 1800) // SBT
	mn.Links[115] = connLink(115, 10, 10, 50, 3, 6, 1600) // SBR
	mn.Links[116] = connLink(116, 10, 10, 50, 4, 7, 1800) // NBT
	mn.Links[117] = connLink(117, 10, 10, 50, 4, 5, 1400) // NBL

	mn.Links[15] = segLink(15, 0, 20, 200, 2, 3600)   // EB=>B (WB approach)
	mn.Links[16] = segLink(16, 0, 20, 200, 2, 3600)   // NB=>B (SB approach)
	mn.Links[17] = segLink(17, 0, 20, 250, 2, 3600)   // D=>B (NB approach)
	mn.Links[25] = segLink(25, 20, 100, 200, 2, 3600) // B=>EB (EB dep)
	mn.Links[26] = segLink(26, 20, 100, 300, 2, 3600) // B=>A (WB dep)
	mn.Links[27] = segLink(27, 20, 100, 200, 2, 3600) // B=>NB (NB dep)
	mn.Links[28] = segLink(28, 20, 100, 250, 2, 3600) // B=>D (SB dep)

	mn.Links[130] = connLink(130, 20, 20, 60, 5, 25, 1800)  // EBT at B
	mn.Links[131] = connLink(131, 20, 20, 60, 5, 27, 1600)  // EBR at B
	mn.Links[132] = connLink(132, 20, 20, 60, 15, 26, 1800) // WBT at B
	mn.Links[133] = connLink(133, 20, 20, 60, 15, 28, 1400) // WBL at B
	mn.Links[134] = connLink(134, 20, 20, 60, 16, 28, 1800) // SBT at B
	mn.Links[135] = connLink(135, 20, 20, 60, 16, 26, 1600) // SBR at B
	mn.Links[136] = connLink(136, 20, 20, 60, 17, 27, 1800) // NBT at B
	mn.Links[137] = connLink(137, 20, 20, 60, 17, 25, 1400) // NBL at B

	net := NewNetwork(mn)

	net.Queues[1] = 15
	net.Queues[2] = 10
	net.Queues[3] = 8
	net.Queues[4] = 6
	net.Queues[5] = 12
	net.Queues[6] = 2
	net.Queues[7] = 4
	net.Queues[8] = 3
	net.Queues[15] = 4
	net.Queues[16] = 10
	net.Queues[17] = 7
	net.Queues[25] = 3
	net.Queues[26] = 2
	net.Queues[27] = 1
	net.Queues[28] = 4

	net.Intersections[50] = &IntersectionState{
		MacroNodeID: 50,
		Stages: []Stage{
			{ID: 0, ConnectorIDs: []gmns.LinkID{110, 111, 112, 113}},
			{ID: 1, ConnectorIDs: []gmns.LinkID{114, 115, 116, 117}},
		},
		ActiveStage:   0,
		PreviousStage: 0,
	}
	net.Intersections[60] = &IntersectionState{
		MacroNodeID: 60,
		Stages: []Stage{
			{ID: 0, ConnectorIDs: []gmns.LinkID{130, 131, 132, 133}},
			{ID: 1, ConnectorIDs: []gmns.LinkID{134, 135, 136, 137}},
		},
		ActiveStage:   1,
		PreviousStage: 1,
	}

	return net
}

func TestIsUpstreamServed_True(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	assert.True(t, net.IsUpstreamServed(130), "EBT at B should detect upstream served")
}

func TestIsUpstreamServed_False(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	assert.False(t, net.IsUpstreamServed(134), "SBT at B: boundary link, no upstream intersection")
}

func TestSmoothedMovementWeight_HasBoost(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	cfg := SmoothingConfig{Alpha: 1.0}

	standard := net.MovementWeight(130)
	smoothed := net.SmoothedMovementWeight(130, cfg)

	assert.Greater(t, smoothed, standard)

	// Xu et al. (2024) boost: xi = Alpha * SatFlow (constant, not queue-proportional)
	expectedBoost := 1.0 * net.SatFlow(130)
	assert.InDelta(t, expectedBoost, smoothed-standard, 0.1)
}

func TestSmoothedMovementWeight_NoBoostAlphaZero(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	standard := net.MovementWeight(130)
	smoothed := net.SmoothedMovementWeight(130, SmoothingConfig{Alpha: 0})
	assert.Equal(t, standard, smoothed)
}

func TestSmoothedSelectPhase_BoostFlipsDecision(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	interB := net.Intersections[60]

	stdPhase, _ := net.SelectPhase(interB)
	smoothPhase, _ := net.SmoothedSelectPhase(interB, SmoothingConfig{Alpha: 1.5})

	assert.Equal(t, StageID(0), smoothPhase, "Smoothing-MP should pick EW to serve arriving platoon")
	t.Logf("Standard MP: sg%d, Smoothing-MP: sg%d", stdPhase, smoothPhase)
}

func TestServedLinks(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	served := net.ServedLinks(net.Intersections[50])

	for _, lid := range []gmns.LinkID{5, 6, 7, 8} {
		assert.True(t, served[lid], "Link %d should be served", lid)
	}
	assert.Len(t, served, 4)
}

func TestOptimizer_StandardMP_SelectsEW(t *testing.T) {
	net := buildIntersectionA()
	opt := NewMPOptimizer(net, MPConfig{
		DeltaT:    5.0,
		Smoothing: SmoothingConfig{Alpha: 0},
	})

	results := opt.Step()
	require.Len(t, results, 1)
	assert.Equal(t, StageID(0), results[0].SelectedStage)
}

func TestOptimizer_QueuesNonNegative(t *testing.T) {
	net := buildIntersectionA()
	opt := NewMPOptimizer(net, MPConfig{
		DeltaT:    5.0,
		Smoothing: SmoothingConfig{Alpha: 0},
	})

	opt.Step()
	for lid, queue := range net.Queues {
		assert.GreaterOrEqual(t, queue, 0.0, "Link %d has negative queue", lid)
	}
}

func TestOptimizer_IntersectionStateUpdated(t *testing.T) {
	net := buildIntersectionA()
	inter := net.Intersections[50]
	inter.ActiveStage = 1

	opt := NewMPOptimizer(net, MPConfig{
		DeltaT:    5.0,
		Smoothing: SmoothingConfig{Alpha: 0},
	})
	opt.Step()

	assert.Equal(t, StageID(1), inter.PreviousStage)
	assert.Equal(t, StageID(0), inter.ActiveStage)
}

func TestOptimizer_BoundaryDepartures(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	opt := NewMPOptimizer(net, MPConfig{DeltaT: 5.0})

	assert.NotEmpty(t, opt.BoundaryDepartures())
	t.Logf("Detected boundary departures: %v", opt.BoundaryDepartures())
}

func TestOptimizer_MultiStep(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	opt := NewMPOptimizer(net, MPConfig{
		DeltaT:    5.0,
		Smoothing: DefaultSmoothingConfig(),
		Demand: ConstantDemand(map[gmns.LinkID]float64{
			1: 1600, 2: 1200, 4: 1000,
		}),
	})

	for step := 0; step < 10; step++ {
		results := opt.Step()
		for _, result := range results {
			assert.GreaterOrEqual(t, int(result.SelectedStage), 0)
		}
		for lid, queue := range net.Queues {
			assert.GreaterOrEqual(t, queue, 0.0, "Step %d, link %d", step, lid)
		}
	}
	t.Logf("After 10 steps (t=%.0fs): total queue = %.1f", opt.Time(), net.TotalQueueLength())
}

func TestOptimizer_Run(t *testing.T) {
	net := buildTwoIntersectionCorridor()
	opt := NewMPOptimizer(net, MPConfig{
		DeltaT:    5.0,
		SimTime:   60.0,
		Smoothing: DefaultSmoothingConfig(),
		Demand:    ConstantDemand(map[gmns.LinkID]float64{1: 1600}),
	})

	allResults := opt.Run()
	// 60/5 = 12 steps
	assert.Len(t, allResults, 12)
	assert.Equal(t, 60.0, opt.Time())
}

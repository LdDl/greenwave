package main

import (
	"fmt"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/meso"
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
	"github.com/LdDl/greenwave/maxpressure"
)

func main() {
	// 2-intersection corridor: [WA] => (A) => (B) => [EB]
	//
	//             [2] NA->A              [4] NB->B
	//              |                      |
	//              v                      v
	// [1] WA->A -> (A) -> [3] A->B ----> (B) -> [7] B->EB
	//              |                      |
	//              v                      v
	//           [5] A->WA              [8] B->NB
	//           [6] A->NA

	fmt.Println("=== Standard MP (alpha=0) ===")
	runScenario(0)

	fmt.Println()
	fmt.Println("=== Smoothing-MP (alpha=0.5) ===")
	runScenario(0.5)

	fmt.Println()
	fmt.Println("=== Realistic 4-phase junctions, Smoothing-MP (alpha=0.5) ===")
	runRealisticScenario(0.5)

	fmt.Println()
	fmt.Println("=== StagesFromJunction: derive stages from 4-phase junction config ===")
	demonstrateStagesFromJunction()
}

// peakDemand returns a time-varying DemandFunc simulating a morning rush:
//
//	0-60s   ramp-up    (50% -> 100%)
//	60-300s peak       (130% of base)
//	300-420s ramp-down (130% -> 100%)
//	420-600s recovery  (100%)
func peakDemand(base map[gmns.LinkID]float64) maxpressure.DemandFunc {
	return func(linkID gmns.LinkID, timeSec float64) float64 {
		rate := base[linkID]
		if rate == 0 {
			return 0
		}
		var multiplier float64
		switch {
		case timeSec < 60:
			multiplier = 0.5 + 0.5*(timeSec/60.0)
		case timeSec < 300:
			multiplier = 1.3
		case timeSec < 420:
			multiplier = 1.3 - 0.3*((timeSec-300.0)/120.0)
		default:
			multiplier = 1.0
		}
		return rate * multiplier
	}
}

func runScenario(alpha float64) {
	mn := buildNetwork()
	net := maxpressure.NewNetwork(mn)

	// Initial queues: residual congestion from previous cycle
	net.Queues[1] = 15 // WA->A: 15 vehicles
	net.Queues[2] = 8  // NA->A: 8 vehicles
	net.Queues[3] = 5  // A->B: 5 vehicles (already in corridor)
	net.Queues[4] = 10 // NB->B: 10 vehicles

	net.Intersections[50] = &maxpressure.IntersectionState{
		MacroNodeID: 50,
		Stages: []maxpressure.Stage{
			{ID: 0, ConnectorIDs: []gmns.LinkID{100, 101}}, // EW
			{ID: 1, ConnectorIDs: []gmns.LinkID{102, 103}}, // NS
		},
	}
	net.Intersections[60] = &maxpressure.IntersectionState{
		MacroNodeID: 60,
		Stages: []maxpressure.Stage{
			{ID: 0, ConnectorIDs: []gmns.LinkID{200, 201}}, // EW
			{ID: 1, ConnectorIDs: []gmns.LinkID{202, 203}}, // NS
		},
	}

	// Effective capacity per approach ~ 800 veh/h (satflow 1600 * ~50% green).
	// Base rates above effective capacity to create realistic congestion.
	// At peak (x1.3): link1=2080, link2=1040, link4=1560 - oversaturated.
	demand := peakDemand(map[gmns.LinkID]float64{
		1: 1600, // WA->A: heavy eastbound corridor
		2: 800,  // NA->A: moderate southbound
		4: 1200, // NB->B: heavy competing flow at B
	})

	opt := maxpressure.NewMPOptimizer(net, maxpressure.MPConfig{
		DeltaT:    5.0,
		SimTime:   600.0, // 10 minutes
		Smoothing: maxpressure.SmoothingConfig{Alpha: alpha},
		Demand:    demand,
	})

	// Tracking
	totalQueue := 0.0
	totalLink3 := 0.0
	maxQueue := 0.0
	maxLink3 := 0.0
	stepsCount := 0
	bEWcount := 0 // how often B chose EW (sg0) - coordination metric
	bNScount := 0

	steps := int(opt.Config.SimTime / opt.Config.DeltaT)
	for i := 0; i < steps; i++ {
		stepResults := opt.Step()
		timeSec := opt.Time()
		queueLen := net.TotalQueueLength()
		link3queue := net.Queues[3]
		totalQueue += queueLen
		totalLink3 += link3queue
		if queueLen > maxQueue {
			maxQueue = queueLen
		}
		if link3queue > maxLink3 {
			maxLink3 = link3queue
		}
		stepsCount++

		for _, r := range stepResults {
			if r.IntersectionID == 60 {
				if r.SelectedStage == 0 {
					bEWcount++
				} else {
					bNScount++
				}
			}
		}

		// Print every 30 seconds and on final step
		if int(timeSec)%30 == 0 || i == steps-1 {
			fmt.Printf("  t=%4.0fs | total %5.1f | link3 %5.1f |", timeSec, queueLen, link3queue)
			for _, r := range stepResults {
				fmt.Printf(" int%d=>sg%d", r.IntersectionID, r.SelectedStage)
			}
			fmt.Println()
		}
	}

	avgQueue := totalQueue / float64(stepsCount)
	avgLink3 := totalLink3 / float64(stepsCount)
	fmt.Println("  ---")
	fmt.Printf("  avg total queue: %5.1f veh | max: %5.1f veh\n", avgQueue, maxQueue)
	fmt.Printf("  avg link3 queue: %5.1f veh | max: %5.1f veh\n", avgLink3, maxLink3)
	fmt.Printf("  B chose EW(sg0): %d times | NS(sg1): %d times  (EW%% = %.0f%%)\n",
		bEWcount, bNScount, 100*float64(bEWcount)/float64(bEWcount+bNScount))
}

func buildNetwork() *meso.Net {
	mn := meso.NewNet()

	// Road segments
	mn.Links[1] = meso.NewLinkFrom(1, 0, 10, meso.WithLengthMeters(200), meso.WithLanesNum(2), meso.WithCapacity(3600))  // WA->A
	mn.Links[2] = meso.NewLinkFrom(2, 0, 10, meso.WithLengthMeters(200), meso.WithLanesNum(2), meso.WithCapacity(3600))  // NA->A
	mn.Links[3] = meso.NewLinkFrom(3, 10, 20, meso.WithLengthMeters(300), meso.WithLanesNum(2), meso.WithCapacity(3600)) // A->B
	mn.Links[4] = meso.NewLinkFrom(4, 0, 20, meso.WithLengthMeters(200), meso.WithLanesNum(2), meso.WithCapacity(3600))  // NB->B
	mn.Links[5] = meso.NewLinkFrom(5, 10, 99, meso.WithLengthMeters(200), meso.WithLanesNum(2), meso.WithCapacity(3600)) // A->WA (departure)
	mn.Links[6] = meso.NewLinkFrom(6, 10, 99, meso.WithLengthMeters(200), meso.WithLanesNum(2), meso.WithCapacity(3600)) // A->NA (departure)
	mn.Links[7] = meso.NewLinkFrom(7, 20, 99, meso.WithLengthMeters(200), meso.WithLanesNum(2), meso.WithCapacity(3600)) // B->EB (departure)
	mn.Links[8] = meso.NewLinkFrom(8, 20, 99, meso.WithLengthMeters(200), meso.WithLanesNum(2), meso.WithCapacity(3600)) // B->NB (departure)

	// Connectors at intersection A (macroNode=50)
	mn.Links[100] = connector(100, 10, 50, 1, 3, 900) // EBT: WA->A => A->B
	mn.Links[101] = connector(101, 10, 50, 1, 6, 700) // EBR: WA->A => A->NA
	mn.Links[102] = connector(102, 10, 50, 2, 5, 900) // SBT: NA->A => A->WA
	mn.Links[103] = connector(103, 10, 50, 2, 3, 700) // SBL: NA->A => A->B

	// Connectors at intersection B (macroNode=60)
	mn.Links[200] = connector(200, 20, 60, 3, 7, 900) // EBT: A->B => B->EB
	mn.Links[201] = connector(201, 20, 60, 3, 8, 700) // EBR: A->B => B->NB
	mn.Links[202] = connector(202, 20, 60, 4, 7, 900) // SBT: NB->B => B->EB
	mn.Links[203] = connector(203, 20, 60, 4, 8, 700) // SBL: NB->B => B->NB

	return mn
}

// buildRealisticJunction creates a 4-phase, 4-group junction.
//
// Groups:
//
//	0  EW through
//	1  EW left turn
//	2  NS through
//	3  NS left turn
//
// Each group is active (GREEN) in exactly one phase and RED in the rest.
// Active phases follow the sequence GREEN -> YELLOW -> RED so every phase ends
// with RED (prohibition). Consecutive phases therefore always share the same
// ending meaning (prohibition -> prohibition), satisfying the no-opposite-ending rule.
func buildRealisticJunction() *junction.Junction {
	return junction.NewJunction(
		[]*junction.Phase{
			// Phase 0: EW through green (35s)
			junction.NewPhase(0, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(30, color.GREEN), junction.NewSignal(3, color.YELLOW), junction.NewSignal(2, color.RED)}},
				{ID: 1, Signals: []*junction.Signal{junction.NewSignal(35, color.RED)}},
				{ID: 2, Signals: []*junction.Signal{junction.NewSignal(35, color.RED)}},
				{ID: 3, Signals: []*junction.Signal{junction.NewSignal(35, color.RED)}},
			}),
			// Phase 1: EW left turn green (20s)
			junction.NewPhase(1, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(20, color.RED)}},
				{ID: 1, Signals: []*junction.Signal{junction.NewSignal(15, color.GREEN), junction.NewSignal(3, color.YELLOW), junction.NewSignal(2, color.RED)}},
				{ID: 2, Signals: []*junction.Signal{junction.NewSignal(20, color.RED)}},
				{ID: 3, Signals: []*junction.Signal{junction.NewSignal(20, color.RED)}},
			}),
			// Phase 2: NS through green (28s)
			junction.NewPhase(2, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(28, color.RED)}},
				{ID: 1, Signals: []*junction.Signal{junction.NewSignal(28, color.RED)}},
				{ID: 2, Signals: []*junction.Signal{junction.NewSignal(23, color.GREEN), junction.NewSignal(3, color.YELLOW), junction.NewSignal(2, color.RED)}},
				{ID: 3, Signals: []*junction.Signal{junction.NewSignal(28, color.RED)}},
			}),
			// Phase 3: NS left turn green (12s)
			junction.NewPhase(3, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(12, color.RED)}},
				{ID: 1, Signals: []*junction.Signal{junction.NewSignal(12, color.RED)}},
				{ID: 2, Signals: []*junction.Signal{junction.NewSignal(12, color.RED)}},
				{ID: 3, Signals: []*junction.Signal{junction.NewSignal(8, color.GREEN), junction.NewSignal(2, color.YELLOW), junction.NewSignal(2, color.RED)}},
			}),
		},
	)
}

// buildRealisticNetwork extends buildNetwork with left-turn departure links and
// connectors at each intersection, supporting 4 signal groups.
//
//	Links added:
//	  9, 10   departure links at A (for EW-left and NS-left turns)
//	  18, 19  departure links at B (for EW-left and NS-left turns)
//
//	Connectors added:
//	  104  EBL at A: WA->A => A->left-dep   (group 1)
//	  105  SBL at A: NA->A => A->left-dep   (group 3)
//	  204  EBL at B: A->B  => B->left-dep   (group 1)
//	  205  NBL at B: NB->B => B->left-dep   (group 3)
func buildRealisticNetwork() *meso.Net {
	mn := buildNetwork()

	// Extra departure links (drain nodes 99 / 98)
	mn.Links[9] = meso.NewLinkFrom(9, 10, 99, meso.WithLengthMeters(150), meso.WithLanesNum(1), meso.WithCapacity(1800))   // A->left-dep (EW-left)
	mn.Links[10] = meso.NewLinkFrom(10, 10, 99, meso.WithLengthMeters(150), meso.WithLanesNum(1), meso.WithCapacity(1800)) // A->left-dep (NS-left)
	mn.Links[18] = meso.NewLinkFrom(18, 20, 98, meso.WithLengthMeters(150), meso.WithLanesNum(1), meso.WithCapacity(1800)) // B->left-dep (EW-left)
	mn.Links[19] = meso.NewLinkFrom(19, 20, 98, meso.WithLengthMeters(150), meso.WithLanesNum(1), meso.WithCapacity(1800)) // B->left-dep (NS-left)

	// Left-turn connectors at A
	mn.Links[104] = connector(104, 10, 50, 1, 9, 600)  // EBL: WA->A => A-left-dep
	mn.Links[105] = connector(105, 10, 50, 2, 10, 600) // SBL: NA->A => A-left-dep

	// Left-turn connectors at B
	mn.Links[204] = connector(204, 20, 60, 3, 18, 600) // EBL: A->B => B-left-dep
	mn.Links[205] = connector(205, 20, 60, 4, 19, 600) // NBL: NB->B => B-left-dep

	return mn
}

func runRealisticScenario(alpha float64) {
	mn := buildRealisticNetwork()
	net := maxpressure.NewNetwork(mn)

	net.Queues[1] = 15 // WA->A
	net.Queues[2] = 8  // NA->A
	net.Queues[3] = 5  // A->B corridor
	net.Queues[4] = 10 // NB->B

	jun := buildRealisticJunction()

	groupConnectorsA := map[junction.GroupID][]gmns.LinkID{
		0: {100, 101}, // EW through
		1: {104},      // EW left turn
		2: {102, 103}, // NS through
		3: {105},      // NS left turn
	}
	groupConnectorsB := map[junction.GroupID][]gmns.LinkID{
		0: {200, 201}, // EW through
		1: {204},      // EW left turn
		2: {202, 203}, // NS through
		3: {205},      // NS left turn
	}

	net.Intersections[50] = &maxpressure.IntersectionState{
		MacroNodeID: 50,
		Stages:      maxpressure.StagesFromJunction(jun, groupConnectorsA),
	}
	net.Intersections[60] = &maxpressure.IntersectionState{
		MacroNodeID: 60,
		Stages:      maxpressure.StagesFromJunction(jun, groupConnectorsB),
	}

	demand := peakDemand(map[gmns.LinkID]float64{
		1: 1600,
		2: 800,
		4: 1200,
	})

	opt := maxpressure.NewMPOptimizer(net, maxpressure.MPConfig{
		DeltaT:    5.0,
		SimTime:   600.0,
		Smoothing: maxpressure.SmoothingConfig{Alpha: alpha},
		Demand:    demand,
	})

	totalQueue, totalLink3, maxQueue, maxLink3 := 0.0, 0.0, 0.0, 0.0
	stageCounts := map[maxpressure.StageID]int{}
	steps := int(opt.Config.SimTime / opt.Config.DeltaT)

	for i := 0; i < steps; i++ {
		stepResults := opt.Step()
		timeSec := opt.Time()
		queueLen := net.TotalQueueLength()
		link3queue := net.Queues[3]
		totalQueue += queueLen
		totalLink3 += link3queue
		if queueLen > maxQueue {
			maxQueue = queueLen
		}
		if link3queue > maxLink3 {
			maxLink3 = link3queue
		}

		for _, r := range stepResults {
			if r.IntersectionID == 60 {
				stageCounts[r.SelectedStage]++
			}
		}

		if int(timeSec)%30 == 0 || i == steps-1 {
			fmt.Printf("  t=%4.0fs | total %5.1f | link3 %5.1f |", timeSec, queueLen, link3queue)
			for _, r := range stepResults {
				fmt.Printf(" int%d=>sg%d", r.IntersectionID, r.SelectedStage)
			}
			fmt.Println()
		}
	}

	avgQueue := totalQueue / float64(steps)
	avgLink3 := totalLink3 / float64(steps)
	total := steps
	fmt.Println("  ---")
	fmt.Printf("  avg total queue: %5.1f veh | max: %5.1f veh\n", avgQueue, maxQueue)
	fmt.Printf("  avg link3 queue: %5.1f veh | max: %5.1f veh\n", avgLink3, maxLink3)
	fmt.Printf("  B stage selection: sg0(EW-thr)=%d  sg1(EW-left)=%d  sg2(NS-thr)=%d  sg3(NS-left)=%d  (total=%d)\n",
		stageCounts[0], stageCounts[1], stageCounts[2], stageCounts[3], total)
}

// demonstrateStagesFromJunction shows how StagesFromJunction derives stages from
// a 4-phase junction config. Each phase has 4 signal groups; only the group with
// GREEN signals contributes connectors to its stage.
func demonstrateStagesFromJunction() {
	jun := buildRealisticJunction()

	groupConnectorsA := map[junction.GroupID][]gmns.LinkID{
		0: {100, 101}, // EW through
		1: {104},      // EW left turn
		2: {102, 103}, // NS through
		3: {105},      // NS left turn
	}

	stages := maxpressure.StagesFromJunction(jun, groupConnectorsA)
	fmt.Printf("  Junction A: %d phases -> %d stages (cycle ~%ds)\n",
		len(jun.Cycle), len(stages), jun.GetTotalDuration())
	for _, s := range stages {
		fmt.Printf("    stage %d: connectors %v\n", s.ID, s.ConnectorIDs)
	}
}

func connector(id gmns.LinkID, node gmns.NodeID, macro gmns.NodeID, upstream, downstream gmns.LinkID, capacity int) *meso.Link {
	return meso.NewLinkFrom(id, node, node,
		meso.WithIsConnection(true),
		meso.WithLineMacroNodeID(macro),
		meso.WithMovementMesoLinkIncome(upstream),
		meso.WithMovementMesoLinkOutcome(downstream),
		meso.WithCapacity(capacity),
		meso.WithControlType(types.CONTROL_TYPE_IS_SIGNAL),
	)
}

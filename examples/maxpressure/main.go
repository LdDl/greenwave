package main

import (
	"fmt"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/meso"
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

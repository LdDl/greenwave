package maxpressure

import "github.com/LdDl/go-gmns/gmns"

// DemandFunc returns traffic intensity (veh/h) for a given link at a given
// simulation time (seconds). Called every step for every link that has demand.
type DemandFunc func(linkID gmns.LinkID, timeSec float64) float64

// ConstantDemand returns a DemandFunc that always returns the same intensity
// for each link, regardless of time.
func ConstantDemand(rates map[gmns.LinkID]float64) DemandFunc {
	return func(linkID gmns.LinkID, _ float64) float64 {
		return rates[linkID]
	}
}

// MPConfig holds configuration for the MP optimizer.
type MPConfig struct {
	DeltaT    float64         // simulation step duration (seconds)
	SimTime   float64         // total simulation time (seconds), 0 = unlimited
	Smoothing SmoothingConfig // Smoothing-MP parameters
	Demand    DemandFunc      // demand provider
}

// MPOptimizer runs max-pressure simulation on a Network.
type MPOptimizer struct {
	Net    *Network
	Config MPConfig

	// Auto-detected boundary departure links (vehicles exit here).
	boundaryDepartures []gmns.LinkID

	// Current simulation time.
	time float64
}

// NewMPOptimizer creates an optimizer, auto-detects boundary departures.
func NewMPOptimizer(net *Network, cfg MPConfig) *MPOptimizer {
	opt := &MPOptimizer{
		Net:    net,
		Config: cfg,
	}
	opt.boundaryDepartures = opt.detectBoundaryDepartures()
	return opt
}

// detectBoundaryDepartures finds links that receive traffic from connectors
// (are someone's MovementMesoLinkOutcome) but no connector takes from them
// (are nobody's MovementMesoLinkIncome). These are network exits.
func (opt *MPOptimizer) detectBoundaryDepartures() []gmns.LinkID {
	isOutcome := make(map[gmns.LinkID]bool)
	isIncome := make(map[gmns.LinkID]bool)

	for _, link := range opt.Net.Meso.Links {
		if !link.IsConnection() {
			continue
		}
		if id := link.MovementMesoLinkOutcome(); id >= 0 {
			isOutcome[id] = true
		}
		if id := link.MovementMesoLinkIncome(); id >= 0 {
			isIncome[id] = true
		}
	}

	var departures []gmns.LinkID
	for lid := range isOutcome {
		if !isIncome[lid] {
			departures = append(departures, lid)
		}
	}
	return departures
}

// BoundaryDepartures returns the auto-detected exit links.
func (opt *MPOptimizer) BoundaryDepartures() []gmns.LinkID {
	return opt.boundaryDepartures
}

// Time returns current simulation time in seconds.
func (opt *MPOptimizer) Time() float64 {
	return opt.time
}

// StepResult holds the outcome of one simulation step for an intersection.
type StepResult struct {
	IntersectionID gmns.NodeID
	SelectedPhase  SignalGroupID
	Pressure       float64
	Boosted        bool
}

// Step runs one simulation step:
//  1. Inject demand based on DemandFunc and current time.
//  2. Drain boundary departures.
//  3. Select signal group at each intersection (standard or Smoothing-MP).
//  4. Discharge vehicles through active connectors.
//  5. Update queues and intersection state.
//  6. Advance time.
func (opt *MPOptimizer) Step() []StepResult {
	net := opt.Net
	dt := opt.Config.DeltaT

	// 1. Inject demand
	if opt.Config.Demand != nil {
		for lid := range net.Meso.Links {
			rate := opt.Config.Demand(lid, opt.time) // veh/h
			if rate > 0 {
				inject := rate * dt / 3600.0
				net.Queues[lid] += inject
				if cap := net.StorageCapacity(lid); cap > 0 && net.Queues[lid] > cap {
					net.Queues[lid] = cap
				}
			}
		}
	}

	// 2. Drain boundary departures
	for _, lid := range opt.boundaryDepartures {
		net.Queues[lid] = 0
	}

	// 3. Select signal groups
	results := make([]StepResult, 0, len(net.Intersections))
	phaseDecisions := make(map[gmns.NodeID]SignalGroupID, len(net.Intersections))

	for nid, inter := range net.Intersections {
		var selected SignalGroupID
		var pressure float64

		if opt.Config.Smoothing.Alpha > 0 {
			selected, pressure = net.SmoothedSelectPhase(inter, opt.Config.Smoothing)
		} else {
			selected, pressure = net.SelectPhase(inter)
		}

		phaseDecisions[nid] = selected
		results = append(results, StepResult{
			IntersectionID: nid,
			SelectedPhase:  selected,
			Pressure:       pressure,
			Boosted:        opt.Config.Smoothing.Alpha > 0,
		})
	}

	// 4. Discharge vehicles through active connectors
	linkDelta := make(map[gmns.LinkID]float64)
	for nid, sgID := range phaseDecisions {
		inter := net.Intersections[nid]
		for i := range inter.SignalGroups {
			if inter.SignalGroups[i].ID != sgID {
				continue
			}
			for _, cid := range inter.SignalGroups[i].ConnectorIDs {
				link, ok := net.Meso.Links[cid]
				if !ok {
					continue
				}
				upID := link.MovementMesoLinkIncome()
				downID := link.MovementMesoLinkOutcome()
				if upID < 0 || downID < 0 {
					continue
				}
				maxDischarge := net.SatFlow(cid) * dt / 3600.0
				discharge := maxDischarge
				if upQueue := net.Queues[upID]; discharge > upQueue {
					discharge = upQueue
				}
				if discharge > 0 {
					linkDelta[upID] -= discharge
					linkDelta[downID] += discharge
				}
			}
			break
		}
	}

	// 5. Apply queue updates
	for lid, delta := range linkDelta {
		net.Queues[lid] += delta
		if net.Queues[lid] < 0 {
			net.Queues[lid] = 0
		}
		if cap := net.StorageCapacity(lid); cap > 0 && net.Queues[lid] > cap {
			net.Queues[lid] = cap
		}
	}

	// Update intersection state
	for nid, sgID := range phaseDecisions {
		inter := net.Intersections[nid]
		inter.PreviousPhase = inter.ActivePhase
		if sgID != inter.ActivePhase {
			inter.ActivePhase = sgID
			inter.ActivePhaseSince = opt.time
		}
	}

	// 6. Advance time
	opt.time += dt

	return results
}

// Run executes the full simulation for SimTime seconds.
// Returns results grouped by step.
func (opt *MPOptimizer) Run() [][]StepResult {
	if opt.Config.SimTime <= 0 {
		return nil
	}
	steps := int(opt.Config.SimTime / opt.Config.DeltaT)
	allResults := make([][]StepResult, 0, steps)
	for i := 0; i < steps; i++ {
		allResults = append(allResults, opt.Step())
	}
	return allResults
}

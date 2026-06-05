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

// PeakDemandConfig parameterises a time-varying peak demand profile.
type PeakDemandConfig struct {
	// BaseRates maps each link ID to its base demand intensity (veh/h).
	BaseRates map[gmns.LinkID]float64
	// RampUpS is the ramp-up phase duration (seconds). During this window the
	// multiplier rises linearly from 0.5 to 1.0. Zero disables the ramp-up.
	RampUpS float64
	// PeakFactor is the demand multiplier during the peak phase. Defaults to 1.3.
	PeakFactor float64
	// PeakDurationS is the duration (seconds) of the peak phase. Zero skips peak.
	PeakDurationS float64
	// RampDownS is the ramp-down phase duration (seconds). During this window the
	// multiplier drops linearly from PeakFactor to 1.0. Zero disables ramp-down.
	RampDownS float64
}

// PeakDemand returns a time-varying DemandFunc with a ramp-up -> peak -> ramp-down -> steady profile.
//
// Timeline (all durations in seconds, starting from t=0):
//
//	[0, RampUpS)                        multiplier: 0.5 -> 1.0  (linear)
//	[RampUpS, RampUpS+PeakDurationS)    multiplier: PeakFactor
//	[…, …+RampDownS)                    multiplier: PeakFactor -> 1.0  (linear)
//	[…, ∞)                              multiplier: 1.0
func PeakDemand(cfg PeakDemandConfig) DemandFunc {
	peakFactor := cfg.PeakFactor
	if peakFactor <= 0 {
		peakFactor = 1.3
	}
	peakStart := cfg.RampUpS
	peakEnd := peakStart + cfg.PeakDurationS
	rampDownEnd := peakEnd + cfg.RampDownS

	return func(linkID gmns.LinkID, timeSec float64) float64 {
		rate := cfg.BaseRates[linkID]
		if rate == 0 {
			return 0
		}
		var multiplier float64
		switch {
		case timeSec < peakStart:
			if peakStart > 0 {
				multiplier = 0.5 + 0.5*(timeSec/peakStart)
			} else {
				multiplier = 1.0
			}
		case timeSec < peakEnd:
			multiplier = peakFactor
		case timeSec < rampDownEnd:
			if cfg.RampDownS > 0 {
				multiplier = peakFactor - (peakFactor-1.0)*((timeSec-peakEnd)/cfg.RampDownS)
			} else {
				multiplier = 1.0
			}
		default:
			multiplier = 1.0
		}
		return rate * multiplier
	}
}

// DrainFunc computes the amount of vehicles (veh) to remove from a boundary
// departure link in one simulation step.
// The return value is capped at the current queue length before subtraction.
// If nil, the optimizer zeros boundary queues each step (auto drain).
type DrainFunc func(linkID gmns.LinkID, queue float64, deltaT float64) float64

// NoDrain returns a DrainFunc that removes nothing. Use it to model a closed
// network where vehicles accumulate at exits rather than leaving.
func NoDrain() DrainFunc {
	return func(_ gmns.LinkID, _ float64, _ float64) float64 {
		return 0
	}
}

// RateDrain returns a DrainFunc that removes vehicles at the given rate (veh/h)
// per link. Links not present in rates are drained fully (auto behaviour).
func RateDrain(rates map[gmns.LinkID]float64) DrainFunc {
	return func(linkID gmns.LinkID, queue float64, deltaT float64) float64 {
		rate, ok := rates[linkID]
		if !ok || rate <= 0 {
			return queue // drain all for unspecified links
		}
		return rate * deltaT / 3600.0
	}
}

// MPConfig holds configuration for the MP optimizer.
type MPConfig struct {
	DeltaT    float64         // simulation step duration (seconds)
	SimTime   float64         // total simulation time (seconds), 0 = unlimited
	Smoothing SmoothingConfig // Smoothing-MP parameters
	Demand    DemandFunc      // demand provider; nil = no demand
	// Drain controls how boundary departure links are drained each step.
	// nil = zero the queue (auto drain); use NoDrain() or RateDrain() for other behaviours.
	Drain DrainFunc
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
	SelectedStage  StageID
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
		if opt.Config.Drain == nil {
			net.Queues[lid] = 0
			continue
		}
		remove := opt.Config.Drain(lid, net.Queues[lid], dt)
		if remove > net.Queues[lid] {
			remove = net.Queues[lid]
		}
		if remove < 0 {
			remove = 0
		}
		net.Queues[lid] -= remove
	}

	// 3. Select signal groups
	results := make([]StepResult, 0, len(net.Intersections))
	phaseDecisions := make(map[gmns.NodeID]StageID, len(net.Intersections))

	for nid, inter := range net.Intersections {
		var selected StageID
		var pressure float64

		if opt.Config.Smoothing.Alpha > 0 {
			selected, pressure = net.SmoothedSelectPhase(inter, opt.Config.Smoothing)
		} else {
			selected, pressure = net.SelectPhase(inter)
		}

		phaseDecisions[nid] = selected
		results = append(results, StepResult{
			IntersectionID: nid,
			SelectedStage:  selected,
			Pressure:       pressure,
			Boosted:        opt.Config.Smoothing.Alpha > 0,
		})
	}

	// 4. Discharge vehicles through active connectors
	linkDelta := make(map[gmns.LinkID]float64)
	for nid, sgID := range phaseDecisions {
		inter := net.Intersections[nid]
		for i := range inter.Stages {
			if inter.Stages[i].ID != sgID {
				continue
			}
			for _, cid := range inter.Stages[i].ConnectorIDs {
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
		inter.PreviousStage = inter.ActiveStage
		if sgID != inter.ActiveStage {
			inter.ActiveStage = sgID
			inter.ActiveStageSince = opt.time
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

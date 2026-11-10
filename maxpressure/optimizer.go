package maxpressure

import (
	"fmt"
	"math"

	"github.com/LdDl/go-gmns/gmns"
)

// MPConfig defines a discrete simulation. FixedPrograms selects baseline replay.
type MPConfig struct {
	DeltaT        float64
	SimTime       float64
	Smoothing     SmoothingConfig
	Demand        DemandFunc
	Drain         DrainFunc
	FixedPrograms map[gmns.NodeID]*SignalProgram
}

type VehicleBalance struct {
	Initial         float64 `json:"initial_veh"`
	Requested       float64 `json:"requested_veh"`
	Admitted        float64 `json:"admitted_veh"`
	Departed        float64 `json:"departed_veh"`
	Remaining       float64 `json:"remaining_veh"`
	BoundaryBacklog float64 `json:"boundary_backlog_veh"`
	Error           float64 `json:"conservation_error_veh"`
}

type MPOptimizer struct {
	Net     *Network
	Config  MPConfig
	time    float64
	balance VehicleBalance
	err     error
}

// NewMPOptimizer validates the model before any simulation work is performed.
func NewMPOptimizer(net *Network, cfg MPConfig) (*MPOptimizer, error) {
	if !finiteNonnegative(cfg.DeltaT) || cfg.DeltaT == 0 || !finiteNonnegative(cfg.SimTime) {
		return nil, fmt.Errorf("delta_t must be positive and sim_time non-negative")
	}
	if !finiteNonnegative(cfg.Smoothing.Alpha) || cfg.Smoothing.Alpha > 1 {
		return nil, fmt.Errorf("alpha must be between 0 and 1")
	}
	if err := net.prepare(cfg.DeltaT); err != nil {
		return nil, err
	}
	if cfg.FixedPrograms != nil {
		for _, id := range net.nodeIDs {
			p := cfg.FixedPrograms[id]
			if err := p.validate(net.Intersections[id]); err != nil {
				return nil, err
			}
		}
	}
	opt := &MPOptimizer{Net: net, Config: cfg}
	opt.balance.Initial = net.TotalQueueLength() + net.BacklogTotal()
	return opt, nil
}

func (opt *MPOptimizer) BoundaryDepartures() []gmns.LinkID {
	return append([]gmns.LinkID(nil), opt.Net.exitIDs...)
}
func (opt *MPOptimizer) Time() float64 { return opt.time }

// Err reports an invalid value returned by a custom demand or drain provider.
func (opt *MPOptimizer) Err() error { return opt.err }
func (opt *MPOptimizer) Balance() VehicleBalance {
	result := opt.balance
	result.Remaining = opt.Net.TotalQueueLength()
	result.BoundaryBacklog = opt.Net.BacklogTotal()
	result.Error = result.Initial + result.Requested - result.Departed - result.Remaining - result.BoundaryBacklog
	return result
}

type StepResult struct {
	IntersectionID gmns.NodeID
	SelectedStage  StageID
	Pressure       float64
	Boosted        bool
	DurationS      float64
}

func (opt *MPOptimizer) stagePressure(stage *Stage) float64 {
	return opt.Net.SmoothedPhasePressure(stage, opt.Config.Smoothing)
}

func (opt *MPOptimizer) selectStage(inter *IntersectionState, exclude StageID) StageID {
	best, pressure := NoStage, math.Inf(-1)
	for i := range inter.Stages {
		stage := &inter.Stages[i]
		if stage.ID == exclude || len(stage.ConnectorIDs) == 0 {
			continue
		}
		p := opt.stagePressure(stage)
		if p > pressure || (p == pressure && (stage.ID == inter.ActiveStage && inter.started || best != inter.ActiveStage && stage.ID < best)) {
			best, pressure = stage.ID, p
		}
	}
	return best
}

func (opt *MPOptimizer) adaptiveStage(inter *IntersectionState) StageID {
	if inter.started && opt.time < inter.clearanceUntil-1e-9 {
		return NoStage
	}
	if inter.pendingStage != NoStage && inter.started && inter.clearanceUntil > 0 {
		inter.ActiveStage = inter.pendingStage
		inter.ActiveStageSince = opt.time
		inter.pendingStage = NoStage
		inter.clearanceUntil = 0
		return inter.ActiveStage
	}
	best := opt.selectStage(inter, NoStage)
	if !inter.started {
		inter.started = true
		inter.ActiveStage, inter.ActiveStageSince, inter.pendingStage = best, opt.time, NoStage
		return best
	}
	dt := opt.Config.DeltaT
	age := opt.time - inter.ActiveStageSince
	if age+1e-9 < math.Ceil(inter.MinGreenS/dt)*dt {
		return inter.ActiveStage
	}
	maxed := inter.MaxGreenS > 0 && age+1e-9 >= math.Floor(inter.MaxGreenS/dt)*dt
	if maxed {
		best = opt.selectStage(inter, inter.ActiveStage)
	}
	if best == inter.ActiveStage && !maxed {
		return best
	}
	if best == NoStage {
		best = inter.ActiveStage
	}
	clearance := math.Ceil(inter.ClearanceS/dt) * dt
	if maxed && best == inter.ActiveStage {
		clearance = math.Max(clearance, dt)
	}
	if clearance > 0 {
		inter.pendingStage = best
		inter.clearanceUntil = opt.time + clearance
		return NoStage
	}
	inter.ActiveStage, inter.ActiveStageSince = best, opt.time
	return best
}

// Step follows store-and-forward timing: decide from x(t), transfer y(t), then
// admit demand into x(t+1). Arrivals cannot traverse two junctions in one step.
func (opt *MPOptimizer) Step() []StepResult { return opt.step(opt.Config.DeltaT) }

func (opt *MPOptimizer) step(dt float64) []StepResult {
	net := opt.Net
	if opt.err != nil {
		return nil
	}
	demands := make(map[gmns.LinkID]float64)
	for _, id := range net.entryIDs {
		if opt.Config.Demand != nil {
			demands[id] = opt.Config.Demand(id, opt.time) * dt / 3600
		}
		if !finiteNonnegative(demands[id]) {
			opt.err = fmt.Errorf("demand on road %d must be finite and non-negative", id)
			return nil
		}
	}
	results := []StepResult{}
	service := make(map[gmns.LinkID]float64)
	decisions := make(map[gmns.NodeID]StageID)
	for _, id := range net.nodeIDs {
		inter := net.Intersections[id]
		if opt.Config.FixedPrograms != nil {
			amounts, windows := opt.Config.FixedPrograms[id].service(opt.time, dt, id)
			for cid, seconds := range amounts {
				service[cid] = seconds
			}
			results = append(results, windows...)
			decisions[id] = windows[len(windows)-1].SelectedStage
			continue
		}
		selected := opt.adaptiveStage(inter)
		result := StepResult{IntersectionID: id, SelectedStage: selected, DurationS: dt}
		for i := range inter.Stages {
			stage := &inter.Stages[i]
			if stage.ID != selected {
				continue
			}
			result.Pressure = opt.stagePressure(stage)
			for _, cid := range stage.ConnectorIDs {
				service[cid] = dt
				if opt.Config.Smoothing.Alpha > 0 && net.IsUpstreamServed(cid) {
					result.Boosted = true
				}
			}
			break
		}
		results = append(results, result)
		decisions[id] = selected
	}
	net.discharge(service)
	for _, id := range net.exitIDs {
		removed := net.Queues[id]
		if opt.Config.Drain != nil {
			amount := opt.Config.Drain(id, removed, dt)
			if !finiteNonnegative(amount) {
				opt.err = fmt.Errorf("drain on road %d must be finite and non-negative", id)
				return nil
			}
			removed = math.Min(removed, amount)
		}
		net.Queues[id] -= removed
		opt.balance.Departed += removed
	}
	for _, id := range net.entryIDs {
		demand := demands[id]
		opt.balance.Requested += demand
		net.EntryBacklogs[id] += demand
		admitted := net.EntryBacklogs[id]
		if net.FiniteStorage {
			admitted = math.Min(admitted, math.Max(0, net.StorageCapacity(id)-net.Queues[id]))
		}
		net.EntryBacklogs[id] -= admitted
		net.addArrivals(id, admitted)
		opt.balance.Admitted += admitted
	}
	net.syncRoadQueues()
	for _, id := range net.nodeIDs {
		net.Intersections[id].PreviousStage = decisions[id]
		net.Intersections[id].HasPreviousStage = true
	}
	opt.time += dt
	return results
}

// Run includes the final partial step when the horizon is not divisible by dt.
func (opt *MPOptimizer) Run() [][]StepResult {
	results := [][]StepResult{}
	for opt.time < opt.Config.SimTime-1e-9 {
		results = append(results, opt.step(math.Min(opt.Config.DeltaT, opt.Config.SimTime-opt.time)))
		if opt.err != nil {
			break
		}
	}
	return results
}

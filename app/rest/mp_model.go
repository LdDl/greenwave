package rest

import (
	"fmt"
	"math"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/junction"
	"github.com/LdDl/greenwave/maxpressure"
)

func validMPNumber(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validateMPRequest(req *dto.MPRunRequest) error {
	if len(req.Network.Links) == 0 || len(req.Network.Links) > 2000 {
		return fmt.Errorf("network.links must contain 1 to 2000 links")
	}
	if len(req.Intersections) == 0 {
		return fmt.Errorf("intersections must not be empty")
	}
	if req.Config.DeltaT == 0 {
		req.Config.DeltaT = 1
	}
	if !validMPNumber(req.Config.DeltaT) || req.Config.DeltaT < 0.1 || req.Config.DeltaT > 60 {
		return fmt.Errorf("config.delta_t must be between 0.1 and 60 seconds")
	}
	if !validMPNumber(req.Config.SimTime) || req.Config.SimTime <= 0 || req.Config.SimTime > 14400 {
		return fmt.Errorf("config.sim_time must be positive and at most 14400 seconds")
	}
	if math.Ceil(req.Config.SimTime/req.Config.DeltaT)*float64(len(req.Network.Links))*4 > 20000000 {
		return fmt.Errorf("simulation is too large; reduce duration or increase delta_t")
	}
	if !validMPNumber(req.Config.Alpha) || req.Config.Alpha > 1 {
		return fmt.Errorf("config.alpha must be between 0 and 1")
	}
	if req.Config.Alpha > 0 && len(req.CoordinatedCorridors) == 0 {
		return fmt.Errorf("coordinated_corridors is required when config.alpha > 0")
	}
	if req.Config.StorageModel == "" {
		req.Config.StorageModel = "point_queue"
	}
	if req.Config.StorageModel != "point_queue" && req.Config.StorageModel != "finite_storage" {
		return fmt.Errorf("unsupported storage_model")
	}
	links := make(map[int]dto.MPLinkDTO)
	income, outcome := map[int]bool{}, map[int]bool{}
	for _, link := range req.Network.Links {
		if _, ok := links[link.ID]; ok || link.ID < 0 {
			return fmt.Errorf("duplicate or invalid link ID %d", link.ID)
		}
		links[link.ID] = link
		if link.Capacity <= 0 || link.Capacity > 100000 {
			return fmt.Errorf("link %d capacity must be in (0,100000] veh/h", link.ID)
		}
		if link.IsConnection {
			if link.MacroNode == nil || link.MovementMesoLinkIncome == nil || link.MovementMesoLinkOutcome == nil {
				return fmt.Errorf("movement %d needs macro_node and both road references", link.ID)
			}
			income[*link.MovementMesoLinkIncome] = true
			outcome[*link.MovementMesoLinkOutcome] = true
		} else if !validMPNumber(link.LengthMeters) || link.LengthMeters == 0 || link.LengthMeters > 100000 || link.Lanes < 1 || link.Lanes > 20 {
			return fmt.Errorf("road %d needs positive length and 1 to 20 lanes", link.ID)
		}
	}
	for _, link := range req.Network.Links {
		if !link.IsConnection {
			continue
		}
		for _, id := range []int{*link.MovementMesoLinkIncome, *link.MovementMesoLinkOutcome} {
			road, ok := links[id]
			if !ok || road.IsConnection {
				return fmt.Errorf("movement %d references missing road %d", link.ID, id)
			}
		}
	}
	nodes := map[int]bool{}
	for _, inter := range req.Intersections {
		if inter.MacroNodeID < 0 || nodes[inter.MacroNodeID] {
			return fmt.Errorf("duplicate or invalid intersection ID")
		}
		nodes[inter.MacroNodeID] = true
		if len(inter.Junction.Cycle) == 0 {
			return fmt.Errorf("intersection %d must have at least one phase", inter.MacroNodeID)
		}
		if len(inter.Junction.Cycle) > 128 {
			return fmt.Errorf("intersection %d has too many phases", inter.MacroNodeID)
		}
		cycle := 0
		for _, phase := range inter.Junction.Cycle {
			if len(phase.SignalGroups) > 128 {
				return fmt.Errorf("too many signal groups")
			}
			for gi, group := range phase.SignalGroups {
				total := 0
				if len(group.Signals) > 128 {
					return fmt.Errorf("too many signals")
				}
				for _, signal := range group.Signals {
					if signal.Duration < 0 || signal.Duration > 3600 {
						return fmt.Errorf("signal duration must be between 0 and 3600 seconds")
					}
					total += signal.Duration
					switch strings.ToUpper(signal.Color) {
					case "GREEN", "GREENPRIORITY", "RED", "YELLOW", "REDYELLOW":
					default:
						return fmt.Errorf("unsupported signal color %q", signal.Color)
					}
				}
				if gi == 0 {
					cycle += total
				}
			}
		}
		if cycle <= 0 || cycle > 3600 {
			return fmt.Errorf("cycle must be between 1 and 3600 seconds")
		}
	}
	for _, link := range req.Network.Links {
		if link.IsConnection && !nodes[*link.MacroNode] {
			return fmt.Errorf("movement %d belongs to an unknown intersection", link.ID)
		}
	}
	for _, rates := range []map[int]float64{req.Demand.Rates, req.Demand.BaseRates} {
		for id, rate := range rates {
			if !income[id] || outcome[id] || !validMPNumber(rate) || rate > 100000 {
				return fmt.Errorf("demand %d must be an entry-road rate between 0 and 100000 veh/h", id)
			}
		}
	}
	for id, rate := range req.Drain.Rates {
		if !outcome[id] || income[id] || !validMPNumber(rate) || rate > 100000 {
			return fmt.Errorf("drain %d must be an exit-road rate between 0 and 100000 veh/h", id)
		}
	}
	for _, value := range []float64{req.Demand.PeakFactor, req.Demand.RampUpS, req.Demand.PeakDurationS, req.Demand.RampDownS} {
		if !validMPNumber(value) {
			return fmt.Errorf("invalid peak demand profile")
		}
	}
	if req.Demand.PeakFactor > 10 {
		return fmt.Errorf("peak_factor must be at most 10")
	}
	for _, queues := range []map[int]float64{req.InitialQueues, req.InitialMovementQueues, req.InitialBacklogs} {
		for id, vehicles := range queues {
			if !validMPNumber(vehicles) || vehicles > 1e9 {
				return fmt.Errorf("initial queue %d must be between 0 and 1000000000 vehicles", id)
			}
		}
	}
	return nil
}

// MP defaults are applied only when bounds were omitted from this API request.
// Explicit equal bounds remain fixed, unlike the old proposal heuristic.
func mpJunction(input dto.JunctionDTO) *junction.Junction {
	jun := dto.JunctionFromDTO(input)
	cycle := jun.GetTotalDuration()
	for pi, phase := range jun.Cycle {
		for gi, group := range phase.SignalGroups {
			for si, signal := range group.Signals {
				original := input.Cycle[pi].SignalGroups[gi].Signals[si]
				green := strings.ToUpper(original.Color) == "GREEN" || strings.ToUpper(original.Color) == "GREENPRIORITY"
				if !green && strings.ToUpper(original.Color) != "RED" {
					continue
				}
				if original.MinDuration == nil {
					signal.MinDuration = 0
					if green {
						signal.MinDuration = min(5, signal.Duration)
					}
				}
				if original.MaxDuration == nil {
					signal.MaxDuration = cycle
				}
			}
		}
	}
	return jun
}

func buildMPRun(req dto.MPRunRequest, fixed bool, alpha float64) (*maxpressure.MPOptimizer, map[gmns.NodeID]*junction.Junction, error) {
	net := maxpressure.NewNetwork(dto.MesoNetFromDTO(req.Network))
	net.FiniteStorage = req.Config.StorageModel == "finite_storage"
	for id, q := range req.InitialQueues {
		net.Queues[gmns.LinkID(id)] = q
	}
	for id, q := range req.InitialMovementQueues {
		net.MovementQueues[gmns.LinkID(id)] = q
	}
	for id, q := range req.InitialBacklogs {
		net.EntryBacklogs[gmns.LinkID(id)] = q
	}
	for id, r := range req.TurningRatios {
		net.TurningRatios[gmns.LinkID(id)] = r
	}
	programs := make(map[gmns.NodeID]*maxpressure.SignalProgram)
	junctions := make(map[gmns.NodeID]*junction.Junction)
	for _, input := range req.Intersections {
		jun := mpJunction(input.Junction)
		mapping := make(map[junction.GroupID][]gmns.LinkID)
		for group, ids := range input.GroupConnectors {
			for _, id := range ids {
				mapping[junction.GroupID(group)] = append(mapping[junction.GroupID(group)], gmns.LinkID(id))
			}
		}
		program, err := maxpressure.CompileProgram(jun, mapping)
		if err != nil {
			return nil, nil, fmt.Errorf("intersection %d: %w", input.MacroNodeID, err)
		}
		node := gmns.NodeID(input.MacroNodeID)
		minGreen, maxGreen, clearance := 5.0, 60.0, 5.0
		if input.MinGreenS != nil {
			minGreen = *input.MinGreenS
		}
		if input.MaxGreenS != nil {
			maxGreen = *input.MaxGreenS
		}
		if input.ClearanceS != nil {
			clearance = *input.ClearanceS
		}
		net.Intersections[node] = &maxpressure.IntersectionState{MacroNodeID: node, Stages: program.Stages, MinGreenS: minGreen, MaxGreenS: maxGreen, ClearanceS: clearance}
		programs[node], junctions[node] = program, jun
	}
	paths := make([][]gmns.LinkID, len(req.CoordinatedCorridors))
	for i, path := range req.CoordinatedCorridors {
		for _, id := range path {
			paths[i] = append(paths[i], gmns.LinkID(id))
		}
	}
	if err := net.SetCoordinatedCorridors(paths); err != nil {
		return nil, nil, err
	}
	cfg := maxpressure.MPConfig{DeltaT: req.Config.DeltaT, SimTime: req.Config.SimTime, Smoothing: maxpressure.SmoothingConfig{Alpha: alpha}}
	rates := make(map[gmns.LinkID]float64)
	switch strings.ToLower(req.Demand.Type) {
	case "", "constant":
		for id, rate := range req.Demand.Rates {
			rates[gmns.LinkID(id)] = rate
		}
		cfg.Demand = maxpressure.ConstantDemand(rates)
	case "peak":
		for id, rate := range req.Demand.BaseRates {
			rates[gmns.LinkID(id)] = rate
		}
		cfg.Demand = maxpressure.PeakDemand(maxpressure.PeakDemandConfig{BaseRates: rates, PeakFactor: req.Demand.PeakFactor, RampUpS: req.Demand.RampUpS, PeakDurationS: req.Demand.PeakDurationS, RampDownS: req.Demand.RampDownS})
	default:
		return nil, nil, fmt.Errorf("unsupported demand type")
	}
	switch strings.ToLower(req.Drain.Type) {
	case "", "auto":
	case "none":
		cfg.Drain = maxpressure.NoDrain()
	case "rate":
		drains := make(map[gmns.LinkID]float64)
		for id, rate := range req.Drain.Rates {
			drains[gmns.LinkID(id)] = rate
		}
		cfg.Drain = maxpressure.RateDrain(drains)
	default:
		return nil, nil, fmt.Errorf("unsupported drain type")
	}
	if fixed {
		cfg.FixedPrograms = programs
	}
	opt, err := maxpressure.NewMPOptimizer(net, cfg)
	return opt, junctions, err
}

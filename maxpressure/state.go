package maxpressure

import (
	"fmt"
	"math"
	"slices"

	"github.com/LdDl/go-gmns/gmns"
)

func finiteNonnegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

// prepare validates and initializes movement state before the first step.
// Initial road totals are split once using turning proportions. Subsequent
// transfers always debit an individual movement queue, including on red turns.
func (net *Network) prepare(dt float64) error {
	if net == nil || net.Meso == nil || len(net.Meso.Links) == 0 {
		return fmt.Errorf("network must contain road links and movements")
	}
	net.outgoing = make(map[gmns.LinkID][]gmns.LinkID)
	net.roadIDs, net.movementIDs, net.entryIDs, net.exitIDs, net.nodeIDs = nil, nil, nil, nil, nil
	incoming := make(map[gmns.LinkID]bool)
	for id, link := range net.Meso.Links {
		if id < 0 || link == nil || link.ID != id {
			return fmt.Errorf("invalid link %d", id)
		}
		if !link.IsConnection() {
			net.roadIDs = append(net.roadIDs, id)
			if net.FiniteStorage && net.StorageCapacity(id) <= 0 {
				return fmt.Errorf("road %d needs positive length and lanes for finite storage", id)
			}
			continue
		}
		up, down := link.MovementMesoLinkIncome(), link.MovementMesoLinkOutcome()
		for _, road := range []gmns.LinkID{up, down} {
			segment, ok := net.Meso.Links[road]
			if !ok || segment == nil || segment.IsConnection() {
				return fmt.Errorf("movement %d references invalid road %d", id, road)
			}
		}
		if up == down {
			return fmt.Errorf("movement %d must join different directed roads", id)
		}
		net.outgoing[up] = append(net.outgoing[up], id)
		incoming[down] = true
		net.movementIDs = append(net.movementIDs, id)
	}
	slices.Sort(net.roadIDs)
	slices.Sort(net.movementIDs)
	assigned := make(map[gmns.LinkID]bool)
	for id, inter := range net.Intersections {
		if inter == nil || inter.MacroNodeID != id || len(inter.Stages) == 0 {
			return fmt.Errorf("intersection %d needs at least one valid stage", id)
		}
		net.nodeIDs = append(net.nodeIDs, id)
		seen := make(map[StageID]bool)
		for _, stage := range inter.Stages {
			if stage.ID < 0 || seen[stage.ID] {
				return fmt.Errorf("intersection %d has an invalid or duplicate stage ID", id)
			}
			seen[stage.ID] = true
			for _, cid := range stage.ConnectorIDs {
				link, ok := net.Meso.Links[cid]
				if !ok || !link.IsConnection() || link.MacroNode() != id {
					return fmt.Errorf("intersection %d has an invalid movement %d", id, cid)
				}
				assigned[cid] = true
			}
		}
		for _, value := range []float64{inter.MinGreenS, inter.MaxGreenS, inter.ClearanceS} {
			if !finiteNonnegative(value) {
				return fmt.Errorf("intersection %d has invalid timing constraints", id)
			}
		}
		if inter.MaxGreenS > 0 && math.Floor(inter.MaxGreenS/dt) < math.Max(1, math.Ceil(inter.MinGreenS/dt)) {
			return fmt.Errorf("intersection %d green limits cannot fit the simulation step", id)
		}
	}
	slices.Sort(net.nodeIDs)
	for _, id := range net.movementIDs {
		if !assigned[id] {
			return fmt.Errorf("movement %d is not assigned to a signal stage", id)
		}
	}
	for id, value := range net.TurningRatios {
		link, ok := net.Meso.Links[id]
		if !ok || !link.IsConnection() || !finiteNonnegative(value) || value > 1 {
			return fmt.Errorf("invalid turning proportion for movement %d", id)
		}
	}
	for _, road := range net.roadIDs {
		ids := net.outgoing[road]
		slices.Sort(ids)
		if len(ids) == 0 {
			if incoming[road] {
				net.exitIDs = append(net.exitIDs, road)
			}
			continue
		}
		if !incoming[road] {
			net.entryIDs = append(net.entryIDs, road)
		}
		if len(ids) == 1 {
			if _, supplied := net.TurningRatios[ids[0]]; !supplied {
				net.TurningRatios[ids[0]] = 1
			}
		}
		sum := 0.0
		for _, id := range ids {
			ratio, ok := net.TurningRatios[id]
			if !ok {
				return fmt.Errorf("road %d needs explicit turning proportions for every outgoing movement", road)
			}
			sum += ratio
		}
		if math.Abs(sum-1) > 1e-9 {
			return fmt.Errorf("turning proportions on road %d must sum to 1", road)
		}
		// Remove harmless decimal rounding so every arrival is fully assigned.
		for _, id := range ids {
			net.TurningRatios[id] /= sum
		}
	}
	for id, value := range net.MovementQueues {
		link, ok := net.Meso.Links[id]
		if !ok || !link.IsConnection() || !finiteNonnegative(value) {
			return fmt.Errorf("invalid initial movement queue %d", id)
		}
	}
	for id, value := range net.Queues {
		link, ok := net.Meso.Links[id]
		if !ok || link.IsConnection() || !finiteNonnegative(value) {
			return fmt.Errorf("invalid initial road queue %d", id)
		}
		if net.initialized {
			continue
		}
		if value == 0 || len(net.outgoing[id]) == 0 {
			continue
		}
		for _, cid := range net.outgoing[id] {
			if _, supplied := net.MovementQueues[cid]; supplied {
				return fmt.Errorf("road %d has both road and movement initial queues", id)
			}
		}
		net.addArrivals(id, value)
	}
	net.syncRoadQueues()
	for _, road := range net.roadIDs {
		if net.FiniteStorage && net.Queues[road] > net.StorageCapacity(road)+1e-9 {
			return fmt.Errorf("initial queue on road %d exceeds storage", road)
		}
	}
	for id, value := range net.EntryBacklogs {
		if !slices.Contains(net.entryIDs, id) || !finiteNonnegative(value) {
			return fmt.Errorf("invalid boundary backlog %d", id)
		}
	}
	net.stepSeconds = dt
	net.initialized = true
	return nil
}

func (net *Network) addArrivals(road gmns.LinkID, vehicles float64) {
	ids := net.outgoing[road]
	if len(ids) == 0 {
		net.Queues[road] += vehicles
		return
	}
	for _, id := range ids {
		net.MovementQueues[id] += vehicles * net.TurningRatios[id]
	}
}

func (net *Network) syncRoadQueues() {
	for _, road := range net.roadIDs {
		if len(net.outgoing[road]) == 0 {
			continue
		}
		net.Queues[road] = 0
		for _, id := range net.outgoing[road] {
			net.Queues[road] += net.MovementQueues[id]
		}
	}
}

func (net *Network) BacklogTotal() float64 {
	total := 0.0
	for _, id := range net.entryIDs {
		total += net.EntryBacklogs[id]
	}
	return total
}

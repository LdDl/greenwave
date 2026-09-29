package maxpressure

import (
	"github.com/LdDl/go-gmns/gmns"
	"math"
)

// discharge debits each movement's own queue, then splits arrivals by the
// destination road's turning proportions, including its red movements.
// All transfers use the same snapshot, as in Xu (1)-(3).
func (net *Network) discharge(greenSeconds map[gmns.LinkID]float64) {
	flows := make(map[gmns.LinkID]float64)
	incoming := make(map[gmns.LinkID]float64)
	for _, id := range net.movementIDs {
		flow := math.Min(net.MovementQueues[id], net.SatFlow(id)*greenSeconds[id]/3600)
		if flow <= 0 {
			continue
		}
		flows[id] = flow
		incoming[net.Meso.Links[id].MovementMesoLinkOutcome()] += flow
	}
	arrivals := make(map[gmns.LinkID]float64)
	for _, id := range net.movementIDs {
		flow := flows[id]
		if flow <= 0 {
			continue
		}
		down := net.Meso.Links[id].MovementMesoLinkOutcome()
		if net.FiniteStorage {
			free := math.Max(0, net.StorageCapacity(down)-net.Queues[down])
			flow *= math.Min(1, free/incoming[down])
		}
		net.MovementQueues[id] = math.Max(0, net.MovementQueues[id]-flow)
		arrivals[down] += flow
	}
	for _, road := range net.roadIDs {
		net.addArrivals(road, arrivals[road])
	}
	net.syncRoadQueues()
}

package maxpressure

import (
	"math"
	"slices"

	"github.com/LdDl/go-gmns/gmns"
)

type movementFlow struct {
	upstream   gmns.LinkID
	downstream gmns.LinkID
	vehicles   float64
}

// dischargeDeltas allocates each queue among active movements in proportion to
// saturation flow, then limits merging flows to the destination's free space.
// Both limits use queues before transfers: arrivals cannot depart in this step,
// and space released by departures becomes available on the next step.
func (net *Network) dischargeDeltas(decisions map[gmns.NodeID]StageID, dt float64) map[gmns.LinkID]float64 {
	active := make(map[gmns.LinkID]bool)
	for nid, selected := range decisions {
		for _, stage := range net.Intersections[nid].Stages {
			if stage.ID != selected {
				continue
			}
			for _, cid := range stage.ConnectorIDs {
				active[cid] = true
			}
			break
		}
	}

	// Stable connector order makes allocation independent of Go map iteration.
	connectorIDs := make([]gmns.LinkID, 0, len(active))
	for cid := range active {
		connectorIDs = append(connectorIDs, cid)
	}
	slices.Sort(connectorIDs)

	flows := make([]movementFlow, 0, len(connectorIDs))
	outgoing := make(map[gmns.LinkID]float64)
	for _, cid := range connectorIDs {
		link, ok := net.Meso.Links[cid]
		if !ok || !link.IsConnection() {
			continue
		}
		upID := link.MovementMesoLinkIncome()
		downID := link.MovementMesoLinkOutcome()
		up, upOK := net.Meso.Links[upID]
		down, downOK := net.Meso.Links[downID]
		if !upOK || !downOK || up.IsConnection() || down.IsConnection() {
			continue
		}
		vehicles := net.SatFlow(cid) * dt / 3600.0
		if vehicles <= 0 || net.Queues[upID] <= 0 {
			continue
		}
		flows = append(flows, movementFlow{upstream: upID, downstream: downID, vehicles: vehicles})
		outgoing[upID] += vehicles
	}

	incoming := make(map[gmns.LinkID]float64)
	for i := range flows {
		flow := &flows[i]
		scale := math.Min(1, net.Queues[flow.upstream]/outgoing[flow.upstream])
		flow.vehicles *= scale
		incoming[flow.downstream] += flow.vehicles
	}

	deltas := make(map[gmns.LinkID]float64)
	for _, flow := range flows {
		if capacity := net.StorageCapacity(flow.downstream); capacity > 0 {
			free := math.Max(0, capacity-net.Queues[flow.downstream])
			flow.vehicles *= math.Min(1, free/incoming[flow.downstream])
		}
		// Rejected flow stays upstream; it is not reassigned to another turn.
		deltas[flow.upstream] -= flow.vehicles
		deltas[flow.downstream] += flow.vehicles
	}
	return deltas
}

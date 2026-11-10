package maxpressure

import (
	"math"

	"github.com/LdDl/go-gmns/gmns"
)

// MovementWeight returns Q_ij * (x_ij - sum_k r_jk*x_jk), using Xu (12).
// Q is service capacity per simulation step, not a rate in vehicles/hour.
func (net *Network) MovementWeight(connectorID gmns.LinkID) float64 {
	link, ok := net.Meso.Links[connectorID]
	if !ok || !link.IsConnection() {
		return 0
	}
	upID := link.MovementMesoLinkIncome()
	downID := link.MovementMesoLinkOutcome()
	if upID < 0 || downID < 0 {
		return 0
	}
	downstream := 0.0
	for _, id := range net.outgoing[downID] {
		downstream += net.TurningRatios[id] * net.MovementQueues[id]
	}
	return net.ServiceCapacity(connectorID) * (net.MovementQueues[connectorID] - downstream)
}

func (net *Network) ServiceCapacity(connectorID gmns.LinkID) float64 {
	return net.SatFlow(connectorID) * net.stepSeconds / 3600
}

// PhasePressure computes the total pressure for a phase at an intersection:
//
//	W(p) = sum_{connector in p} w_{connector}
func (net *Network) PhasePressure(phase *Stage) float64 {
	total := 0.0
	seen := make(map[gmns.LinkID]bool)
	for _, cid := range phase.ConnectorIDs {
		if seen[cid] {
			continue
		}
		seen[cid] = true
		total += net.MovementWeight(cid)
	}
	return total
}

// PhasePressures computes pressures for all phases of an intersection.
func (net *Network) PhasePressures(inter *IntersectionState) map[StageID]float64 {
	result := make(map[StageID]float64, len(inter.Stages))
	for i := range inter.Stages {
		result[inter.Stages[i].ID] = net.PhasePressure(&inter.Stages[i])
	}
	return result
}

// SelectPhase returns the phase with maximum pressure.
// Ties are broken by lower StageID.
func (net *Network) SelectPhase(inter *IntersectionState) (StageID, float64) {
	if len(inter.Stages) == 0 {
		return NoStage, 0
	}
	bestPhase := inter.Stages[0].ID
	bestPressure := math.Inf(-1)

	for i := range inter.Stages {
		p := net.PhasePressure(&inter.Stages[i])
		if p > bestPressure || (p == bestPressure && inter.Stages[i].ID < bestPhase) {
			bestPressure = p
			bestPhase = inter.Stages[i].ID
		}
	}
	return bestPhase, bestPressure
}

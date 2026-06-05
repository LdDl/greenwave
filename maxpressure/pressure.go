package maxpressure

import (
	"math"

	"github.com/LdDl/go-gmns/gmns"
)

// MovementWeight computes the pressure weight for a single connector link (movement):
//
//	w_{u,d} = S_{u,d} * (x_u/K_u - x_d/K_d)
//
// connectorID must be a meso connector link (IsConnection == true).
func (net *Network) MovementWeight(connectorID gmns.LinkID) float64 {
	link, ok := net.Meso.Links[connectorID]
	if !ok {
		return 0
	}
	upID := link.MovementMesoLinkIncome()
	downID := link.MovementMesoLinkOutcome()
	if upID < 0 || downID < 0 {
		return 0
	}
	upOcc := net.NormalizedOccupancy(upID)
	downOcc := net.NormalizedOccupancy(downID)
	return net.SatFlow(connectorID) * (upOcc - downOcc)
}

// PhasePressure computes the total pressure for a phase at an intersection:
//
//	W(p) = sum_{connector in p} w_{connector}
func (net *Network) PhasePressure(phase *Stage) float64 {
	total := 0.0
	for _, cid := range phase.ConnectorIDs {
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

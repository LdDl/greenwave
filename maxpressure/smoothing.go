package maxpressure

import (
	"math"

	"github.com/LdDl/go-gmns/gmns"
)

// SmoothingConfig holds parameters for an experimental coordination boost
// inspired by Smoothing-MP (Xu et al., 2024).
//
// The boost for a corridor movement (u,d) whose configured predecessor had
// green during the last completed step is:
//
//	xi_{u,d} = Alpha * Q_{u,d}
//
// Here Q_{u,d} is the connector's saturation flow in veh/h. This scaling of
// Alpha is a project convention, not the paper's per-step capacity Q.
// Alpha = 0 reduces to standard max-pressure.
// The paper's stability guarantee has not been established for this normalized,
// finite-storage model without explicit turning proportions.
type SmoothingConfig struct {
	// Alpha scales the experimental coordination bonus; zero disables it.
	Alpha float64
}

// DefaultSmoothingConfig returns a default configuration with Alpha=1.0.
func DefaultSmoothingConfig() SmoothingConfig {
	return SmoothingConfig{Alpha: 1.0}
}

// ServedLinks returns roads with an actuated incoming movement in the last
// completed step. A green signal does not imply positive discharged flow.
func (net *Network) ServedLinks(inter *IntersectionState) map[gmns.LinkID]bool {
	served := make(map[gmns.LinkID]bool)
	if !inter.HasPreviousStage {
		return served
	}
	for i := range inter.Stages {
		if inter.Stages[i].ID != inter.PreviousStage {
			continue
		}
		for _, cid := range inter.Stages[i].ConnectorIDs {
			if link, ok := net.Meso.Links[cid]; ok {
				if downID := link.MovementMesoLinkOutcome(); downID >= 0 {
					served[downID] = true
				}
			}
		}
		break
	}
	return served
}

// IsUpstreamServed implements the corridor indicator from Xu et al. (2024),
// section 3.5: c_jk(t+1) = s_ij(t). It tests actuation, not actual discharge.
// With overlapping corridors, any actuated configured predecessor sets c=1.
func (net *Network) IsUpstreamServed(connectorID gmns.LinkID) bool {
	for _, predecessor := range net.coordinationPredecessors[connectorID] {
		if net.wasActuated(predecessor) {
			return true
		}
	}
	return false
}

// SmoothedMovementWeight computes the prototype's candidate-stage bonus.
// See PAPER_MODEL.md for the ambiguity in the printed objective (13).
//
//	w_smooth = Q * w + xi * c
//
// where c = 1 if upstream served, 0 otherwise, and xi = Alpha * Q.
// Since MovementWeight already returns Q * w (satflow × normalized pressure),
// the boost simplifies to:
//
//	if upstream served: w_smooth = w_standard + Alpha * SatFlow
//	otherwise:          w_smooth = w_standard
func (net *Network) SmoothedMovementWeight(connectorID gmns.LinkID, cfg SmoothingConfig) float64 {
	w := net.MovementWeight(connectorID)
	if cfg.Alpha <= 0 || !net.IsUpstreamServed(connectorID) {
		return w
	}
	return w + cfg.Alpha*net.SatFlow(connectorID)
}

// SmoothedPhasePressure computes pressure for a phase with Smoothing-MP boost.
func (net *Network) SmoothedPhasePressure(phase *Stage, cfg SmoothingConfig) float64 {
	total := 0.0
	for _, cid := range phase.ConnectorIDs {
		total += net.SmoothedMovementWeight(cid, cfg)
	}
	return total
}

// SmoothedSelectPhase returns the phase with maximum smoothed pressure.
func (net *Network) SmoothedSelectPhase(inter *IntersectionState, cfg SmoothingConfig) (StageID, float64) {
	bestPhase := inter.Stages[0].ID
	bestPressure := math.Inf(-1)
	for i := range inter.Stages {
		p := net.SmoothedPhasePressure(&inter.Stages[i], cfg)
		if p > bestPressure || (p == bestPressure && inter.Stages[i].ID < bestPhase) {
			bestPressure = p
			bestPhase = inter.Stages[i].ID
		}
	}
	return bestPhase, bestPressure
}

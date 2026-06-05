package maxpressure

import (
	"math"

	"github.com/LdDl/go-gmns/gmns"
)

// SmoothingConfig holds parameters for the Smoothing-MP coordination boost
// from Xu et al. (2024).
//
// The boost for a connector (u,d) when the upstream intersection just served
// a movement into link u is:
//
//	xi_{u,d} = Alpha * Q_{u,d}
//
// where Q_{u,d} is the saturation flow (capacity) of the connector.
// Alpha = 0 reduces to standard max-pressure.
// Stability is preserved when xi_{u,d} <= Q_{u,d}^2 (i.e. Alpha <= Q_{u,d}).
type SmoothingConfig struct {
	Alpha float64 // dimensionless coordination coefficient (>= 0)
}

// DefaultSmoothingConfig returns a default configuration with Alpha=1.0.
func DefaultSmoothingConfig() SmoothingConfig {
	return SmoothingConfig{Alpha: 1.0}
}

// ServedLinks returns downstream road link IDs that were served by the
// given intersection in its previous phase.
func (net *Network) ServedLinks(inter *IntersectionState) map[gmns.LinkID]bool {
	served := make(map[gmns.LinkID]bool)
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

// IsUpstreamServed checks whether the upstream road link of a connector was
// served by its upstream intersection in the previous step.
//
// Logic: for connector C at intersection J with upstream road link U,
// scan all connectors in meso.Net that discharge into U. If any of them
// belongs to a different intersection whose PreviousStage matches, return true.
func (net *Network) IsUpstreamServed(connectorID gmns.LinkID) bool {
	link, ok := net.Meso.Links[connectorID]
	if !ok {
		return false
	}
	upRoadID := link.MovementMesoLinkIncome()
	if upRoadID < 0 {
		return false
	}
	myMacroNode := link.MacroNode()

	// Find any connector at a DIFFERENT intersection that discharges into upRoadID
	for _, otherLink := range net.Meso.Links {
		if !otherLink.IsConnection() {
			continue
		}
		if otherLink.MacroNode() == myMacroNode {
			continue // same intersection
		}
		if otherLink.MovementMesoLinkOutcome() != upRoadID {
			continue
		}
		// Found a connector at upstream intersection that feeds into our approach.
		// Check if that intersection's previous phase included it.
		upInter, ok := net.Intersections[otherLink.MacroNode()]
		if !ok {
			continue
		}
		for i := range upInter.Stages {
			if upInter.Stages[i].ID != upInter.PreviousStage {
				continue
			}
			for _, pid := range upInter.Stages[i].ConnectorIDs {
				if pid == otherLink.ID {
					return true
				}
			}
			break
		}
	}
	return false
}

// SmoothedMovementWeight computes the Smoothing-MP weight for a connector
// following Xu et al. (2024), eq. 13:
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

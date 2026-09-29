package maxpressure

import (
	"fmt"
	"slices"

	"github.com/LdDl/go-gmns/gmns"
)

type roadPair struct {
	from gmns.LinkID
	to   gmns.LinkID
}

// SetCoordinatedCorridors configures directed road-link paths, not connector IDs.
// Xu et al. (2024), section 3.5, restricts c_jk(t+1) = s_ij(t) to consecutive
// movements (i,j), (j,k) in an explicitly coordinated corridor.
// An empty list disables coordination. Invalid paths leave the old setup intact.
// Call again after changing topology or stage assignments.
func (net *Network) SetCoordinatedCorridors(corridors [][]gmns.LinkID) error {
	byRoads := make(map[roadPair][]gmns.LinkID)
	for _, inter := range net.Intersections {
		for _, stage := range inter.Stages {
			for _, cid := range stage.ConnectorIDs {
				link, ok := net.Meso.Links[cid]
				if !ok || !link.IsConnection() || link.MacroNode() != inter.MacroNodeID {
					continue
				}
				pair := roadPair{from: link.MovementMesoLinkIncome(), to: link.MovementMesoLinkOutcome()}
				if !slices.Contains(byRoads[pair], cid) {
					byRoads[pair] = append(byRoads[pair], cid)
				}
			}
		}
	}

	predecessors := make(map[gmns.LinkID][]gmns.LinkID)
	for index, path := range corridors {
		if len(path) < 3 {
			return fmt.Errorf("coordinated_corridors[%d] must contain at least 3 directed road links", index)
		}
		for _, id := range path {
			link, ok := net.Meso.Links[id]
			if !ok || link.IsConnection() {
				return fmt.Errorf("coordinated_corridors[%d]: road link %d does not exist", index, id)
			}
		}
		for i := 0; i+1 < len(path); i++ {
			if len(byRoads[roadPair{from: path[i], to: path[i+1]}]) == 0 {
				return fmt.Errorf("coordinated_corridors[%d]: no signal-controlled movement from road %d to %d", index, path[i], path[i+1])
			}
		}
		for i := 0; i+2 < len(path); i++ {
			upstream := byRoads[roadPair{from: path[i], to: path[i+1]}]
			downstream := byRoads[roadPair{from: path[i+1], to: path[i+2]}]
			for _, down := range downstream {
				for _, up := range upstream {
					if net.Meso.Links[up].MacroNode() == net.Meso.Links[down].MacroNode() {
						return fmt.Errorf("coordinated_corridors[%d]: consecutive movements must belong to different intersections", index)
					}
					if !slices.Contains(predecessors[down], up) {
						predecessors[down] = append(predecessors[down], up)
					}
				}
			}
		}
	}
	for _, ids := range predecessors {
		slices.Sort(ids)
	}
	net.coordinationPredecessors = predecessors
	return nil
}

func (net *Network) wasActuated(connectorID gmns.LinkID) bool {
	link, ok := net.Meso.Links[connectorID]
	if !ok || !link.IsConnection() {
		return false
	}
	inter, ok := net.Intersections[link.MacroNode()]
	if !ok || !inter.HasPreviousStage {
		return false
	}
	for _, stage := range inter.Stages {
		if stage.ID == inter.PreviousStage {
			return slices.Contains(stage.ConnectorIDs, connectorID)
		}
	}
	return false
}

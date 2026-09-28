package rest

import (
	"fmt"

	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/junction"
)

// Resolve both selections before constructing junctions or calculating waves.
func coordinationGroups(junctions []dto.JunctionDTO, forward, reverse map[int]junction.GroupID) (map[int]junction.GroupID, map[int]junction.GroupID, error) {
	forwardGroups := make(map[int]junction.GroupID, len(junctions))
	reverseGroups := make(map[int]junction.GroupID, len(junctions))
	for _, jun := range junctions {
		forwardID := forward[jun.ID]
		reverseID, ok := reverse[jun.ID]
		if !ok {
			reverseID = forwardID
		}
		if len(jun.Cycle) == 0 {
			return nil, nil, fmt.Errorf("junction %d: a program needs phases", jun.ID)
		}
		for _, phase := range jun.Cycle {
			foundForward, foundReverse := false, false
			for _, group := range phase.SignalGroups {
				foundForward = foundForward || group.ID == int(forwardID)
				foundReverse = foundReverse || group.ID == int(reverseID)
			}
			if !foundForward || !foundReverse {
				return nil, nil, fmt.Errorf("junction %d, phase %d: selected forward or reverse group does not exist", jun.ID, phase.ID)
			}
		}
		forwardGroups[jun.ID] = forwardID
		reverseGroups[jun.ID] = reverseID
	}
	for _, selections := range []map[int]junction.GroupID{forward, reverse} {
		for id := range selections {
			if _, ok := forwardGroups[id]; !ok {
				return nil, nil, fmt.Errorf("group selection references missing junction %d", id)
			}
		}
	}
	return forwardGroups, reverseGroups, nil
}

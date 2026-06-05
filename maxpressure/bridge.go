package maxpressure

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
)

// StagesFromJunction builds []Stage for a maxpressure.IntersectionState
// from a junction's phase configuration.
//
// groupConnectors maps each signal GroupID to the meso connector link IDs
// that this group controls at this intersection (e.g. northbound through movement
// -> connector link IDs in the meso graph). The caller provides this mapping
// from GMNS graph topology  the bridge does not infer it automatically.
//
// Each junction phase becomes one Stage. A Stage's ConnectorIDs are the union
// of connector IDs for all groups that have at least one GREEN or GREENPRIORITY
// signal in their sequence within that phase.
// Groups with no entry in groupConnectors are silently skipped.
func StagesFromJunction(jun *junction.Junction, groupConnectors map[junction.GroupID][]gmns.LinkID) []Stage {
	stages := make([]Stage, len(jun.Cycle))
	for i, phase := range jun.Cycle {
		var connectorIDs []gmns.LinkID
		for _, sg := range phase.SignalGroups {
			if !groupIsGreen(sg) {
				continue
			}
			connectorIDs = append(connectorIDs, groupConnectors[sg.ID]...)
		}
		stages[i] = Stage{
			ID:           StageID(phase.ID),
			ConnectorIDs: connectorIDs,
		}
	}
	return stages
}

// groupIsGreen reports whether a signal group has at least one GREEN or
// GREENPRIORITY signal in its sequence within a phase.
func groupIsGreen(sg junction.SignalGroup) bool {
	for _, sig := range sg.Signals {
		if sig.Color == color.GREEN || sig.Color == color.GREENPRIORITY {
			return true
		}
	}
	return false
}

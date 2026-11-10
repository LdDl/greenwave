package dto

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/meso"
)

// MesoNetFromDTO builds a meso.Net from the network DTO.
func MesoNetFromDTO(d MPNetworkDTO) *meso.Net {
	mn := meso.NewNet()
	for _, l := range d.Links {
		opts := make([]func(*meso.Link), 0, 6)
		if l.LengthMeters > 0 {
			opts = append(opts, meso.WithLengthMeters(l.LengthMeters))
		}
		if l.Lanes > 0 {
			opts = append(opts, meso.WithLanesNum(l.Lanes))
		}
		if l.Capacity > 0 {
			opts = append(opts, meso.WithCapacity(l.Capacity))
		}
		if l.IsConnection {
			opts = append(opts, meso.WithIsConnection(true))
			opts = append(opts, meso.WithControlType(types.CONTROL_TYPE_IS_SIGNAL))
			if l.MacroNode != nil {
				opts = append(opts, meso.WithLineMacroNodeID(gmns.NodeID(*l.MacroNode)))
			}
			if l.MovementMesoLinkIncome != nil {
				opts = append(opts, meso.WithMovementMesoLinkIncome(gmns.LinkID(*l.MovementMesoLinkIncome)))
			}
			if l.MovementMesoLinkOutcome != nil {
				opts = append(opts, meso.WithMovementMesoLinkOutcome(gmns.LinkID(*l.MovementMesoLinkOutcome)))
			}
		}
		mn.Links[gmns.LinkID(l.ID)] = meso.NewLinkFrom(
			gmns.LinkID(l.ID),
			gmns.NodeID(l.SourceNode),
			gmns.NodeID(l.TargetNode),
			opts...,
		)
	}
	return mn
}

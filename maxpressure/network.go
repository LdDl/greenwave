package maxpressure

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/meso"
)

const (
	DEFAULT_SATURATION_FLOW  = 1800.0 // veh/h fallback
	DEFAULT_VEHICLE_LENGTH   = 7.0    // meters
	VEHICLE_DEFAULT_LENGTH_M = 7.0
)

// SignalGroupID identifies a phase within an intersection.
type SignalGroupID int

// SignalGroup is a set of non-conflicting connector link IDs (movements)
// that can be served simultaneously at an intersection.
type SignalGroup struct {
	ID           SignalGroupID
	ConnectorIDs []gmns.LinkID // meso connector link IDs belonging to this group
}

// IntersectionState holds runtime state for a signalized intersection.
type IntersectionState struct {
	MacroNodeID  gmns.NodeID
	SignalGroups []SignalGroup

	MinGreenS  float64
	MaxGreenS  float64
	ClearanceS float64

	ActivePhase      SignalGroupID
	ActivePhaseSince float64
	PreviousPhase    SignalGroupID
}

// Network wraps a meso.Net and adds queue lengths + phase assignments
// for max-pressure control. The meso graph already contains all topology
// (segments, connectors, upstream/downstream references).
type Network struct {
	Meso           *meso.Net
	Queues         map[gmns.LinkID]float64
	Intersections  map[gmns.NodeID]*IntersectionState
	VehicleLengthM float64
}

// NewNetwork creates a Network from an existing meso.Net.
func NewNetwork(mesoNet *meso.Net) *Network {
	return &Network{
		Meso:           mesoNet,
		Queues:         make(map[gmns.LinkID]float64),
		Intersections:  make(map[gmns.NodeID]*IntersectionState),
		VehicleLengthM: DEFAULT_VEHICLE_LENGTH,
	}
}

// StorageCapacity returns max vehicles for a road link: length * lanes / vehicleLen.
func (net *Network) StorageCapacity(linkID gmns.LinkID) float64 {
	link, ok := net.Meso.Links[linkID]
	if !ok {
		return 0
	}
	length := link.LengthMeters()
	lanes := link.LanesNum()
	if length <= 0 || lanes <= 0 {
		return 0
	}
	vl := net.VehicleLengthM
	if vl <= 0 {
		vl = DEFAULT_VEHICLE_LENGTH
	}
	return (length * float64(lanes)) / vl
}

// NormalizedOccupancy returns queue / capacity for a road link.
func (net *Network) NormalizedOccupancy(linkID gmns.LinkID) float64 {
	k := net.StorageCapacity(linkID)
	if k <= 0 {
		return 0
	}
	return net.Queues[linkID] / k
}

// SatFlow returns the saturation flow for a connector link.
// Priority: connector's Capacity() => upstream link Capacity() => 1800 fallback.
func (net *Network) SatFlow(linkID gmns.LinkID) float64 {
	link, ok := net.Meso.Links[linkID]
	if !ok {
		return 0
	}
	if c := link.Capacity(); c > 0 {
		return float64(c)
	}
	if link.IsConnection() {
		if up, ok2 := net.Meso.Links[link.MovementMesoLinkIncome()]; ok2 {
			if c := up.Capacity(); c > 0 {
				return float64(c)
			}
		}
	}
	return DEFAULT_SATURATION_FLOW
}

// TotalQueueLength returns sum of all queues in the network.
func (net *Network) TotalQueueLength() float64 {
	total := 0.0
	for _, q := range net.Queues {
		total += q
	}
	return total
}

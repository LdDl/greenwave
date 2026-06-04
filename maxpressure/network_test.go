package maxpressure

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/meso"
	"github.com/stretchr/testify/assert"
)

// segLink creates a non-connector meso link (road segment).
func segLink(id gmns.LinkID, src, tgt gmns.NodeID, lengthM float64, lanes int, capacity int) *meso.Link {
	return meso.NewLinkFrom(id, src, tgt,
		meso.WithLengthMeters(lengthM),
		meso.WithLanesNum(lanes),
		meso.WithCapacity(capacity),
	)
}

// connLink creates a connector meso link (movement) at a macro node.
func connLink(id gmns.LinkID, src, tgt gmns.NodeID, macroNode gmns.NodeID, upstreamLink, downstreamLink gmns.LinkID, capacity int) *meso.Link {
	return meso.NewLinkFrom(id, src, tgt,
		meso.WithIsConnection(true),
		meso.WithLineMacroNodeID(macroNode),
		meso.WithMovementMesoLinkIncome(upstreamLink),
		meso.WithMovementMesoLinkOutcome(downstreamLink),
		meso.WithCapacity(capacity),
		meso.WithControlType(types.CONTROL_TYPE_IS_SIGNAL),
	)
}

func TestStorageCapacity(t *testing.T) {
	mn := meso.NewNet()
	mn.Links[1] = segLink(1, 0, 1, 300, 2, 3600)

	net := NewNetwork(mn)
	assert.InDelta(t, 300.0*2.0/7.0, net.StorageCapacity(1), 0.01)
}

func TestNormalizedOccupancy(t *testing.T) {
	mn := meso.NewNet()
	mn.Links[1] = segLink(1, 0, 1, 200, 2, 3600)

	net := NewNetwork(mn)
	net.Queues[1] = 15

	expected := 15.0 / (200.0 * 2.0 / 7.0)
	assert.InDelta(t, expected, net.NormalizedOccupancy(1), 0.001)
}

func TestTotalQueueLength(t *testing.T) {
	mn := meso.NewNet()
	mn.Links[1] = segLink(1, 0, 1, 200, 2, 3600)
	mn.Links[2] = segLink(2, 1, 2, 200, 2, 3600)

	net := NewNetwork(mn)
	net.Queues[1] = 10
	net.Queues[2] = 5

	assert.Equal(t, 15.0, net.TotalQueueLength())
}

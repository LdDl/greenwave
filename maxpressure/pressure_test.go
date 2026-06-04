package maxpressure

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/meso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildIntersectionA builds the worked example from note6_smoothing_mp.tex:
// Intersection A (macroNode=50) with 4 approaches, 8 movements, 2 signal groups.
func buildIntersectionA() *Network {
	mn := meso.NewNet()

	mn.Links[1] = segLink(1, 0, 10, 200, 2, 3600) // WA=>A (EB), K≈57
	mn.Links[2] = segLink(2, 0, 10, 300, 2, 3600)  // B=>A  (WB), K≈86
	mn.Links[3] = segLink(3, 0, 10, 200, 2, 3600)  // NA=>A (SB), K≈57
	mn.Links[4] = segLink(4, 0, 10, 250, 2, 3600)  // C=>A  (NB), K≈71

	mn.Links[5] = segLink(5, 10, 20, 300, 2, 3600) // A=>B  (EB dep), K≈86
	mn.Links[6] = segLink(6, 10, 20, 200, 2, 3600) // A=>WA (WB dep), K≈57
	mn.Links[7] = segLink(7, 10, 20, 200, 2, 3600) // A=>NA (NB dep), K≈57
	mn.Links[8] = segLink(8, 10, 20, 250, 2, 3600) // A=>C  (SB dep), K≈71

	mn.Links[110] = connLink(110, 10, 10, 50, 1, 5, 1800) // EBT
	mn.Links[111] = connLink(111, 10, 10, 50, 1, 7, 1600) // EBR
	mn.Links[112] = connLink(112, 10, 10, 50, 2, 6, 1800) // WBT
	mn.Links[113] = connLink(113, 10, 10, 50, 2, 8, 1400) // WBL

	mn.Links[120] = connLink(120, 10, 10, 50, 3, 8, 1800) // SBT
	mn.Links[121] = connLink(121, 10, 10, 50, 3, 6, 1600) // SBR
	mn.Links[122] = connLink(122, 10, 10, 50, 4, 7, 1800) // NBT
	mn.Links[123] = connLink(123, 10, 10, 50, 4, 5, 1400) // NBL

	net := NewNetwork(mn)
	net.Queues[1] = 15
	net.Queues[2] = 10
	net.Queues[3] = 8
	net.Queues[4] = 6
	net.Queues[5] = 5
	net.Queues[6] = 2
	net.Queues[7] = 4
	net.Queues[8] = 3

	net.Intersections[50] = &IntersectionState{
		MacroNodeID: 50,
		SignalGroups: []SignalGroup{
			{ID: 0, ConnectorIDs: []gmns.LinkID{110, 111, 112, 113}},
			{ID: 1, ConnectorIDs: []gmns.LinkID{120, 121, 122, 123}},
		},
	}

	return net
}

func TestMovementWeight_EBT(t *testing.T) {
	net := buildIntersectionA()
	expected := 1800.0 * (15.0/(200.0*2/7.0) - 5.0/(300.0*2/7.0))
	assert.InDelta(t, expected, net.MovementWeight(110), 0.1)
}

func TestMovementWeight_NBT(t *testing.T) {
	net := buildIntersectionA()
	expected := 1800.0 * (6.0/(250.0*2/7.0) - 4.0/(200.0*2/7.0))
	assert.InDelta(t, expected, net.MovementWeight(122), 0.1)
}

func TestPhasePressure_P0_GreaterThan_P1(t *testing.T) {
	net := buildIntersectionA()
	inter := net.Intersections[50]

	p0 := net.PhasePressure(&inter.SignalGroups[0])
	p1 := net.PhasePressure(&inter.SignalGroups[1])

	assert.Greater(t, p0, p1)
	assert.InDelta(t, 927.0, p0, 100.0, "p0 (EW) should be ~927")
	assert.InDelta(t, 409.0, p1, 100.0, "p1 (NS) should be ~409")
}

func TestSelectPhase_ChoosesP0(t *testing.T) {
	net := buildIntersectionA()
	inter := net.Intersections[50]

	phase, pressure := net.SelectPhase(inter)
	assert.Equal(t, SignalGroupID(0), phase)
	assert.Positive(t, pressure)
}

func TestPhasePressures(t *testing.T) {
	net := buildIntersectionA()
	inter := net.Intersections[50]

	pressures := net.PhasePressures(inter)
	require.Len(t, pressures, 2)
	assert.Greater(t, pressures[0], pressures[1])
}

func TestMovementWeight_MissingLink(t *testing.T) {
	net := NewNetwork(meso.NewNet())
	assert.Equal(t, 0.0, net.MovementWeight(999))
}

func TestSelectPhase_TieBreaksLowerID(t *testing.T) {
	mn := meso.NewNet()
	mn.Links[1] = segLink(1, 0, 10, 200, 2, 3600)
	mn.Links[2] = segLink(2, 0, 10, 200, 2, 3600)
	mn.Links[3] = segLink(3, 10, 20, 200, 2, 3600)
	mn.Links[4] = segLink(4, 10, 20, 200, 2, 3600)
	mn.Links[100] = connLink(100, 10, 10, 50, 1, 3, 1800)
	mn.Links[200] = connLink(200, 10, 10, 50, 2, 4, 1800)

	net := NewNetwork(mn)
	net.Queues[1] = 10
	net.Queues[2] = 10
	net.Queues[3] = 5
	net.Queues[4] = 5

	net.Intersections[50] = &IntersectionState{
		MacroNodeID: 50,
		SignalGroups: []SignalGroup{
			{ID: 0, ConnectorIDs: []gmns.LinkID{100}},
			{ID: 1, ConnectorIDs: []gmns.LinkID{200}},
		},
	}

	phase, _ := net.SelectPhase(net.Intersections[50])
	assert.Equal(t, SignalGroupID(0), phase, "tie should break to lower ID")
}

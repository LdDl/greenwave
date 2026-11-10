package junction

import (
	"testing"

	"github.com/LdDl/greenwave/greeninterval"
	"github.com/stretchr/testify/assert"
)

func TestCycleDurationCorrectness(t *testing.T) {
	junctions := BasicTestJunctions()
	correctDuration := 85
	for i, jun := range junctions {
		assert.Equalf(t, correctDuration, jun.GetTotalDuration(),
			"Junction at position %d has incorrect total duration", i)
	}
}

func TestGetGreenIntervals(t *testing.T) {
	junctions := BasicTestJunctions()
	correctGreenIntervals := [][]*greeninterval.GreenInterval{
		{greeninterval.New(0, 0, 30), greeninterval.New(1, 50, 70)},
		{greeninterval.New(0, 20, 55), greeninterval.New(1, 70, 80)},
		{greeninterval.New(0, 45, 55), greeninterval.New(1, 62, 80)},
		{greeninterval.New(0, 40, 55), greeninterval.New(1, 65, 85)},
	}
	// Pick group for each junction (in this case, we know that all junctions have group #0)
	groups := map[int]GroupID{
		100500: 0,
		42:     0,
		78:     0,
		256:    0,
	}
	for i, jun := range junctions {
		intervals := jun.GetGreenIntervals(groups[jun.ID])
		assert.Equalf(t, len(correctGreenIntervals[i]), len(intervals),
			"Mismatch in green intervals length for junction %d", i)
		for j, gi := range intervals {
			assert.Equalf(t, correctGreenIntervals[i][j].Start, gi.Start, "Mismatch in start time for junction %d interval %d", i, j)
			assert.Equalf(t, correctGreenIntervals[i][j].End, gi.End, "Mismatch in end time for junction %d interval %d", i, j)
			assert.Equalf(t, correctGreenIntervals[i][j].PhaseIdx, gi.PhaseIdx, "Mismatch in phase index for junction %d interval %d", i, j)
		}
	}
}

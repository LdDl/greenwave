package greenwave

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGreenInternvals(t *testing.T) {
	junctions := basicTestJuntions()
	correctGreenIntervals := [][]*GreenInterval{
		{NewGreenInterval(0, 0, 30), NewGreenInterval(1, 50, 70)},
		{NewGreenInterval(0, 20, 55), NewGreenInterval(1, 70, 80)},
		{NewGreenInterval(0, 45, 55), NewGreenInterval(1, 62, 80)},
		{NewGreenInterval(0, 40, 55), NewGreenInterval(1, 65, 85)},
	}
	for i, junction := range junctions {
		greenIntervals := junction.GetGreenIntervals()
		assert.Equalf(t, len(greenIntervals), len(correctGreenIntervals[i]), "Mismatch in green intervals length for junction %d", i)
		for j, greenInterval := range greenIntervals {
			assert.Equalf(t, correctGreenIntervals[i][j].Start, greenInterval.Start, "Mismatch in start time for junction %d interval %d", i, j)
			assert.Equalf(t, correctGreenIntervals[i][j].End, greenInterval.End, "Mismatch in end time for junction %d interval %d", i, j)
			assert.Equalf(t, correctGreenIntervals[i][j].PhaseIdx, greenInterval.PhaseIdx, "Mismatch in phase index for junction %d interval %d", i, j)
		}
	}
}

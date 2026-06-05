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

func TestCanConnect(t *testing.T) {
	intervalOne := NewGreenInterval(0, 20, 48)
	intervalTwo := NewGreenInterval(0, 22.5, 32.5)
	connectedInterval := intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 22.5, connectedInterval.Start, 0.01, "Expected start time to be 22.5")
	assert.InDelta(t, 32.5, connectedInterval.End, 0.01, "Expected end time to be 32.5")

	intervalOne = NewGreenInterval(0, 20, 48)
	intervalTwo = NewGreenInterval(0, 39.5, 55)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 39.5, connectedInterval.Start, 0.01, "Expected start time to be 39.5")
	assert.InDelta(t, 48, connectedInterval.End, 0.01, "Expected end time to be 48")

	intervalOne = NewGreenInterval(0, 45, 55)
	intervalTwo = NewGreenInterval(0, 51.5, 55)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 51.5, connectedInterval.Start, 0.01, "Expected start time to be 51.5")
	assert.InDelta(t, 55, connectedInterval.End, 0.01, "Expected end time to be 55")

	intervalOne = NewGreenInterval(1, 62, 70.5)
	intervalTwo = NewGreenInterval(1, 62, 71.5)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 1, connectedInterval.PhaseIdx, "Expected phase index to be 1")
	assert.InDelta(t, 62, connectedInterval.Start, 0.01, "Expected start time to be 62")
	assert.InDelta(t, 70.5, connectedInterval.End, 0.01, "Expected end time to be 70.5")

	intervalOne = NewGreenInterval(0, 45, 55)
	intervalTwo = NewGreenInterval(0, 51.5, 55)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 51.5, connectedInterval.Start, 0.01, "Expected start time to be 51.5")
	assert.InDelta(t, 55, connectedInterval.End, 0.01, "Expected end time to be 55")

	intervalOne = NewGreenInterval(0, 20, 48)
	intervalTwo = NewGreenInterval(0, 29, 32.5)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 29, connectedInterval.Start, 0.01, "Expected start time to be 29")
	assert.InDelta(t, 32.5, connectedInterval.End, 0.01, "Expected end time to be 32.5")

	intervalOne = NewGreenInterval(1, 62, 70.5)
	intervalTwo = NewGreenInterval(1, 62, 70.5)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 1, connectedInterval.PhaseIdx, "Expected phase index to be 1")
	assert.InDelta(t, 62, connectedInterval.Start, 0.01, "Expected start time to be 62")
	assert.InDelta(t, 70.5, connectedInterval.End, 0.01, "Expected end time to be 70.5")

	intervalOne = NewGreenInterval(0, 20, 48)
	intervalTwo = NewGreenInterval(0, 39.5, 48)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 39.5, connectedInterval.Start, 0.01, "Expected start time to be 39.5")
	assert.InDelta(t, 48, connectedInterval.End, 0.01, "Expected end time to be 48")
}

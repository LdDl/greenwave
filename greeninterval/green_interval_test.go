package greeninterval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanConnect(t *testing.T) {
	intervalOne := New(0, 20, 48)
	intervalTwo := New(0, 22.5, 32.5)
	connectedInterval := intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 22.5, connectedInterval.Start, 0.01, "Expected start time to be 22.5")
	assert.InDelta(t, 32.5, connectedInterval.End, 0.01, "Expected end time to be 32.5")

	intervalOne = New(0, 20, 48)
	intervalTwo = New(0, 39.5, 55)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 39.5, connectedInterval.Start, 0.01, "Expected start time to be 39.5")
	assert.InDelta(t, 48, connectedInterval.End, 0.01, "Expected end time to be 48")

	intervalOne = New(0, 45, 55)
	intervalTwo = New(0, 51.5, 55)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 51.5, connectedInterval.Start, 0.01, "Expected start time to be 51.5")
	assert.InDelta(t, 55, connectedInterval.End, 0.01, "Expected end time to be 55")

	intervalOne = New(1, 62, 70.5)
	intervalTwo = New(1, 62, 71.5)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 1, connectedInterval.PhaseIdx, "Expected phase index to be 1")
	assert.InDelta(t, 62, connectedInterval.Start, 0.01, "Expected start time to be 62")
	assert.InDelta(t, 70.5, connectedInterval.End, 0.01, "Expected end time to be 70.5")

	intervalOne = New(0, 45, 55)
	intervalTwo = New(0, 51.5, 55)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 51.5, connectedInterval.Start, 0.01, "Expected start time to be 51.5")
	assert.InDelta(t, 55, connectedInterval.End, 0.01, "Expected end time to be 55")

	intervalOne = New(0, 20, 48)
	intervalTwo = New(0, 29, 32.5)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 29, connectedInterval.Start, 0.01, "Expected start time to be 29")
	assert.InDelta(t, 32.5, connectedInterval.End, 0.01, "Expected end time to be 32.5")

	intervalOne = New(1, 62, 70.5)
	intervalTwo = New(1, 62, 70.5)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 1, connectedInterval.PhaseIdx, "Expected phase index to be 1")
	assert.InDelta(t, 62, connectedInterval.Start, 0.01, "Expected start time to be 62")
	assert.InDelta(t, 70.5, connectedInterval.End, 0.01, "Expected end time to be 70.5")

	intervalOne = New(0, 20, 48)
	intervalTwo = New(0, 39.5, 48)
	connectedInterval = intervalOne.CanConnect(intervalTwo)
	assert.NotNil(t, connectedInterval, "Expected intervals to connect")
	assert.Equal(t, 0, connectedInterval.PhaseIdx, "Expected phase index to be 0")
	assert.InDelta(t, 39.5, connectedInterval.Start, 0.01, "Expected start time to be 39.5")
	assert.InDelta(t, 48, connectedInterval.End, 0.01, "Expected end time to be 48")
}

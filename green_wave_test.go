package greenwave

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindGreenWavesBetweenIntervals(t *testing.T) {
	// Case 1
	greenIntervalsOne := []*GreenInterval{
		NewGreenInterval(0, 0, 30),
		NewGreenInterval(1, 50, 70),
	}

	greenIntervalsTwo := []*GreenInterval{
		NewGreenInterval(0, 18, 53),
		NewGreenInterval(1, 68, 78),
	}

	distanceMeters := 200.0
	travelTimeSeconds := 18.0

	correctGreenWaves := []*GreenWave{
		NewGreenWave(
			NewGreenInterval(0, 0, 30),
			NewGreenInterval(0, 18, 48),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			NewGreenInterval(1, 50, 60),
			NewGreenInterval(1, 68, 78),
			distanceMeters,
			travelTimeSeconds,
		),
	}

	greenWaves := FindGreenWavesBetweenIntervals(greenIntervalsOne, greenIntervalsTwo, distanceMeters, travelTimeSeconds)
	assert.Equalf(t, len(correctGreenWaves), len(greenWaves), "Case1. Expected %d green waves, got %d", len(correctGreenWaves), len(greenWaves))
	for i, greenWave := range greenWaves {
		assert.Equalf(t, correctGreenWaves[i], greenWave, "Case 1. Expected green wave %d to be %v, got %v", i, correctGreenWaves[i], greenWave)
	}

	// Case 2
	greenIntervalsOne = []*GreenInterval{
		NewGreenInterval(0, 18, 53),
		NewGreenInterval(1, 68, 78),
	}

	greenIntervalsTwo = []*GreenInterval{
		NewGreenInterval(0, 44, 54),
		NewGreenInterval(1, 61, 79),
	}

	distanceMeters = 250.0
	travelTimeSeconds = 22.5

	correctGreenWaves = []*GreenWave{
		NewGreenWave(
			NewGreenInterval(0, 21.5, 31.5),
			NewGreenInterval(0, 44, 54),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			NewGreenInterval(0, 38.5, 53),
			NewGreenInterval(1, 61, 75.5),
			distanceMeters,
			travelTimeSeconds,
		),
	}

	greenWaves = FindGreenWavesBetweenIntervals(greenIntervalsOne, greenIntervalsTwo, distanceMeters, travelTimeSeconds)
	assert.Equalf(t, len(correctGreenWaves), len(greenWaves), "Case 2. Expected %d green waves, got %d", len(correctGreenWaves), len(greenWaves))
	for i, greenWave := range greenWaves {
		assert.Equalf(t, correctGreenWaves[i], greenWave, "Case 2. Expected green wave %d to be %v, got %v", i, correctGreenWaves[i], greenWave)
	}

	// Case 3
	greenIntervalsOne = []*GreenInterval{
		NewGreenInterval(0, 44, 54),
		NewGreenInterval(1, 61, 79),
	}

	greenIntervalsTwo = []*GreenInterval{
		NewGreenInterval(0, 38, 53),
		NewGreenInterval(1, 63, 83),
	}

	distanceMeters = 150.0
	travelTimeSeconds = 13.5

	correctGreenWaves = []*GreenWave{
		NewGreenWave(
			NewGreenInterval(0, 49.5, 54.0),
			NewGreenInterval(1, 63, 67.5),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			NewGreenInterval(1, 61, 69.5),
			NewGreenInterval(1, 74.5, 83),
			distanceMeters,
			travelTimeSeconds,
		),
	}

	greenWaves = FindGreenWavesBetweenIntervals(greenIntervalsOne, greenIntervalsTwo, distanceMeters, travelTimeSeconds)
	assert.Equalf(t, len(correctGreenWaves), len(greenWaves), "Case 3. Expected %d green waves, got %d", len(correctGreenWaves), len(greenWaves))
	for i, greenWave := range greenWaves {
		assert.Equalf(t, correctGreenWaves[i], greenWave, "Case 3. Expected green wave %d to be %v, got %v", i, correctGreenWaves[i], greenWave)
	}
}

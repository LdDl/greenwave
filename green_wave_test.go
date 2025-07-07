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
		NewGreenInterval(0, 20, 55),
		NewGreenInterval(1, 70, 80),
	}

	distanceMeters := 200.0
	travelTimeSeconds := 18.0

	correctGreenWaves := []*GreenWave{
		NewGreenWave(
			NewGreenInterval(0, 2, 30),
			NewGreenInterval(0, 20, 48),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			NewGreenInterval(1, 52, 62),
			NewGreenInterval(1, 70, 80),
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
		NewGreenInterval(0, 20, 55),
		NewGreenInterval(1, 70, 80),
	}

	greenIntervalsTwo = []*GreenInterval{
		NewGreenInterval(0, 45, 55),
		NewGreenInterval(1, 62, 80),
	}

	distanceMeters = 250.0
	travelTimeSeconds = 22.5

	correctGreenWaves = []*GreenWave{
		NewGreenWave(
			NewGreenInterval(0, 22.5, 32.5),
			NewGreenInterval(0, 45, 55),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			NewGreenInterval(0, 39.5, 55),
			NewGreenInterval(1, 62, 77.5),
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
		NewGreenInterval(0, 45, 55),
		NewGreenInterval(1, 62, 80),
	}

	greenIntervalsTwo = []*GreenInterval{
		NewGreenInterval(0, 40, 55),
		NewGreenInterval(1, 65, 85),
	}

	distanceMeters = 150.0
	travelTimeSeconds = 13.5

	correctGreenWaves = []*GreenWave{
		NewGreenWave(
			NewGreenInterval(0, 51.5, 55.0),
			NewGreenInterval(1, 65, 68.5),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			NewGreenInterval(1, 62, 71.5),
			NewGreenInterval(1, 75.5, 85),
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

func TestFindGreenWaves(t *testing.T) {
	junctions := basicTestJuntions()
	desiredSpeedKmh := 40.0
	greenWaves := FindGreenWaves(junctions, desiredSpeedKmh)
	correctGreenWaves := [][]*GreenWave{
		// Segment 0
		{
			NewGreenWave(
				NewGreenInterval(0, 2, 30),
				NewGreenInterval(0, 20, 48),
				200,
				18.0,
			),
			NewGreenWave(
				NewGreenInterval(1, 52, 62),
				NewGreenInterval(1, 70, 80),
				200,
				18.0,
			),
		},
		// Segment 1
		{
			NewGreenWave(
				NewGreenInterval(0, 22.5, 32.5),
				NewGreenInterval(0, 45, 55),
				250,
				22.5,
			),
			NewGreenWave(
				NewGreenInterval(0, 39.5, 55),
				NewGreenInterval(1, 62, 77.5),
				250,
				22.5,
			),
		},
		// Segment 2
		{
			NewGreenWave(
				NewGreenInterval(0, 51.5, 55.0),
				NewGreenInterval(1, 65, 68.5),
				150,
				13.5,
			),
			NewGreenWave(
				NewGreenInterval(1, 62, 71.5),
				NewGreenInterval(1, 75.5, 85),
				150,
				13.5,
			),
		},
	}

	assert.Equalf(t, len(correctGreenWaves), len(greenWaves), "Expected %d segments, got %d", len(correctGreenWaves), len(greenWaves))
	for i, segmentGreenWaves := range greenWaves {
		assert.Equalf(t, len(correctGreenWaves[i]), len(segmentGreenWaves), "Segment %d: Expected %d green waves, got %d", i, len(correctGreenWaves[i]), len(segmentGreenWaves))
		for j, greenWave := range segmentGreenWaves {
			assert.Equalf(t, correctGreenWaves[i][j], greenWave, "Segment %d, Green Wave %d: Expected %v, got %v", i, j, correctGreenWaves[i][j], greenWave)
		}
	}
}

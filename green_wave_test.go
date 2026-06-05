package greenwave

import (
	"testing"

	"github.com/LdDl/greenwave/greeninterval"
	"github.com/LdDl/greenwave/junction"
	"github.com/stretchr/testify/assert"
)

func TestFindGreenWavesBetweenIntervals(t *testing.T) {
	// Case 1
	greenIntervalsOne := []*greeninterval.GreenInterval{
		greeninterval.New(0, 0, 30),
		greeninterval.New(1, 50, 70),
	}

	greenIntervalsTwo := []*greeninterval.GreenInterval{
		greeninterval.New(0, 20, 55),
		greeninterval.New(1, 70, 80),
	}

	distanceMeters := 200.0
	travelTimeSeconds := 18.0

	correctGreenWaves := []*GreenWave{
		NewGreenWave(
			greeninterval.New(0, 2, 30),
			greeninterval.New(0, 20, 48),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			greeninterval.New(1, 52, 62),
			greeninterval.New(1, 70, 80),
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
	greenIntervalsOne = []*greeninterval.GreenInterval{
		greeninterval.New(0, 20, 55),
		greeninterval.New(1, 70, 80),
	}

	greenIntervalsTwo = []*greeninterval.GreenInterval{
		greeninterval.New(0, 45, 55),
		greeninterval.New(1, 62, 80),
	}

	distanceMeters = 250.0
	travelTimeSeconds = 22.5

	correctGreenWaves = []*GreenWave{
		NewGreenWave(
			greeninterval.New(0, 22.5, 32.5),
			greeninterval.New(0, 45, 55),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			greeninterval.New(0, 39.5, 55),
			greeninterval.New(1, 62, 77.5),
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
	greenIntervalsOne = []*greeninterval.GreenInterval{
		greeninterval.New(0, 45, 55),
		greeninterval.New(1, 62, 80),
	}

	greenIntervalsTwo = []*greeninterval.GreenInterval{
		greeninterval.New(0, 40, 55),
		greeninterval.New(1, 65, 85),
	}

	distanceMeters = 150.0
	travelTimeSeconds = 13.5

	correctGreenWaves = []*GreenWave{
		NewGreenWave(
			greeninterval.New(0, 51.5, 55.0),
			greeninterval.New(1, 65, 68.5),
			distanceMeters,
			travelTimeSeconds,
		),
		NewGreenWave(
			greeninterval.New(1, 62, 71.5),
			greeninterval.New(1, 75.5, 85),
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
	junctions := junction.BasicTestJunctions()
	desiredSpeedKmh := 40.0
	// Pick group for each junction (in this case, we know that all junctions have group #0)
	groupIDs := map[int]junction.GroupID{
		100500: 0,
		42:     0,
		78:     0,
		256:    0,
	}
	greenWaves := FindGreenWaves(junctions, groupIDs, desiredSpeedKmh)
	correctGreenWaves := [][]*GreenWave{
		// Segment 0
		{
			NewGreenWave(
				greeninterval.New(0, 2, 30),
				greeninterval.New(0, 20, 48),
				200,
				18.0,
			),
			NewGreenWave(
				greeninterval.New(1, 52, 62),
				greeninterval.New(1, 70, 80),
				200,
				18.0,
			),
		},
		// Segment 1
		{
			NewGreenWave(
				greeninterval.New(0, 22.5, 32.5),
				greeninterval.New(0, 45, 55),
				250,
				22.5,
			),
			NewGreenWave(
				greeninterval.New(0, 39.5, 55),
				greeninterval.New(1, 62, 77.5),
				250,
				22.5,
			),
		},
		// Segment 2
		{
			NewGreenWave(
				greeninterval.New(0, 51.5, 55.0),
				greeninterval.New(1, 65, 68.5),
				150,
				13.5,
			),
			NewGreenWave(
				greeninterval.New(1, 62, 71.5),
				greeninterval.New(1, 75.5, 85),
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

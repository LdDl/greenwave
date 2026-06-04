package greenwave

import (
	"testing"

	"github.com/LdDl/greenwave/greeninterval"
	"github.com/LdDl/greenwave/junction"
	"github.com/stretchr/testify/assert"
)

func TestMergeGreenWaves(t *testing.T) {
	junctions := junction.BasicTestJunctions()
	desiredSpeedKmh := 40.0
	greenWaves := FindGreenWaves(junctions, desiredSpeedKmh)
	throughGreenWaves := MergeGreenWaves(greenWaves)

	correctThroughGreenWaves := []*ThroughGreenWave{
		NewThroughGreenWave(
			[]*greeninterval.GreenInterval{
				greeninterval.New(0, 11, 14.5),
				greeninterval.New(0, 29, 32.5),
				greeninterval.New(0, 51.5, 55),
				greeninterval.New(1, 65, 68.5),
			},
		),
		NewThroughGreenWave(
			[]*greeninterval.GreenInterval{
				greeninterval.New(0, 21.5, 30),
				greeninterval.New(0, 39.5, 48),
				greeninterval.New(1, 62, 70.5),
				greeninterval.New(1, 75.5, 84),
			},
		),
	}
	assert.Equalf(t, len(correctThroughGreenWaves), len(throughGreenWaves), "Expected %d through green waves, got %d", len(correctThroughGreenWaves), len(throughGreenWaves))
	for i, throughGreenWave := range throughGreenWaves {
		assert.Equalf(t, correctThroughGreenWaves[i], throughGreenWave, "Expected through green wave %d to be %v, got %v", i, correctThroughGreenWaves[i], throughGreenWave)
		for j, interval := range throughGreenWave.intervals {
			assert.Equalf(t, correctThroughGreenWaves[i].intervals[j], interval, "Through green wave %d interval %d is incorrect: got %v, want %v", i, j, interval, correctThroughGreenWaves[i].intervals[j])
		}
	}
}

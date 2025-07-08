package greenwave

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeGreenWaves(t *testing.T) {
	junctions := basicTestJuntions()
	desiredSpeedKmh := 40.0
	greenWaves := FindGreenWaves(junctions, desiredSpeedKmh)
	throughGreenWaves := MergeGreenWaves(greenWaves)

	correctThroughGreenWaves := []*ThroughGreenWave{
		NewThroughGreenWave(
			[]*GreenInterval{
				NewGreenInterval(0, 11, 14.5),
				NewGreenInterval(0, 29, 32.5),
				NewGreenInterval(0, 51.5, 55),
				NewGreenInterval(1, 65, 68.5),
			},
		),
		NewThroughGreenWave(
			[]*GreenInterval{
				NewGreenInterval(0, 21.5, 30),
				NewGreenInterval(0, 39.5, 48),
				NewGreenInterval(1, 62, 70.5),
				NewGreenInterval(1, 75.5, 84),
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

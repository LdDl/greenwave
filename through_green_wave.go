package greenwave

import "math"

// ThroughGreenWave represents a green wave that passes through a series of green intervals between traffic lights.
type ThroughGreenWave struct {
	// Intervals of green lights that form the green wave
	intervals []*GreenInterval
	// Number of junctions which could be passed through
	depth int
	// Bandwidth of the green wave, which is the minimum duration of the green intervals
	bandWidth float64
}

// NewThroughGreenWave creates a new ThroughGreenWave from a slice of GreenInterval.
// It calculates the minimum bandwidth of the green wave, which is the shortest duration of the green intervals.
func NewThroughGreenWave(intervals []*GreenInterval) *ThroughGreenWave {
	minBandWidth := math.Inf(1)
	for _, interval := range intervals {
		bandWidth := interval.End - interval.Start
		if bandWidth < minBandWidth {
			minBandWidth = bandWidth
		}
	}
	if math.IsInf(minBandWidth, 1) {
		minBandWidth = 0
	}
	return &ThroughGreenWave{
		intervals: intervals,
		depth:     len(intervals),
		bandWidth: minBandWidth,
	}
}

// Depth returns the number of junctions that can be passed through in this green wave.
func (tgw *ThroughGreenWave) Depth() int {
	return tgw.depth
}

// Bandwitdh returns the bandwidth of the green wave, which is the minimum duration of the green intervals.
func (tgw *ThroughGreenWave) Bandwidth() float64 {
	return tgw.bandWidth
}

// GetIntervals returns the intervals of the green wave.
func (tgw *ThroughGreenWave) GetIntervals() []*GreenInterval {
	return tgw.intervals
}

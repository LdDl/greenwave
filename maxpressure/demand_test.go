package maxpressure

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/stretchr/testify/require"
)

type demandProfilePoint struct {
	name string
	time float64
	rate float64
}

func TestPeakDemandBoundaries(t *testing.T) {
	demand := PeakDemand(PeakDemandConfig{
		BaseRates:     map[gmns.LinkID]float64{1: 3600},
		RampUpS:       10,
		PeakFactor:    1.4,
		PeakDurationS: 20,
		RampDownS:     10,
	})
	points := []demandProfilePoint{
		{name: "ramp start", time: 0, rate: 1800},
		{name: "ramp midpoint", time: 5, rate: 2700},
		{name: "peak start", time: 10, rate: 5040},
		{name: "during peak", time: 29, rate: 5040},
		{name: "ramp down start", time: 30, rate: 5040},
		{name: "ramp down midpoint", time: 35, rate: 4320},
		{name: "steady start", time: 40, rate: 3600},
	}
	for _, point := range points {
		require.InDelta(t, point.rate, demand(1, point.time), 1e-9, point.name)
		require.Zero(t, demand(2, point.time), point.name)
	}
}

func TestPeakDemandDefaultFactorAndSkippedRamps(t *testing.T) {
	demand := PeakDemand(PeakDemandConfig{
		BaseRates:     map[gmns.LinkID]float64{1: 3600},
		PeakDurationS: 10,
	})
	require.InDelta(t, 4680, demand(1, 0), 1e-9)
	require.InDelta(t, 3600, demand(1, 10), 1e-9)
	steady := PeakDemand(PeakDemandConfig{BaseRates: map[gmns.LinkID]float64{1: 3600}})
	require.Equal(t, 3600.0, steady(1, 0))
}

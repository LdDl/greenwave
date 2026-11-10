package greenwave

import (
	"math"
	"testing"

	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/geom"
	"github.com/LdDl/greenwave/junction"
	"github.com/stretchr/testify/require"
)

func periodicTestJunction(id, cycle, green, offset int, distance float64) *junction.Junction {
	phase := junction.NewPhase(0, []junction.SignalGroup{{ID: 0, Signals: []*junction.Signal{
		junction.NewSignal(green, color.GREEN),
		junction.NewSignal(cycle-green, color.RED),
	}}})
	jun := junction.NewJunction([]*junction.Phase{phase}, junction.WithID(id), junction.WithPoint(geom.Point{Y: distance}))
	jun.SetOffset(offset)
	return jun
}

func TestPeriodicWavesAcrossCycles(t *testing.T) {
	junctions := []*junction.Junction{
		periodicTestJunction(1, 100, 40, 0, 0),
		periodicTestJunction(2, 100, 40, 9, 394.8),
		periodicTestJunction(3, 100, 40, 94, 701.6),
	}
	waves := FindGreenWaves(junctions, nil, 13)
	through := MergeGreenWaves(waves)
	require.Len(t, through, 1)
	require.Equal(t, 3, through[0].Depth())
	require.InDelta(t, 39.67076923076923, through[0].Bandwidth(), 1e-8)
	for i, arrival := range []float64{0, 109.32923076923077, 194.28923076923076} {
		require.InDelta(t, arrival, through[0].GetIntervals()[i].Start, 1e-8)
	}
	for i := range junctions {
		junctions[i].SetOffset(0)
	}
	baseline := MergeGreenWaves(FindGreenWaves(junctions, nil, 13))
	require.Len(t, baseline, 1)
	require.InDelta(t, 24.96, baseline[0].Bandwidth(), 1e-8)
}

func TestTravelAcrossNineThousandCyclesDoesNotExpandTheResult(t *testing.T) {
	for _, travel := range []float64{9, 900009} {
		junctions := []*junction.Junction{
			periodicTestJunction(1, 100, 40, 0, 0),
			periodicTestJunction(2, 100, 40, 9, travel),
			periodicTestJunction(3, 100, 40, 18, travel*2),
		}
		waves := FindGreenWaves(junctions, nil, 3.6)
		require.Len(t, waves[0], 1)
		require.Len(t, waves[1], 1)
		through := MergeGreenWaves(waves)
		require.Len(t, through, 1)
		require.Equal(t, 40.0, through[0].Bandwidth())
		require.Equal(t, travel*2, through[0].GetIntervals()[2].Start)
	}
}

func TestFullCycleGreenAndWrappedOffsets(t *testing.T) {
	for _, offset := range []int{0, 60, 99} {
		junctions := []*junction.Junction{
			periodicTestJunction(1, 100, 100, offset, 0),
			periodicTestJunction(2, 100, 100, offset, 900009),
		}
		waves := FindGreenWaves(junctions, nil, 3.6)
		band := 0.0
		for _, wave := range waves[0] {
			band += wave.Bandwidth()
		}
		require.Equal(t, 100.0, band)
		through := MergeGreenWaves(waves)
		require.Len(t, through, 1)
		require.Equal(t, 100.0, through[0].Bandwidth())
		for _, wave := range through {
			require.Equal(t, 2, wave.Depth())
		}
	}
}

func signalIsGreen(jun *junction.Junction, time float64) bool {
	cycle := float64(jun.GetTotalDuration())
	position := math.Mod(math.Mod(time-float64(jun.GetOffset()), cycle)+cycle, cycle)
	for _, phase := range jun.Cycle {
		for _, signal := range phase.SignalGroups[0].Signals {
			if position < float64(signal.Duration) {
				return signal.Color == color.GREEN
			}
			position -= float64(signal.Duration)
		}
	}
	return false
}

func TestPeriodicWavesMatchSignalStatesIncludingUnequalCycles(t *testing.T) {
	// Compare interval membership with direct signal lookup at independent sample times.
	for scenario := 0; scenario < 48; scenario++ {
		junctions := []*junction.Junction{
			periodicTestJunction(1, 19, 8, scenario%19, 0),
			periodicTestJunction(2, 7+scenario%5, scenario%7, scenario%9, float64(3+scenario*11)),
			periodicTestJunction(3, 13, 6, scenario%13, float64(9+scenario*31)),
			periodicTestJunction(4, 17, 8, scenario%17, float64(15+scenario*37)),
		}
		for _, route := range [][]*junction.Junction{junctions, ReverseJunctions(junctions)} {
			waves := FindGreenWaves(route, nil, 3.6)
			through := MergeGreenWaves(waves)
			for departure := 0.125; departure < float64(route[0].GetTotalDuration()); departure += 0.25 {
				arrival := departure
				allGreen := signalIsGreen(route[0], arrival)
				for i, segment := range waves {
					nextArrival := arrival + math.Abs(route[i+1].GetPoint().Y-route[i].GetPoint().Y)
					want := signalIsGreen(route[i], arrival) && signalIsGreen(route[i+1], nextArrival)
					matches := 0
					for _, wave := range segment {
						if wave.IntervalJunOne().Start <= arrival && arrival < wave.IntervalJunOne().End {
							matches++
							actual := wave.IntervalJunTwo().Start + arrival - wave.IntervalJunOne().Start
							require.InDelta(t, nextArrival, actual, 1e-8)
						}
					}
					require.LessOrEqual(t, matches, 1, "duplicate segment at scenario %d", scenario)
					require.Equal(t, want, matches == 1, "scenario %d, segment %d, departure %v", scenario, i, departure)
					allGreen = allGreen && signalIsGreen(route[i+1], nextArrival)
					arrival = nextArrival
				}
				matches := 0
				for _, wave := range through {
					first := wave.GetIntervals()[0]
					if wave.Depth() == len(route) && first.Start <= departure && departure < first.End {
						matches++
					}
				}
				require.LessOrEqual(t, matches, 1)
				require.Equal(t, allGreen, matches == 1, "scenario %d, departure %v", scenario, departure)
			}
		}
	}
}

func TestFitnessRewardsAnIsolatedLaterSegment(t *testing.T) {
	junctions := []*junction.Junction{
		periodicTestJunction(1, 100, 0, 0, 0),
		periodicTestJunction(2, 100, 40, 9, 109),
		periodicTestJunction(3, 100, 40, 18, 218),
	}
	waves := FindGreenWaves(junctions, nil, 3.6)
	require.Empty(t, waves[0])
	require.Len(t, waves[1], 1)
	require.Empty(t, MergeGreenWaves(waves))
	require.InDelta(t, 40.0*4/9, calculateDirectionalFitness(junctions, nil, 3.6), 1e-8)
}

func TestEmptyWaveInputs(t *testing.T) {
	require.Empty(t, FindGreenWaves(nil, nil, 13))
	require.Empty(t, MergeGreenWaves(nil))
}

func BenchmarkPeriodicWaves(b *testing.B) {
	for _, travel := range []float64{9, 900009} {
		name := "short"
		if travel > 100 {
			name = "9000_cycles"
		}
		b.Run(name, func(b *testing.B) {
			junctions := []*junction.Junction{
				periodicTestJunction(1, 100, 40, 0, 0),
				periodicTestJunction(2, 100, 40, 9, travel),
				periodicTestJunction(3, 100, 40, 18, travel*2),
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				MergeGreenWaves(FindGreenWaves(junctions, nil, 3.6))
			}
		})
	}
}

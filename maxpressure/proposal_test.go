package maxpressure

import (
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
	"github.com/stretchr/testify/require"
	"testing"
)

func proposalGreen(duration int) *junction.Signal {
	return junction.NewSignal(duration, color.GREEN, junction.WithMinDuration(5), junction.WithMaxDuration(65))
}
func proposalRed(duration int) *junction.Signal {
	return junction.NewSignal(duration, color.RED, junction.WithMinDuration(0), junction.WithMaxDuration(70))
}
func buildProposalJunction() *junction.Junction {
	return junction.NewJunction([]*junction.Phase{
		junction.NewPhase(0, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{proposalGreen(40), junction.NewSignal(3, color.YELLOW), junction.NewSignal(2, color.RED)}},
			{ID: 1, Signals: []*junction.Signal{proposalRed(45)}},
		}),
		junction.NewPhase(1, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{proposalRed(25)}},
			{ID: 1, Signals: []*junction.Signal{proposalGreen(20), junction.NewSignal(3, color.YELLOW), junction.NewSignal(2, color.RED)}},
		}),
	})
}

func TestProposalPreservesCycleOffsetBoundsAndClearance(t *testing.T) {
	jun := buildProposalJunction()
	jun.SetOffset(13)
	result, err := SynthesizeProgram(jun, map[int]float64{0: 50, 1: 50})
	require.NoError(t, err)
	require.Equal(t, 70, result.GetTotalDuration())
	require.Equal(t, 13, result.GetOffset())
	require.Equal(t, 30, result.Cycle[0].SignalGroups[0].Signals[0].Duration)
	require.Equal(t, 30, result.Cycle[1].SignalGroups[1].Signals[0].Duration)
	for i, phase := range result.Cycle {
		for gi, group := range phase.SignalGroups {
			total := 0
			for si, signal := range group.Signals {
				original := jun.Cycle[i].SignalGroups[gi].Signals[si]
				require.NotSame(t, original, signal)
				require.Equal(t, original.MinDuration, signal.MinDuration)
				require.Equal(t, original.MaxDuration, signal.MaxDuration)
				require.GreaterOrEqual(t, signal.Duration, signal.MinDuration)
				require.LessOrEqual(t, signal.Duration, signal.MaxDuration)
				if original.Color == color.YELLOW {
					require.Equal(t, original.Duration, signal.Duration)
				}
				total += signal.Duration
			}
			require.Equal(t, 35, total)
		}
	}
	require.Equal(t, 40, jun.Cycle[0].SignalGroups[0].Signals[0].Duration)
}

func TestProposalHonorsLowerUpperAndFixedBounds(t *testing.T) {
	jun := buildProposalJunction()
	jun.Cycle[1].SignalGroups[1].Signals[0].MinDuration = 12
	result, err := SynthesizeProgram(jun, map[int]float64{0: 100})
	require.NoError(t, err)
	require.Equal(t, 12, result.Cycle[1].SignalGroups[1].Signals[0].Duration)
	require.Equal(t, 48, result.Cycle[0].SignalGroups[0].Signals[0].Duration)
	jun.Cycle[1].SignalGroups[1].Signals[0].MaxDuration = 25
	result, err = SynthesizeProgram(jun, map[int]float64{1: 100})
	require.NoError(t, err)
	require.Equal(t, 25, result.Cycle[1].SignalGroups[1].Signals[0].Duration)
	fixed := jun.Cycle[0].SignalGroups[0].Signals[0]
	fixed.MinDuration, fixed.MaxDuration = 40, 40
	result, err = SynthesizeProgram(jun, map[int]float64{1: 100})
	require.NoError(t, err)
	require.Equal(t, 40, result.Cycle[0].SignalGroups[0].Signals[0].Duration)
	require.Equal(t, 20, result.Cycle[1].SignalGroups[1].Signals[0].Duration)
}

func TestProposalResizesSimultaneousGroupsTogether(t *testing.T) {
	jun := buildProposalJunction()
	groups := jun.Cycle[0].SignalGroups
	groups = append(groups, junction.SignalGroup{ID: 2, Signals: []*junction.Signal{proposalGreen(40), junction.NewSignal(3, color.YELLOW), junction.NewSignal(2, color.RED)}})
	jun.Cycle[0] = junction.NewPhase(0, groups)
	jun.Cycle[1] = junction.NewPhase(1, append(jun.Cycle[1].SignalGroups, junction.SignalGroup{ID: 2, Signals: []*junction.Signal{proposalRed(25)}}))
	result, err := SynthesizeProgram(jun, map[int]float64{0: 1, 1: 1})
	require.NoError(t, err)
	require.Equal(t, 30, result.Cycle[0].SignalGroups[0].Signals[0].Duration)
	require.Equal(t, 30, result.Cycle[0].SignalGroups[2].Signals[0].Duration)
	require.Equal(t, 35, result.Cycle[1].SignalGroups[2].Signals[0].Duration)
}

func TestProposalNoWeightsReturnsIndependentUnchangedProgram(t *testing.T) {
	jun := buildProposalJunction()
	result, err := SynthesizeProgram(jun, nil)
	require.NoError(t, err)
	require.Equal(t, jun, result)
	require.NotSame(t, jun, result)
	result.Cycle[0].SignalGroups[0].Signals[0].Duration = 7
	require.Equal(t, 40, jun.Cycle[0].SignalGroups[0].Signals[0].Duration)
	_, err = SynthesizeProgram(jun, map[int]float64{0: -1})
	require.Error(t, err)
}

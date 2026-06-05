package maxpressure

import (
	"testing"

	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
	"github.com/stretchr/testify/assert"
)

// buildProposalJunction creates a 2-phase junction with G->Y->R sequences.
//
//	Phase 0: Group 0 - GREEN(40) + YELLOW(3) + RED(2) = 45s, Group 1 - RED(45) = 45s
//	Phase 1: Group 0 - RED(25) = 25s, Group 1 - GREEN(20) + YELLOW(3) + RED(2) = 25s
//	Total cycle: 70s
//	Clearance phase 0: Y(3) + R(2) = 5s
//	Clearance phase 1: Y(3) + R(2) = 5s
//	Available green time: 70 - 5 - 5 = 60s
func buildProposalJunction() *junction.Junction {
	return junction.NewJunction([]*junction.Phase{
		junction.NewPhase(0, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{
				junction.NewSignal(40, color.GREEN),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
			{ID: 1, Signals: []*junction.Signal{junction.NewSignal(45, color.RED)}},
		}),
		junction.NewPhase(1, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{junction.NewSignal(25, color.RED)}},
			{ID: 1, Signals: []*junction.Signal{
				junction.NewSignal(20, color.GREEN),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
		}),
	})
}

// greenDuration returns the duration of the first GREEN signal in phase phaseIdx, group grpIdx.
func greenDuration(jun *junction.Junction, phaseIdx, grpIdx int) int {
	for _, sig := range jun.Cycle[phaseIdx].SignalGroups[grpIdx].Signals {
		if sig.Color == color.GREEN || sig.Color == color.GREENPRIORITY {
			return sig.Duration
		}
	}
	return 0
}

// phaseTotalDuration returns the total duration of grpIdx in phaseIdx (sum of all signals).
func phaseTotalDuration(jun *junction.Junction, phaseIdx, grpIdx int) int {
	total := 0
	for _, sig := range jun.Cycle[phaseIdx].SignalGroups[grpIdx].Signals {
		total += sig.Duration
	}
	return total
}

// TestSynthesizeProposal_EqualFractions verifies that equal stage fractions yield equal green times.
//
// Available green time = 60s, fractions = 0.5 / 0.5 -> desired = 30s each.
// Phase 0 goes from GREEN(40) to GREEN(30); Phase 1 from GREEN(20) to GREEN(30).
// Total cycle stays at 70s (clearance preserved).
func TestSynthesizeProposal_EqualFractions(t *testing.T) {
	jun := buildProposalJunction()
	counts := map[StageID]int{0: 50, 1: 50}

	result := SynthesizeProposal(jun, counts, 100)

	assert.Equal(t, 30, greenDuration(result, 0, 0), "phase 0 green")
	assert.Equal(t, 30, greenDuration(result, 1, 1), "phase 1 green")
	// Clearance (Y+R = 5s) must be preserved in active groups
	assert.Equal(t, 35, phaseTotalDuration(result, 0, 0), "phase 0 total (30+3+2)")
	assert.Equal(t, 35, phaseTotalDuration(result, 1, 1), "phase 1 total (30+3+2)")
	// Total cycle = 35 + 35 = 70s (unchanged)
	assert.Equal(t, 70, result.GetTotalDuration())
}

// TestSynthesizeProposal_AllWeightOnPhase0 verifies normalization when one phase dominates.
//
// fraction_0 = 1.0, fraction_1 = 0.0 -> desired_0=60, desired_1=0->clamped to 5 (DefaultMinGreenS).
// sum = 65 > available(60) -> scale down phase 0 by excess=5:
//
//	scalable = 60-5 = 55  ->  green_0 = 60 - 5*(60-5)/55 = 55,  green_1 = 5.
func TestSynthesizeProposal_AllWeightOnPhase0(t *testing.T) {
	jun := buildProposalJunction()
	counts := map[StageID]int{0: 100, 1: 0}

	result := SynthesizeProposal(jun, counts, 100)

	assert.Equal(t, 55, greenDuration(result, 0, 0), "phase 0 green capped after normalization")
	assert.Equal(t, 5, greenDuration(result, 1, 1), "phase 1 green at DefaultMinGreenS floor")
	// Total green = 55+5 = 60 = available green time
	assert.Equal(t, 60, greenDuration(result, 0, 0)+greenDuration(result, 1, 1))
}

// TestSynthesizeProposal_DefaultMinGreenSFloor verifies that a phase with zero count
// still receives at least DefaultMinGreenS seconds of green, never less.
func TestSynthesizeProposal_DefaultMinGreenSFloor(t *testing.T) {
	jun := buildProposalJunction()
	counts := map[StageID]int{0: 100, 1: 0}

	result := SynthesizeProposal(jun, counts, 100)

	assert.GreaterOrEqual(t, greenDuration(result, 1, 1), int(DefaultMinGreenS), "phase 1 green must not go below DefaultMinGreenS")
}

// TestSynthesizeProposal_ExplicitMinDurationRespected verifies that an explicitly set
// Signal.MinDuration (< Duration) raises the per-phase floor above DefaultMinGreenS
// when it is larger.
//
//	Phase 1 GREEN signal has MinDuration=12 > DefaultMinGreenS(5) -> floor = 12.
func TestSynthesizeProposal_ExplicitMinDurationRespected(t *testing.T) {
	jun := junction.NewJunction([]*junction.Phase{
		junction.NewPhase(0, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{
				junction.NewSignal(40, color.GREEN),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
			{ID: 1, Signals: []*junction.Signal{junction.NewSignal(45, color.RED)}},
		}),
		junction.NewPhase(1, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{junction.NewSignal(25, color.RED)}},
			{ID: 1, Signals: []*junction.Signal{
				// MinDuration=12 explicitly set (< Duration=20, > DefaultMinGreenS=5)
				junction.NewSignal(20, color.GREEN, junction.WithMinDuration(12)),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
		}),
	})
	counts := map[StageID]int{0: 100, 1: 0}

	result := SynthesizeProposal(jun, counts, 100)

	assert.GreaterOrEqual(t, greenDuration(result, 1, 1), 12, "phase 1 green must respect Signal.MinDuration=12")
}

// TestSynthesizeProposal_ExplicitMinDurationBelowDefaultStillFloored verifies that
// Signal.MinDuration=3 (< DefaultMinGreenS=5) is floored up to DefaultMinGreenS.
func TestSynthesizeProposal_ExplicitMinDurationBelowDefaultStillFloored(t *testing.T) {
	jun := junction.NewJunction([]*junction.Phase{
		junction.NewPhase(0, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{
				junction.NewSignal(40, color.GREEN),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
			{ID: 1, Signals: []*junction.Signal{junction.NewSignal(45, color.RED)}},
		}),
		junction.NewPhase(1, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{junction.NewSignal(25, color.RED)}},
			{ID: 1, Signals: []*junction.Signal{
				// MinDuration=3 < DefaultMinGreenS=5 -> floor must be DefaultMinGreenS
				junction.NewSignal(20, color.GREEN, junction.WithMinDuration(3)),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
		}),
	})
	counts := map[StageID]int{0: 100, 1: 0}

	result := SynthesizeProposal(jun, counts, 100)

	assert.GreaterOrEqual(t, greenDuration(result, 1, 1), int(DefaultMinGreenS), "DefaultMinGreenS is the absolute floor even when Signal.MinDuration < DefaultMinGreenS")
}

// TestSynthesizeProposal_NoActiveGroupUnchanged verifies that a phase with no GREEN signal
// is returned unchanged.
func TestSynthesizeProposal_NoActiveGroupUnchanged(t *testing.T) {
	jun := junction.NewJunction([]*junction.Phase{
		junction.NewPhase(0, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{junction.NewSignal(30, color.RED)}},
		}),
		junction.NewPhase(1, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{junction.NewSignal(30, color.GREEN)}},
		}),
	})
	counts := map[StageID]int{0: 50, 1: 50}

	result := SynthesizeProposal(jun, counts, 100)

	// Phase 0 has no GREEN: must stay RED(30) untouched
	assert.Equal(t, color.RED, result.Cycle[0].SignalGroups[0].Signals[0].Color)
	assert.Equal(t, 30, result.Cycle[0].SignalGroups[0].Signals[0].Duration)
}

// TestSynthesizeProposal_ZeroStepsReturnsOriginal verifies that passing totalSteps=0
// returns the junction unchanged (guard against division by zero).
func TestSynthesizeProposal_ZeroStepsReturnsOriginal(t *testing.T) {
	jun := buildProposalJunction()

	result := SynthesizeProposal(jun, nil, 0)

	assert.Same(t, jun, result, "original junction must be returned when totalSteps=0")
}

// TestSynthesizeProposal_RoundingDriftPreservedCycle verifies that the total cycle
// duration is preserved exactly even when green times do not divide evenly.
//
// 3-phase junction: clearance 5s per phase, total cycle 76s, availableGreen = 61s.
// Equal fractions -> desired = 61/3 ≈ 20.333s per phase.
// Naive rounding: 20+20+20 = 60 ≠ 61 (drift = +1).
// The corrective pass must add the 1s to one phase so:
//
//	total green = 61s  and  total cycle = 76s.
func TestSynthesizeProposal_RoundingDriftPreservedCycle(t *testing.T) {
	jun := junction.NewJunction([]*junction.Phase{
		junction.NewPhase(0, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{
				junction.NewSignal(25, color.GREEN),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
			{ID: 1, Signals: []*junction.Signal{junction.NewSignal(30, color.RED)}},
			{ID: 2, Signals: []*junction.Signal{junction.NewSignal(30, color.RED)}},
		}),
		junction.NewPhase(1, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{junction.NewSignal(25, color.RED)}},
			{ID: 1, Signals: []*junction.Signal{
				junction.NewSignal(20, color.GREEN),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
			{ID: 2, Signals: []*junction.Signal{junction.NewSignal(25, color.RED)}},
		}),
		junction.NewPhase(2, []junction.SignalGroup{
			{ID: 0, Signals: []*junction.Signal{junction.NewSignal(21, color.RED)}},
			{ID: 1, Signals: []*junction.Signal{junction.NewSignal(21, color.RED)}},
			{ID: 2, Signals: []*junction.Signal{
				junction.NewSignal(16, color.GREEN),
				junction.NewSignal(3, color.YELLOW),
				junction.NewSignal(2, color.RED),
			}},
		}),
	})
	counts := map[StageID]int{0: 1, 1: 1, 2: 1}

	result := SynthesizeProposal(jun, counts, 3)

	totalGreen := greenDuration(result, 0, 0) + greenDuration(result, 1, 1) + greenDuration(result, 2, 2)
	assert.Equal(t, 61, totalGreen, "total green must equal availableGreen despite rounding drift")
	assert.Equal(t, 76, result.GetTotalDuration(), "total cycle must remain 76s")
}

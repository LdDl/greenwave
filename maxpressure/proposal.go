package maxpressure

import (
	"math"

	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
)

// DefaultMinGreenS is the fallback minimum green time used in SynthesizeProposal
// when a signal's MinDuration was not explicitly constrained.
const DefaultMinGreenS = 5.0

// phaseAnalysis holds the pre-computed structural properties of one junction phase
// needed for green time redistribution in SynthesizeProposal.
type phaseAnalysis struct {
	// activeGrpIdx is the index of the signal group that has a GREEN signal in this phase.
	// -1 if no group is active (no GREEN signal found).
	activeGrpIdx int
	// greenSigIdx is the index of the GREEN signal within the active group's Signals slice.
	greenSigIdx int
	// clearance is the total duration (seconds) of all non-GREEN signals in the active group
	// (e.g. YELLOW + RED). This time is kept fixed during redistribution.
	clearance float64
	// minGreen is the minimum allowable green duration (seconds) for this phase.
	// Derived from Signal.MinDuration if explicitly constrained, floored at DefaultMinGreenS.
	minGreen float64
}

// analyzePhase scans a phase and returns its structural properties for green time redistribution.
// It finds the first signal group containing a GREEN or GREENPRIORITY signal (the "active" group),
// computes the clearance (sum of all non-green signal durations in that group), and derives the
// per-phase minimum green floor from Signal.MinDuration, bounded below by DefaultMinGreenS.
func analyzePhase(phase *junction.Phase) phaseAnalysis {
	pa := phaseAnalysis{activeGrpIdx: -1, greenSigIdx: -1, minGreen: DefaultMinGreenS}
	for gi, sg := range phase.SignalGroups {
		for si, sig := range sg.Signals {
			if sig.Color != color.GREEN && sig.Color != color.GREENPRIORITY {
				continue
			}
			pa.activeGrpIdx = gi
			pa.greenSigIdx = si
			for _, s := range sg.Signals {
				if s.Color != color.GREEN && s.Color != color.GREENPRIORITY {
					pa.clearance += float64(s.Duration)
				}
			}
			// Use Signal.MinDuration as floor only when it was explicitly set to a value
			// smaller than Duration (otherwise it equals Duration by default in NewSignal).
			if sig.MinDuration > 0 && sig.MinDuration < sig.Duration {
				pa.minGreen = math.Max(float64(sig.MinDuration), DefaultMinGreenS)
			}
			return pa
		}
	}
	return pa
}

// SynthesizeProposal rebuilds a junction's green times using MP stage fractions.
//
// Algorithm:
//  1. For each phase, find the active signal group (first group with a GREEN signal).
//  2. Clearance per phase = sum of non-GREEN signal durations in the active group.
//  3. minGreen per phase = Signal.MinDuration of the GREEN signal if MinDuration < Duration
//     (explicitly constrained), otherwise DefaultMinGreenS.
//  4. Available green time = total_cycle - sum(clearance_i).
//  5. desired_green_i = fraction_i * availableGreen  (where fraction_i = counts_i / totalSteps).
//  6. actual_green_i = max(desired_green_i, minGreen_i).
//  7. If sum(actual_green_i) > availableGreen, scale down proportionally (keeping per-phase floors).
//  8. Rebuild phases with new green durations; non-active groups adjusted to match total.
func SynthesizeProposal(jun *junction.Junction, stageCounts map[StageID]int, totalSteps int) *junction.Junction {
	analyses := make([]phaseAnalysis, len(jun.Cycle))
	totalClearance := 0.0

	for i, phase := range jun.Cycle {
		analyses[i] = analyzePhase(phase)
		totalClearance += analyses[i].clearance
	}

	totalCycle := float64(jun.GetTotalDuration())
	availableGreen := totalCycle - totalClearance
	if availableGreen <= 0 || totalSteps <= 0 {
		return jun
	}

	// Compute desired green times from stage fractions
	greenTimes := make([]float64, len(jun.Cycle))
	for i, phase := range jun.Cycle {
		if analyses[i].activeGrpIdx < 0 {
			continue
		}
		frac := float64(stageCounts[StageID(phase.ID)]) / float64(totalSteps)
		g := frac * availableGreen
		if g < analyses[i].minGreen {
			g = analyses[i].minGreen
		}
		greenTimes[i] = g
	}

	// Scale down if sum exceeds availableGreen (keep relative ratios, floor at per-phase minGreen)
	sum := 0.0
	for _, g := range greenTimes {
		sum += g
	}
	if sum > availableGreen+0.5 {
		excess := sum - availableGreen
		scalable := 0.0
		for i, g := range greenTimes {
			if analyses[i].activeGrpIdx >= 0 && g > analyses[i].minGreen {
				scalable += g - analyses[i].minGreen
			}
		}
		if scalable > 0 {
			for i, g := range greenTimes {
				if analyses[i].activeGrpIdx >= 0 && g > analyses[i].minGreen {
					greenTimes[i] = g - excess*(g-analyses[i].minGreen)/scalable
				}
			}
		}
	}

	// Round all green times to integers; then correct rounding drift so that
	// the total cycle duration is preserved exactly.
	newGreens := make([]int, len(jun.Cycle))
	for i := range jun.Cycle {
		if analyses[i].activeGrpIdx < 0 {
			continue
		}
		newGreens[i] = int(math.Max(math.Round(greenTimes[i]), analyses[i].minGreen))
	}
	roundedSum := 0
	for _, g := range newGreens {
		roundedSum += g
	}
	if drift := int(availableGreen) - roundedSum; drift != 0 {
		// Apply drift to the active phase with the largest green time.
		maxIdx, maxVal := -1, -1
		for i, g := range newGreens {
			if analyses[i].activeGrpIdx >= 0 && g > maxVal {
				maxIdx, maxVal = i, g
			}
		}
		if maxIdx >= 0 {
			adjusted := newGreens[maxIdx] + drift
			if adjusted < int(analyses[maxIdx].minGreen) {
				adjusted = int(analyses[maxIdx].minGreen)
			}
			newGreens[maxIdx] = adjusted
		}
	}

	// Rebuild phases with updated durations
	newPhases := make([]*junction.Phase, len(jun.Cycle))
	for i, phase := range jun.Cycle {
		pa := analyses[i]
		if pa.activeGrpIdx < 0 {
			newPhases[i] = phase
			continue
		}

		newGreen := newGreens[i]
		newPhaseDur := newGreen + int(math.Round(pa.clearance))

		newSGs := make([]junction.SignalGroup, len(phase.SignalGroups))
		for gi, sg := range phase.SignalGroups {
			newSigs := make([]*junction.Signal, len(sg.Signals))
			if gi == pa.activeGrpIdx {
				// Update GREEN signal; keep YELLOW/RED as-is
				for si, sig := range sg.Signals {
					dur := sig.Duration
					if si == pa.greenSigIdx {
						dur = newGreen
					}
					newSigs[si] = &junction.Signal{
						Duration:    dur,
						MinDuration: sig.MinDuration,
						MaxDuration: sig.MaxDuration,
						Color:       sig.Color,
					}
				}
			} else {
				// Non-active group: adjust last signal to match new total phase duration
				currentTotal := 0
				for _, sig := range sg.Signals {
					currentTotal += sig.Duration
				}
				diff := newPhaseDur - currentTotal
				for si, sig := range sg.Signals {
					dur := sig.Duration
					if si == len(sg.Signals)-1 {
						dur += diff
						if dur < 0 {
							dur = 0
						}
					}
					newSigs[si] = &junction.Signal{
						Duration:    dur,
						MinDuration: sig.MinDuration,
						MaxDuration: sig.MaxDuration,
						Color:       sig.Color,
					}
				}
			}
			newSGs[gi] = junction.SignalGroup{ID: sg.ID, Signals: newSigs}
		}
		newPhases[i] = junction.NewPhase(phase.ID, newSGs)
	}

	pt := jun.GetPoint()
	return junction.NewJunction(newPhases,
		junction.WithID(jun.ID),
		junction.WithLabel(jun.Label),
		junction.WithPoint(pt),
	)
}

package maxpressure

import (
	"fmt"
	"math"
	"slices"

	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
)

const DefaultMinGreenS = 5.0

type phaseAdjustment struct {
	indices []int
	width   int
	minimum int
	maximum int
}

// adjustableWindow locates a common green/red interval that can be resized in
// every group together. Yellow and red-yellow intervals are never modified.
func adjustableWindow(phase *junction.Phase) phaseAdjustment {
	bounds := []int{0}
	for _, group := range phase.SignalGroups {
		elapsed := 0
		for _, signal := range group.Signals {
			elapsed += signal.Duration
			bounds = append(bounds, elapsed)
		}
	}
	slices.Sort(bounds)
	bounds = slices.Compact(bounds)
	best := phaseAdjustment{}
	for i := 0; i+1 < len(bounds); i++ {
		start, width := bounds[i], bounds[i+1]-bounds[i]
		candidate := phaseAdjustment{width: width, minimum: 1 - width, maximum: math.MaxInt32}
		green, allowed := false, true
		for _, group := range phase.SignalGroups {
			elapsed, index := 0, -1
			for si, signal := range group.Signals {
				if elapsed <= start && start < elapsed+signal.Duration {
					index = si
					break
				}
				elapsed += signal.Duration
			}
			if index < 0 {
				allowed = false
				break
			}
			signal := group.Signals[index]
			if signal.Color != color.RED && !isGreen(signal.Color) {
				allowed = false
				break
			}
			green = green || isGreen(signal.Color)
			candidate.indices = append(candidate.indices, index)
			candidate.minimum = max(candidate.minimum, signal.MinDuration-signal.Duration)
			candidate.maximum = min(candidate.maximum, signal.MaxDuration-signal.Duration)
		}
		if allowed && green && width > best.width {
			best = candidate
		}
	}
	return best
}

// SynthesizeProgram preserves the cycle, offset, group alignment and every
// signal's explicit bounds. It is a fixed-plan heuristic evaluated separately
// from the adaptive controller; no performance improvement is assumed.
func SynthesizeProgram(jun *junction.Junction, phaseSeconds map[int]float64) (*junction.Junction, error) {
	if jun == nil || len(jun.Cycle) == 0 {
		return nil, fmt.Errorf("cannot synthesize an empty program")
	}
	if _, err := compileProgram(jun, nil, false); err != nil {
		return nil, err
	}
	adjustments := make([]phaseAdjustment, len(jun.Cycle))
	widths := make([]int, len(jun.Cycle))
	budget, totalWeight := 0, 0.0
	for i, phase := range jun.Cycle {
		adjustments[i] = adjustableWindow(phase)
		widths[i] = adjustments[i].width
		budget += widths[i]
		weight := phaseSeconds[phase.ID]
		if !finiteNonnegative(weight) {
			return nil, fmt.Errorf("phase weights must be finite and non-negative")
		}
		if widths[i] > 0 {
			totalWeight += weight
		}
	}
	if totalWeight > 0 {
		target := make([]float64, len(widths))
		assigned := 0
		// Project proportional targets onto bounded integer durations with exact sum.
		for i, phase := range jun.Cycle {
			a := adjustments[i]
			target[i] = phaseSeconds[phase.ID] / totalWeight * float64(budget)
			widths[i] = a.width + a.minimum
			if a.width == 0 {
				widths[i] = 0
			}
			assigned += widths[i]
		}
		for assigned < budget {
			best, gap := -1, math.Inf(-1)
			for i, a := range adjustments {
				if a.width == 0 || widths[i] >= a.width+a.maximum {
					continue
				}
				distance := target[i] - float64(widths[i])
				if distance > gap {
					best, gap = i, distance
				}
			}
			if best < 0 {
				return nil, fmt.Errorf("signal bounds cannot preserve cycle duration")
			}
			widths[best]++
			assigned++
		}
	}
	phases := make([]*junction.Phase, len(jun.Cycle))
	for i, phase := range jun.Cycle {
		groups := make([]junction.SignalGroup, len(phase.SignalGroups))
		delta := widths[i] - adjustments[i].width
		for gi, group := range phase.SignalGroups {
			signals := make([]*junction.Signal, len(group.Signals))
			for si, original := range group.Signals {
				signal := *original
				if len(adjustments[i].indices) > 0 && si == adjustments[i].indices[gi] {
					signal.Duration += delta
				}
				if signal.Duration < signal.MinDuration || signal.Duration > signal.MaxDuration {
					return nil, fmt.Errorf("proposal violates signal bounds")
				}
				signals[si] = &signal
			}
			groups[gi] = junction.SignalGroup{ID: group.ID, Signals: signals}
		}
		phases[i] = junction.NewPhase(phase.ID, groups)
	}
	result := junction.NewJunction(phases, junction.WithID(jun.ID), junction.WithLabel(jun.Label), junction.WithPoint(jun.GetPoint()))
	result.SetOffset(jun.GetOffset())
	if result.GetTotalDuration() != jun.GetTotalDuration() {
		return nil, fmt.Errorf("proposal changed cycle duration")
	}
	return result, nil
}

// SynthesizeProposal retains the older count-based Go entry point. New callers
// should use SynthesizeProgram with phase green seconds and handle its error.
func SynthesizeProposal(jun *junction.Junction, stageCounts map[StageID]int, totalSteps int) *junction.Junction {
	weights := make(map[int]float64)
	if totalSteps > 0 {
		for stage, count := range stageCounts {
			weights[int(stage)] = float64(count)
		}
	}
	result, err := SynthesizeProgram(jun, weights)
	if err != nil {
		return jun
	}
	return result
}

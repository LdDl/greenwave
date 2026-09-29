package maxpressure

import (
	"fmt"
	"math"
	"slices"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
)

type ProgramWindow struct {
	StartS       float64
	EndS         float64
	StageID      StageID
	ConnectorIDs []gmns.LinkID
}

// SignalProgram retains exact simultaneous green windows and the original offset.
type SignalProgram struct {
	CycleS  float64
	OffsetS float64
	Stages  []Stage
	Windows []ProgramWindow
}

func isGreen(c color.Color) bool { return c == color.GREEN || c == color.GREENPRIORITY }

// CompileProgram splits phases at every group's signal boundary. Two groups
// whose greens never overlap can never appear together in an adaptive stage.
func CompileProgram(jun *junction.Junction, mapping map[junction.GroupID][]gmns.LinkID) (*SignalProgram, error) {
	return compileProgram(jun, mapping, true)
}

func compileProgram(jun *junction.Junction, mapping map[junction.GroupID][]gmns.LinkID, requireMovements bool) (*SignalProgram, error) {
	if jun == nil || len(jun.Cycle) == 0 {
		return nil, fmt.Errorf("junction must have at least one phase")
	}
	p := &SignalProgram{OffsetS: float64(jun.GetOffset())}
	groups := make(map[junction.GroupID]bool)
	phaseIDs := make(map[int]bool)
	nextID := 0
	for _, phase := range jun.Cycle {
		if phase == nil || phase.ID < 0 || phaseIDs[phase.ID] {
			return nil, fmt.Errorf("invalid or duplicate phase ID")
		}
		phaseIDs[phase.ID] = true
		nextID = max(nextID, phase.ID+1)
	}
	for pi, phase := range jun.Cycle {
		seen := make(map[junction.GroupID]bool)
		boundaries := []int{0}
		duration := -1
		for _, group := range phase.SignalGroups {
			if seen[group.ID] || group.ID < 0 || len(group.Signals) == 0 {
				return nil, fmt.Errorf("phase %d has invalid groups", phase.ID)
			}
			seen[group.ID] = true
			if pi == 0 {
				groups[group.ID] = true
			} else if !groups[group.ID] {
				return nil, fmt.Errorf("signal groups must be present in every phase")
			}
			total := 0
			for _, signal := range group.Signals {
				if signal == nil || signal.Duration < 0 || signal.MinDuration < 0 || signal.MaxDuration < signal.MinDuration || signal.Duration < signal.MinDuration || signal.Duration > signal.MaxDuration {
					return nil, fmt.Errorf("phase %d group %d has invalid signal duration bounds", phase.ID, group.ID)
				}
				switch signal.Color {
				case color.RED, color.YELLOW, color.GREEN, color.GREENPRIORITY, color.REDYELLOW:
				default:
					return nil, fmt.Errorf("phase %d group %d has an unsupported signal color", phase.ID, group.ID)
				}
				total += signal.Duration
				boundaries = append(boundaries, total)
			}
			if total <= 0 || (duration >= 0 && total != duration) {
				return nil, fmt.Errorf("phase %d group durations must be equal and positive", phase.ID)
			}
			duration = total
		}
		if len(seen) == 0 || len(seen) != len(groups) {
			return nil, fmt.Errorf("signal groups must be present in every phase")
		}
		slices.Sort(boundaries)
		boundaries = slices.Compact(boundaries)
		firstGreen := true
		for i := 0; i+1 < len(boundaries); i++ {
			start, end := boundaries[i], boundaries[i+1]
			active := make(map[gmns.LinkID]bool)
			for _, group := range phase.SignalGroups {
				elapsed := 0
				for _, signal := range group.Signals {
					if elapsed <= start && start < elapsed+signal.Duration && isGreen(signal.Color) {
						for _, id := range mapping[group.ID] {
							active[id] = true
						}
					}
					elapsed += signal.Duration
				}
			}
			ids := make([]gmns.LinkID, 0, len(active))
			for id := range active {
				ids = append(ids, id)
			}
			slices.Sort(ids)
			stageID := NoStage
			if len(ids) > 0 {
				stageID = StageID(phase.ID)
				if !firstGreen {
					stageID = StageID(nextID)
					nextID++
				}
				firstGreen = false
				p.Stages = append(p.Stages, Stage{ID: stageID, PhaseID: phase.ID, ConnectorIDs: ids})
			}
			p.Windows = append(p.Windows, ProgramWindow{StartS: p.CycleS + float64(start), EndS: p.CycleS + float64(end), StageID: stageID, ConnectorIDs: ids})
		}
		p.CycleS += float64(duration)
	}
	assigned := make(map[gmns.LinkID]bool)
	for group, ids := range mapping {
		if !groups[group] {
			return nil, fmt.Errorf("mapped group %d is absent from the program", group)
		}
		for _, id := range ids {
			if assigned[id] {
				return nil, fmt.Errorf("movement %d is assigned more than once", id)
			}
			assigned[id] = true
		}
	}
	if requireMovements && len(p.Stages) == 0 {
		return nil, fmt.Errorf("junction has no assigned green movement")
	}
	return p, nil
}

func (p *SignalProgram) validate(inter *IntersectionState) error {
	if p == nil || !finiteNonnegative(p.CycleS) || p.CycleS == 0 || math.IsNaN(p.OffsetS) || math.IsInf(p.OffsetS, 0) || len(p.Windows) == 0 {
		return fmt.Errorf("missing or invalid fixed program for intersection %d", inter.MacroNodeID)
	}
	end := 0.0
	for _, window := range p.Windows {
		if window.StartS != end || !finiteNonnegative(window.EndS) || window.EndS <= end || window.EndS > p.CycleS {
			return fmt.Errorf("fixed program windows must cover the cycle without gaps or overlaps")
		}
		end = window.EndS
		if window.StageID == NoStage && len(window.ConnectorIDs) == 0 {
			continue
		}
		found := false
		for _, stage := range inter.Stages {
			if stage.ID == window.StageID && slices.Equal(stage.ConnectorIDs, window.ConnectorIDs) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("fixed program window references an unknown stage or movement set")
		}
	}
	if end != p.CycleS {
		return fmt.Errorf("fixed program windows do not cover the complete cycle")
	}
	return nil
}

// service integrates green time exactly across boundaries, offsets and cycles.
func (p *SignalProgram) service(time, dt float64, node gmns.NodeID) (map[gmns.LinkID]float64, []StepResult) {
	service := make(map[gmns.LinkID]float64)
	results := []StepResult{}
	for remaining := dt; remaining > 1e-9; {
		local := math.Mod(time-p.OffsetS, p.CycleS)
		if local < 0 {
			local += p.CycleS
		}
		for _, window := range p.Windows {
			if local < window.StartS || local >= window.EndS {
				continue
			}
			seconds := math.Min(remaining, window.EndS-local)
			for _, id := range window.ConnectorIDs {
				service[id] += seconds
			}
			results = append(results, StepResult{IntersectionID: node, SelectedStage: window.StageID, DurationS: seconds})
			remaining -= seconds
			time += seconds
			break
		}
	}
	return service, results
}

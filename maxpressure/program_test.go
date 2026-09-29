package maxpressure

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProgramNeverCombinesDisjointGreenWindows(t *testing.T) {
	jun := junction.NewJunction([]*junction.Phase{junction.NewPhase(7, []junction.SignalGroup{
		{ID: 10, Signals: []*junction.Signal{junction.NewSignal(10, color.GREEN), junction.NewSignal(10, color.RED)}},
		{ID: 20, Signals: []*junction.Signal{junction.NewSignal(10, color.RED), junction.NewSignal(10, color.GREEN)}},
	})})
	jun.SetOffset(3)
	program, err := CompileProgram(jun, map[junction.GroupID][]gmns.LinkID{10: {100}, 20: {101}})
	require.NoError(t, err)
	require.Len(t, program.Stages, 2)
	for _, stage := range program.Stages {
		require.Len(t, stage.ConnectorIDs, 1)
	}
	service, _ := program.service(11, 5, 10)
	require.Equal(t, 2.0, service[100])
	require.Equal(t, 3.0, service[101])
	service, _ = program.service(21, 25, 10)
	require.Equal(t, 13.0, service[100])
	require.Equal(t, 12.0, service[101])
}

func TestProgramRejectsUnequalGroupsAndUnknownColors(t *testing.T) {
	jun := buildProposalJunction()
	jun.Cycle[0].SignalGroups[1].Signals[0].Duration = 44
	_, err := CompileProgram(jun, map[junction.GroupID][]gmns.LinkID{0: {100}, 1: {101}})
	require.ErrorContains(t, err, "group durations")
	jun = buildProposalJunction()
	jun.Cycle[0].SignalGroups[0].Signals[0].Color = color.UNDEFINED
	_, err = CompileProgram(jun, map[junction.GroupID][]gmns.LinkID{0: {100}, 1: {101}})
	require.ErrorContains(t, err, "unsupported signal color")
}

func TestFixedProgramRejectsGapsBeforeReplay(t *testing.T) {
	program := &SignalProgram{CycleS: 20, Windows: []ProgramWindow{
		{StartS: 0, EndS: 5, StageID: NoStage},
		{StartS: 6, EndS: 20, StageID: NoStage},
	}}
	inter := &IntersectionState{MacroNodeID: 1}
	require.ErrorContains(t, program.validate(inter), "without gaps")
	program.Windows[1].StartS = 5
	require.NoError(t, program.validate(inter))
	program.Windows[1].EndS = 19
	require.ErrorContains(t, program.validate(inter), "complete cycle")
}

func TestSynthesisRejectsInvalidProgramsWithoutPanicking(t *testing.T) {
	jun := buildProposalJunction()
	jun.Cycle[0].SignalGroups[0].Signals[0] = nil
	_, err := SynthesizeProgram(jun, nil)
	require.Error(t, err)
	jun.Cycle[0] = nil
	_, err = SynthesizeProgram(jun, nil)
	require.Error(t, err)
}

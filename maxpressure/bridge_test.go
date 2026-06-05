package maxpressure

import (
	"testing"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/junction"
	"github.com/stretchr/testify/assert"
)

// buildTwoPhaseJunction creates a junction with 2 signal groups and 2 phases:
//   - Phase 0: group 0 (EW) is GREEN, group 1 (NS) is RED
//   - Phase 1: group 0 (EW) is RED,   group 1 (NS) is GREEN
func buildTwoPhaseJunction() *junction.Junction {
	return junction.NewJunction(
		[]*junction.Phase{
			junction.NewPhase(0, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(55, color.GREEN)}},
				{ID: 1, Signals: []*junction.Signal{junction.NewSignal(55, color.RED)}},
			}),
			junction.NewPhase(1, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(25, color.RED)}},
				{ID: 1, Signals: []*junction.Signal{junction.NewSignal(25, color.GREEN)}},
			}),
		},
	)
}

func TestStagesFromJunction_Basic(t *testing.T) {
	jun := buildTwoPhaseJunction()
	groupConnectors := map[junction.GroupID][]gmns.LinkID{
		0: {100, 101}, // EW connectors
		1: {102, 103}, // NS connectors
	}

	stages := StagesFromJunction(jun, groupConnectors)

	assert.Len(t, stages, 2)

	// Phase 0: group 0 is GREEN -> connectors 100, 101
	assert.Equal(t, StageID(0), stages[0].ID)
	assert.ElementsMatch(t, []gmns.LinkID{100, 101}, stages[0].ConnectorIDs)

	// Phase 1: group 1 is GREEN -> connectors 102, 103
	assert.Equal(t, StageID(1), stages[1].ID)
	assert.ElementsMatch(t, []gmns.LinkID{102, 103}, stages[1].ConnectorIDs)
}

func TestStagesFromJunction_BothGroupsGreen(t *testing.T) {
	jun := junction.NewJunction(
		[]*junction.Phase{
			junction.NewPhase(0, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(30, color.GREEN)}},
				{ID: 1, Signals: []*junction.Signal{junction.NewSignal(30, color.GREEN)}},
			}),
		},
	)
	groupConnectors := map[junction.GroupID][]gmns.LinkID{
		0: {100},
		1: {101},
	}

	stages := StagesFromJunction(jun, groupConnectors)

	assert.Len(t, stages, 1)
	assert.ElementsMatch(t, []gmns.LinkID{100, 101}, stages[0].ConnectorIDs)
}

func TestStagesFromJunction_NoGreenGroup(t *testing.T) {
	jun := junction.NewJunction(
		[]*junction.Phase{
			junction.NewPhase(0, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(30, color.RED)}},
			}),
		},
	)
	groupConnectors := map[junction.GroupID][]gmns.LinkID{
		0: {100},
	}

	stages := StagesFromJunction(jun, groupConnectors)

	assert.Len(t, stages, 1)
	assert.Empty(t, stages[0].ConnectorIDs)
}

func TestStagesFromJunction_UnmappedGroup(t *testing.T) {
	// Group 1 is green but has no entry in groupConnectors  silently skipped
	jun := junction.NewJunction(
		[]*junction.Phase{
			junction.NewPhase(0, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(30, color.GREEN)}},
				{ID: 1, Signals: []*junction.Signal{junction.NewSignal(30, color.GREEN)}},
			}),
		},
	)
	groupConnectors := map[junction.GroupID][]gmns.LinkID{
		0: {100},
		// group 1 intentionally not mapped
	}

	stages := StagesFromJunction(jun, groupConnectors)

	assert.Len(t, stages, 1)
	assert.Equal(t, []gmns.LinkID{100}, stages[0].ConnectorIDs)
}

func TestStagesFromJunction_GreenPriority(t *testing.T) {
	jun := junction.NewJunction(
		[]*junction.Phase{
			junction.NewPhase(0, []junction.SignalGroup{
				{ID: 0, Signals: []*junction.Signal{junction.NewSignal(30, color.GREENPRIORITY)}},
			}),
		},
	)
	groupConnectors := map[junction.GroupID][]gmns.LinkID{
		0: {100},
	}

	stages := StagesFromJunction(jun, groupConnectors)

	assert.Len(t, stages, 1)
	assert.Equal(t, []gmns.LinkID{100}, stages[0].ConnectorIDs)
}

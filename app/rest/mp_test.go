package rest

import (
	"net/http"
	"testing"

	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/stretchr/testify/require"
)

func mpTestConnector(id, node, upstream, downstream int) dto.MPLinkDTO {
	return dto.MPLinkDTO{
		ID: id, IsConnection: true, MacroNode: &node,
		MovementMesoLinkIncome: &upstream, MovementMesoLinkOutcome: &downstream,
		Capacity: 1800,
	}
}

func mpTestIntersection(id, connector int) dto.MPIntersectionConfigDTO {
	return dto.MPIntersectionConfigDTO{
		MacroNodeID:     id,
		GroupConnectors: map[int][]int{0: {connector}},
		Junction: dto.JunctionDTO{
			ID: id,
			Cycle: []dto.PhaseDTO{{
				ID: 0,
				SignalGroups: []dto.SignalGroupDTO{{
					ID:      0,
					Signals: []dto.SignalDTO{{Duration: 25, Color: "GREEN"}, {Duration: 3, Color: "YELLOW"}, {Duration: 2, Color: "RED"}},
				}},
			}},
		},
	}
}

func mpTestRequest() dto.MPRunRequest {
	return dto.MPRunRequest{
		Network: dto.MPNetworkDTO{Links: []dto.MPLinkDTO{
			{ID: 1, LengthMeters: 70, Lanes: 1, Capacity: 1800},
			{ID: 2, LengthMeters: 70, Lanes: 1, Capacity: 1800},
			{ID: 3, LengthMeters: 70, Lanes: 1, Capacity: 1800},
			mpTestConnector(100, 10, 1, 2),
			mpTestConnector(200, 20, 2, 3),
		}},
		Intersections:        []dto.MPIntersectionConfigDTO{mpTestIntersection(10, 100), mpTestIntersection(20, 200)},
		CoordinatedCorridors: [][]int{{1, 2, 3}},
		InitialQueues:        map[int]float64{1: 3},
		Drain:                dto.MPDrainDTO{Type: "none"},
		Config:               dto.MPSimConfigDTO{DeltaT: 1, SimTime: 4, Alpha: 1},
	}
}

func TestMPRunAcceptsExplicitCorridor(t *testing.T) {
	var response dto.MPRunResponse
	callCoordinationAPI(t, RequestMPRun(), mpTestRequest(), http.StatusOK, &response)
	require.Len(t, response.Evaluation.PerIntersection, 2)
	require.Len(t, response.Proposal, 2)
	// No demand or drain: every completed step still contains three vehicles.
	require.InDelta(t, 12, response.Evaluation.TotalDelayVehS, 1e-9)
}

func TestStandardMPRunDoesNotRequireCorridor(t *testing.T) {
	request := mpTestRequest()
	request.Config.Alpha = 0
	request.CoordinatedCorridors = nil
	callCoordinationAPI(t, RequestMPRun(), request, http.StatusOK, nil)
}

type invalidMPTestCase struct {
	name    string
	change  func(*dto.MPRunRequest)
	message string
}

func TestMPRunRejectsInvalidCoordination(t *testing.T) {
	tests := []invalidMPTestCase{
		{
			name:    "missing corridor",
			change:  func(req *dto.MPRunRequest) { req.CoordinatedCorridors = nil },
			message: "coordinated_corridors is required",
		},
		{
			name:    "short corridor",
			change:  func(req *dto.MPRunRequest) { req.CoordinatedCorridors = [][]int{{1, 2}} },
			message: "at least 3 directed road links",
		},
		{
			name:    "missing road",
			change:  func(req *dto.MPRunRequest) { req.CoordinatedCorridors = [][]int{{1, 999, 3}} },
			message: "road link 999 does not exist",
		},
		{
			name:    "reverse path has no reverse movements",
			change:  func(req *dto.MPRunRequest) { req.CoordinatedCorridors = [][]int{{3, 2, 1}} },
			message: "no signal-controlled movement",
		},
		{
			name:    "unassigned movement",
			change:  func(req *dto.MPRunRequest) { req.Intersections[1].GroupConnectors = nil },
			message: "no signal-controlled movement",
		},
		{
			name:    "empty phases",
			change:  func(req *dto.MPRunRequest) { req.Intersections[0].Junction.Cycle = nil },
			message: "must have at least one phase",
		},
		{
			name:    "negative coefficient",
			change:  func(req *dto.MPRunRequest) { req.Config.Alpha = -1 },
			message: "config.alpha must be non-negative",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mpTestRequest()
			tt.change(&request)
			var response map[string]string
			callCoordinationAPI(t, RequestMPRun(), request, http.StatusBadRequest, &response)
			require.Contains(t, response["Error"], tt.message)
		})
	}
}

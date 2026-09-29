package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/maxpressure"
	"github.com/stretchr/testify/require"
)

func TestMPComparisonReplaysProposalAndConservesDemand(t *testing.T) {
	body, err := os.ReadFile("../../examples/maxpressure/corridor.json")
	require.NoError(t, err)
	var request dto.MPRunRequest
	require.NoError(t, json.Unmarshal(body, &request))
	var response dto.MPRunResponse
	callCoordinationAPI(t, RequestMPRun(), request, http.StatusOK, &response)
	require.Equal(t, 2, response.ModelVersion)
	require.True(t, response.ProposalAccepted)
	require.Less(t, response.ProposalEvaluation.TotalDelayVehS, response.Baseline.TotalDelayVehS)
	require.GreaterOrEqual(t, response.ProposalEvaluation.Balance.Departed, response.Baseline.Balance.Departed)
	for _, result := range []*maxpressure.Evaluation{response.Baseline, response.StandardMP, response.Evaluation, response.ProposalEvaluation} {
		require.InDelta(t, 0, result.Balance.Error, 1e-8)
		require.Equal(t, request.Config.SimTime, result.DurationS)
		require.InDelta(t, response.Baseline.Balance.Requested, result.Balance.Requested, 1e-9)
		require.InDelta(t, response.Baseline.Balance.Initial, result.Balance.Initial, 1e-9)
		require.LessOrEqual(t, len(result.Samples), 202)
	}
	for i, proposal := range response.Proposal {
		require.Equal(t, request.Intersections[i].Junction.Offset, proposal.Junction.Offset)
		require.Equal(t, mpJunction(request.Intersections[i].Junction).GetTotalDuration(), proposal.Junction.TotalDuration)
		request.Intersections[i].Junction = proposal.Junction
	}
	opt, _, err := buildMPRun(request, true, 0)
	require.NoError(t, err)
	replayed, err := opt.Evaluate(context.Background())
	require.NoError(t, err)
	require.Equal(t, response.ProposalEvaluation, replayed)
}

func TestMPUnhelpfulProposalRetainsOriginalPrograms(t *testing.T) {
	request := mpTestRequest()
	var response dto.MPRunResponse
	callCoordinationAPI(t, RequestMPRun(), request, http.StatusOK, &response)
	require.False(t, response.ProposalAccepted)
	require.Equal(t, response.Baseline, response.ProposalEvaluation)
	for i, proposal := range response.Proposal {
		require.Equal(t, request.Intersections[i].Junction, proposal.Junction)
	}
}

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
			name:    "oversized initial queue",
			change:  func(req *dto.MPRunRequest) { req.InitialQueues[1] = 1e300 },
			message: "initial queue",
		},
		{
			name:    "duplicate road",
			change:  func(req *dto.MPRunRequest) { req.Network.Links = append(req.Network.Links, req.Network.Links[0]) },
			message: "duplicate or invalid link",
		},
		{
			name:    "road and movement queues mixed",
			change:  func(req *dto.MPRunRequest) { req.InitialMovementQueues = map[int]float64{100: 2} },
			message: "both road and movement initial queues",
		},
		{
			name:    "internal road demand",
			change:  func(req *dto.MPRunRequest) { req.Demand.Rates = map[int]float64{2: 500} },
			message: "entry-road rate",
		},
		{
			name:    "finite storage overflow",
			change:  func(req *dto.MPRunRequest) { req.Config.StorageModel = "finite_storage"; req.InitialQueues[1] = 100 },
			message: "exceeds storage",
		},
		{
			name: "invalid signal bounds",
			change: func(req *dto.MPRunRequest) {
				value := 30
				req.Intersections[0].Junction.Cycle[0].SignalGroups[0].Signals[0].MinDuration = &value
			},
			message: "invalid signal duration bounds",
		},
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
			message: "no assigned green movement",
		},
		{
			name:    "empty phases",
			change:  func(req *dto.MPRunRequest) { req.Intersections[0].Junction.Cycle = nil },
			message: "must have at least one phase",
		},
		{
			name:    "negative coefficient",
			change:  func(req *dto.MPRunRequest) { req.Config.Alpha = -1 },
			message: "config.alpha must be between 0 and 1",
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

package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/junction"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type corridorExample struct {
	DesiredSpeed    float64                  `json:"desiredSpeed"`
	Junctions       []dto.JunctionDTO        `json:"junctions"`
	GroupIDs        map[int]junction.GroupID `json:"groupIds"`
	ReverseGroupIDs map[int]junction.GroupID `json:"reverseGroupIds"`
}

func exampleRequest(t *testing.T) GreenWavesRequest {
	t.Helper()
	data, err := os.ReadFile("../../web/greenwave-ui/tests/fixtures/bidirectional-turns.json")
	require.NoError(t, err)
	var example corridorExample
	require.NoError(t, json.Unmarshal(data, &example))
	return GreenWavesRequest{Junctions: example.Junctions, DesiredSpeedKmh: example.DesiredSpeed, Direction: "bidirectional", GroupIDs: example.GroupIDs, ReverseGroupIDs: example.ReverseGroupIDs}
}

func coordinatedExampleRequest(t *testing.T) GreenWavesRequest {
	t.Helper()
	request := exampleRequest(t)
	for i, offset := range []int{0, 9, 19, 30} {
		request.Junctions[i].Offset = offset
	}
	return request
}

func TestUncoordinatedExampleStartsWithoutGreenWaves(t *testing.T) {
	var response GreenWavesResponse
	callCoordinationAPI(t, ExtractGreenWaves(), exampleRequest(t), 200, &response)
	for _, direction := range [][][]dto.GreenWaveDTO{response.GreenWaves, response.ReverseGreenWaves} {
		require.Len(t, direction, 3)
		for _, segment := range direction {
			require.Empty(t, segment)
		}
	}
	require.Empty(t, response.ThroughGreenWaves)
	require.Empty(t, response.ReverseThroughGreenWaves)
}

func callCoordinationAPI(t *testing.T, handler echo.HandlerFunc, request interface{}, status int, response interface{}) {
	t.Helper()
	body, err := json.Marshal(request)
	require.NoError(t, err)
	context := echo.New()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	require.NoError(t, handler(context.NewContext(req, recorder)))
	require.Equal(t, status, recorder.Code, recorder.Body.String())
	if response != nil {
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), response))
	}
}

func TestExtractionUsesReverseGroupIDsAndLegacyFallback(t *testing.T) {
	request := coordinatedExampleRequest(t)
	var both GreenWavesResponse
	callCoordinationAPI(t, ExtractGreenWaves(), request, 200, &both)
	require.Len(t, both.ThroughGreenWaves, 1)
	require.Len(t, both.ReverseThroughGreenWaves, 1)
	require.Equal(t, len(request.Junctions), both.ThroughGreenWaves[0].Depth)
	require.Equal(t, len(request.Junctions), both.ReverseThroughGreenWaves[0].Depth)
	require.InDelta(t, 28, both.ThroughGreenWaves[0].Bandwidth, 1e-8)
	require.InDelta(t, 20, both.ReverseThroughGreenWaves[0].Bandwidth, 1e-8)
	request.Direction = "forward"
	request.GroupIDs = request.ReverseGroupIDs
	for left, right := 0, len(request.Junctions)-1; left < right; left, right = left+1, right-1 {
		request.Junctions[left], request.Junctions[right] = request.Junctions[right], request.Junctions[left]
	}
	var reversed GreenWavesResponse
	callCoordinationAPI(t, ExtractGreenWaves(), request, 200, &reversed)
	require.Equal(t, reversed.GreenWaves, both.ReverseGreenWaves)
	require.Equal(t, reversed.ThroughGreenWaves, both.ReverseThroughGreenWaves)
	require.Len(t, both.ReverseGreenWaves, 3)

	request = coordinatedExampleRequest(t)
	request.ReverseGroupIDs = nil
	var legacy GreenWavesResponse
	callCoordinationAPI(t, ExtractGreenWaves(), request, 200, &legacy)
	require.Equal(t, both.GreenWaves, legacy.GreenWaves)
	require.NotEqual(t, both.ReverseGreenWaves, legacy.ReverseGreenWaves)
	request.ReverseGroupIDs = request.GroupIDs
	var explicit GreenWavesResponse
	callCoordinationAPI(t, ExtractGreenWaves(), request, 200, &explicit)
	require.Equal(t, legacy, explicit)
}

func TestCoordinatedExampleHasFullCorridorBandsInBothDirections(t *testing.T) {
	request := exampleRequest(t)
	for i, offset := range []int{0, 9, 19, 30} {
		request.Junctions[i].Offset = offset
	}
	var response GreenWavesResponse
	callCoordinationAPI(t, ExtractGreenWaves(), request, 200, &response)
	require.Len(t, response.ThroughGreenWaves, 1)
	require.Len(t, response.ReverseThroughGreenWaves, 1)
	require.Equal(t, 4, response.ThroughGreenWaves[0].Depth)
	require.Equal(t, 4, response.ReverseThroughGreenWaves[0].Depth)
	require.InDelta(t, 28, response.ThroughGreenWaves[0].Bandwidth, 1e-8)
	require.InDelta(t, 20, response.ReverseThroughGreenWaves[0].Bandwidth, 1e-8)
}

func TestOptimizationIncludesTheCurrentPlan(t *testing.T) {
	for _, origin := range []int{0, 7} {
		input := coordinatedExampleRequest(t)
		for i := range input.Junctions {
			input.Junctions[i].Offset += origin
		}
		request := OptimizeRequest{Junctions: input.Junctions, DesiredSpeedKmh: input.DesiredSpeedKmh, Direction: input.Direction, GroupIDs: input.GroupIDs, ReverseGroupIDs: input.ReverseGroupIDs, OptimizerType: "genetic", OptimizerParams: map[string]interface{}{"population_size": 1, "generations": 2}}
		var optimized OptimizeResponse
		callCoordinationAPI(t, RequestOptimize(), request, 200, &optimized)
		require.Equal(t, []float64{0, 9, 19, 30}, optimized.BestOffsets)
		require.Len(t, optimized.ThroughGreenWaves, 1)
		require.Len(t, optimized.ReverseThroughGreenWaves, 1)
		require.Equal(t, 4, optimized.ThroughGreenWaves[0].Depth)
		require.Equal(t, 4, optimized.ReverseThroughGreenWaves[0].Depth)
	}
}

func TestOptimizationReturnsBothGroupsAtOneSetOfOffsets(t *testing.T) {
	input := exampleRequest(t)
	request := OptimizeRequest{Junctions: input.Junctions, DesiredSpeedKmh: input.DesiredSpeedKmh, Direction: input.Direction, GroupIDs: input.GroupIDs, ReverseGroupIDs: input.ReverseGroupIDs, OptimizerType: "genetic", OptimizerParams: map[string]interface{}{"population_size": 8, "generations": 2, "tournament_size": 2}}
	var optimized OptimizeResponse
	callCoordinationAPI(t, RequestOptimize(), request, 200, &optimized)
	require.Len(t, optimized.BestOffsets, len(input.Junctions))
	for i := range input.Junctions {
		input.Junctions[i].Offset = int(optimized.BestOffsets[i])
	}
	var extracted GreenWavesResponse
	callCoordinationAPI(t, ExtractGreenWaves(), input, 200, &extracted)
	require.Equal(t, extracted.GreenWaves, optimized.GreenWaves)
	require.Equal(t, extracted.ThroughGreenWaves, optimized.ThroughGreenWaves)
	require.Equal(t, extracted.ReverseGreenWaves, optimized.ReverseGreenWaves)
	require.Equal(t, extracted.ReverseThroughGreenWaves, optimized.ReverseThroughGreenWaves)

}

func TestInvalidReverseSelectionsAreRejectedByBothEndpoints(t *testing.T) {
	for _, selections := range []map[int]junction.GroupID{{0: 999}, {99: 10}} {
		request := exampleRequest(t)
		request.ReverseGroupIDs = selections
		callCoordinationAPI(t, ExtractGreenWaves(), request, 400, nil)
		callCoordinationAPI(t, RequestOptimize(), OptimizeRequest{Junctions: request.Junctions, DesiredSpeedKmh: 40, Direction: "bidirectional", GroupIDs: request.GroupIDs, ReverseGroupIDs: selections, OptimizerType: "genetic"}, 400, nil)
	}
	request := exampleRequest(t)
	forward, reverse, err := coordinationGroups(request.Junctions, request.GroupIDs, map[int]junction.GroupID{0: 30})
	require.NoError(t, err)
	require.Equal(t, request.GroupIDs, forward)
	require.Equal(t, map[int]junction.GroupID{0: 30, 1: 10, 2: 10, 3: 10}, reverse)
}

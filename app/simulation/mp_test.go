package simulation_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/app/simulation"
	"github.com/stretchr/testify/require"
)

func exampleMPRequest(t *testing.T) dto.MPRunRequest {
	t.Helper()
	body, err := os.ReadFile("../../examples/maxpressure/corridor.json")
	require.NoError(t, err)
	var request dto.MPRunRequest
	require.NoError(t, json.Unmarshal(body, &request))
	return request
}

func TestRunMaxPressureKeepsInputReusable(t *testing.T) {
	request := exampleMPRequest(t)
	request.Config.DeltaT = 0
	request.Config.StorageModel = ""
	before, err := json.Marshal(request)
	require.NoError(t, err)
	first, err := simulation.RunMaxPressure(context.Background(), request)
	require.NoError(t, err)
	second, err := simulation.RunMaxPressure(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, first, second)
	after, err := json.Marshal(request)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestRunMaxPressurePropagatesCancellation(t *testing.T) {
	request := exampleMPRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response, err := simulation.RunMaxPressure(ctx, request)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, response)
}

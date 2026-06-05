package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/LdDl/greenwave"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/junction"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// OptimizeRequest represents the request structure for optimization requests.
// swagger:model
type OptimizeRequest struct {
	// List of junctions with their phases and signals
	Junctions []dto.JunctionDTO `json:"junctions"`
	// Desired speed in km/h for calculating green waves
	DesiredSpeedKmh float64 `json:"desired_speed_kmh"`
	// Specifies which optimizer to use
	OptimizerType string `json:"optimizer_type"`
	// Contains parameters for the optimizer
	OptimizerParams map[string]interface{} `json:"optimizer_params"`
	// Direction for optimization: "forward" (default) or "bidirectional"
	Direction string `json:"direction"`
	// GroupIDs maps each junction ID to the signal group used for green wave coordination in this corridor.
	// A junction may have multiple signal groups (e.g. northbound, eastbound, pedestrian); only one group represents
	// the through-movement for a given corridor. The caller is responsible for providing the correct group per junction.
	GroupIDs map[int]junction.GroupID `json:"group_ids"`
}

// OptimizeResponse represents the response structure for optimization requests.
// swagger:model
type OptimizeResponse struct {
	// Contains the optimal offsets for each junction
	BestOffsets []float64 `json:"best_offsets"`
	// Additional information about the optimization process
	OptimizerExtra OptimizerExtra `json:"optimizer_extra"`
	// List of segments of green waves between junctions considering the optimal offsets (forward direction).
	// Ordered from first junction to last: [0] = J0->J1, [1] = J1->J2, etc.
	GreenWaves [][]dto.GreenWaveDTO `json:"green_waves"`
	// List of through green waves (forward direction, so they can be passed through multiple junctions).
	// Intervals ordered from first junction to last.
	ThroughGreenWaves []dto.ThroughGreenWaveDTO `json:"through_green_waves"`
	// List of segments of green waves (reverse direction, only for `bidirectional` case).
	// Ordered from last junction to first: [0] = JN->JN-1, [1] = JN-1->JN-2, etc.
	ReverseGreenWaves [][]dto.GreenWaveDTO `json:"reverse_green_waves"`
	// List of through green waves (reverse direction, only for `bidirectional` case).
	// Intervals ordered from last junction to first.
	ReverseThroughGreenWaves []dto.ThroughGreenWaveDTO `json:"reverse_through_green_waves"`
}

// OptimizerExtra contains additional information about the optimization process.
// swagger:model
type OptimizerExtra struct {
	// Contains the fitness evolution over generations
	// Will be represented in case of genetic algorithm
	// Each value is the best fitness of the population in that generation
	FitnessHistory []float64 `json:"fitness_history"`
}

// RequestOptimize return best offsets with green waves for traffic lights configuration.
// @Summary Request optimization
// @Description Requests the optimization of green waves for traffic lights configuration
// @Tags Optimize
// @Produce json
// @Param POST-body body rest.OptimizeRequest true "Traffic lights configuration"
// @Success 200 {object} rest.OptimizeResponse
// @Failure 400 {object} codes.Error400
// @Failure 500 {object} codes.Error500
// @Router /api/greenwave/optimize [POST]
func RequestOptimize() func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		bodyBytes, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			errReason := "Can't read body"
			log.Error().Err(err).Str("scope", "api").Str("method", ctx.Request().Method).Str("route", ctx.Request().URL.Path).RawJSON("req_body", bodyBytes).Msg(errReason)
			return ctx.JSON(400, echo.Map{
				"Error": err.Error(),
			})
		}
		requestData := OptimizeRequest{}
		err = json.Unmarshal(bodyBytes, &requestData)
		if err != nil {
			errReason := "Can't unmarshal request data"
			log.Error().Err(err).Str("scope", "api").Str("method", ctx.Request().Method).Str("route", ctx.Request().URL.Path).RawJSON("req_body", bodyBytes).Msg(errReason)
			return ctx.JSON(400, echo.Map{
				"Error": err.Error(),
			})
		}

		// Validate input
		if len(requestData.Junctions) < 2 {
			return ctx.JSON(400, echo.Map{
				"Error": "At least 2 junctions are required",
			})
		}
		if requestData.DesiredSpeedKmh <= 0 {
			return ctx.JSON(400, echo.Map{
				"Error": "Desired speed must be greater than 0",
			})
		}

		var optimizationMode greenwave.OptimizationMode
		switch strings.ToLower(requestData.Direction) {
		case "", "forward":
			optimizationMode = greenwave.OPTIMIZATION_FORWARD
		case "bidirectional":
			optimizationMode = greenwave.OPTIMIZATION_BIDIRECTIONAL
		default:
			return ctx.JSON(400, echo.Map{
				"Error": "Direction must be either 'forward' or 'bidirectional'",
			})
		}

		// Convert DTOs to domain objects
		junctions := make([]*junction.Junction, len(requestData.Junctions))
		for i, junctionDTO := range requestData.Junctions {
			junctions[i] = dto.JunctionFromDTO(junctionDTO)
		}

		// Create optimizer based on type
		optimizer, err := createOptimizer(requestData.OptimizerType, junctions, requestData.GroupIDs, requestData.DesiredSpeedKmh, requestData.OptimizerParams, optimizationMode)
		if err != nil {
			return ctx.JSON(400, echo.Map{
				"Error": err.Error(),
			})
		}

		// Run optimization
		bestOffsets := optimizer.Optimize()
		// Apply best offsets to junctions
		for i, jun := range junctions {
			jun.SetOffset(int(bestOffsets[i]))
		}
		// Calculate green waves with optimized offsets
		greenWaves := greenwave.FindGreenWaves(junctions, requestData.GroupIDs, requestData.DesiredSpeedKmh)
		throughGreenWaves := greenwave.MergeGreenWaves(greenWaves)

		optimizerExtra := OptimizerExtra{}
		switch opt := optimizer.(type) {
		case *greenwave.OptimizerGenetic:
			optimizerExtra.FitnessHistory = opt.BestFitnessHistory()
		}

		response := OptimizeResponse{
			BestOffsets:              bestOffsets,
			OptimizerExtra:           optimizerExtra,
			GreenWaves:               convertGreenWavesToDTO(greenWaves),
			ThroughGreenWaves:        convertThroughGreenWavesToDTO(throughGreenWaves),
			ReverseGreenWaves:        [][]dto.GreenWaveDTO{},
			ReverseThroughGreenWaves: []dto.ThroughGreenWaveDTO{},
		}

		// If bidirectional, also calculate reverse waves
		if optimizationMode == greenwave.OPTIMIZATION_BIDIRECTIONAL {
			reversedJunctions := greenwave.ReverseJunctions(junctions)
			reverseGreenWaves := greenwave.FindGreenWaves(reversedJunctions, requestData.GroupIDs, requestData.DesiredSpeedKmh)
			reverseThroughGreenWaves := greenwave.MergeGreenWaves(reverseGreenWaves)

			response.ReverseGreenWaves = convertGreenWavesToDTO(reverseGreenWaves)
			response.ReverseThroughGreenWaves = convertThroughGreenWavesToDTO(reverseThroughGreenWaves)
		}

		return ctx.JSON(200, response)
	}
}

// createOptimizer creates an optimizer based on the specified type and parameters.
// groupIDs maps each junction ID to the signal group used for green wave coordination in this corridor.
func createOptimizer(optimizerType string, junctions []*junction.Junction, groupIDs map[int]junction.GroupID, speedKmh float64, params map[string]interface{}, optimizationMode greenwave.OptimizationMode) (greenwave.Optimizer, error) {
	switch strings.ToLower(optimizerType) {
	case "genetic":
		return createGeneticOptimizer(junctions, groupIDs, speedKmh, params, optimizationMode)
	default:
		return nil, fmt.Errorf("unsupported optimizer type: %s", optimizerType)
	}
}

// createGeneticOptimizer creates a genetic algorithm optimizer with flexible parameters.
// groupIDs maps each junction ID to the signal group used for green wave coordination in this corridor.
func createGeneticOptimizer(junctions []*junction.Junction, groupIDs map[int]junction.GroupID, speedKmh float64, params map[string]interface{}, optimizationMode greenwave.OptimizationMode) (greenwave.Optimizer, error) {
	// Helper function to get parameter with default value
	getParam := func(key string, defaultValue interface{}) interface{} {
		if val, exists := params[key]; exists {
			return val
		}
		return defaultValue
	}

	// Helper function to convert interface{} to specific types with validation
	getIntParam := func(key string, defaultValue int) (int, error) {
		val := getParam(key, defaultValue)
		switch v := val.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		default:
			return defaultValue, nil
		}
	}

	getFloatParam := func(key string, defaultValue float64) (float64, error) {
		val := getParam(key, defaultValue)
		switch v := val.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		default:
			return defaultValue, nil
		}
	}

	getStringParam := func(key string, defaultValue string) string {
		val := getParam(key, defaultValue)
		if str, ok := val.(string); ok {
			return str
		}
		return defaultValue
	}

	// Extract parameters with defaults
	populationSize, err := getIntParam("population_size", 50)
	if err != nil {
		return nil, fmt.Errorf("invalid population_size parameter: %v", err)
	}

	generations, err := getIntParam("generations", 100)
	if err != nil {
		return nil, fmt.Errorf("invalid generations parameter: %v", err)
	}

	mutationRate, err := getFloatParam("mutation_rate", 0.1)
	if err != nil {
		return nil, fmt.Errorf("invalid mutation_rate parameter: %v", err)
	}

	tournamentSize, err := getIntParam("tournament_size", 3)
	if err != nil {
		return nil, fmt.Errorf("invalid tournament_size parameter: %v", err)
	}

	crossoverTypeStr := getStringParam("crossover_type", "blend")

	// Parse crossover type
	var crossoverType greenwave.CrossoverType
	switch strings.ToLower(crossoverTypeStr) {
	case "uniform":
		crossoverType = greenwave.CROSSOVER_UNIFORM
	case "blend":
		crossoverType = greenwave.CROSSOVER_BLEND
	default:
		return nil, fmt.Errorf("unsupported crossover type: %s", crossoverTypeStr)
	}

	// Validate parameters
	if populationSize <= 0 {
		return nil, fmt.Errorf("population_size must be greater than 0")
	}
	if generations <= 0 {
		return nil, fmt.Errorf("generations must be greater than 0")
	}
	if mutationRate < 0 || mutationRate > 1 {
		return nil, fmt.Errorf("mutation_rate must be between 0 and 1")
	}
	if tournamentSize <= 0 {
		return nil, fmt.Errorf("tournament_size must be greater than 0")
	}

	return greenwave.NewOptimizerGenetic(
		junctions,
		groupIDs,
		speedKmh,
		populationSize,
		generations,
		mutationRate,
		tournamentSize,
		crossoverType,
		optimizationMode,
	), nil
}

package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/LdDl/greenwave"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// OptimizeRequest represents the request structure for optimization requests.
// swagger:model
type OptimizeRequest struct {
	// Junctions is a list of junctions with their phases and signals
	Junctions []dto.JunctionDTO `json:"junctions"`
	// DesiredSpeedKmh is the desired speed in km/h for calculating green waves
	DesiredSpeedKmh float64 `json:"desired_speed_kmh"`
	// OptimizerType specifies which optimizer to use
	OptimizerType string `json:"optimizer_type"`
	// OptimizerParams contains parameters for the optimizer
	OptimizerParams map[string]interface{} `json:"optimizer_params"`
}

// OptimizeResponse represents the response structure for optimization requests.
// swagger:model
type OptimizeResponse struct {
	// BestOffsets contains the optimal offsets for each junction
	BestOffsets []float64 `json:"best_offsets"`
	// Additional information about the optimization process
	OptimizerExtra OptimizerExtra `json:"optimizer_extra"`
	// GreenWaves is the green waves calculated with optimal offsets
	GreenWaves [][]dto.GreenWaveDTO `json:"green_waves"`
	// ThroughGreenWaves is the through green waves with optimal offsets
	ThroughGreenWaves []dto.ThroughGreenWaveDTO `json:"through_green_waves"`
}

// OptimizerExtra contains additional information about the optimization process.
// swagger:model
type OptimizerExtra struct {
	// FitnessHistory contains the fitness evolution over generations
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

		// Convert DTOs to domain objects
		junctions := make([]*greenwave.Junction, len(requestData.Junctions))
		for i, junctionDTO := range requestData.Junctions {
			junctions[i] = dto.JunctionFromDTO(junctionDTO)
		}

		// Create optimizer based on type
		optimizer, err := createOptimizer(requestData.OptimizerType, junctions, requestData.DesiredSpeedKmh, requestData.OptimizerParams)
		if err != nil {
			return ctx.JSON(400, echo.Map{
				"Error": err.Error(),
			})
		}

		// Run optimization
		bestOffsets := optimizer.Optimize()
		// Apply best offsets to junctions
		for i, junction := range junctions {
			junction.SetOffset(int(bestOffsets[i]))
		}
		// Calculate green waves with optimized offsets
		greenWaves := greenwave.FindGreenWaves(junctions, requestData.DesiredSpeedKmh)
		throughGreenWaves := greenwave.MergeGreenWaves(greenWaves)

		optimizerExtra := OptimizerExtra{}
		switch opt := optimizer.(type) {
		case *greenwave.OptimizerGenetic:
			optimizerExtra.FitnessHistory = opt.BestFitnessHistory()
		}

		response := OptimizeResponse{
			BestOffsets:       bestOffsets,
			OptimizerExtra:    optimizerExtra,
			GreenWaves:        convertGreenWavesToDTO(greenWaves),
			ThroughGreenWaves: convertThroughGreenWavesToDTO(throughGreenWaves),
		}

		return ctx.JSON(200, response)
	}
}

// createOptimizer creates an optimizer based on the specified type and parameters
func createOptimizer(optimizerType string, junctions []*greenwave.Junction, speedKmh float64, params map[string]interface{}) (greenwave.Optimizer, error) {
	switch strings.ToLower(optimizerType) {
	case "genetic":
		return createGeneticOptimizer(junctions, speedKmh, params)
	default:
		return nil, fmt.Errorf("unsupported optimizer type: %s", optimizerType)
	}
}

// createGeneticOptimizer creates a genetic algorithm optimizer with flexible parameters
func createGeneticOptimizer(junctions []*greenwave.Junction, speedKmh float64, params map[string]interface{}) (greenwave.Optimizer, error) {
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

	fmt.Println("Creating Genetic Optimizer with parameters:")
	fmt.Printf("  Population Size: %d\n", populationSize)
	fmt.Printf("  Generations: %d\n", generations)
	fmt.Printf("  Mutation Rate: %.2f\n", mutationRate)
	fmt.Printf("  Tournament Size: %d\n", tournamentSize)
	fmt.Printf("  Crossover Type: %s\n", crossoverTypeStr)

	return greenwave.NewOptimizerGenetic(
		junctions,
		speedKmh,
		populationSize,
		generations,
		mutationRate,
		tournamentSize,
		crossoverType,
	), nil
}

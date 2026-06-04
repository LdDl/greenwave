package rest

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/LdDl/greenwave"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/junction"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// GreenWavesRequest represents the request structure for green waves requests.
// swagger:model
type GreenWavesRequest struct {
	// List of junctions with their phases and signals
	Junctions []dto.JunctionDTO `json:"junctions"`
	// Desired speed in km/h for calculating green waves
	DesiredSpeedKmh float64 `json:"desired_speed_kmh"`
	// Direction for green wave calculation: "forward" (default) or "bidirectional"
	Direction string `json:"direction"`
}

// GreenWavesResponse represents the response structure for green waves requests.
// swagger:model
type GreenWavesResponse struct {
	// List of segments of green waves between junctions (forward direction).
	// Ordered from first junction to last: [0] = J0->J1, [1] = J1->J2, etc.
	GreenWaves [][]dto.GreenWaveDTO `json:"green_waves"`
	// List of through green waves (forward direction).
	// Intervals ordered from first junction to last.
	ThroughGreenWaves []dto.ThroughGreenWaveDTO `json:"through_green_waves"`
	// List of segments of green waves (reverse direction, only for `bidirectional` case).
	// Ordered from last junction to first: [0] = JN->JN-1, [1] = JN-1->JN-2, etc.
	ReverseGreenWaves [][]dto.GreenWaveDTO `json:"reverse_green_waves"`
	// List of through green waves (reverse direction, only for `bidirectional` case).
	// Intervals ordered from last junction to first.
	ReverseThroughGreenWaves []dto.ThroughGreenWaveDTO `json:"reverse_through_green_waves"`
}

// ExtractGreenWaves returns green waves for traffic lights configuration.
// @Summary Extract green waves
// @Description Requests the calculation of green waves for traffic lights configuration
// @Tags Reference
// @Produce json
// @Param POST-body body rest.GreenWavesRequest true "Traffic lights configuration"
// @Success 200 {object} rest.GreenWavesResponse
// @Failure 400 {object} codes.Error400
// @Failure 500 {object} codes.Error500
// @Router /api/greenwave/extract [POST]
func ExtractGreenWaves() func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		bodyBytes, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			errReason := "Can't read body"
			log.Error().Err(err).Str("scope", "api").Str("method", ctx.Request().Method).Str("route", ctx.Request().URL.Path).RawJSON("req_body", bodyBytes).Msg(errReason)
			return ctx.JSON(400, echo.Map{
				"Error": err,
			})
		}
		requestData := GreenWavesRequest{}
		err = json.Unmarshal(bodyBytes, &requestData)
		if err != nil {
			errReason := "Can't unmarshal request data"
			log.Error().Err(err).Str("scope", "api").Str("method", ctx.Request().Method).Str("route", ctx.Request().URL.Path).RawJSON("req_body", bodyBytes).Msg(errReason)
			return ctx.JSON(400, echo.Map{
				"Error": err,
			})
		}

		if len(requestData.Junctions) < 2 {
			return ctx.JSON(400, echo.Map{
				"Error": "At least 2 junctions are required",
			})
		}

		junctions := make([]*junction.Junction, len(requestData.Junctions))
		for i, junctionDTO := range requestData.Junctions {
			junctions[i] = dto.JunctionFromDTO(junctionDTO)
		}

		// Extract forward green waves
		greenWaves := greenwave.FindGreenWaves(junctions, requestData.DesiredSpeedKmh)
		throughGreenWaves := greenwave.MergeGreenWaves(greenWaves)

		response := GreenWavesResponse{
			GreenWaves:               convertGreenWavesToDTO(greenWaves),
			ThroughGreenWaves:        convertThroughGreenWavesToDTO(throughGreenWaves),
			ReverseGreenWaves:        [][]dto.GreenWaveDTO{},
			ReverseThroughGreenWaves: []dto.ThroughGreenWaveDTO{},
		}

		// If bidirectional, also calculate reverse waves
		if strings.ToLower(requestData.Direction) == "bidirectional" {
			reversedJunctions := greenwave.ReverseJunctions(junctions)
			reverseGreenWaves := greenwave.FindGreenWaves(reversedJunctions, requestData.DesiredSpeedKmh)
			reverseThroughGreenWaves := greenwave.MergeGreenWaves(reverseGreenWaves)

			response.ReverseGreenWaves = convertGreenWavesToDTO(reverseGreenWaves)
			response.ReverseThroughGreenWaves = convertThroughGreenWavesToDTO(reverseThroughGreenWaves)
		}

		return ctx.JSON(200, response)
	}
}

// convertGreenWavesToDTO converts a slice of GreenWave to GreenWaveDTO
func convertGreenWavesToDTO(greenWaves [][]*greenwave.GreenWave) [][]dto.GreenWaveDTO {
	result := make([][]dto.GreenWaveDTO, len(greenWaves))
	for i, segment := range greenWaves {
		result[i] = make([]dto.GreenWaveDTO, len(segment))
		for j, wave := range segment {
			result[i][j] = dto.GreenWaveToDTO(wave)
		}
	}
	return result
}

// convertThroughGreenWavesToDTO converts a slice of ThroughGreenWave to ThroughGreenWaveDTO
func convertThroughGreenWavesToDTO(throughWaves []*greenwave.ThroughGreenWave) []dto.ThroughGreenWaveDTO {
	result := make([]dto.ThroughGreenWaveDTO, len(throughWaves))
	for i, wave := range throughWaves {
		result[i] = dto.ThroughGreenWaveToDTO(wave)
	}
	return result
}

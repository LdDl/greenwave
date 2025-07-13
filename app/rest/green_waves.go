package rest

import (
	"encoding/json"
	"io"

	"github.com/LdDl/greenwave"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// GreenWavesRequest represents the request structure for green waves requests.
// swagger:model
type GreenWavesRequest struct {
	// Junctions is a list of junctions with their phases and signals
	Junctions []dto.JunctionDTO `json:"junctions"`
	// DesiredSpeedKmh is the desired speed in km/h for calculating green waves
	DesiredSpeedKmh float64 `json:"desired_speed_kmh"`
}

// GreenWavesResponse represents the response structure for green waves requests.
// swagger:model
type GreenWavesResponse struct {
	// GreenWaves is a list of segments of green waves between junctions
	GreenWaves [][]dto.GreenWaveDTO `json:"green_waves"`
	// ThroughGreenWaves is a list of through green waves (so they can be passed through multiple junctions)
	ThroughGreenWaves []dto.ThroughGreenWaveDTO `json:"through_green_waves"`
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

		junctions := make([]*greenwave.Junction, len(requestData.Junctions))
		for i, junctionDTO := range requestData.Junctions {
			junctions[i] = dto.JunctionFromDTO(junctionDTO)
		}

		// Extract green waves
		greenWaves := greenwave.FindGreenWaves(junctions, requestData.DesiredSpeedKmh)
		throughGreenWaves := greenwave.MergeGreenWaves(greenWaves)

		response := GreenWavesResponse{
			GreenWaves:        convertGreenWavesToDTO(greenWaves),
			ThroughGreenWaves: convertThroughGreenWavesToDTO(throughGreenWaves),
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

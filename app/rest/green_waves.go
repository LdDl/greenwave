package rest

import "github.com/labstack/echo/v4"

// GreenWavesRequest represents the request structure for green waves requests.
// swagger:model
type GreenWavesRequest struct {
}

// GreenWavesResponse represents the response structure for green waves requests.
// swagger:model
type GreenWavesResponse struct {
}

// RequestGreenWaves returns green waves for traffic lights configuration.
// @Summary Extract green waves
// @Description Requests the calculation of green waves for traffic lights configuration
// @Tags Reference
// @Produce json
// @Param POST-body body rest.GreenWavesRequest true "Traffic lights configuration"
// @Success 200 {object} rest.GreenWavesResponse
// @Failure 400 {object} codes.Error400
// @Failure 500 {object} codes.Error500
// @Router /api/greenwave/calculate_waves [POST]
func RequestGreenWaves() func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		response := GreenWavesResponse{}
		return ctx.JSON(200, response)
	}
}

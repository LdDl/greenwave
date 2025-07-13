package rest

import "github.com/labstack/echo/v4"

// OptimizeRequest represents the request structure for optimization requests.
// swagger:model
type OptimizeRequest struct {
}

// OptimizeResponse represents the response structure for optimization requests.
// swagger:model
type OptimizeResponse struct {
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
		response := OptimizeResponse{}
		return ctx.JSON(200, response)
	}
}

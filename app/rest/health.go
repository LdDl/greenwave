package rest

import (
	"github.com/labstack/echo/v4"
)

// HealthResponse represents the response structure for health checks.
// swagger:model
type HealthResponse struct {
	// Message contains the health status message.
	Message string `json:"message"`
}

// GetHealth returns the health status of the microservice.
// @Summary Get health status
// @Description Returns a simple message indicating the microservice is running
// @Tags Health
// @Produce json
// @Success 200 {object} rest.HealthResponse
// @Failure 400 {object} codes.Error400
// @Failure 500 {object} codes.Error500
// @Router /api/greenwave/health [GET]
func GetHealth() func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		response := HealthResponse{
			Message: "Microservice is running",
		}
		return ctx.JSON(200, response)
	}
}

package rest

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/app/simulation"
	"github.com/labstack/echo/v4"
)

// RequestMPRun compares the fixed baseline, ordinary MP, coordinated MP and a
// separately simulated fixed timing proposal under identical traffic inputs.
//
// @Summary Compare movement-queue max-pressure controllers
// @Description Model version 2 uses explicit turning proportions and per-movement queues. Adaptive and fixed-plan metrics are evaluated separately.
// @Tags MaxPressure
// @Accept json
// @Produce json
// @Param POST-body body dto.MPRunRequest true "MP simulation request"
// @Success 200 {object} dto.MPRunResponse
// @Failure 400 {object} codes.Error400
// @Failure 413 {object} codes.Error400
// @Router /api/maxpressure/run [POST]
func RequestMPRun() func(echo.Context) error {
	return func(ctx echo.Context) error {
		const bodyLimit = 8 << 20
		body, err := io.ReadAll(io.LimitReader(ctx.Request().Body, bodyLimit+1))
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, echo.Map{"Error": err.Error()})
		}
		if len(body) > bodyLimit {
			return ctx.JSON(http.StatusRequestEntityTooLarge, echo.Map{"Error": "request exceeds 8 MiB"})
		}
		var request dto.MPRunRequest
		if err := json.Unmarshal(body, &request); err != nil {
			return ctx.JSON(http.StatusBadRequest, echo.Map{"Error": err.Error()})
		}
		response, err := simulation.RunMaxPressure(ctx.Request().Context(), request)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, echo.Map{"Error": err.Error()})
		}
		return ctx.JSON(http.StatusOK, response)
	}
}

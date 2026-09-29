package rest

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/maxpressure"
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
		var req dto.MPRunRequest
		if err = json.Unmarshal(body, &req); err != nil {
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		if err = validateMPRequest(&req); err != nil {
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		adaptive, junctions, err := buildMPRun(req, false, req.Config.Alpha)
		if err != nil {
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		evaluation, err := adaptive.Evaluate(ctx.Request().Context())
		if err != nil {
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		baselineOpt, _, err := buildMPRun(req, true, 0)
		if err != nil {
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		baseline, err := baselineOpt.Evaluate(ctx.Request().Context())
		if err != nil {
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		standard := evaluation
		if req.Config.Alpha > 0 {
			standardOpt, _, buildErr := buildMPRun(req, false, 0)
			if buildErr != nil {
				return ctx.JSON(400, echo.Map{"Error": buildErr.Error()})
			}
			standard, err = standardOpt.Evaluate(ctx.Request().Context())
			if err != nil {
				return ctx.JSON(400, echo.Map{"Error": err.Error()})
			}
		}
		weights := make(map[gmns.NodeID]map[int]float64)
		for _, stats := range evaluation.PerIntersection {
			weights[gmns.NodeID(stats.NodeID)] = stats.PhaseSeconds
		}
		candidate := req
		candidate.Intersections = append([]dto.MPIntersectionConfigDTO(nil), req.Intersections...)
		proposals := []dto.MPProposalDTO{}
		for i, input := range candidate.Intersections {
			node := gmns.NodeID(input.MacroNodeID)
			program, synthErr := maxpressure.SynthesizeProgram(junctions[node], weights[node])
			if synthErr != nil {
				return ctx.JSON(400, echo.Map{"Error": synthErr.Error()})
			}
			updated := dto.JunctionToDTO(program)
			candidate.Intersections[i].Junction = updated
			proposals = append(proposals, dto.MPProposalDTO{MacroNodeID: int(node), Junction: updated})
		}
		proposedOpt, _, err := buildMPRun(candidate, true, 0)
		if err != nil {
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		proposed, err := proposedOpt.Evaluate(ctx.Request().Context())
		if err != nil {
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		accepted := proposed.TotalDelayVehS < baseline.TotalDelayVehS-1e-6 && proposed.Balance.Departed >= baseline.Balance.Departed-1e-6
		warnings := []string{"Queueing simulation: road travel time and individual vehicle trajectories are not modeled."}
		if !accepted {
			proposed = baseline
			proposals = nil
			for _, input := range req.Intersections {
				proposals = append(proposals, dto.MPProposalDTO{MacroNodeID: input.MacroNodeID, Junction: input.Junction})
			}
			warnings = append(warnings, "The synthesized fixed plan did not improve waiting time without reducing departures; the original programs were retained.")
		}
		return ctx.JSON(200, dto.MPRunResponse{ModelVersion: 2, Evaluation: evaluation, Baseline: baseline, StandardMP: standard, ProposalEvaluation: proposed, Proposal: proposals, ProposalAccepted: accepted, Warnings: warnings})
	}
}

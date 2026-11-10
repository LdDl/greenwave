package simulation

import (
	"context"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/maxpressure"
)

// RunMaxPressure compares controllers and evaluates a fixed timing proposal
// from independent copies of the same traffic scenario. It does not use HTTP.
func RunMaxPressure(ctx context.Context, req dto.MPRunRequest) (*dto.MPRunResponse, error) {
	if err := validateMPRequest(&req); err != nil {
		return nil, err
	}
	adaptive, junctions, err := buildMPRun(req, false, req.Config.Alpha)
	if err != nil {
		return nil, err
	}
	evaluation, err := adaptive.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	baselineOpt, _, err := buildMPRun(req, true, 0)
	if err != nil {
		return nil, err
	}
	baseline, err := baselineOpt.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	standard := evaluation
	if req.Config.Alpha > 0 {
		standardOpt, _, buildErr := buildMPRun(req, false, 0)
		if buildErr != nil {
			return nil, buildErr
		}
		standard, err = standardOpt.Evaluate(ctx)
		if err != nil {
			return nil, err
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
			return nil, synthErr
		}
		updated := dto.JunctionToDTO(program)
		candidate.Intersections[i].Junction = updated
		proposals = append(proposals, dto.MPProposalDTO{MacroNodeID: int(node), Junction: updated})
	}
	proposedOpt, _, err := buildMPRun(candidate, true, 0)
	if err != nil {
		return nil, err
	}
	proposed, err := proposedOpt.Evaluate(ctx)
	if err != nil {
		return nil, err
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
	return &dto.MPRunResponse{
		ModelVersion:       2,
		Evaluation:         evaluation,
		Baseline:           baseline,
		StandardMP:         standard,
		ProposalEvaluation: proposed,
		Proposal:           proposals,
		ProposalAccepted:   accepted,
		Warnings:           warnings,
	}, nil
}

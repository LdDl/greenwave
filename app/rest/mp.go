package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/greenwave/app/rest/dto"
	"github.com/LdDl/greenwave/junction"
	"github.com/LdDl/greenwave/maxpressure"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

const defaultMPDeltaT = 5.0

// RequestMPRun runs a max-pressure simulation and returns evaluation metrics
// together with a proposal for redistributed green times.
//
// @Summary Run max-pressure simulation
// @Description Runs a Smoothing-MP simulation on a meso network and returns
// @Description evaluation metrics and a synthesized signal timing proposal.
// @Tags MaxPressure
// @Accept json
// @Produce json
// @Param POST-body body dto.MPRunRequest true "MP simulation request"
// @Success 200 {object} dto.MPRunResponse
// @Failure 400 {object} codes.Error400
// @Failure 500 {object} codes.Error500
// @Router /api/maxpressure/run [POST]
func RequestMPRun() func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		bodyBytes, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			log.Error().Err(err).Str("scope", "api").Str("route", ctx.Request().URL.Path).Msg("Can't read body")
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}
		var req dto.MPRunRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			log.Error().Err(err).Str("scope", "api").Str("route", ctx.Request().URL.Path).Msg("Can't unmarshal request")
			return ctx.JSON(400, echo.Map{"Error": err.Error()})
		}

		if len(req.Network.Links) == 0 {
			return ctx.JSON(400, echo.Map{"Error": "network.links must not be empty"})
		}
		if len(req.Intersections) == 0 {
			return ctx.JSON(400, echo.Map{"Error": "intersections must not be empty"})
		}
		if req.Config.SimTime <= 0 {
			return ctx.JSON(400, echo.Map{"Error": "config.sim_time must be greater than 0"})
		}
		deltaT := req.Config.DeltaT
		if deltaT <= 0 {
			deltaT = defaultMPDeltaT
		}

		// Build meso network
		mesoNet := dto.MesoNetFromDTO(req.Network)

		net := maxpressure.NewNetwork(mesoNet)

		// Set initial queues
		for linkID, q := range req.InitialQueues {
			net.Queues[gmns.LinkID(linkID)] = q
		}

		// Build intersections from junction configs
		for _, intCfg := range req.Intersections {
			jun := dto.JunctionFromDTO(intCfg.Junction)
			groupConnectors := make(map[junction.GroupID][]gmns.LinkID, len(intCfg.GroupConnectors))
			for gidInt, connIDs := range intCfg.GroupConnectors {
				linkIDs := make([]gmns.LinkID, len(connIDs))
				for i, cid := range connIDs {
					linkIDs[i] = gmns.LinkID(cid)
				}
				groupConnectors[junction.GroupID(gidInt)] = linkIDs
			}
			stages := maxpressure.StagesFromJunction(jun, groupConnectors)
			net.Intersections[gmns.NodeID(intCfg.MacroNodeID)] = &maxpressure.IntersectionState{
				MacroNodeID: gmns.NodeID(intCfg.MacroNodeID),
				Stages:      stages,
			}
		}

		// Build demand function
		var demandFn maxpressure.DemandFunc
		switch strings.ToLower(req.Demand.Type) {
		case "", "constant":
			if len(req.Demand.Rates) > 0 {
				rates := make(map[gmns.LinkID]float64, len(req.Demand.Rates))
				for linkID, rate := range req.Demand.Rates {
					rates[gmns.LinkID(linkID)] = rate
				}
				demandFn = maxpressure.ConstantDemand(rates)
			}
		case "peak":
			baseRates := make(map[gmns.LinkID]float64, len(req.Demand.BaseRates))
			for linkID, rate := range req.Demand.BaseRates {
				baseRates[gmns.LinkID(linkID)] = rate
			}
			demandFn = maxpressure.PeakDemand(maxpressure.PeakDemandConfig{
				BaseRates:     baseRates,
				RampUpS:       req.Demand.RampUpS,
				PeakFactor:    req.Demand.PeakFactor,
				PeakDurationS: req.Demand.PeakDurationS,
				RampDownS:     req.Demand.RampDownS,
			})
		default:
			return ctx.JSON(400, echo.Map{"Error": fmt.Sprintf("unsupported demand type: %s", req.Demand.Type)})
		}

		// Build drain function
		var drainFn maxpressure.DrainFunc
		switch strings.ToLower(req.Drain.Type) {
		case "", "auto":
			// nil = auto drain (zero queue at boundary exits)
		case "none":
			drainFn = maxpressure.NoDrain()
		case "rate":
			rates := make(map[gmns.LinkID]float64, len(req.Drain.Rates))
			for linkID, rate := range req.Drain.Rates {
				rates[gmns.LinkID(linkID)] = rate
			}
			drainFn = maxpressure.RateDrain(rates)
		default:
			return ctx.JSON(400, echo.Map{"Error": fmt.Sprintf("unsupported drain type: %s", req.Drain.Type)})
		}

		opt := maxpressure.NewMPOptimizer(net, maxpressure.MPConfig{
			DeltaT:    deltaT,
			SimTime:   req.Config.SimTime,
			Smoothing: maxpressure.SmoothingConfig{Alpha: req.Config.Alpha},
			Demand:    demandFn,
			Drain:     drainFn,
		})

		// Run simulation and collect metrics
		steps := int(req.Config.SimTime / deltaT)

		linkSumQueue := make(map[gmns.LinkID]float64)
		linkMaxQueue := make(map[gmns.LinkID]float64)
		stageCounts := make(map[gmns.NodeID]map[maxpressure.StageID]int, len(net.Intersections))
		for nid := range net.Intersections {
			stageCounts[nid] = make(map[maxpressure.StageID]int)
		}
		totalDelay := 0.0

		for range steps {
			stepResults := opt.Step()
			// Accumulate queue stats for road links only
			for linkID, link := range mesoNet.Links {
				if link.IsConnection() {
					continue
				}
				q := net.Queues[linkID]
				linkSumQueue[linkID] += q
				if q > linkMaxQueue[linkID] {
					linkMaxQueue[linkID] = q
				}
				totalDelay += q * deltaT
			}
			// Count stage selections per intersection
			for _, r := range stepResults {
				stageCounts[r.IntersectionID][r.SelectedStage]++
			}
		}

		// Build per-link stats
		perLink := make([]dto.MPLinkStatsDTO, 0, len(linkSumQueue))
		for linkID := range linkSumQueue {
			avg := 0.0
			if steps > 0 {
				avg = linkSumQueue[linkID] / float64(steps)
			}
			perLink = append(perLink, dto.MPLinkStatsDTO{
				LinkID:      int(linkID),
				AvgQueueVeh: avg,
				MaxQueueVeh: linkMaxQueue[linkID],
			})
		}

		// Build per-intersection stats
		perIntersection := make([]dto.MPIntersectionStatsDTO, 0, len(stageCounts))
		for nid, counts := range stageCounts {
			fractions := make(map[int]float64, len(counts))
			for stageID, count := range counts {
				fractions[int(stageID)] = float64(count) / float64(steps)
			}
			perIntersection = append(perIntersection, dto.MPIntersectionStatsDTO{
				MacroNodeID:    int(nid),
				StageFractions: fractions,
			})
		}

		// Synthesize proposal for each intersection
		proposals := make([]dto.MPProposalDTO, 0, len(req.Intersections))
		for _, intCfg := range req.Intersections {
			nid := gmns.NodeID(intCfg.MacroNodeID)
			counts := stageCounts[nid]
			jun := dto.JunctionFromDTO(intCfg.Junction)
			proposedJun := maxpressure.SynthesizeProposal(jun, counts, steps)
			proposals = append(proposals, dto.MPProposalDTO{
				MacroNodeID: intCfg.MacroNodeID,
				Junction:    dto.JunctionToDTO(proposedJun),
			})
		}

		return ctx.JSON(200, dto.MPRunResponse{
			Evaluation: dto.MPEvaluationDTO{
				TotalDelayVehS:  totalDelay,
				PerLink:         perLink,
				PerIntersection: perIntersection,
			},
			Proposal: proposals,
		})
	}
}


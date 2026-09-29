package maxpressure

import (
	"context"
	"fmt"
	"math"

	"github.com/LdDl/go-gmns/gmns"
)

type QueueStats struct {
	ID      int     `json:"id"`
	Average float64 `json:"avg_queue_veh"`
	Maximum float64 `json:"max_queue_veh"`
	Final   float64 `json:"final_queue_veh"`
}

type IntersectionStats struct {
	NodeID         int                 `json:"macro_node_id"`
	StageFractions map[StageID]float64 `json:"stage_fractions"`
	PhaseSeconds   map[int]float64     `json:"phase_green_seconds"`
	ClearanceS     float64             `json:"clearance_s"`
}

type SimulationSample struct {
	TimeS           float64 `json:"time_s"`
	Queue           float64 `json:"queue_veh"`
	BoundaryBacklog float64 `json:"boundary_backlog_veh"`
	Departed        float64 `json:"departed_veh"`
}

type Evaluation struct {
	DurationS        float64             `json:"duration_s"`
	TotalDelayVehS   float64             `json:"total_delay_veh_s"`
	BoundaryWaitVehS float64             `json:"boundary_wait_veh_s"`
	Balance          VehicleBalance      `json:"vehicle_balance"`
	PerLink          []QueueStats        `json:"per_link"`
	PerMovement      []QueueStats        `json:"per_movement"`
	PerIntersection  []IntersectionStats `json:"per_intersection"`
	Samples          []SimulationSample  `json:"samples"`
}

// Evaluate integrates start-of-step queue occupancy, including external backlog.
// This is point-queue waiting time, not a microscopic travel-time estimate.
func (opt *MPOptimizer) Evaluate(ctx context.Context) (*Evaluation, error) {
	if opt.time != 0 {
		return nil, fmt.Errorf("evaluation requires a fresh optimizer")
	}
	e := &Evaluation{}
	net := opt.Net
	roadStats := make(map[gmns.LinkID]*QueueStats)
	movementStats := make(map[gmns.LinkID]*QueueStats)
	intersectionStats := make(map[gmns.NodeID]*IntersectionStats)
	for _, id := range net.roadIDs {
		roadStats[id] = &QueueStats{ID: int(id)}
	}
	for _, id := range net.movementIDs {
		movementStats[id] = &QueueStats{ID: int(id)}
	}
	for _, id := range net.nodeIDs {
		intersectionStats[id] = &IntersectionStats{NodeID: int(id), StageFractions: map[StageID]float64{}, PhaseSeconds: map[int]float64{}}
	}
	e.Samples = append(e.Samples, opt.sample())
	stride := math.Max(opt.Config.DeltaT, opt.Config.SimTime/200)
	nextSample := stride
	for opt.time < opt.Config.SimTime-1e-9 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		dt := math.Min(opt.Config.DeltaT, opt.Config.SimTime-opt.time)
		for _, id := range net.roadIDs {
			q := net.Queues[id]
			roadStats[id].Average += q * dt
			roadStats[id].Maximum = math.Max(roadStats[id].Maximum, q)
			e.TotalDelayVehS += q * dt
		}
		for _, id := range net.movementIDs {
			q := net.MovementQueues[id]
			movementStats[id].Average += q * dt
			movementStats[id].Maximum = math.Max(movementStats[id].Maximum, q)
		}
		waiting := net.BacklogTotal() * dt
		e.BoundaryWaitVehS += waiting
		e.TotalDelayVehS += waiting
		results := opt.step(dt)
		if opt.err != nil {
			return nil, opt.err
		}
		for _, result := range results {
			stats := intersectionStats[result.IntersectionID]
			if result.SelectedStage == NoStage {
				stats.ClearanceS += result.DurationS
				continue
			}
			stats.StageFractions[result.SelectedStage] += result.DurationS
			for _, stage := range net.Intersections[result.IntersectionID].Stages {
				if stage.ID == result.SelectedStage {
					stats.PhaseSeconds[stage.PhaseID] += result.DurationS
					break
				}
			}
		}
		if opt.time >= nextSample-1e-9 || opt.time >= opt.Config.SimTime-1e-9 {
			e.Samples = append(e.Samples, opt.sample())
			nextSample = opt.time + stride
		}
	}
	e.DurationS = opt.time
	e.Balance = opt.Balance()
	for _, id := range net.roadIDs {
		stats := roadStats[id]
		if e.DurationS > 0 {
			stats.Average /= e.DurationS
		}
		stats.Final = net.Queues[id]
		stats.Maximum = math.Max(stats.Maximum, stats.Final)
		e.PerLink = append(e.PerLink, *stats)
	}
	for _, id := range net.movementIDs {
		stats := movementStats[id]
		if e.DurationS > 0 {
			stats.Average /= e.DurationS
		}
		stats.Final = net.MovementQueues[id]
		stats.Maximum = math.Max(stats.Maximum, stats.Final)
		e.PerMovement = append(e.PerMovement, *stats)
	}
	for _, id := range net.nodeIDs {
		stats := intersectionStats[id]
		for stage := range stats.StageFractions {
			if e.DurationS > 0 {
				stats.StageFractions[stage] /= e.DurationS
			}
		}
		e.PerIntersection = append(e.PerIntersection, *stats)
	}
	return e, nil
}

func (opt *MPOptimizer) sample() SimulationSample {
	return SimulationSample{
		TimeS:           opt.time,
		Queue:           opt.Net.TotalQueueLength(),
		BoundaryBacklog: opt.Net.BacklogTotal(),
		Departed:        opt.balance.Departed,
	}
}

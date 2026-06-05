package dto

// MPRunRequest is the request body for POST /maxpressure/run.
// swagger:model
type MPRunRequest struct {
	// Network is the meso-level graph (road segments + connector links).
	Network MPNetworkDTO `json:"network"`
	// Intersections defines the signal configuration and group-to-connector mapping.
	Intersections []MPIntersectionConfigDTO `json:"intersections"`
	// Demand defines the traffic input per entry link.
	Demand MPDemandDTO `json:"demand"`
	// Drain controls how vehicles leave the network at boundary exits.
	// Omit or set type to "auto" for the default behaviour (zero queue each step).
	Drain MPDrainDTO `json:"drain"`
	// InitialQueues sets starting queue length (vehicles) per link ID. Optional.
	InitialQueues map[int]float64 `json:"initial_queues"`
	// Config holds simulation parameters.
	Config MPSimConfigDTO `json:"config"`
}

// MPNetworkDTO describes the meso topology.
// swagger:model
type MPNetworkDTO struct {
	// Links is the list of meso links (road segments and connectors).
	Links []MPLinkDTO `json:"links"`
}

// MPLinkDTO represents a single meso link.
// If IsConnection is true the link is a turn connector between two road segments.
// swagger:model
type MPLinkDTO struct {
	// ID is the unique link identifier.
	ID int `json:"id"`
	// SourceNode is the source meso node ID.
	SourceNode int `json:"source_node"`
	// TargetNode is the target meso node ID.
	TargetNode int `json:"target_node"`
	// MacroNode is the parent macro-node ID (connector links only).
	MacroNode *int `json:"macro_node"`
	// IsConnection marks whether this link connects two other mesoscopic links.
	IsConnection bool `json:"is_connection"`
	// MovementMesoLinkIncome is the incoming mesoscopic link identifier (connector links only).
	MovementMesoLinkIncome *int `json:"movement_meso_link_income"`
	// MovementMesoLinkOutcome is the outgoing mesoscopic link identifier (connector links only).
	MovementMesoLinkOutcome *int `json:"movement_meso_link_outcome"`
	// LengthMeters is the link length in meters (road segments).
	LengthMeters float64 `json:"length_meters"`
	// Lanes is the number of lanes (road segments).
	Lanes int `json:"lanes"`
	// Capacity is the saturation flow in veh/h.
	Capacity int `json:"capacity"`
}

// MPIntersectionConfigDTO pairs a junction signal plan with its group-to-connector mapping.
// swagger:model
type MPIntersectionConfigDTO struct {
	// MacroNodeID is the macro-graph node ID identifying this intersection.
	MacroNodeID int `json:"macro_node_id"`
	// Junction is the full signal plan for this intersection.
	Junction JunctionDTO `json:"junction"`
	// GroupConnectors maps signal group ID to the meso connector link IDs it controls.
	// JSON keys are group IDs as strings (e.g. "0", "1").
	GroupConnectors map[int][]int `json:"group_connectors"`
}

// MPDemandDTO specifies the traffic demand injected into the network.
//
// Supported types:
//   - "constant" (default): fixed rate per link, use Rates field.
//   - "peak": time-varying morning-peak profile, use BaseRates + peak shape fields.
//
// swagger:model
type MPDemandDTO struct {
	// Type is the demand model: "constant" or "peak".
	Type string `json:"type"`
	// Rates maps link ID to constant demand intensity in veh/h. Used for type "constant".
	Rates map[int]float64 `json:"rates"`
	// BaseRates maps link ID to base demand intensity in veh/h. Used for type "peak".
	BaseRates map[int]float64 `json:"base_rates"`
	// PeakFactor is the demand multiplier at peak. Defaults to 1.3 when 0. Used for type "peak".
	PeakFactor float64 `json:"peak_factor"`
	// RampUpS is the ramp-up phase duration in seconds. Used for type "peak".
	RampUpS float64 `json:"ramp_up_s"`
	// PeakDurationS is the peak phase duration in seconds. Used for type "peak".
	PeakDurationS float64 `json:"peak_duration_s"`
	// RampDownS is the ramp-down phase duration in seconds. Used for type "peak".
	RampDownS float64 `json:"ramp_down_s"`
}

// MPDrainDTO controls how vehicles leave the network at boundary exits.
//
// Supported types:
//   - "auto" (default): zero the queue at each boundary departure link every step.
//   - "none": no drain; vehicles accumulate at exits (closed-network test scenario).
//   - "rate": drain at the specified rate (veh/h) per link; links absent from Rates are drained fully.
//
// swagger:model
type MPDrainDTO struct {
	// Type is the drain model: "auto", "none", or "rate".
	Type string `json:"type"`
	// Rates maps link ID to drain intensity in veh/h. Used for type "rate".
	Rates map[int]float64 `json:"rates"`
}

// MPSimConfigDTO holds max-pressure simulation parameters.
// swagger:model
type MPSimConfigDTO struct {
	// DeltaT is the simulation step duration in seconds. Defaults to 5.0 if not set.
	DeltaT float64 `json:"delta_t"`
	// SimTime is the total simulation duration in seconds. Must be > 0.
	SimTime float64 `json:"sim_time"`
	// Alpha is the Smoothing-MP coordination coefficient (>= 0). 0 = standard MP.
	Alpha float64 `json:"alpha"`
}

// MPRunResponse is the response from POST /maxpressure/run.
// swagger:model
type MPRunResponse struct {
	// Evaluation contains simulation metrics measured over the full SimTime.
	Evaluation MPEvaluationDTO `json:"evaluation"`
	// Proposal contains re-synthesized signal timings based on MP stage fractions.
	Proposal []MPProposalDTO `json:"proposal"`
}

// MPEvaluationDTO aggregates simulation performance metrics.
// swagger:model
type MPEvaluationDTO struct {
	// TotalDelayVehS is the total accumulated queue (veh * s) summed over all road
	// links and all simulation steps (sum of queue_i * delta_t at each step).
	TotalDelayVehS float64 `json:"total_delay_veh_s"`
	// PerLink contains per-link queue statistics (road segments only).
	PerLink []MPLinkStatsDTO `json:"per_link"`
	// PerIntersection contains per-intersection stage selection statistics.
	PerIntersection []MPIntersectionStatsDTO `json:"per_intersection"`
}

// MPLinkStatsDTO reports average and peak queue for one road link.
// swagger:model
type MPLinkStatsDTO struct {
	// LinkID is the meso link identifier.
	LinkID int `json:"link_id"`
	// AvgQueueVeh is the time-averaged queue length in vehicles.
	AvgQueueVeh float64 `json:"avg_queue_veh"`
	// MaxQueueVeh is the peak queue length in vehicles.
	MaxQueueVeh float64 `json:"max_queue_veh"`
}

// MPIntersectionStatsDTO reports stage selection fractions for one intersection.
// swagger:model
type MPIntersectionStatsDTO struct {
	// MacroNodeID identifies the intersection.
	MacroNodeID int `json:"macro_node_id"`
	// StageFractions maps stage ID to the fraction of steps it was selected [0, 1].
	// JSON keys are stage IDs as strings.
	StageFractions map[int]float64 `json:"stage_fractions"`
}

// MPProposalDTO contains synthesized signal timings for one intersection.
// Green times are redistributed proportionally to stage selection fractions.
// swagger:model
type MPProposalDTO struct {
	// MacroNodeID identifies the intersection.
	MacroNodeID int `json:"macro_node_id"`
	// Junction is the updated signal plan with redistributed green times.
	Junction JunctionDTO `json:"junction"`
}

# Max-pressure model and original sources

Model version 2 follows the movement-queue formulation in the original papers stored in `tex`. The implementation includes a documented interpretation of Xu's coordination objective and practical extensions. It does not claim an exact numerical reproduction of the paper's experiments or their stability guarantees.

| Source | Relevant formulation | Implementation |
|:---|:---|:---|
| Varaiya (2013), [PDF](../tex/varaiya2013.pdf), [publication](https://doi.org/10.1016/j.trc.2013.08.014), section 3.2, equations (21), (22) | Movement queues and downstream queues weighted by turning proportions | Independent movement queues and weighted downstream pressure |
| Xu, Barman, Levin (2024), [PDF](../tex/xu_2024.pdf), [publication](https://doi.org/10.1016/j.trc.2024.104760), sections 3.2-3.5 | Movement conservation, explicit corridors, one-step actuation history and coordination bonus | Queue equations, corridor scope and signal history implemented; objective interpretation below |
| Kouvelas et al. (2014), [PDF](../tex/Kouvelas_2014.pdf), [publication](https://doi.org/10.3141/2421-15), p. 134 | Normalized road queues, turning rates and green allocation | Reference for a different formulation; the former normalized-road approximation has been removed |
| Liu and Gayah (2022), [PDF](../tex/Liu_2022.pdf), [preprint](https://arxiv.org/abs/2202.03290) | Delay-based MP | Separate controller, not implemented here |

The user's `note*.tex/pdf` files are not normative sources for these equations.

## Movement state, pressure and units

A connector represents movement `(i,j)`: vehicles on directed road `i` intending to enter directed road `j`. Its queue is independent of other movements leaving the same road. In Xu (1)-(3), (12):

```text
Q_ij = saturation_veh_per_hour * delta_t / 3600
y_ij(t) = min(Q_ij * s_ij(t), x_ij(t))
x_ij(t+1) = x_ij(t) - y_ij(t) + sum_h(y_hi(t)) * r_ij
w_ij(t) = x_ij(t) - sum_k(r_jk * x_jk(t))
ordinary_stage_score = sum_active(Q_ij * w_ij)
```

At entry roads the arrival term is external demand multiplied by the turning proportions. All arrivals split across all outgoing movements, including red ones. Saturation flows do not determine turning proportions. Every branching road requires explicit non-negative proportions summing to one; a sole outgoing movement defaults to one. Fractional vehicles represent continuous flow.

`Network.MovementQueues` is authoritative after initialization. `Network.Queues` holds aggregate road occupancy and accepts initial road totals, split once by turning proportions. Do not supply both road and movement initial queues for the same road. After initialization, update movement queues rather than overwriting aggregate road queues. Boundary departures have an aggregate queue without outgoing movements.

Each step chooses from the start-of-step state, transfers traffic simultaneously, drains exits and admits external demand into the next state. An arrival cannot cross two controlled movements in one step. `Run` and `Evaluate` include a final partial step; service uses its actual duration and stage scoring uses the configured nominal step. This is a store-and-forward queueing model without travel times, trajectories, lane changing or individual vehicles.

## Coordination and the printed objective

Xu section 3.5 defines `c_jk(t+1) = s_ij(t)` on explicitly selected corridors. An upstream green counts even if no vehicle discharged. The indicator uses the previous completed step, not the current neighboring decision, and has no travel-time delay. A fresh simulation has no preceding actuation unless the Go caller supplies `PreviousStage` and `HasPreviousStage`.

`SetCoordinatedCorridors` accepts directed road paths of at least three roads, such as `[1,3,7]`. This coordinates movement `(3,7)` with `(1,3)` and excludes side turns. Reverse travel needs a separate path. Unknown roads, disconnected paths and movements absent from control stages are rejected. Multiple configured predecessors combine using a binary OR; this overlap convention does not multiply the bonus.

On page 6, printed equation (13) contains `xi_ij * c_ij(t)` without multiplying it by candidate actuation `s_ij(t)`. Since `c` is fixed by the preceding step, a literal additive term would be constant in the current maximization and could not alter stage selection. The implementation uses the candidate-stage interpretation of the stated prioritization intent:

```text
xi_ij = alpha * Q_ij^2, alpha in [0,1]
smoothed_stage_score = sum_active(Q_ij * w_ij + xi_ij * c_ij)
```

The scaling uses the suggested `Q^2` upper bound discussed after (13), with `Q` measured in vehicles per step. This replaces the former `alpha * saturation_veh_per_hour` convention. The candidate-stage interpretation still needs confirmation from author code or an erratum before claiming exact reproduction. No corresponding stability proof is claimed for this code or its extensions.

## Timing and physical storage extensions

`CompileProgram` splits the supplied program at every signal boundary. Adaptive stages contain only movements whose green intervals actually overlap. Groups that turn green at different times within one phase are not combined into an unsafe union. Original phase IDs remain available for reporting and fixed-plan synthesis.

Adaptive control enforces minimum green, maximum green and all-red clearance. Defaults in REST are 5/60/5 seconds. Explicit zero disables the respective bound. Minimum green and clearance round up to a whole number of steps; maximum green rounds down. Incompatible limits are rejected. Ties retain the active stage, then favor the smallest stage ID. With only one available stage, reaching maximum green inserts at least one all-red step before restarting it. This is not a controller-specific yellow transition sequence.

The default `point_queue` model has unlimited road storage. Optional `finite_storage` uses `length * lanes / 7` vehicles of storage and limits merging inflows proportionally to available receiving space. Blocked movement traffic remains upstream. Space freed by outgoing traffic becomes available for incoming transfers on the next step. Entry demand exceeding available space waits in a separate boundary backlog. Initial queues exceeding finite storage must be supplied differently, not silently clipped. These are engineering extensions, not Xu's point-queue assumptions.

Saturation is supplied per movement. Shared-lane service competition is not inferred from road lane counts; input capacities and stage assignments must describe the intended service model. The editor checks its geometric movement conflicts, while raw API clients provide their own feasible movement sets.

## Fixed programs and comparable evaluation

The API simulates four cases from independent copies of the same initial state, demand and storage model: the original fixed program, ordinary MP, smoothing-MP and a synthesized fixed program. Fixed replay integrates actual green windows and offsets across each step; yellow, red and red-yellow provide no service.

Synthesis uses adaptive phase service durations as weights. It resizes one common green/red interval per phase, changes all groups consistently, respects every explicit signal bound, leaves yellow/red-yellow durations unchanged and preserves exact integer cycle length and offset. Omitted REST bounds permit green in `[min(5,current),cycle]` and red in `[0,cycle]`; omitted yellow/red-yellow bounds remain fixed. Explicit equal bounds remain fixed. This is a project heuristic, not a result proved in the paper.

The candidate is replayed as a fixed program. It is accepted only if waiting decreases and departures do not decrease relative to the original fixed program, with tolerance `1e-6`. Otherwise the original programs and their evaluation are returned with `proposal_accepted=false`. Adaptive performance is never substituted for the fixed proposal's performance.

`Evaluate` requires a fresh optimizer. Waiting is the integral of start-of-step network queues plus external backlog, in vehicle-seconds. It is not microscopic travel delay. Results include road and movement average/maximum/final queues, phase service, all-red time, bounded queue samples and vehicle accounting:

```text
initial + requested = departed + remaining_in_network + boundary_backlog
```

`admitted` separately reports boundary admissions during the run. Invalid custom demand/drain results halt a Go optimizer with `Err()`. The API validates topology, assignment, proportions, program consistency, limits, numerical inputs and request size; context cancellation stops evaluations.

## Compatibility and verification

REST responses now contain `model_version: 2`. Clients must consume `baseline`, `standard_mp`, `evaluation`, `proposal_evaluation`, `proposal_accepted` and the vehicle balance fields. Branching networks must provide `turning_ratios`; `alpha` is now restricted to `[0,1]` and has changed scale. Default `delta_t` is one second. Positive alpha requires explicit `coordinated_corridors`. Go callers must handle the error returned by `NewMPOptimizer`.

Regression tests cover red left-turn queues, weighted downstream pressure, simultaneous diverging/merging transfers, finite storage and boundary backlog, conservation, offsets and partial steps, non-overlapping group greens, timing constraints, corridor history, fixed-plan bounds and independent replay. [The runnable example](../examples/maxpressure/README.md) calls `app/simulation.RunMaxPressure`, the same Go function used by the REST handler, and provides both API and shared-editor JSON inputs. The REST handler handles HTTP decoding and responses; scenario validation, model construction and controller comparison live in `app/simulation`. Demand and drain profiles use named methods, and evaluation records snapshots through `MPOptimizer.sample`. [The editor guide](../web/greenwave-ui/MAX_PRESSURE.md) describes the complete user workflow.

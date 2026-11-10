# Max-pressure control

Each intersection chooses which movements get green from the queues around it. Standard MP balances upstream demand against downstream queues. Smoothing-MP adds priority along selected corridors based on the previous upstream green.

This describes model version 2. For setup and results in the UI, see [Max pressure](../web/greenwave-ui/MAX_PRESSURE.md). The [runnable example](../examples/maxpressure/README.md) includes API and editor inputs.

## Standard MP

A movement `(i,j)` goes from directed road `i` to road `j`. Through traffic and turns have separate queues, even when they share an incoming road.

| Symbol | Meaning |
| --- | --- |
| `x_ij` | Vehicles waiting for movement `(i,j)` |
| `r_ij` | Share of arrivals on road `i` that will use movement `(i,j)` |
| `Q_ij` | Movement capacity in vehicles per simulation step |
| `s_ij` | 1 when the movement has green, otherwise 0 |

[pressure.go](pressure.go) implements the movement weights from Varaiya (21)-(23) and Xu (12):

```text
Q_ij = saturation_veh_per_hour * delta_t / 3600
w_ij = x_ij - sum_k(r_jk * x_jk)
score(stage) = sum_(i,j in stage)(Q_ij * w_ij)
```

A larger upstream queue increases priority. A larger weighted downstream queue reduces it. For example, with `Q_ij = 1`, an upstream queue of 12 and weighted downstream queue of 5 contribute 7 to the stage score.

A stage contains movements that can receive green together. [CompileProgram](program.go) splits the supplied program at every signal boundary, preserving the green combinations actually present. Groups with non-overlapping green intervals stay in separate stages.

Each intersection selects the highest-scoring stage allowed by its timing constraints. Ties retain the active stage, then favor the smallest stage ID.

## Smoothing-MP

For consecutive roads `i, j, k`, Xu section 3.5 supplies the previous-green indicator. The implementation uses it in the following experimental score:

```text
c_jk(t+1) = s_ij(t)
xi_ij = alpha * Q_ij^2, alpha in [0,1]
smoothed_score(stage) = sum_(i,j in stage)(Q_ij * w_ij + xi_ij * c_ij)
```

An upstream green gives the next movement a bonus on the following step. This uses signal history, including green with an empty queue. There is no travel-time delay. API runs start without previous-green history. With `alpha = 0`, the score is Standard MP.

[SetCoordinatedCorridors](coordination.go) takes directed road paths of at least three roads. `[1,3,7]` gives movement `(3,7)` priority after `(1,3)` had green. Reverse travel needs its own path. When several configured predecessors feed one movement, any previous green sets its indicator to 1.

[smoothing.go](smoothing.go) uses `alpha` to scale the `Q^2` reference discussed after Xu (13). `Q` uses the configured step duration, so changing `delta_t` also changes the bonus scale.

Xu's printed equation (13) adds `xi_ij * c_ij(t)` without multiplying that term by the candidate signal `s_ij(t)`. With the indicator fixed by the previous step, that additive term cannot change the maximizing choice. The code sums the bonus only over movements green in the candidate stage. This is the project's interpretation of the coordination intent. Exact reproduction of the paper and its stability guarantees are not claimed for this implementation or its extensions.

## From adaptive control to a fixed program

Adaptive MP reacts to queues at each step and can change the order and duration of green stages. A fixed program repeats its phase sequence and offset. Compressing adaptive decisions into that repeating program can lose their benefit.

[RunMaxPressure](../app/simulation/mp.go) evaluates the original fixed program, Standard MP, Smoothing-MP and a fixed proposal from the same initial queues, demand and storage settings. Fixed replay uses the actual green intervals and offsets. Yellow, red and red-yellow provide no service.

[SynthesizeProgram](proposal.go) uses adaptive service seconds per original phase as weights. It reallocates a common green/red window in each phase, changes all groups consistently and respects signal bounds. Phase order, group structure, yellow/red-yellow durations, integer cycle length and offset are preserved. This synthesis is a project heuristic.

When REST bounds are omitted, green permits `[min(5,current),cycle]` seconds and red permits `[0,cycle]`. Yellow/red-yellow stay fixed. Explicit equal bounds fix a signal's duration.

The proposal is replayed independently and accepted only when:

```text
proposal_waiting < original_waiting - 1e-6
proposal_departed >= original_departed - 1e-6
```

Otherwise the API returns the original programs and their evaluation with `proposal_accepted=false`. Better adaptive results alone do not make a fixed proposal acceptable.

## Queueing model and limits

For a full step with point queues, Xu (1)-(3) give the transferred vehicles `y_ij` and updated queue:

```text
y_ij(t) = min(Q_ij * s_ij(t), x_ij(t))
x_ij(t+1) = x_ij(t) - y_ij(t) + r_ij * sum_h(y_hi(t))
```

At an entry, external arrivals replace the upstream-flow sum. Turning shares split arrivals across all outgoing movements, including red ones. Branching roads require explicit shares summing to one. A sole outgoing movement defaults to one. Shares stay fixed during a run, while demand can vary with time. Fractional vehicles represent continuous flow.

Each step selects signals from its starting queues, transfers traffic simultaneously, drains exits and admits external demand. Arrivals cannot cross two controlled movements in one step. The final partial step uses its actual duration for service and waiting.

| Feature | Implementation |
| --- | --- |
| Signal timing | Minimum green, maximum green and all-red clearance. REST defaults are 5/60/5 seconds. Zero disables the respective bound. Minimum green and clearance round up to whole steps, maximum green rounds down. |
| One available stage | Reaching maximum green inserts at least one all-red step before restarting it. |
| Point queues | Unlimited road storage. |
| Finite storage | Capacity is `length * lanes / vehicle_length`, with a default vehicle length of 7 m. Merging inflows share receiving space proportionally. Blocked traffic stays upstream, and unadmitted entry demand waits outside. Space freed by departures is available to internal transfers on the next step. |
| Movement capacity | Uses per-movement saturation flows. Xu (3) uses `Q_ij = min(Q_i,Q_j)` from adjoining link capacities. Shared-lane competition is not inferred from lane counts. |

Timing constraints, finite storage and fixed-program synthesis are extensions to the paper model. Road travel times, vehicle trajectories and lane changes are absent, so queue improvements do not establish a stop-free green wave. API callers supply feasible movement sets. The editor also checks geometric movement conflicts.

`Network.MovementQueues` holds the movement state, while `Network.Queues` mirrors road totals. Initial road totals are split once by turning shares. Road and movement initial queues cannot both be supplied for the same road. Initial queues above finite storage are rejected.

[Evaluate](evaluation.go) integrates start-of-step network queues plus external backlog in vehicle-seconds. It requires a fresh optimizer. Vehicle accounting follows:

```text
initial + requested = departed + remaining_in_network + boundary_backlog
```

`admitted` separately counts traffic admitted during the run. REST responses carry `model_version: 2` and separate adaptive results from the fixed-program evaluation.

## References

1. Varaiya, P. (2013). [Max pressure control of a network of signalized intersections](https://doi.org/10.1016/j.trc.2013.08.014), section 3.2, equations (21)-(23). Movement queues and pressure-based selection. [PDF](../tex/varaiya2013.pdf).
2. Xu, T., Barman, S., Levin, M.W. (2024). [Smoothing-MP: A novel max-pressure signal control considering signal coordination to smooth traffic in urban networks](https://doi.org/10.1016/j.trc.2024.104760), sections 3.3 and 3.5. Queue evolution and corridor coordination, with the objective interpretation described above. [PDF](../tex/xu_2024.pdf).
3. Kouvelas, A., Lioris, J., Fayazi, S.A., Varaiya, P. (2014). [Maximum pressure controller for stabilizing queues in signalized arterial networks](https://doi.org/10.3141/2421-15). Related formulation using normalized road queues. [PDF](../tex/Kouvelas_2014.pdf).
4. Liu, H., Gayah, V.V. (2022). [A novel max pressure algorithm based on traffic delay](https://arxiv.org/abs/2202.03290). Related controller using delay measurements. [PDF](../tex/Liu_2022.pdf).

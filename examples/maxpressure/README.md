# Movement-queue MaxPressure example

This example compares an original fixed program, ordinary MP, smoothing-MP and an independently replayed fixed-program proposal. The original Varaiya and Xu papers in `tex` provide the movement-queue model. [PAPER_MODEL.md](../../maxpressure/PAPER_MODEL.md) documents equations, units, timing/storage extensions and an unresolved ambiguity in Xu's printed coordination objective. The example does not claim reproduction of the paper's experiments.

## Run

```sh
go run ./examples/maxpressure
go run ./examples/maxpressure -request
go run ./examples/maxpressure -json
```

The executable calls `simulation.RunMaxPressure` directly, the same Go function used by the REST handler. No HTTP context, test server or running service is required. `-request` prints the complete request and `-json` prints all comparison results. With a running service:

```sh
curl -H 'Content-Type: application/json' --data-binary @examples/maxpressure/corridor.json http://localhost:36000/api/maxpressure/run
```

[corridor.json](corridor.json) is the REST request. [project.json](project.json) is the same traffic scenario mapped to the shared editor: open it through "Project > Open JSON", select "Max pressure" and run the comparison. Programs and traffic settings are saved with the project. The UI guide is [MAX_PRESSURE.md](../../web/greenwave-ui/MAX_PRESSURE.md).

## Scenario

Pushkin and Nekrasov intersections are connected by road 3. Each program has four phases and four groups: main through/right movements, a separate main left turn, side through movements and a separate side left turn. Every group has its own signals in every phase. Cycles are 100 seconds, including 20 seconds green, 3 yellow and 2 red per active phase. Offsets are 0 and 35 seconds.

| Junction | Connector | Incoming road | Outgoing road | Group | Turning share | Saturation, veh/h |
|:---|---:|---:|---:|---:|---:|---:|
| Pushkin | 100 | 1 | 3 | 0 | 0.65 | 1800 |
| Pushkin | 101 | 1 | 6 | 0 | 0.15 | 600 |
| Pushkin | 104 | 1 | 9 | 1 | 0.20 | 900 |
| Pushkin | 102 | 2 | 5 | 2 | 0.80 | 1800 |
| Pushkin | 103 | 2 | 3 | 3 | 0.20 | 900 |
| Nekrasov | 200 | 3 | 7 | 0 | 0.65 | 1800 |
| Nekrasov | 201 | 3 | 8 | 0 | 0.15 | 600 |
| Nekrasov | 204 | 3 | 18 | 1 | 0.20 | 900 |
| Nekrasov | 202 | 4 | 7 | 2 | 0.80 | 1800 |
| Nekrasov | 203 | 4 | 8 | 3 | 0.20 | 900 |

Demand is constant: road 1 receives 850 veh/h, road 2 receives 250 veh/h and road 4 receives 300 veh/h. Initial road queues are 12, 4, 5 and 7 vehicles respectively. These totals are split once into independent movement queues according to turning shares. A left-turn queue keeps waiting during through green.

The coordinated directed path is `[1,3,7]`. Only movement 200 receives the bonus following movement 100's green; side movements do not. The run lasts 1200 seconds with a one-second step, point queues, alpha 0.5, minimum green 5 seconds, maximum green 45 seconds and clearance 5 seconds. Exits drain automatically. Initial history is empty, so the first decision has no coordination bonus.

## Reading the result

An observed run of model version 2 gives:

| Controller | Waiting, veh·s | Departed, veh | Remaining, veh |
|:---|---:|---:|---:|
| Original program | 61235.8 | 410.7 | 84.0 |
| Standard MP | 57075.2 | 420.7 | 73.9 |
| Smoothing-MP | 56812.1 | 421.2 | 73.5 |
| Returned fixed program | 27412.1 | 470.4 | 24.3 |

The fixed candidate is accepted in this scenario because waiting decreases without reducing departures. It is a separate timing heuristic and can outperform or underperform the adaptive controller. Other inputs can produce no accepted fixed improvement, in which case the API returns the original programs. Cycle lengths, offsets, yellow intervals, group structure and explicit signal bounds are preserved.

Queues, external backlog and departures are all counted. Check `vehicle_balance.conservation_error_veh`, which should be zero within floating-point tolerance. `total_delay_veh_s` integrates waiting in the network and outside full entrances; it does not represent road travel time. The model cannot establish a stop-free green wave or animate actual vehicle trajectories.

## Finite storage and API changes

Set `config.storage_model` to `finite_storage` to enable storage limits. Unadmitted demand remains in `initial_backlogs`/the evolving boundary backlog rather than disappearing. The default is `point_queue`, matching the paper's storage assumption. Supply `initial_movement_queues` for independently measured turn queues instead of `initial_queues` for that road.

This is a version 2 API example. Explicit turning ratios are required at branching roads, alpha uses per-step `Q²` scaling in `[0,1]`, and the default step is now one second. The response separates adaptive evaluation from original and proposed fixed-program replay. Restart the backend after updating the code before testing the frontend.

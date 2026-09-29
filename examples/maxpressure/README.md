# Max-Pressure Example

Demonstrates an experimental normalized max-pressure controller with a coordination boost on a 2-intersection corridor. The current model is an adaptation inspired by the papers below, not a validated reproduction of their models or stability guarantees.

Based on: Varaiya, P. (2013). [Max pressure control of a network of signalized intersections](https://doi.org/10.1016/j.trc.2013.08.014). Transportation Research Part C, 36, 177-195.

Normalized pressure (Modified MP): Kouvelas, A., Lioris, J., Fayazi, S.A., Varaiya, P. (2014). [Maximum pressure controller for stabilizing queues in signalized arterial networks](https://doi.org/10.3141/2421-15). Transportation Research Record, 2421(1), 133-141.

Coordination enhancement: Xu, T., Barman, S., Levin, M.W. (2024). [Smoothing-MP: A novel max-pressure signal control considering signal coordination to smooth traffic in urban networks](https://doi.org/10.1016/j.trc.2024.104760). Transportation Research Part C, 166, 104760.

## Current status and next steps

The original PDFs in `tex` have now been checked directly. [PAPER_MODEL.md](../../maxpressure/PAPER_MODEL.md) maps the implementation to the source equations and records an ambiguity in Xu's printed objective (13). Corridor membership and one-step actuation history follow section 3.5; the queue model and coordination score still contain explicit project-specific approximations.

Transfers between road links now conserve vehicles: diverging movements share the upstream queue, merging movements share receiving space, and rejected flow remains upstream. Regression tests cover both alpha=0 and alpha=1, full and partially full destinations, duplicate connectors, and repeated simulation steps without external demand or drainage.

The remaining work before connecting this controller to the shared network editor is:

1. Define movement demand and turning proportions. Queues currently belong to roads, so separate left-turn and through demand cannot be represented. Graph topology lists possible movements but does not specify their traffic shares.
2. Account for demand that cannot enter a full road. Injection currently clips at road capacity without retaining a boundary backlog or reporting rejected arrivals. Queue totals alone cannot establish throughput or stability under overload.
3. Implement timing constraints. `MinGreenS`, `MaxGreenS`, and `ClearanceS` exist in the model but are not enforced. `StagesFromJunction` collects all groups with any green in a phase, without checking that their green intervals overlap.
4. Resolve the objective's ambiguity and replace legacy road pressures with movement pressures. The indicator now follows the immediately preceding signal actuation only on configured corridors. Xu's indicator uses the green signal, not measured discharge or travel time.
5. Validate REST inputs and synthesized programs. The current stage-frequency proposal is not a simulation of the resulting fixed program; its timing constraints, multi-group consistency, and offset preservation need review.
6. Map the shared editor's directed roads and assigned movements to meso links and connectors, then compare controllers on identical demand. Corridor intensity alone does not supply turning proportions or demand for every network entrance.

The source of truth for signal programs and group assignments remains the shared editor project. The existing green-wave calculation and its UI are separate from this MP simulation.

## Run

```bash
go run ./examples/maxpressure/main.go
```

## Network

```
            [2] NA->A              [4] NB->B
             |                     |
             v                     v
[1] WA->A -> (A) -> [3] A->B ----> (B) -> [7] B->EB
             |                     |
             v                     v
          [5] A->WA              [8] B->NB
          [6] A->NA
```

- `(A)` - intersection, macroNode=50
- `(B)` - intersection, macroNode=60
- `[N]` - meso segment (road link) with ID=N
- Links 5, 6, 7, 8 - boundary departures (vehicles exit the network)
- Link 3 (A->B) - coordination link, the key segment connecting both intersections

## Meso-level detail (inside each intersection)

Each intersection has connector links (movements) that connect upstream segments to downstream segments. These are the arcs inside the intersection box in the TeX figure.

Intersection A (macroNode=50):

| Connector ID | Movement | Upstream -> Downstream | $S$ (veh/h) | Stage |
|:---:|:---:|:---:|:---:|:---:|
| 100 | EBT | [1] WA->A -> [3] A->B | 900 | sg0 (EW) |
| 101 | EBR | [1] WA->A -> [6] A->NA | 700 | sg0 (EW) |
| 102 | SBT | [2] NA->A -> [5] A->WA | 900 | sg1 (NS) |
| 103 | SBL | [2] NA->A -> [3] A->B | 700 | sg1 (NS) |

Intersection B (macroNode=60):

| Connector ID | Movement | Upstream -> Downstream | $S$ (veh/h) | Stage |
|:---:|:---:|:---:|:---:|:---:|
| 200 | EBT | [3] A->B -> [7] B->EB | 900 | sg0 (EW) |
| 201 | EBR | [3] A->B -> [8] B->NB | 700 | sg0 (EW) |
| 202 | SBT | [4] NB->B -> [7] B->EB | 900 | sg1 (NS) |
| 203 | SBL | [4] NB->B -> [8] B->NB | 700 | sg1 (NS) |

## Why meso-level graph

The meso graph from go-gmns represents movements as connector links between road links. The current implementation uses the following simplified data model:

| Aspect | Macro (Varaiya / Xu) | Meso (ours) |
|:---|:---|:---|
| Movement | Abstract pair (i,j) + turning ratio | Connector link with ID, satflow, type |
| Turning ratios | Required as external input | Not represented; topology only lists possible turns |
| Queue model | Per-movement, point queue | Per-link, finite capacity $K$ |
| Link capacity | Infinite (assumed) | Finite: $K = length \cdot lanes / L_{veh}$ |

Pressure uses road occupancy $x/K$. When active connectors compete for a queue, discharge is allocated in proportion to their saturation flows. This is an explicit approximation, not measured turning demand. The coordination indicator $c_{u,d}$ uses graph queries (`MovementMesoLinkOutcome` / `MovementMesoLinkIncome`) and stored stage history.

## Scenario

The example simulates a 10-minute period with time-varying demand (morning rush):

| Phase | Time | Demand multiplier | Description |
|:---:|:---:|:---:|:---|
| Ramp-up | 0-60s | 0.5x -> 1.0x | Traffic builds |
| Peak | 60-300s | 1.3x | Rush hour, network oversaturated |
| Ramp-down | 300-420s | 1.3x -> 1.0x | Peak subsides |
| Recovery | 420-600s | 1.0x | Normal load |

Base demand rates:
- Link 1 (WA->A): 1600 veh/h - heavy eastbound corridor
- Link 2 (NA->A): 800 veh/h - moderate southbound
- Link 4 (NB->B): 1200 veh/h - heavy competing flow at B

At peak (x1.3), link 1 reaches 2080 veh/h - well above effective capacity (~800 veh/h per approach with 50% green share). The network is stressed, queues grow, and the algorithm must make real trade-offs.

Initial queues represent residual congestion: link1=15, link2=8, link3=5, link4=10 vehicles.

Both scenarios (Standard MP and Smoothing-MP) run on identical input data.

The coordinated corridor is explicitly `[1, 3, 7]`: movement 100 at A can give priority to movement 200 at B on the next step. Movement 201, which turns off the corridor, receives no coordination bonus. No previous signal state is supplied, so the first step has no bonus. REST clients with `alpha > 0` must now supply `coordinated_corridors`; see [the request contract](../../maxpressure/PAPER_MODEL.md#coordination-implemented-in-this-step).

## Realistic 4-phase scenario

The `runRealisticScenario` function extends the basic 2-stage model to a realistic 4-phase, 4-group signal plan derived from a `junction.Junction` via `StagesFromJunction`.

### Signal plan

Each intersection has 4 signal groups:

| Group | Movement |
|:---:|:---|
| 0 | EW through |
| 1 | EW left turn |
| 2 | NS through |
| 3 | NS left turn |

4 phases per cycle (total cycle = 95s). Each active group follows GREEN->YELLOW->RED; inactive groups stay RED throughout the phase.

| Phase | Duration | Group 0 | Group 1 | Group 2 | Group 3 |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 0 | 35s | G(30)->Y(3)->R(2) | R(35) | R(35) | R(35) |
| 1 | 20s | R(20) | G(15)->Y(3)->R(2) | R(20) | R(20) |
| 2 | 28s | R(28) | R(28) | G(23)->Y(3)->R(2) | R(28) |
| 3 | 12s | R(12) | R(12) | R(12) | G(8)->Y(2)->R(2) |

**Signal transition rule**: for each group, consecutive phases must end with the same signal meaning -> prohibition (R, Y) or permission (G). All phases here end with R, so all transitions are prohibition->prohibition. This prevents e.g. a phase ending GREEN immediately followed by a phase starting GREEN for the same group.

### Network extension

Links 9, 10 (at A) and 18, 19 (at B) are added as left-turn departure links (boundary drains). Four new connectors map left-turn groups to these links:

| ID | Movement | Upstream -> Downstream | S (veh/h) | Group | Stage |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 104 | EBL | [1] WA->A -> [9] A-left | 600 | 1 | sg1 |
| 105 | SBL | [2] NA->A -> [10] A-left | 600 | 3 | sg3 |
| 204 | EBL | [3] A->B -> [18] B-left | 600 | 1 | sg1 |
| 205 | NBL | [4] NB->B -> [19] B-left | 600 | 3 | sg3 |

### Stages derived via StagesFromJunction

Instead of listing connector IDs manually, stages are built from the junction config. `StagesFromJunction` inspects each phase's signal groups and collects connectors for every group with at least one GREEN signal:

```
Junction A: 4 phases -> 4 stages (cycle ~95s)
  stage 0: connectors [100 101]   (Group 0 GREEN in phase 0)
  stage 1: connectors [104]       (Group 1 GREEN in phase 1)
  stage 2: connectors [102 103]   (Group 2 GREEN in phase 2)
  stage 3: connectors [105]       (Group 3 GREEN in phase 3)
```

### Realistic scenario output

Current summary from the four-phase example:

```
  avg total queue: 156.7 veh | max: 222.5 veh
  avg link3 queue:  31.6 veh | max:  58.1 veh
  B stage selection: sg0(EW-thr)=38  sg1(EW-left)=0  sg2(NS-thr)=82  sg3(NS-left)=0  (total=120)
```

### Interpreting the realistic results

The left-turn stages are never selected in this example. This exposes the shared road-queue approximation: through and left-turn movements compete using the same upstream queue, and the through stages have higher combined saturation flow. Separate demand for vehicles waiting to turn left is not modeled, so these results cannot demonstrate acceptable left-turn service.

The four-phase example currently produces the same totals as the two-stage scenario. It exercises the junction-to-stage mapping, but does not validate phase transitions or intergreen timing.

## Algorithm step-by-step

### Step 1: Inject demand

Each step, the optimizer converts intensities (veh/h) to vehicles per step:

$$\text{inject}_l(k) = \frac{q_l(t) \cdot \Delta t}{3600}$$

where $q_l(t)$ is demand intensity on link $l$ at time $t$ (veh/h) and $\Delta t$ is the step duration (seconds).

Example: WA->A at peak ($q = 2080$ veh/h), $\Delta t = 5$ s $\Rightarrow$ inject $= 2080 \cdot 5 / 3600 = 2.89$ veh/step.

### Step 2: Drain boundary departures

Links 5, 6, 7, 8 are auto-detected as boundary departures (they are `MovementMesoLinkOutcome` of some connector but `MovementMesoLinkIncome` of none). Their queues are set to 0 each step - vehicles that arrive here have exited the network.

### Step 3: Compute pressure for each stage

The current implementation normalizes road queues by storage capacity $K$. It does not use per-movement queues or turning proportions; these are model limitations, not information supplied implicitly by the meso graph.

For each connector link (movement) connecting upstream segment $u$ to downstream segment $d$, compute the movement weight:

$$w_{u,d} = S_{u,d} \cdot \left(\frac{x_u}{K_u} - \frac{x_d}{K_d}\right)$$

where:
- $x_u, x_d$ - queue lengths on upstream and downstream segments (vehicles)
- $K_l = \dfrac{\text{length}_l \cdot \text{lanes}_l}{L_{\text{veh}}}$ - storage capacity ($L_{\text{veh}} = 7$ m)
- $S_{u,d}$ - saturation flow of the connector (veh/h)

Then sum weights per stage $p$:

$$W(p) = \sum_{(u,d) \in p} w_{u,d}$$

Example (initial state, intersection A, $K_1 = 57.1$, $K_3 = 85.7$):

| Connector | $S$ | $x_u$ | $K_u$ | $x_d$ | $K_d$ | $x_u/K_u$ | $x_d/K_d$ | $w$ |
|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| 100 (EBT) | 900 | 15 | 57.1 | 5 | 85.7 | 0.263 | 0.058 | 184.1 |
| 101 (EBR) | 700 | 15 | 57.1 | 0 | 57.1 | 0.263 | 0.000 | 183.8 |
| sg0 total | | | | | | | | 367.9 |
| 102 (SBT) | 900 | 8 | 57.1 | 0 | 57.1 | 0.140 | 0.000 | 126.0 |
| 103 (SBL) | 700 | 8 | 57.1 | 5 | 85.7 | 0.140 | 0.058 | 57.3 |
| sg1 total | | | | | | | | 183.3 |

Phase selection - activate the stage with maximum pressure:

$$p^* = \arg\max_p W(p)$$

Decision: $W(\text{sg0}) = 367.9 > W(\text{sg1}) = 183.3$ $\Rightarrow$ activate sg0 (EW) at intersection A.

### Step 4: Discharge vehicles

For each active connector $m=(u,d)$, first compute the requested discharge:

$$r_m = \frac{S_m \cdot \Delta t}{3600}$$

All active movements leaving the same road share its queue proportionally:

$$a_m = r_m \cdot \min\!\left(1,\frac{x_u}{\sum_{j:\,u(j)=u} r_j}\right)$$

All movements entering the same road then share its available space:

$$d_m = a_m \cdot \min\!\left(1,\frac{\max(0,K_d-x_d)}{\sum_{j:\,d(j)=d} a_j}\right)$$

Only positive requested flows participate. Queues $x$ are measured after demand injection and boundary drainage, before any internal transfer. A road with unspecified storage capacity retains the existing unlimited receiving behavior. Repeated references to the same active connector discharge only once.

For example, one vehicle shared by two equal-capacity movements produces 0.5 vehicles on each destination. If one destination is full, its 0.5 vehicles stay upstream instead of disappearing or being reassigned to the other turn. Fractional vehicle counts represent continuous traffic flow.

Transfers are simultaneous: arrivals cannot leave a second road in the same step, and room freed by outgoing traffic becomes available on the next step. This conservative discretization is not a travel-time model.

### Step 5: Update queues

Starting with the queues after injection and drainage:

$$x_l(k+1) = x_l^{\text{pre-transfer}}(k) - \sum_{\text{out}} d_{\text{out}}(k) + \sum_{\text{in}} d_{\text{in}}(k)$$

### Step 6: Update intersection state

After every intersection has chosen its stage, record the selected stage in `PreviousStage`, mark `HasPreviousStage=true`, update the active stage, and advance simulation time by $\Delta t$. The next decision uses this completed step's actuation. A zero-value stage ID does not count as known history before the first step.

## Smoothing-MP enhancement

When $\alpha > 0$, the prototype adds a constant coordination bonus to candidate-stage movements with an actuated predecessor on a configured corridor:

$$c_{u,d}(k) = \begin{cases} 1 & \text{if a configured corridor predecessor had green at step } k-1 \\ 0 & \text{otherwise} \end{cases}$$

Then the smoothed movement weight becomes:

$$w^{\text{smooth}}_{u,d} = w_{u,d} + \alpha \cdot S_{u,d} \cdot c_{u,d}$$

The boost $\alpha \cdot S_{u,d}$ is independent of queue length. The paper's stability result has not been established for this implementation, which uses normalized road queues, finite receiving space, and no explicit turning proportions.

When $c_{u,d} = 0$, this reduces to standard max-pressure. When $c_{u,d} = 1$, the additional term $\alpha \cdot S_{u,d}$ biases the downstream intersection toward giving green to the arriving platoon.

In this example, movement 100 feeding link 3 can increase the weight of connector 200 at B. This biases B toward EW. Other movements feeding link 3 do not activate that corridor's indicator. A green stage does not itself prove that a platoon was released, and the current simulation cannot establish stop-free passage along a road.

## Output

Recorded after configuring the directed corridor and correcting actuation history. Intersection columns may appear in a different order because results are collected from a Go map.

```
=== Standard MP (alpha=0) ===
  t=  30s | total  40.2 | link3  10.0 | int60=>sg1 int50=>sg1
  t=  60s | total  46.1 | link3  10.6 | int50=>sg0 int60=>sg0
  t=  90s | total  65.6 | link3  15.3 | int50=>sg1 int60=>sg1
  t= 120s | total  84.9 | link3  20.3 | int50=>sg0 int60=>sg1
  t= 150s | total 104.4 | link3  23.1 | int50=>sg0 int60=>sg1
  t= 180s | total 124.0 | link3  28.1 | int50=>sg0 int60=>sg1
  t= 210s | total 143.6 | link3  30.8 | int60=>sg0 int50=>sg0
  t= 240s | total 163.1 | link3  35.8 | int50=>sg0 int60=>sg1
  t= 270s | total 177.1 | link3  40.3 | int50=>sg1 int60=>sg1
  t= 300s | total 182.9 | link3  42.2 | int50=>sg1 int60=>sg0
  t= 330s | total 188.2 | link3  46.7 | int50=>sg0 int60=>sg1
  t= 360s | total 192.3 | link3  50.8 | int50=>sg0 int60=>sg1
  t= 390s | total 197.6 | link3  52.8 | int50=>sg1 int60=>sg1
  t= 420s | total 201.7 | link3  55.0 | int50=>sg1 int60=>sg1
  t= 450s | total 205.0 | link3  57.2 | int50=>sg1 int60=>sg1
  t= 480s | total 208.3 | link3  59.4 | int50=>sg1 int60=>sg1
  t= 510s | total 209.2 | link3  61.7 | int50=>sg0 int60=>sg1
  t= 540s | total 212.5 | link3  61.7 | int50=>sg0 int60=>sg0
  t= 570s | total 215.8 | link3  63.9 | int50=>sg0 int60=>sg0
  t= 600s | total 219.2 | link3  66.1 | int50=>sg0 int60=>sg0
  ---
  avg total queue: 155.4 veh | max: 219.9 veh
  avg link3 queue:  40.2 veh | max:  67.1 veh
  B chose EW(sg0): 34 times | NS(sg1): 86 times  (EW% = 28%)

=== Smoothing-MP (alpha=0.5) ===
  t=  30s | total  40.2 | link3   1.1 | int50=>sg1 int60=>sg0
  t=  60s | total  46.9 | link3   2.5 | int50=>sg0 int60=>sg0
  t=  90s | total  66.4 | link3   5.3 | int50=>sg0 int60=>sg0
  t= 120s | total  86.0 | link3  10.0 | int50=>sg1 int60=>sg1
  t= 150s | total 105.6 | link3  15.0 | int50=>sg1 int60=>sg1
  t= 180s | total 124.8 | link3  17.8 | int50=>sg0 int60=>sg1
  t= 210s | total 144.4 | link3  22.8 | int50=>sg0 int60=>sg1
  t= 240s | total 163.9 | link3  25.6 | int50=>sg0 int60=>sg0
  t= 270s | total 177.9 | link3  30.3 | int50=>sg0 int60=>sg1
  t= 300s | total 183.8 | link3  34.4 | int50=>sg0 int60=>sg1
  t= 330s | total 189.1 | link3  38.6 | int50=>sg0 int60=>sg1
  t= 360s | total 195.6 | link3  40.6 | int50=>sg1 int60=>sg0
  t= 390s | total 198.4 | link3  45.0 | int60=>sg1 int50=>sg0
  t= 420s | total 202.5 | link3  46.9 | int50=>sg1 int60=>sg1
  t= 450s | total 205.8 | link3  46.9 | int60=>sg0 int50=>sg1
  t= 480s | total 209.2 | link3  49.2 | int50=>sg1 int60=>sg0
  t= 510s | total 212.5 | link3  51.4 | int50=>sg1 int60=>sg0
  t= 540s | total 215.8 | link3  53.6 | int50=>sg1 int60=>sg0
  t= 570s | total 219.2 | link3  55.8 | int60=>sg1 int50=>sg1
  t= 600s | total 222.5 | link3  58.1 | int60=>sg1 int50=>sg1
  ---
  avg total queue: 156.7 veh | max: 222.5 veh
  avg link3 queue:  31.6 veh | max:  58.1 veh
  B chose EW(sg0): 38 times | NS(sg1): 82 times  (EW% = 32%)
```

## Reading the output

Each line shows a snapshot every 30 seconds:

- `t=120s` - simulation time
- `total 84.9` - sum of all queues across the network (vehicles)
- `link3 20.3` - queue on link 3 (A->B), the coordination link between intersections
- `int50=>sg0` - intersection A activated stage 0 (EW) at this step
- `int60=>sg1` - intersection B activated stage 1 (NS) at this step

Summary metrics:
- `avg total queue` / `max` - network-wide congestion over the simulation
- `avg link3 queue` / `max` - congestion on the coordination link specifically
- `B chose EW%` - how often intersection B selected the eastbound stage

## Results comparison

These are descriptive results from the current simplified example, not a validation of the paper or a measure of all requested traffic being served.

| Metric | Standard MP | Smoothing-MP ($\alpha$=0.5) |
|:---|---:|---:|
| avg total queue | 155.4 veh | 156.7 veh |
| max total queue | 219.9 veh | 222.5 veh |
| avg link3 queue | 40.2 veh | 31.6 veh |
| max link3 queue | 67.1 veh | 58.1 veh |
| B chose EW (sg0) | 28% | 32% |

In this run, smoothing reduces the queue on the connecting road while increasing the average total queue. More green selections for EW describe the controller's preference; they do not by themselves demonstrate coordination quality. Demand rejected at full entrances is not included in these totals, so a plateau in queue length must not be interpreted as network stability.

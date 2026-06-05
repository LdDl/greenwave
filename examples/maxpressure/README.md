# Max-Pressure Example

Demonstrates the Smoothing-MP algorithm on a 2-intersection corridor.

Based on: Varaiya, P. (2013). [Max pressure control of a network of signalized intersections](https://doi.org/10.1016/j.trc.2013.08.014). Transportation Research Part C, 36, 184-195.

Normalized pressure (Modified MP): Kouvelas, A., Lioris, J., Fayazi, S.A., Varaiya, P. (2014). [Maximum pressure controller for stabilizing queues in signalized arterial networks](https://doi.org/10.3141/2421-15). Transportation Research Record, 2421(1), 133-141.

Coordination enhancement: Xu, H., Barman, S., Levin, M.W. (2024). [Smoothing-MP: A novel max-pressure signal control considering signal coordination](https://doi.org/10.1016/j.trc.2024.104782). Transportation Research Part C, 166, 104782.

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

The original MP papers (Varaiya 2013, Xu et al. 2024) use a macro graph: one node per intersection, one edge per road. Movements are abstract pairs with external turning ratios, and queues are per-movement point queues with infinite capacity. Our meso-level graph from go-gmns provides structural advantages:

| Aspect | Macro (Varaiya / Xu) | Meso (ours) |
|:---|:---|:---|
| Movement | Abstract pair (i,j) + turning ratio | Connector link with ID, satflow, type |
| Turning ratios | Required as external input | Encoded in graph structure |
| Queue model | Per-movement, point queue | Per-link, finite capacity $K$ |
| Link capacity | Infinite (assumed) | Finite: $K = length \cdot lanes / L_{veh}$ |

Consequences: turning ratios do not appear in the pressure formula (each connector = one movement). Finite capacity enables Kouvelas normalization ($x/K$). The coordination indicator $c_{u,d}$ maps directly to graph queries (`MovementMesoLinkOutcome` / `MovementMesoLinkIncome`).

## Scenario

The example simulates a 10-minute period with time-varying demand (morning rush):

| Phase | Time | Demand multiplier | Description |
|:---:|:---:|:---:|:---|
| Ramp-up | 0--60s | 0.5x -> 1.0x | Traffic builds |
| Peak | 60--300s | 1.3x | Rush hour, network oversaturated |
| Ramp-down | 300--420s | 1.3x -> 1.0x | Peak subsides |
| Recovery | 420--600s | 1.0x | Normal load |

Base demand rates:
- Link 1 (WA->A): 1600 veh/h - heavy eastbound corridor
- Link 2 (NA->A): 800 veh/h - moderate southbound
- Link 4 (NB->B): 1200 veh/h - heavy competing flow at B

At peak (x1.3), link 1 reaches 2080 veh/h - well above effective capacity (~800 veh/h per approach with 50% green share). The network is stressed, queues grow, and the algorithm must make real trade-offs.

Initial queues represent residual congestion: link1=15, link2=8, link3=5, link4=10 vehicles.

Both scenarios (Standard MP and Smoothing-MP) run on identical input data.

## Realistic 4-phase scenario

The `runRealisticScenario` function extends the basic 2-stage model to a realistic
4-phase, 4-group signal plan derived from a `junction.Junction` via `StagesFromJunction`.

### Signal plan

Each intersection has 4 signal groups:

| Group | Movement |
|:---:|:---|
| 0 | EW through |
| 1 | EW left turn |
| 2 | NS through |
| 3 | NS left turn |

4 phases per cycle (total cycle = 95s). Each active group follows GREEN->YELLOW->RED;
inactive groups stay RED throughout the phase.

| Phase | Duration | Group 0 | Group 1 | Group 2 | Group 3 |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 0 | 35s | G(30)->Y(3)->R(2) | R(35) | R(35) | R(35) |
| 1 | 20s | R(20) | G(15)->Y(3)->R(2) | R(20) | R(20) |
| 2 | 28s | R(28) | R(28) | G(23)->Y(3)->R(2) | R(28) |
| 3 | 12s | R(12) | R(12) | R(12) | G(8)->Y(2)->R(2) |

**Signal transition rule**: for each group, consecutive phases must end with the same
signal meaning -> prohibition (R, Y) or permission (G). All phases here end with R,
so all transitions are prohibition->prohibition. This prevents e.g. a phase ending
GREEN immediately followed by a phase starting GREEN for the same group.

### Network extension

Links 9, 10 (at A) and 18, 19 (at B) are added as left-turn departure links (boundary
drains). Four new connectors map left-turn groups to these links:

| ID | Movement | Upstream -> Downstream | S (veh/h) | Group | Stage |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 104 | EBL | [1] WA->A -> [9] A-left | 600 | 1 | sg1 |
| 105 | SBL | [2] NA->A -> [10] A-left | 600 | 3 | sg3 |
| 204 | EBL | [3] A->B -> [18] B-left | 600 | 1 | sg1 |
| 205 | NBL | [4] NB->B -> [19] B-left | 600 | 3 | sg3 |

### Stages derived via StagesFromJunction

Instead of listing connector IDs manually, stages are built from the junction config.
`StagesFromJunction` inspects each phase's signal groups and collects connectors for
every group with at least one GREEN signal:

```
Junction A: 4 phases -> 4 stages (cycle ~95s)
  stage 0: connectors [100 101]   (Group 0 GREEN in phase 0)
  stage 1: connectors [104]       (Group 1 GREEN in phase 1)
  stage 2: connectors [102 103]   (Group 2 GREEN in phase 2)
  stage 3: connectors [105]       (Group 3 GREEN in phase 3)
```

### Realistic scenario output

```
=== Realistic 4-phase junctions, Smoothing-MP (alpha=0.5) ===
  t=  30s | total  40.3 | link3   0.7 | int50=>sg2 int60=>sg0
  t=  60s | total  53.1 | link3   0.4 | int50=>sg0 int60=>sg0
  ...
  t= 600s | total 209.1 | link3  42.6 | int50=>sg0 int60=>sg2
  ---
  avg total queue: 161.6 veh | max: 211.0 veh
  avg link3 queue:  27.0 veh | max:  42.6 veh
  B stage selection: sg0(EW-thr)=49  sg1(EW-left)=0  sg2(NS-thr)=71  sg3(NS-left)=0  (total=120)
```

### Interpreting the realistic results

**sg1 and sg3 are never selected.** This is expected and correct. Left-turn departure
links (9, 10, 18, 19) are boundary drains -> always flushed to zero. With $x_d = 0$,
the movement weight simplifies to $w = S \cdot x_u / K_u$. However, the competing
through stages (sg0, sg2) have higher combined satflow (900+700=1600 vs 600 for sg1),
so through movements always outweigh left turns under the same upstream queue.

**Left turns would activate when:** the downstream through-link (e.g. link 3) becomes
heavily congested, reducing sg0's pressure via the $-x_d/K_d$ term. In a full network
without boundary drains, left-turn departure links accumulate queue and create genuine
competition. The current example uses isolated departures intentionally to keep the
corridor dynamics clean.

**The 49/71 split (sg0 vs sg2 at B) is identical to the 2-stage Smoothing-MP scenario.**
Adding left-turn stages does not disturb coordination -> MP correctly ignores empty stages
and focuses on competing through movements. This confirms that `StagesFromJunction`
produces stages that are behaviorally equivalent to the manually built 2-stage case
when left-turn demand is absent.

## Algorithm step-by-step

### Step 1: Inject demand

Each step, the optimizer converts intensities (veh/h) to vehicles per step:

$$\text{inject}_l(k) = \frac{q_l(t) \cdot \Delta t}{3600}$$

where $q_l(t)$ is demand intensity on link $l$ at time $t$ (veh/h) and $\Delta t$ is the step duration (seconds).

Example: WA->A at peak ($q = 2080$ veh/h), $\Delta t = 5$ s $\Rightarrow$ inject $= 2080 \cdot 5 / 3600 = 2.89$ veh/step.

### Step 2: Drain boundary departures

Links 5, 6, 7, 8 are auto-detected as boundary departures (they are `MovementMesoLinkOutcome` of some connector but `MovementMesoLinkIncome` of none). Their queues are set to 0 each step - vehicles that arrive here have exited the network.

### Step 3: Compute pressure for each stage

Note: the Original-MP (Varaiya, 2013) uses absolute queue lengths with infinite link capacity. We use the Modified MP formulation (Kouvelas et al., 2014) which normalizes queues by storage capacity $K$, accounting for finite link lengths. On the meso graph, turning ratios are encoded in the graph structure (each connector = one movement), so they do not appear in the formula.

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

For the active stage, each connector discharges vehicles from upstream to downstream:

$$d_{u,d}(k) = \min\!\left(\frac{S_{u,d} \cdot \Delta t}{3600},\; x_u(k)\right)$$

Example: connector 100 (EBT, $S = 900$), $\Delta t = 5$ s $\Rightarrow$ max discharge $= 900 \cdot 5 / 3600 = 1.25$ veh.

### Step 5: Update queues

$$x_l(k+1) = x_l(k) - \sum_{\text{out}} d_{\text{out}}(k) + \sum_{\text{in}} d_{\text{in}}(k) + \text{inject}_l(k)$$

### Step 6: Update intersection state

Record which stage was active (needed for Smoothing-MP boost in next step). Advance simulation time by $\Delta t$.

## Smoothing-MP enhancement

Following Xu et al. (2024), when $\alpha > 0$, the optimizer adds a constant coordination boost to movements whose upstream intersection just released a platoon. Define the coordination indicator:

$$c_{u,d}(k) = \begin{cases} 1 & \text{if a connector at an upstream intersection discharged into } u \text{ at step } k-1 \\ 0 & \text{otherwise} \end{cases}$$

Then the smoothed movement weight becomes:

$$w^{\text{smooth}}_{u,d} = w_{u,d} + \alpha \cdot S_{u,d} \cdot c_{u,d}$$

The boost $\alpha \cdot S_{u,d}$ is a constant (not queue-dependent) -- this matches the proven formulation from Xu et al. where stability is preserved when $\xi_{u,d} = \alpha \cdot S_{u,d} \leq Q_{u,d}^2$.

When $c_{u,d} = 0$, this reduces to standard max-pressure. When $c_{u,d} = 1$, the additional term $\alpha \cdot S_{u,d}$ biases the downstream intersection toward giving green to the arriving platoon.

In this example: when intersection A gives green to EW (sg0), vehicles discharge from link 1 into link 3. On the next step, connectors 200 and 201 at intersection B see that their upstream link (link 3) was just served by A. The constant boost increases their pressure, making B more likely to also give green to EW -- letting the platoon pass through without stopping.

## Output

```
=== Standard MP (alpha=0) ===
  t=  30s | total  40.2 | link3  10.0 | int50=>sg1 int60=>sg1
  t=  60s | total  46.1 | link3  10.6 | int50=>sg0 int60=>sg0
  t=  90s | total  65.6 | link3  15.3 | int50=>sg1 int60=>sg1
  t= 120s | total  84.9 | link3  20.3 | int50=>sg0 int60=>sg1
  t= 150s | total 104.4 | link3  23.1 | int50=>sg0 int60=>sg1
  t= 180s | total 124.0 | link3  28.1 | int50=>sg0 int60=>sg1
  t= 210s | total 143.6 | link3  30.8 | int50=>sg0 int60=>sg0
  t= 240s | total 163.1 | link3  35.8 | int60=>sg1 int50=>sg0
  t= 270s | total 177.1 | link3  40.3 | int50=>sg1 int60=>sg1
  t= 300s | total 182.9 | link3  42.2 | int50=>sg1 int60=>sg0
  t= 330s | total 188.2 | link3  46.7 | int60=>sg1 int50=>sg0
  t= 360s | total 192.3 | link3  50.8 | int50=>sg0 int60=>sg1
  t= 390s | total 197.6 | link3  52.8 | int50=>sg1 int60=>sg1
  t= 420s | total 201.7 | link3  55.0 | int50=>sg1 int60=>sg1
  t= 450s | total 205.0 | link3  57.2 | int50=>sg1 int60=>sg1
  t= 480s | total 208.3 | link3  59.4 | int50=>sg1 int60=>sg1
  t= 510s | total 209.2 | link3  61.7 | int50=>sg0 int60=>sg1
  t= 540s | total 212.5 | link3  61.7 | int60=>sg0 int50=>sg0
  t= 570s | total 215.8 | link3  63.9 | int50=>sg0 int60=>sg0
  t= 600s | total 219.2 | link3  66.1 | int50=>sg0 int60=>sg0
  ---
  avg total queue: 155.4 veh | max: 219.9 veh
  avg link3 queue:  40.2 veh | max:  67.1 veh
  B chose EW(sg0): 34 times | NS(sg1): 86 times  (EW% = 28%)

=== Smoothing-MP (alpha=0.5) ===
  t=  30s | total  40.3 | link3   0.7 | int60=>sg0 int50=>sg1
  t=  60s | total  53.1 | link3   0.4 | int50=>sg0 int60=>sg0
  t=  90s | total  75.2 | link3   2.9 | int50=>sg0 int60=>sg0
  t= 120s | total  94.8 | link3   7.9 | int50=>sg0 int60=>sg1
  t= 150s | total 114.3 | link3  12.6 | int50=>sg1 int60=>sg1
  t= 180s | total 133.6 | link3  15.4 | int50=>sg0 int60=>sg0
  t= 210s | total 153.1 | link3  20.4 | int50=>sg0 int60=>sg1
  t= 240s | total 172.7 | link3  25.4 | int50=>sg0 int60=>sg1
  t= 270s | total 189.2 | link3  27.9 | int50=>sg1 int60=>sg1
  t= 300s | total 195.0 | link3  32.1 | int50=>sg1 int60=>sg1
  t= 330s | total 200.3 | link3  36.2 | int50=>sg1 int60=>sg1
  t= 360s | total 204.4 | link3  38.2 | int50=>sg1 int60=>sg1
  t= 390s | total 207.2 | link3  40.4 | int50=>sg0 int60=>sg0
  t= 420s | total 209.1 | link3  42.6 | int50=>sg0 int60=>sg1
  t= 450s | total 209.1 | link3  42.6 | int50=>sg0 int60=>sg1
  t= 480s | total 209.1 | link3  42.6 | int50=>sg0 int60=>sg1
  t= 510s | total 209.1 | link3  42.6 | int50=>sg0 int60=>sg1
  t= 540s | total 209.1 | link3  42.6 | int50=>sg0 int60=>sg1
  t= 570s | total 209.1 | link3  42.6 | int50=>sg0 int60=>sg1
  t= 600s | total 209.1 | link3  42.6 | int60=>sg1 int50=>sg0
  ---
  avg total queue: 161.6 veh | max: 211.0 veh
  avg link3 queue:  27.0 veh | max:  42.6 veh
  B chose EW(sg0): 49 times | NS(sg1): 71 times  (EW% = 41%)
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
- `B chose EW%` - how often intersection B gave green to the eastbound corridor (higher = better coordination with A)

## Results comparison

| Metric | Standard MP | Smoothing-MP ($\alpha$=0.5) | Difference |
|:---|---:|---:|---:|
| avg total queue | 155.4 veh | 161.6 veh | +4% |
| max total queue | 219.9 veh | 211.0 veh | -4% |
| avg link3 queue | 40.2 veh | 27.0 veh | -33% |
| max link3 queue | 67.1 veh | 42.6 veh | -36% |
| B chose EW (sg0) | 28% | 41% | +13 p.p. |

Observations:

1. The coordination link (link 3, A->B) shows a 33% reduction in average queue and 36% in peak queue. This is the core benefit: Smoothing-MP keeps the inter-intersection corridor clearer by coordinating B's green with A's platoon release.

2. Intersection B chose EW (sg0) 41% of the time with smoothing vs 28% without. The constant boost $\alpha \cdot S$ makes B respond to platoons arriving from A.

3. Total network queue is slightly higher (+4% average) -- a trade-off. The NS approaches at B get less green time because EW is boosted. This is expected: coordination improves corridor throughput at the cost of cross-street delay.

4. Smoothing-MP stabilizes earlier (total queue stops growing at t=420s vs continuing through t=600s with standard MP). Peak total queue is lower (211 vs 220).

5. During recovery (t>420s), standard MP still has link3 queue growing (55 -> 66), while Smoothing-MP plateaus at ~43. The coordination effect is most visible after peak demand subsides.

## Code to math mapping

| Formula | Code function | File |
|:---|:---|:---|
| $w_{u,d}$ (movement weight) | `Network.MovementWeight()` | `pressure.go` |
| $W(p)$ (phase pressure) | `Network.PhasePressure()` | `pressure.go` |
| $p^* = \arg\max W$ (phase selection) | `Network.SelectPhase()` | `pressure.go` |
| $w^{\text{smooth}}_{u,d} = w + \alpha \cdot S \cdot c$ (Xu et al.) | `Network.SmoothedMovementWeight()` | `smoothing.go` |
| $c_{u,d}$ (coordination indicator) | `Network.IsUpstreamServed()` | `smoothing.go` |
| Queue dynamics $x_l(k+1)$ | `MPOptimizer.Step()` | `optimizer.go` |
| $\text{inject}_l(k)$ | `DemandFunc` / `ConstantDemand()` | `optimizer.go` |
| Boundary drain | `MPOptimizer.detectBoundaryDepartures()` | `optimizer.go` |

# Shared project model

Developer reference for the document shared by `/`, `/network` and `/pressure`. For editing instructions, see [Network editor](NETWORK_EDITOR.md) and [Max pressure](MAX_PRESSURE.md).

## Project JSON

The format is `"greenwave-project"`, version `1`. Parsing and validation live in [shared-project.js](src/lib/utils/shared-project.js).

| Field | Contents |
| --- | --- |
| `name` | Project name |
| `junctions` | IDs, labels, offsets and `cycle[].signal_groups[].signals` programs |
| `positions` | Canvas `{ x, y }` coordinates keyed by junction ID |
| `roads` | `id`, `from`, `to`, `lanes_fwd`, `lanes_back`, `lengthMeters` |
| `movements` | Junction ID to movement ID to permanent signal group ID, or `null` for forbidden movements |
| `corridors`, `activeCorridorId` | Routes and the selected route ID. An empty project contains one empty route |
| `mpScenarios`, `activeMPScenarioId` | Optional non-empty list of `{ id, name, settings }` and its selected ID |

A corridor stores `id`, `name`, ordered distinct `nodeIds`, `desiredSpeed`, `desiredIntensity`, `direction`, `origin`, `automatic`, and optional `groupIds` and `reverseGroupIds` keyed by junction ID. Missing reverse selections inherit forward groups. `automatic` allows generated input chains to adjust return lanes when direction changes. Manual topology or route edits disable it.

Programs, offsets and movement assignments belong to physical junctions. Corridors reference those junctions and choose groups for each direction. MP scenarios share the same network and programs, with separate traffic and control settings. Their settings and validation are defined in [mp-scenario.js](src/lib/utils/mp-scenario.js).

Canvas coordinates are independent of road lengths in meters. Each junction pair has at most one road, with 1 to 12 forward lanes and 0 to 12 reverse lanes. An unknown length is `null`.

## State and projections

- [corridor.js](src/lib/stores/corridor.js) owns the project and editing commands. `snapshot()` copies the whole document, while `inputSnapshot()` copies the active corridor input.
- [core.js](src/lib/stores/core.js) exposes writable views of the active corridor through those commands.
- `graphView()` returns the complete graph. `corridorProjection()` returns `{ input, error }`, using a common `point.x` and accumulated road lengths for `point.y`, starting at the corridor's `origin`.
- [mp-network.js](src/lib/utils/mp-network.js) builds MP requests from stable road-direction and movement keys. Generated mesoscopic IDs belong to the request.

Saving a project and calculating it have different validation requirements. Missing route connections, unknown lengths and incomplete turning totals can be saved. Calculation checks reject these and inconsistent movement/group selections.

## History, storage and results

Project commands share up to 100 Undo/Redo steps. `begin()` and `finish()` group continuous edits into one step. Junction dialogs keep local drafts and apply them as one project command.

The root layout connects the store to localStorage under `greenwave.project.v1`. An invalid saved project is kept without automatic replacement. Results and history stay in memory and do not survive reloads.

Green-wave `calculationRevision` tracks the active corridor, projected programs, distances, speed, direction, group choices and route errors. Layout and intensity changes do not advance it. `applyOffsets()` requires a matching revision and changes junction offsets.

MP results use a serialized snapshot of the whole project. `applyMPPrograms()` requires that snapshot to match and validates proposed durations against the existing program structure, cycles, offsets and timing constraints. Either application is one undoable change to the shared junctions.

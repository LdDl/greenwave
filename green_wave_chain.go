package greenwave

// GreenWaveChain represents a chain of green waves. Wraps a slice of GreenWave basically
type GreenWaveChain struct {
	greenWaves []*GreenWave
}

// WaveID is a struct that uniquely identifies a wave within a segment.
type WaveID struct {
	SegmentIdx int // Index of the segment
	WaveIdx    int // Index of the wave within the segment
}

// FindWaveConnections builds a connection map between waves using their indices.
func FindWaveConnections(segmentsWaves [][]*GreenWave) map[WaveID][]WaveID {
	// Initialize connections map
	connections := make(map[WaveID][]WaveID)
	// Iterate through segments
	for segIdx := 0; segIdx < len(segmentsWaves)-1; segIdx++ {
		currentSegment := segmentsWaves[segIdx]
		nextSegment := segmentsWaves[segIdx+1]
		for waveFromIdx, waveFrom := range currentSegment {
			waveFromID := WaveID{SegmentIdx: segIdx, WaveIdx: waveFromIdx}
			connections[waveFromID] = []WaveID{}
			for waveToIdx, waveTo := range nextSegment {
				// Waves can only connect if they are in the same phase (@todo: consider phase transitions)
				if waveFrom.intervalJunTwo.PhaseIdx != waveTo.intervalJunOne.PhaseIdx {
					continue
				}
				// Check if intervals can connect
				if intersection := waveFrom.intervalJunTwo.CanConnect(waveTo.intervalJunOne); intersection != nil {
					connections[waveFromID] = append(connections[waveFromID], WaveID{SegmentIdx: segIdx + 1, WaveIdx: waveToIdx})
				}
			}
		}
	}
	return connections
}

// BuildChainFunctional is a pure recursive function that builds chains of waves without mutation.
func BuildChainFunctional(waveID WaveID, connections map[WaveID][]WaveID, currentPath []WaveID) [][]WaveID {
	// Base case: if no connections, return current path
	if nextWaves, exists := connections[waveID]; !exists || len(nextWaves) == 0 {
		return [][]WaveID{currentPath}
	}
	var allChains [][]WaveID
	for _, nextWaveID := range connections[waveID] {
		newPath := make([]WaveID, len(currentPath)+1)
		copy(newPath, currentPath)
		newPath[len(currentPath)] = nextWaveID
		chains := BuildChainFunctional(nextWaveID, connections, newPath)
		allChains = append(allChains, chains...)
	}
	return allChains
}

// AdjustWaveByConnection adjusts a wave based on the connection with another wave and an overlap interval.
func AdjustWaveByConnection(waveFrom *GreenWave, waveTo *GreenWave, overlap *GreenInterval) *GreenWave {
	// Create a new adjusted wave based on waveTo
	adjustedWave := waveTo.Clone()
	// Calculate deltas for start and end
	deltaStart := overlap.Start - waveTo.intervalJunOne.Start
	deltaEnd := overlap.End - waveTo.intervalJunOne.End
	// Apply adjustments to the adjusted wave
	adjustedWave.intervalJunOne = overlap
	adjustedWave.intervalJunTwo.Start += deltaStart
	adjustedWave.intervalJunTwo.End += deltaEnd
	adjustedWave.bandwidth = adjustedWave.intervalJunTwo.End - adjustedWave.intervalJunTwo.Start
	return adjustedWave
}

// CreateAdjustedSegments creates adjusted segments with corrected wave connections.
func CreateAdjustedSegments(segmentsWaves [][]*GreenWave) [][]*GreenWave {
	// Start with cloned segments
	adjustedSegments := make([][]*GreenWave, len(segmentsWaves))
	for i, segmentWaves := range segmentsWaves {
		adjustedSegments[i] = make([]*GreenWave, len(segmentWaves))
		for j, wave := range segmentWaves {
			adjustedSegments[i][j] = wave.Clone()
		}
	}
	// Process each segment pair
	for segIdx := 0; segIdx < len(adjustedSegments)-1; segIdx++ {
		currentSegment := adjustedSegments[segIdx]
		nextSegment := adjustedSegments[segIdx+1]
		newNextSegment := []*GreenWave{}
		for _, waveFrom := range currentSegment {
			for _, waveTo := range nextSegment {
				if waveFrom.intervalJunTwo.PhaseIdx != waveTo.intervalJunOne.PhaseIdx {
					continue
				}
				if intersection := waveFrom.intervalJunTwo.CanConnect(waveTo.intervalJunOne); intersection != nil {
					adjustedWave := AdjustWaveByConnection(waveFrom, waveTo, intersection)
					newNextSegment = append(newNextSegment, adjustedWave)
				}
			}
		}
		// Update the next segment with adjusted waves
		adjustedSegments[segIdx+1] = newNextSegment
	}
	return adjustedSegments
}

// ExtractChains extracts chains of green waves from segments using functional programming principles.
func ExtractChains(segmentsWaves [][]*GreenWave) []GreenWaveChain {
	// Create adjusted segments with proper connections
	adjustedSegments := CreateAdjustedSegments(segmentsWaves)
	// Build connection map using indices
	connections := FindWaveConnections(adjustedSegments)
	// Find all chains starting from first segment
	allChains := make([][]WaveID, 0, len(adjustedSegments[0])*2) // Rough estimate
	for waveIdx := 0; waveIdx < len(adjustedSegments[0]); waveIdx++ {
		startWaveID := WaveID{SegmentIdx: 0, WaveIdx: waveIdx}
		chains := BuildChainFunctional(startWaveID, connections, []WaveID{startWaveID})
		allChains = append(allChains, chains...)
	}
	// Convert chains from indices back to GreenWave objects
	resultChains := make([]GreenWaveChain, 0, len(allChains))
	for _, chain := range allChains {
		if len(chain) < 2 { // Skip single-wave chains
			continue
		}
		waveObjects := make([]*GreenWave, len(chain))
		for i, waveID := range chain {
			waveObjects[i] = adjustedSegments[waveID.SegmentIdx][waveID.WaveIdx]
		}
		resultChains = append(resultChains, GreenWaveChain{greenWaves: waveObjects})
	}
	return resultChains
}

// MergeGreenWaves merges chains of connected green waves into through green waves.
func MergeGreenWaves(segmentsWaves [][]*GreenWave) []*ThroughGreenWave {
	possibleChains := ExtractChains(segmentsWaves)
	throughWaves := make([]*ThroughGreenWave, 0, len(possibleChains))
	for _, chain := range possibleChains {
		// A through wave must pass through at least 2 segments
		if len(chain.greenWaves) < 2 {
			continue
		}
		// Prepare adjusted waves
		adjustedWaves := make([]*GreenWave, len(chain.greenWaves))
		for i, wave := range chain.greenWaves {
			adjustedWaves[i] = wave.Clone()
		}
		for i := len(adjustedWaves) - 1; i > 0; i-- {
			current := adjustedWaves[i]    // Current wave (later)
			previous := adjustedWaves[i-1] // Previous wave (earlier)
			if intersection := previous.intervalJunTwo.CanConnect(current.intervalJunOne); intersection != nil {
				current.intervalJunOne = intersection
				deltaStart := intersection.Start - previous.intervalJunTwo.Start
				deltaEnd := intersection.End - previous.intervalJunTwo.End
				previous.intervalJunTwo.Start = intersection.Start
				previous.intervalJunTwo.End = intersection.End
				previous.intervalJunOne.Start += deltaStart
				previous.intervalJunOne.End += deltaEnd
				previous.bandwidth = previous.intervalJunTwo.End - previous.intervalJunTwo.Start
			}
		}
		var intervals []*GreenInterval
		intervals = append(intervals, adjustedWaves[0].intervalJunOne)
		for _, wave := range adjustedWaves {
			intervals = append(intervals, wave.intervalJunTwo)
		}
		throughWaves = append(throughWaves, NewThroughGreenWave(intervals))
	}
	return throughWaves
}

package dto

import "github.com/LdDl/greenwave"

// JunctionToDTO converts a Junction to a DTO
func JunctionToDTO(junction *greenwave.Junction) JunctionDTO {
	cycleDTO := make([]PhaseDTO, len(junction.Cycle))
	for i, phase := range junction.Cycle {
		cycleDTO[i] = PhaseToDTO(phase)
	}

	point := junction.GetPoint()
	return JunctionDTO{
		ID:            junction.ID,
		Label:         junction.Label,
		Cycle:         cycleDTO,
		TotalDuration: junction.GetTotalDuration(),
		Offset:        junction.GetOffset(),
		Point:         PointDTO{X: point.X, Y: point.Y},
	}
}

// PhaseToDTO converts a Phase to a DTO
func PhaseToDTO(phase *greenwave.Phase) PhaseDTO {
	signalsDTO := make([]SignalDTO, len(phase.Signals))
	for i, signal := range phase.Signals {
		signalsDTO[i] = SignalToDTO(signal)
	}

	return PhaseDTO{
		ID:           phase.ID,
		Signals:      signalsDTO,
		TotalSeconds: phase.GetTotalSeconds(),
	}
}

// SignalToDTO converts a Signal to a DTO
func SignalToDTO(signal *greenwave.Signal) SignalDTO {
	return SignalDTO{
		Duration:    signal.Duration,
		MinDuration: &signal.MinDuration,
		MaxDuration: &signal.MaxDuration,
		Color:       signal.Color.String(),
	}
}

// GreenIntervalToDTO converts a GreenInterval to a DTO
func GreenIntervalToDTO(interval *greenwave.GreenInterval) *GreenIntervalDTO {
	if interval == nil {
		return nil
	}
	return &GreenIntervalDTO{
		PhaseIdx: interval.PhaseIdx,
		Start:    interval.Start,
		End:      interval.End,
	}
}

// GreenWaveToDTO converts a GreenWave to a DTO
func GreenWaveToDTO(wave *greenwave.GreenWave) GreenWaveDTO {
	wave.Clone()
	return GreenWaveDTO{
		IntervalJunOne: GreenIntervalToDTO(wave.IntervalJunOne()),
		IntervalJunTwo: GreenIntervalToDTO(wave.IntervalJunTwo()),
		Distance:       wave.Distance(),
		TravelTime:     wave.TravelTime(),
		BandWidth:      wave.Bandwidth(),
	}
}

// ThroughGreenWaveToDTO converts a ThroughGreenWave to a DTO
func ThroughGreenWaveToDTO(wave *greenwave.ThroughGreenWave) ThroughGreenWaveDTO {
	intervals := wave.GetIntervals()
	intervalDTOs := make([]GreenIntervalDTO, len(intervals))
	for i, interval := range intervals {
		intervalDTOs[i] = *GreenIntervalToDTO(interval)
	}
	return ThroughGreenWaveDTO{
		Intervals: intervalDTOs,
		Depth:     wave.Depth(),
		Bandwidth: wave.Bandwidth(),
	}
}

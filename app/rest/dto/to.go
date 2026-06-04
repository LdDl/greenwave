package dto

import (
	"github.com/LdDl/greenwave"
	"github.com/LdDl/greenwave/greeninterval"
	"github.com/LdDl/greenwave/junction"
)

// JunctionToDTO converts a Junction to a DTO
func JunctionToDTO(jun *junction.Junction) JunctionDTO {
	cycleDTO := make([]PhaseDTO, len(jun.Cycle))
	for i, phase := range jun.Cycle {
		cycleDTO[i] = PhaseToDTO(phase)
	}

	point := jun.GetPoint()
	return JunctionDTO{
		ID:            jun.ID,
		Label:         jun.Label,
		Cycle:         cycleDTO,
		TotalDuration: jun.GetTotalDuration(),
		Offset:        jun.GetOffset(),
		Point:         PointDTO{X: point.X, Y: point.Y},
	}
}

// PhaseToDTO converts a Phase to a DTO
func PhaseToDTO(phase *junction.Phase) PhaseDTO {
	signalsDTO := make([]SignalDTO, len(phase.Signals))
	for i, sig := range phase.Signals {
		signalsDTO[i] = SignalToDTO(sig)
	}

	return PhaseDTO{
		ID:           phase.ID,
		Signals:      signalsDTO,
		TotalSeconds: phase.GetTotalSeconds(),
	}
}

// SignalToDTO converts a Signal to a DTO
func SignalToDTO(sig *junction.Signal) SignalDTO {
	return SignalDTO{
		Duration:    sig.Duration,
		MinDuration: &sig.MinDuration,
		MaxDuration: &sig.MaxDuration,
		Color:       sig.Color.String(),
	}
}

// GreenIntervalToDTO converts a GreenInterval to a DTO
func GreenIntervalToDTO(interval *greeninterval.GreenInterval) *GreenIntervalDTO {
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

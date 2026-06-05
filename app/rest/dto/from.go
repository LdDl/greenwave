package dto

import (
	"strings"

	"github.com/LdDl/greenwave"
	"github.com/LdDl/greenwave/color"
)

// JunctionFromDTO creates a Junction from a DTO
func JunctionFromDTO(dto JunctionDTO) *greenwave.Junction {
	cycle := make([]*greenwave.Phase, len(dto.Cycle))
	for i, phaseDTO := range dto.Cycle {
		cycle[i] = PhaseFromDTO(phaseDTO)
	}

	junction := greenwave.NewJunction(cycle,
		greenwave.WithID(dto.ID),
		greenwave.WithLabel(dto.Label),
		greenwave.WithPoint(greenwave.Point{X: dto.Point.X, Y: dto.Point.Y}))

	// Set offset if provided
	junction.SetOffset(dto.Offset)

	return junction
}

// PhaseFromDTO creates a Phase from a DTO
func PhaseFromDTO(dto PhaseDTO) *greenwave.Phase {
	signals := make([]*greenwave.Signal, len(dto.Signals))
	for i, signalDTO := range dto.Signals {
		signals[i] = SignalFromDTO(signalDTO)
	}
	return greenwave.NewPhase(dto.ID, signals)
}

// SignalFromDTO creates a Signal from a DTO
func SignalFromDTO(dto SignalDTO) *greenwave.Signal {
	var signalColor color.Color
	switch strings.ToUpper(dto.Color) {
	case "UNDEFINED":
		signalColor = color.UNDEFINED
	case "RED":
		signalColor = color.RED
	case "YELLOW":
		signalColor = color.YELLOW
	case "GREEN":
		signalColor = color.GREEN
	case "GREENPRIORITY":
		signalColor = color.GREENPRIORITY
	case "GREENRIGHT":
		signalColor = color.GREENRIGHT
	case "REDYELLOW":
		signalColor = color.REDYELLOW
	case "BLINKING":
		signalColor = color.BLINKING
	case "NO":
		signalColor = color.NO
	default:
		signalColor = color.UNDEFINED // Default fallback
	}

	signal := greenwave.NewSignal(dto.Duration, signalColor)
	if dto.MinDuration != nil {
		greenwave.WithMinDuration(*dto.MinDuration)(signal)
	}
	if dto.MaxDuration != nil {
		greenwave.WithMaxDuration(*dto.MaxDuration)(signal)
	}
	return signal
}

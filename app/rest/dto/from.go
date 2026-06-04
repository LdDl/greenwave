package dto

import (
	"strings"

	"github.com/LdDl/greenwave/color"
	"github.com/LdDl/greenwave/geom"
	"github.com/LdDl/greenwave/junction"
)

// JunctionFromDTO creates a Junction from a DTO
func JunctionFromDTO(dto JunctionDTO) *junction.Junction {
	cycle := make([]*junction.Phase, len(dto.Cycle))
	for i, phaseDTO := range dto.Cycle {
		cycle[i] = PhaseFromDTO(phaseDTO)
	}

	jun := junction.NewJunction(cycle,
		junction.WithID(dto.ID),
		junction.WithLabel(dto.Label),
		junction.WithPoint(geom.Point{X: dto.Point.X, Y: dto.Point.Y}))

	// Set offset if provided
	jun.SetOffset(dto.Offset)

	return jun
}

// PhaseFromDTO creates a Phase from a DTO
func PhaseFromDTO(dto PhaseDTO) *junction.Phase {
	signals := make([]*junction.Signal, len(dto.Signals))
	for i, signalDTO := range dto.Signals {
		signals[i] = SignalFromDTO(signalDTO)
	}
	return junction.NewPhase(dto.ID, signals)
}

// SignalFromDTO creates a Signal from a DTO
func SignalFromDTO(dto SignalDTO) *junction.Signal {
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

	sig := junction.NewSignal(dto.Duration, signalColor)
	if dto.MinDuration != nil {
		junction.WithMinDuration(*dto.MinDuration)(sig)
	}
	if dto.MaxDuration != nil {
		junction.WithMaxDuration(*dto.MaxDuration)(sig)
	}
	return sig
}

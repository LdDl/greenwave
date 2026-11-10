package maxpressure

import "github.com/LdDl/go-gmns/gmns"

// DemandFunc returns traffic intensity (veh/h) for an entry road at a given
// simulation time (seconds). The optimizer calls it once per step and entry.
type DemandFunc func(linkID gmns.LinkID, timeSec float64) float64

type constantDemand map[gmns.LinkID]float64

// ConstantDemand supplies a time-independent intensity for each entry road.
func ConstantDemand(rates map[gmns.LinkID]float64) DemandFunc {
	return constantDemand(rates).intensity
}

func (rates constantDemand) intensity(linkID gmns.LinkID, _ float64) float64 {
	return rates[linkID]
}

// PeakDemandConfig parameterises a time-varying peak demand profile.
type PeakDemandConfig struct {
	// BaseRates maps each link ID to its base demand intensity (veh/h).
	BaseRates map[gmns.LinkID]float64
	// RampUpS is the duration of a linear rise from 0.5 to 1.0 times base demand.
	// Zero disables the ramp-up.
	RampUpS float64
	// PeakFactor is the demand multiplier during the peak phase. Defaults to 1.3.
	PeakFactor float64
	// PeakDurationS is the duration at peak demand. Zero skips the peak.
	PeakDurationS float64
	// RampDownS is the duration of a linear decline from PeakFactor to 1.0.
	// Zero disables the ramp-down.
	RampDownS float64
}

// PeakDemand supplies a ramp-up, peak, ramp-down and steady demand profile.
func PeakDemand(cfg PeakDemandConfig) DemandFunc {
	if cfg.PeakFactor <= 0 {
		cfg.PeakFactor = 1.3
	}
	return cfg.intensity
}

func (cfg PeakDemandConfig) intensity(linkID gmns.LinkID, timeSec float64) float64 {
	rate := cfg.BaseRates[linkID]
	if rate == 0 {
		return 0
	}
	peakStart := cfg.RampUpS
	peakEnd := peakStart + cfg.PeakDurationS
	rampDownEnd := peakEnd + cfg.RampDownS
	var multiplier float64
	switch {
	case timeSec < peakStart:
		if peakStart > 0 {
			multiplier = 0.5 + 0.5*(timeSec/peakStart)
		} else {
			multiplier = 1.0
		}
	case timeSec < peakEnd:
		multiplier = cfg.PeakFactor
	case timeSec < rampDownEnd:
		if cfg.RampDownS > 0 {
			multiplier = cfg.PeakFactor - (cfg.PeakFactor-1.0)*((timeSec-peakEnd)/cfg.RampDownS)
		} else {
			multiplier = 1.0
		}
	default:
		multiplier = 1.0
	}
	return rate * multiplier
}

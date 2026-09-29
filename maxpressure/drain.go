package maxpressure

import "github.com/LdDl/go-gmns/gmns"

// DrainFunc computes vehicles to remove from a boundary exit in one step.
// The optimizer caps the result at the current queue. Nil drains the entire queue.
type DrainFunc func(linkID gmns.LinkID, queue float64, deltaT float64) float64

// NoDrain keeps vehicles at exits instead of letting them leave the network.
func NoDrain() DrainFunc {
	return noDrain
}

func noDrain(_ gmns.LinkID, _ float64, _ float64) float64 {
	return 0
}

type rateDrain map[gmns.LinkID]float64

// RateDrain removes vehicles at the specified veh/h rate for each exit.
// A missing rate drains the entire queue; an explicit zero blocks that exit.
func RateDrain(rates map[gmns.LinkID]float64) DrainFunc {
	return rateDrain(rates).discharge
}

func (rates rateDrain) discharge(linkID gmns.LinkID, queue float64, deltaT float64) float64 {
	rate, ok := rates[linkID]
	if !ok {
		return queue
	}
	if rate <= 0 {
		return 0
	}
	return rate * deltaT / 3600.0
}

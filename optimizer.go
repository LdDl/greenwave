package greenwave

// Optimizer is an interface that defines a method for optimizing offsets for traffic lights signal timing.
type Optimizer interface {
	// Optimize calculates the optimal offsets for traffic lights and returns them as a slice of float64.
	Optimize() []float64
}

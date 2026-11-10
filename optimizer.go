package greenwave

// Optimizer is an interface that defines a method for optimizing offsets for traffic lights signal timing.
type Optimizer interface {
	// Optimize returns the selected offsets in whole seconds, normalized to each
	// junction's cycle. The float64 slice preserves the public return type.
	Optimize() []float64
}

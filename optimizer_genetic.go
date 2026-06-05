package greenwave

import (
	"math"
	"math/rand/v2"

	"github.com/LdDl/greenwave/junction"
)

// OptimizerGenetic implements a genetic algorithm for optimizing traffic light offsets
type CrossoverType uint8

const (
	// CROSSOVER_BLEND uses a blend crossover method where offspring are created by blending the parents' offsets
	CROSSOVER_BLEND CrossoverType = iota
	// CROSSOVER_UNIFORM uses a uniform crossover method where offspring are created by randomly selecting offsets from each parent
	CROSSOVER_UNIFORM

	// @todo: consider making CONVERGENCE_GENERATIONS configurable
	// Number of generations without fitness improvement before early stopping (convergence)
	CONVERGENCE_GENERATIONS = 15
)

var crossoverTypeToStr = [...]string{"blend", "uniform"}

// String returns the string representation of the CrossoverType
func (ioutIndex CrossoverType) String() string {
	return crossoverTypeToStr[ioutIndex]
}

// OptimizationMode defines the direction(s) for green wave optimization
type OptimizationMode uint8

const (
	// OPTIMIZATION_FORWARD optimizes green waves for forward direction only (junction 0 -> 1 -> 2 -> ...)
	OPTIMIZATION_FORWARD OptimizationMode = iota
	// OPTIMIZATION_BIDIRECTIONAL optimizes green waves for both forward and reverse directions
	OPTIMIZATION_BIDIRECTIONAL
)

var optimizationModeToStr = [...]string{"forward", "bidirectional"}

// String returns the string representation of the OptimizationMode
func (mode OptimizationMode) String() string {
	return optimizationModeToStr[mode]
}

// Individual represents a single solution in the genetic algorithm
type Individual struct {
	Offsets []float64
	Fitness float64
}

// OptimizerGenetic is a genetic algorithm optimizer for traffic light offsets
type OptimizerGenetic struct {
	// contains the traffic junctions to optimize
	junctions []*junction.Junction
	// speedKhm is the speed in kilometers per hour used for calculating offsets
	speedKhm float64
	// populationSize is the number of individuals in the population
	populationSize int
	// generations is the number of generations to run the genetic algorithm
	generations int
	// mutationRate is the probability of a mutation occurring in an individual
	mutationRate float64
	// tournamentSize is the number of individuals to select for tournament selection
	tournamentSize int
	// crossoverType defines the type of crossover to use in the genetic algorithm
	crossoverType CrossoverType
	// optimizationMode defines whether to optimize for forward only or bidirectional traffic
	optimizationMode OptimizationMode
	// crossoverFunc is the function used for crossover between two parents
	crossoverFunc func(cycleLengths []float64, parent1, parent2 *Individual) *Individual
	// cycleLengths contains the total duration of each junction in seconds
	cycleLengths []float64
	// groupIDs maps each junction ID to the signal group used for green wave coordination in this corridor.
	// A junction may have multiple signal groups; only one group represents the through-movement per corridor.
	groupIDs map[int]junction.GroupID
	// bestFitenessHistory keeps track of the best fitness value in each generation
	bestFitenessHistory []float64
}

// NewOptimizerGenetic creates a new instance of OptimizerGenetic with the provided parameters.
// groupIDs maps each junction ID to the signal group to use for green wave coordination in this corridor.
// A junction may have multiple signal groups (e.g. northbound, eastbound, pedestrian); only one group represents
// the through-movement for a given corridor. The caller is responsible for providing the correct group per junction.
func NewOptimizerGenetic(junctions []*junction.Junction, groupIDs map[int]junction.GroupID, speedKhm float64, populationSize int, generations int, mutationRate float64, tournamentSize int, crossoverType CrossoverType, optimizationMode OptimizationMode) Optimizer {
	cycleLengths := make([]float64, len(junctions))
	for i, jun := range junctions {
		cycleLengths[i] = float64(jun.GetTotalDuration())
	}
	crossoverFunc := blendCrossover
	if crossoverType == CROSSOVER_UNIFORM {
		crossoverFunc = uniformCrossover
	}
	return &OptimizerGenetic{
		junctions:           junctions,
		groupIDs:            groupIDs,
		speedKhm:            speedKhm,
		populationSize:      populationSize,
		generations:         generations,
		mutationRate:        mutationRate,
		tournamentSize:      tournamentSize,
		crossoverType:       crossoverType,
		optimizationMode:    optimizationMode,
		crossoverFunc:       crossoverFunc,
		cycleLengths:        cycleLengths,
		bestFitenessHistory: make([]float64, 0, generations),
	}
}

func randomFloat(min, max float64) float64 {
	// Generate a random float64 between min and max
	return min + (max-min)*rand.Float64()
}

func (optga *OptimizerGenetic) createIndividual() *Individual {
	// Create a new individual with random offsets
	offsets := make([]float64, len(optga.cycleLengths))
	offsets[0] = 0.0 // The first offset is always 0.0
	for i := 1; i < len(offsets); i++ {
		offsets[i] = randomFloat(0, optga.cycleLengths[i])
	}
	return &Individual{Offsets: offsets, Fitness: 0.0}
}

// EvaluateFitness calculates the fitness of an individual based on the traffic light offsets
func (optga *OptimizerGenetic) evaluateFitness(individual *Individual) float64 {
	// Apply the offsets to the junctions
	for i, jun := range optga.junctions {
		jun.SetOffset(int(individual.Offsets[i]))
	}

	// Calculate forward fitness
	forwardFitness := calculateDirectionalFitness(optga.junctions, optga.groupIDs, optga.speedKhm)

	// If bidirectional mode, also calculate reverse fitness
	if optga.optimizationMode == OPTIMIZATION_BIDIRECTIONAL {
		reversedJunctions := ReverseJunctions(optga.junctions)
		reverseFitness := calculateDirectionalFitness(reversedJunctions, optga.groupIDs, optga.speedKhm)
		// Combine forward and reverse fitness (equal weight)
		return forwardFitness + reverseFitness
	}

	return forwardFitness
}

// calculateDirectionalFitness calculates fitness for a given direction (order of junctions).
// groupIDs maps each junction ID to the signal group to use for green wave coordination in this corridor.
func calculateDirectionalFitness(junctions []*junction.Junction, groupIDs map[int]junction.GroupID, speedKhm float64) float64 {
	greenWavs := FindGreenWaves(junctions, groupIDs, speedKhm)
	throughGreenWaves := MergeGreenWaves(greenWavs)
	if len(throughGreenWaves) == 0 {
		return 0.0 // No green waves found
	}
	// Calculate total fitness based on the depth and band size of the green waves
	maxDepth := len(junctions)
	totalFitness := 0.0
	for _, wave := range throughGreenWaves {
		depthRatio := float64(wave.Depth()) / float64(maxDepth)
		// Square the depth ratio to emphasize deeper wave
		waveFitness := depthRatio * depthRatio * float64(wave.Bandwidth())
		totalFitness += waveFitness
	}
	return totalFitness
}

// ReverseJunctions returns a new slice with junctions in reverse order
// Note: it contains pointers to the same Junction objects
func ReverseJunctions(junctions []*junction.Junction) []*junction.Junction {
	n := len(junctions)
	reversed := make([]*junction.Junction, n)
	for i := 0; i < n; i++ {
		reversed[i] = junctions[n-1-i]
	}
	return reversed
}

func (optga *OptimizerGenetic) selectParent(population []*Individual) *Individual {
	// Select a parent using tournament selection
	tournament := make([]*Individual, optga.tournamentSize)
	for i := range tournament {
		tournament[i] = population[rand.IntN(len(population))]
	}
	// Return the individual with the highest fitness in the tournament
	bestParent := tournament[0]
	for _, ind := range tournament[1:] {
		if ind.Fitness > bestParent.Fitness {
			bestParent = ind
		}
	}
	return bestParent
}

// blendCrossover performs a blend crossover between two parents
func blendCrossover(cycleLengths []float64, parent1, parent2 *Individual) *Individual {
	// Create a child by blending the offsets of the parents
	childOffsets := make([]float64, len(cycleLengths))
	childOffsets[0] = 0.0 // The first offset is always 0.0
	for i := 1; i < len(childOffsets); i++ {
		weight := rand.Float64() // Random weight between 0 and 1
		offset := weight*parent1.Offsets[i] + (1-weight)*parent2.Offsets[i]
		childOffsets[i] = math.Mod(offset, cycleLengths[i]) // Ensure offset is within cycle length
	}
	return &Individual{Offsets: childOffsets, Fitness: 0.0}
}

// uniformCrossover performs a uniform crossover between two parents
func uniformCrossover(cycleLengths []float64, parent1, parent2 *Individual) *Individual {
	// Create a child by randomly selecting offsets from each parent
	childOffsets := make([]float64, len(cycleLengths))
	childOffsets[0] = 0.0 // The first offset is always 0.0
	for i := 1; i < len(childOffsets); i++ {
		if rand.Float64() < 0.5 {
			childOffsets[i] = parent1.Offsets[i]
		} else {
			childOffsets[i] = parent2.Offsets[i]
		}
		childOffsets[i] = math.Mod(childOffsets[i], cycleLengths[i]) // Ensure offset is within cycle length
	}
	return &Individual{Offsets: childOffsets, Fitness: 0.0}
}

// mutate applies mutation to an individual based on the current generation
func (optga *OptimizerGenetic) mutate(individual *Individual, currentGeneration int) {
	// Calculate the mutation range based on the current generation
	progress := float64(currentGeneration) / float64(optga.generations)
	// Mutation step is range [-5; 5]
	maxDelta := 5*(1-progress) + 0.5*progress // Decrease mutation range over generations
	// Mutate each offset with a probability of mutationRate
	for i := 1; i < len(individual.Offsets); i++ {
		if rand.Float64() < optga.mutationRate {
			delta := randomFloat(-maxDelta, maxDelta)
			individual.Offsets[i] = math.Mod(individual.Offsets[i]+delta, optga.cycleLengths[i])
		}
	}
}

// Optimize runs the genetic algorithm to calculate the optimal offsets for the traffic lights
func (optga *OptimizerGenetic) Optimize() []float64 {
	// Generate the initial population
	population := make([]*Individual, optga.populationSize)
	for i := range population {
		population[i] = optga.createIndividual()
	}

	bestFitness := -1.0
	var bestIndividual *Individual
	generationsWithoutImprovement := 0
	previousBestFitness := -1.0

	for generation := 0; generation < optga.generations; generation++ {
		// Evaluate fitness for each individual in the population
		for _, individual := range population {
			individual.Fitness = optga.evaluateFitness(individual)
			if individual.Fitness > bestFitness || bestIndividual == nil {
				bestFitness = individual.Fitness
				bestIndividual = individual
			}
		}

		// Check for convergence
		if bestFitness > previousBestFitness {
			generationsWithoutImprovement = 0
			previousBestFitness = bestFitness
		} else {
			generationsWithoutImprovement++
			if generationsWithoutImprovement >= CONVERGENCE_GENERATIONS {
				// Early stopping due to convergence
				break
			}
		}

		// Create the next generation
		newPopulation := make([]*Individual, 0, optga.populationSize)
		newPopulation = append(newPopulation, bestIndividual) // Keep the best individual

		for len(newPopulation) < optga.populationSize {
			// Select two parents using tournament selection
			parent1 := optga.selectParent(population)
			parent2 := optga.selectParent(population)
			// Perform crossover to create a child
			child := optga.crossoverFunc(optga.cycleLengths, parent1, parent2)
			// Mutate the child
			optga.mutate(child, generation)
			// Add the child to the new population
			newPopulation = append(newPopulation, child)
		}

		population = newPopulation
		optga.bestFitenessHistory = append(optga.bestFitenessHistory, bestFitness)

	}

	// Safety check: if no best individual found, return zero offsets
	if bestIndividual == nil {
		zeroOffsets := make([]float64, len(optga.junctions))
		return zeroOffsets
	}
	return bestIndividual.Offsets
}

// BestFitnessHistory returns the history of the best fitness values across generations
// Returns slice, do not modify it
func (optga *OptimizerGenetic) BestFitnessHistory() []float64 {
	return optga.bestFitenessHistory
}

package greenwave

import (
	"math"
	"math/rand/v2"
)

// OptimizerGenetic implements a genetic algorithm for optimizing traffic light offsets
type CrossoverType uint8

const (
	// CROSSOVER_BLEND uses a blend crossover method where offspring are created by blending the parents' offsets
	CROSSOVER_BLEND CrossoverType = iota
	// CROSSOVER_UNIFORM uses a uniform crossover method where offspring are created by randomly selecting offsets from each parent
	CROSSOVER_UNIFORM
)

var crossoverTypeToStr = [...]string{"blend", "uniform"}

// String returns the string representation of the CrossoverType
func (ioutIndex CrossoverType) String() string {
	return crossoverTypeToStr[ioutIndex]
}

// Individual represents a single solution in the genetic algorithm
type Individual struct {
	Offsets []float64
	Fitness float64
}

// OptimizerGenetic is a genetic algorithm optimizer for traffic light offsets
type OptimizerGenetic struct {
	// contains the traffic junctions to optimize
	junctions []*Junction
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
	// crossoverFunc is the function used for crossover between two parents
	crossoverFunc func(cycleLengths []float64, parent1, parent2 *Individual) *Individual
	// cycleLengths contains the total duration of each junction in seconds
	cycleLengths []float64
	// bestFitenessHistory keeps track of the best fitness value in each generation
	bestFitenessHistory []float64
}

// NewOptimizerGenetic creates a new instance of OptimizerGenetic with the provided parameters
func NewOptimizerGenetic(junctions []*Junction, speedKhm float64, populationSize int, generations int, mutationRate float64, tournamentSize int, crossoverType CrossoverType) Optimizer {
	cycleLengths := make([]float64, len(junctions))
	for i, junction := range junctions {
		cycleLengths[i] = float64(junction.totalDuration)
	}
	crossoverFunc := blendCrossover
	if crossoverType == CROSSOVER_UNIFORM {
		crossoverFunc = uniformCrossover
	}
	return &OptimizerGenetic{
		junctions:           junctions,
		speedKhm:            speedKhm,
		populationSize:      populationSize,
		generations:         generations,
		mutationRate:        mutationRate,
		tournamentSize:      tournamentSize,
		crossoverType:       crossoverType,
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
	for i, junction := range optga.junctions {
		junction.SetOffset(int(individual.Offsets[i]))
	}
	// Find green waves
	greenWavs := FindGreenWaves(optga.junctions, optga.speedKhm)
	throughGreenWaves := MergeGreenWaves(greenWavs)
	if len(throughGreenWaves) == 0 {
		return 0.0 // No green waves found
	}
	// Calculate total fitness based on the depth and band size of the green waves
	maxDepth := len(optga.junctions)
	totalFitness := 0.0
	for _, wave := range throughGreenWaves {
		depthRatio := float64(wave.Depth()) / float64(maxDepth)
		// Square the depth ratio to emphasize deeper wave
		waveFitness := depthRatio * depthRatio * float64(wave.Bandwidth())
		totalFitness += waveFitness
	}
	return totalFitness
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

	bestFitness := 0.0
	var bestIndividual *Individual

	for generation := 0; generation < optga.generations; generation++ {
		// Evaluate fitness for each individual in the population
		for _, individual := range population {
			individual.Fitness = optga.evaluateFitness(individual)
			if individual.Fitness > bestFitness {
				bestFitness = individual.Fitness
				bestIndividual = individual
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
	return bestIndividual.Offsets
}

// BestFitnessHistory returns the history of the best fitness values across generations
// Returns slice, do not modify it
func (optga *OptimizerGenetic) BestFitnessHistory() []float64 {
	return optga.bestFitenessHistory
}

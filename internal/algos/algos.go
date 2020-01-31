package algos

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zedseven/steg/internal/util"
)

// Algorithm definitions

// Defines a supported algorithm type.
type Algo int

// Simply determines whether a given algorithm is valid.
func (algo Algo) IsValid() bool {
	return algo > AlgoUnknown && algo <= maxAlgoVal
}

// Returns the name of the algorithm, or "<unknown>" if unknown.
func (algo Algo) String() string {
	switch algo {
	case AlgoSequential:
		return "sequential"
	case AlgoPattern:
		return "pattern"
	default:
		return "<unknown>"
	}
}

const (
	AlgoUnknown    Algo = iota     // An unknown algorithm type.
	AlgoSequential Algo = iota     // An algorithm that works sequentially, from 0 to Max.
	AlgoPattern    Algo = iota     // An algorithm that returns unique, random addresses in the range of 0 to Max.
	maxAlgoVal     Algo = iota - 1 // The maximum algorithm value, used for validity checking.
)

// Error types

// Thrown when an unknown algorithm type is provided.
type UnknownAlgoError struct {
	Algorithm Algo
}

func (e UnknownAlgoError) Error() string {
	return fmt.Sprintf("The specified algorithm (%d) does not exist.", e.Algorithm)
}

// Thrown when an algorithm addressor is called but it's pool of available addresses to hand out is empty.
type EmptyPoolError struct {}

func (e EmptyPoolError) Error() string {
	return "The pool of bit addresses is empty."
}

// Algorithm closures

// An algorithm that works sequentially, from 0 to Max.
func SequentialAddressor(channels int64, bitsPerChannel uint8) func() (int64, error) {
	pos := int64(-1)
	posMax := channels * int64(bitsPerChannel)
	return func() (int64, error) {
		pos++
		if pos >= posMax {
			return -1, &EmptyPoolError{}
		}
		return pos, nil
	}
}

// An algorithm that returns unique, random addresses in the range of 0 to Max.
func PatternAddressor(seed, channels int64, bitsPerChannel uint8) func() (int64, error) {
	poolSize := channels * int64(bitsPerChannel)
	pool := util.MakeRange(poolSize)
	rand.Seed(seed)
	//An implementation of the Fisher-Yates shuffling algorithm, slightly re-purposed
	return func() (int64, error) {
		if poolSize <= 0 {
			return -1, &EmptyPoolError{}
		}

		j := rand.Int63n(poolSize) //I'm aware this isn't crypto/rand, but I needed to be able to seed it

		poolSize--

		p := pool[j]

		pool[j] = pool[poolSize]
		pool = pool[:poolSize]

		return p, nil
	}//, &pool
}

// Algorithm type interfacing methods

// Facilitates a running different algorithm addressors at runtime based on a provided algo value.
func AlgoAddressor(algo Algo, seed, channels int64, bitsPerChannel uint8) (func() (int64, error), error) {
	switch algo {
	case AlgoSequential:
		return SequentialAddressor(channels, bitsPerChannel), nil
	case AlgoPattern:
		return PatternAddressor(seed, channels, bitsPerChannel), nil
	default:
		return nil, &UnknownAlgoError{algo}
	}
}

// Simply parses a string into an algorithm type, or AlgoUnknown if the string is not recognized.
func StringToAlgo(str string) Algo {
	switch strings.ToLower(str) {
	case "sequential":
		return AlgoSequential
	case "pattern":
		return AlgoPattern
	default:
		return AlgoUnknown
	}
}
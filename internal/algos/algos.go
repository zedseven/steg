package algos

import (
	"fmt"
	"math/rand"

	"github.com/zedseven/steg/internal/util"
)

// Algorithm definitions

type Algo int

const (
	AlgoUnknown    Algo = iota
	AlgoSequential Algo = iota
	AlgoPattern    Algo = iota
	MaxAlgoVal     Algo = iota - 1
)

// Error types

type UnknownAlgoError struct {
	Algorithm Algo
}

func (e UnknownAlgoError) Error() string {
	return fmt.Sprintf("The specified algorithm (%d) does not exist.", e.Algorithm)
}


type EmptyPoolError struct {}

func (e EmptyPoolError) Error() string {
	return "The pool of bit addresses is empty."
}

// Algorithm closures

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

func StringToAlgo(str string) Algo {
	switch str {
	case "sequential":
		return AlgoSequential
	case "pattern":
		return AlgoPattern
	default:
		return AlgoUnknown
	}
}

func AlgoToString(algo Algo) string {
	switch algo {
	case AlgoSequential:
		return "sequential"
	case AlgoPattern:
		return "pattern"
	default:
		return "<Unknown>"
	}
}
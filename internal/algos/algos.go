package algos

import (
	"math/rand"

	"github.com/zedseven/steg/internal/util"
)

// Error types

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
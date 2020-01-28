package op

import (
	"fmt"
	"math/rand"
)

const (
	bitsPerByte uint8 = 8
	encodeChunkSize uint8 = 32
	encodeHeaderSize uint8 = 32
	encodeHeaderSeparator string = ";"
	versionMax uint8 = 0
	versionMid uint8 = 9
	versionMin uint8 = 0
)

var maxBitsPerChannel uint8 = 1
var encodeAlpha = true
var encodeLsb = true //For debugging

func makeRange(max int64) []int64 {
	r := make([]int64, max)
	for i := range r {
		r[i] = int64(i)
	}
	return r
}

type emptyPoolError struct {}

func (e emptyPoolError) Error() string {
	return "The pool of bit addresses is empty."
}

func patternAddressor(seed, channels int64, bitsPerChannel uint8) func() (int64, error) {
	poolSize := channels * int64(bitsPerChannel)
	pool := makeRange(poolSize)
	rand.Seed(seed)
	fmt.Println("poolSize:", poolSize)
	//An implementation of the Fisher-Yates shuffling algorithm, slightly re-purposed
	return func() (int64, error) {
		if poolSize <= 0 {
			return -1, &emptyPoolError{}
		}

		j := rand.Int63n(poolSize) //I'm aware this isn't crypto/rand, but I needed to be able to seed it

		poolSize--

		p := pool[j]

		pool[j] = pool[poolSize]
		pool = pool[:poolSize]

		return p, nil
	}//, &pool
}

func sequentialAddressor(channels int64, bitsPerChannel uint8) func() (int64, error) {
	pos := int64(-1)
	posMax := channels * int64(bitsPerChannel)
	return func() (int64, error) {
		pos++
		if pos >= posMax {
			return -1, &emptyPoolError{}
		}
		return pos, nil
	}
}
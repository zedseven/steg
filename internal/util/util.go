// Package util provides some basic utility functions.
package util

// MakeRange returns a new array of length max, with contents (0, ..., max - 1)
func MakeRange(max int64) []int64 {
	r := make([]int64, max)
	for i := range r {
		r[i] = int64(i)
	}
	return r
}

// Min returns the smallest of a and b.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the largest of a and b.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Clamp returns val if val is within min and max, min if val < min, or max if val > max.
// This clamps val to the range defined by min and max.
func Clamp(min, max, val int) int {
	return Min(max, Max(min, val))
}
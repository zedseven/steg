package util

func MakeRange(max int64) []int64 {
	r := make([]int64, max)
	for i := range r {
		r[i] = int64(i)
	}
	return r
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Clamp(min, max, val int) int {
	return Min(max, Max(min, val))
}
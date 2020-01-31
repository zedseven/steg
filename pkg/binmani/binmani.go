// Package binmani provides rudimentary binary manipulation functions.
package binmani

// Bit manipulation functions

// GetMask creates a bitmask of size shifted left index bits.
// 	GetMask(2, 3) -> 00011100
func GetMask(index, size uint8) uint16 {
	return ((1 << size) - 1) << index
}

// ReadFrom reads a specified bit or set of consecutive bits from data.
func ReadFrom(data uint16, index, size uint8) uint16 {
	return (data & GetMask(index, size)) >> index
}

// WriteTo writes a value to a specified bit or set of consecutive bits in data, and returns the result.
func WriteTo(data uint16, index, size uint8, value uint16) uint16 {
	return (data & (^GetMask(index, size))) | (value << index)
}
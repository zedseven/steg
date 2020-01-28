package binmani

// Bit manipulation functions

func GetMask(index, size uint8) uint16 {
	return ((1 << size) - 1) << index
}

func ReadFrom(data uint16, index, size uint8) uint16 {
	return (data & GetMask(index, size)) >> index
}

func WriteTo(data uint16, index, size uint8, value uint16) uint16 {
	return (data & (^GetMask(index, size))) | (value << index)
}
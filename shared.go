package steg

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"io"
	"math"
	"os"
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

// Shared types

type pixel []uint16

type fmtInfo struct {
	Model color.Model
	ChannelsPerPix uint8
	BitsPerChannel uint8
}

func (info *fmtInfo) BytesPerChannel() uint8 {
	return uint8(math.Ceil(float64(info.BitsPerChannel / bitsPerByte)))
}

func (info *fmtInfo) String() string {
	return fmt.Sprintf("{%v %d %d}", colourModelToStr(info.Model), info.ChannelsPerPix, info.BitsPerChannel)
}

type imgInfo struct {
	W, H   uint
	Format fmtInfo
}

type unknownColourModelError struct {}

func (e unknownColourModelError) Error() string {
	return "The colour model of the provided Image is unknown."
}

// Shared methods

func hashPatternFile(patternPath string) int64 {
	f, err := os.Open(patternPath)
	if err != nil {
		fmt.Errorf("Unable to open the pattern file '%v': %v\n", patternPath, err.Error())
	}

	h := fnv.New64()

	b := make([]byte, 1024)
	for {
		n, err := f.Read(b)
		if n > 0 {
			h.Write(b[0:n])
		}
		if err != nil {
			if err != io.EOF {
				fmt.Println("An error occurred while reading the file:", err.Error())
			}
			break
		}
	}

	return int64(h.Sum64())
}

//PCB = pixel, Channel, Bit #
func bitAddrToPCB(addr int64, channels, bitsPerChannel uint8) (pix int64, channel, bit uint8) {
	pix = addr / int64(channels * bitsPerChannel)
	channel = uint8(addr / int64(bitsPerChannel)) % channels
	bit = uint8(addr % int64(channels * bitsPerChannel)) % bitsPerChannel
	return
}

func posToXY(pos int64, w int) (x, y int) {
	x = int(pos % int64(w))
	//Would normally floor here, but since all values are >= 0, integer division handles this for us
	y = int(pos / int64(w))
	return
}
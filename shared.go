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
	bitsPerByte           uint8  = 8
	encodeChunkSize       uint8  = 32
	encodeHeaderSize      uint8  = 32
	encodeHeaderSeparator string = ";"
	VersionMax            uint8  = 0
	VersionMid            uint8  = 9
	VersionMin            uint8  = 0
)

// Shared types

// The levels of output supported by the package.
type OutputLevel int

const (
	OutputNothing OutputLevel = iota // Output nothing at all to stdout.
	OutputSteps   OutputLevel = iota // Output operation progress at each significant step of the process.
	OutputInfo    OutputLevel = iota // Output operation progress at each significant step of the process, and include additional information.
	OutputDebug   OutputLevel = iota // Output formatted debug information.
)


type pixel []uint16


type fmtInfo struct {
	Model          color.Model
	ChannelsPerPix uint8
	BitsPerChannel uint8
}

func (info *fmtInfo) bytesPerChannel() uint8 {
	return uint8(math.Ceil(float64(info.BitsPerChannel / bitsPerByte)))
}

func (info *fmtInfo) alphaChannel() int8 {
	switch info.Model {
	case color.RGBAModel, color.RGBA64Model, color.NRGBAModel, color.NRGBA64Model:
		return 3
	case color.AlphaModel, color.Alpha16Model:
		return 0
	default:
		return -1
	}
}

func (info *fmtInfo) supportsAlpha() bool {
	return info.alphaChannel() >= 0
}

func (info *fmtInfo) String() string {
	return fmt.Sprintf("{%v %d %d}", colourModelToStr(info.Model), info.ChannelsPerPix, info.BitsPerChannel)
}


type imgInfo struct {
	W, H   uint
	Format fmtInfo
}

// Error types

type unknownColourModelError struct {}

func (e unknownColourModelError) Error() string {
	return "The colour model of the provided Image is unknown."
}

// Thrown when provided data is of an invalid format.
type InvalidFormatError struct {
	ErrorDesc string // A description of the problem. If empty, a default message is used.
}

func (e InvalidFormatError) Error() string {
	if len(e.ErrorDesc) > 0 {
		return e.ErrorDesc
	}
	return "The provided data is of an invalid format."
}

// Thrown when the provided image does not have enough room to hide the provided file using the provided configuration.
type InsufficientHidingSpotsError struct {
	AdditionalInfo string // Additional information about the problem.
	InnerError     error  // An inner error involved in the issue to provide more information.
}

func (e *InsufficientHidingSpotsError) Error() string {
	ret := "There is not enough space available to store the provided file within the provided image."
	if len(e.AdditionalInfo) > 0 && e.InnerError != nil {
		return fmt.Sprintf("%v Additional info: %v Inner error: %v", ret, e.AdditionalInfo, e.InnerError.Error())
	} else if len(e.AdditionalInfo) > 0 {
		return fmt.Sprintf("%v Additional info: %v", ret, e.AdditionalInfo)
	} else if e.InnerError != nil {
		return fmt.Sprintf("%v Inner error: %v", ret, e.InnerError.Error())
	}
	return ret
}

// Library methods

// Returns the library version in a pretty string format.
// Format: Max.Mid.Min
func Version() string {
	return fmt.Sprintf("%02d.%02d.%02d", VersionMax, VersionMid, VersionMin)
}

// Shared methods

func hashPatternFile(patternPath string) (int64, error) {
	f, err := os.Open(patternPath)
	if err != nil {
		return -1, err
	}

	h := fnv.New64()

	b := make([]byte, 1024)
	for {
		n, err := f.Read(b)
		if n > 0 {
			if _, werr := h.Write(b[0:n]); werr != nil {
				return -1, werr
			}
		}
		if err != nil {
			if err != io.EOF {
				return -1, err
			}
			break
		}
	}

	return int64(h.Sum64()), nil
}

// PCB = Pixel, Channel, Bit
func bitAddrToPCB(addr int64, channels, bitsPerChannel uint8) (pix int64, channel, bit uint8) {
	// Would normally floor here, but since all values are >= 0, integer division handles this for us
	pix = addr / int64(channels * bitsPerChannel)
	channel = uint8((addr / int64(bitsPerChannel)) % int64(channels))
	bit = uint8(addr % int64(bitsPerChannel))
	return
}

func posToXY(pos int64, w int) (x, y int) {
	x = int(pos % int64(w))
	y = int(pos / int64(w))
	return
}

func printlnLvl(level, minLevel OutputLevel, val ...interface{}) {
	if level >= minLevel {
		fmt.Println(val...)
	}
}
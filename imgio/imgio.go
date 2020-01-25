package imgio

import (
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
)

type Pixel struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

func LoadImage(imgPath string) (pixels [][]Pixel, e error) {
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)

	imgFile, err := os.Open(imgPath)

	if err != nil {
		fmt.Println("Unable to open the image!", err.Error())
		return
	}

	defer imgFile.Close()

	pixels, err = readPixels(imgFile)
	//fmt.Println(pixels)

	if err != nil {
		fmt.Println("The image couldn't be decoded.", err.Error())
		return
	}

	return
}

func readPixels(imgFile io.Reader) (pixels [][]Pixel, e error) {
	img, _, err := image.Decode(imgFile)

	if err != nil {
		return nil, err
	}

	dims := img.Bounds()
	w, h := dims.Max.X, dims.Max.Y
	//TODO: Potentially check for 0x0 dims

	for y := 0; y < h; y++ {
		row := make([]Pixel, 0, w)
		for x := 0; x < w; x++ {
			row = append(row, rgbaToPixel(img.At(x, y).RGBA()))
		}
		pixels = append(pixels, row)
	}

	return
}

func WriteImage(pixels *[][]Pixel, outPath string) {
	w, h := len((*pixels)[0]), len(*pixels)

	img := image.NewRGBA(image.Rectangle{Max: image.Point{w, h}})
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, pixelToRgba((*pixels)[y][x]))
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("There was an error creating the file '%v': %v\n", outPath, err.Error())
	}

	err = png.Encode(f, img)
	if err != nil {
		fmt.Printf("There was an error encoding the image to the new file: %v\n", err.Error())
	}
}

func rgbaToPixel(r, g, b, a uint32) Pixel {
	return Pixel{uint8(r), uint8(g), uint8(b), uint8(a)}
}

func pixelToRgba(pix Pixel) color.Color {
	return color.RGBA{pix.R, pix.G, pix.B, pix.A}
}

//PCB = Pixel, Channel, Bit #
func BitAddrToPCB(addr int64, channels, bitsPerChannel uint8) (pix int64, channel, bit uint8) {
	/*pix = addr / int64(channels)
	channel = uint8(addr % int64(channels))*/
	pix = addr / int64(channels * bitsPerChannel)
	channel = uint8(addr / int64(bitsPerChannel)) % channels
	bit = uint8(addr % int64(channels * bitsPerChannel)) % bitsPerChannel
	return
}

func PosToXY(pos int64, w int) (x, y int) {
	x = int(pos % int64(w))
	//Would normally floor here, but since all values are >= 0, integer division handles this for us
	y = int(pos / int64(w))
	return
}

func HashPatternFile(patternPath string) int64 {
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

//Bit manipulation functions

func GetMask(index, size uint8) uint8 {
	return ((1 << size) - 1) << index
}

func ReadFrom(data uint8, index, size uint8) uint8 {
	return (data & GetMask(index, size)) >> index
}

func WriteTo(data uint8, index, size uint8, value uint8) uint8 {
	return (data & (^GetMask(index, size))) | (value << index)
}
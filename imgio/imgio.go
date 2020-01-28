package imgio

import (
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
)

//TODO: Potentially change this to type Pixel []uint16
type Pixel struct {
	Channels []uint16
}

type FmtInfo struct {
	Model color.Model
	ChannelsPerPix uint8
	BitsPerChannel uint8
}

func (info *FmtInfo) BytesPerChannel() uint8 {
	return uint8(math.Ceil(float64(info.BitsPerChannel / bitsPerByte)))
}

func (info *FmtInfo) String() string {
	return fmt.Sprintf("{%v %d %d}", colorModelToStr(info.Model), info.ChannelsPerPix, info.BitsPerChannel)
}

type ImgInfo struct {
	W, H   uint
	Format FmtInfo
}

const (
	bitsPerByte uint8 = 8
	pngHeader string = "\x89PNG\r\n\x1a\n"
)

type unknownColourModelError struct {}

func (e unknownColourModelError) Error() string {
	return "The colour model of the provided Image is unknown."
}

func LoadImage(imgPath string) (pixels *[]Pixel, info ImgInfo, e error) {
	image.RegisterFormat("png", pngHeader, png.Decode, png.DecodeConfig)

	imgFile, err := os.Open(imgPath)

	if err != nil {
		fmt.Println("Unable to open the image!", err.Error())
		return
	}

	defer func() {
		if err = imgFile.Close(); err != nil {
			fmt.Println("Error closing the file:", err.Error())
		}
	}()

	pixels, info, err = readPixels(imgFile)
	//fmt.Println(pixels)

	if err != nil {
		fmt.Println("The image couldn't be decoded:", err.Error())
		return
	}

	return
}

func readPixels(imgFile io.Reader) (pixels *[]Pixel, info ImgInfo, e error) {
	img, _, err := image.Decode(imgFile)
	//img, err := png.Decode(imgFile)

	if err != nil {
		return nil, ImgInfo{}, err
	}

	dims := img.Bounds()
	w, h := dims.Max.X, dims.Max.Y

	info = ImgInfo{W: uint(w), H:uint(h)}

	fmt.Println("Colour model:", colorModelToStr(img.ColorModel()))
	//Handle each image type independently, parsing out the pixel channel values
	switch img.(type) {
	case *image.Alpha16:
		info.Format = FmtInfo{color.Alpha16Model, 4, 16}
		simg := img.(*image.Alpha16)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.Alpha:
		info.Format = FmtInfo{color.AlphaModel, 4, 8}
		simg := img.(*image.Alpha)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.CMYK:
		info.Format = FmtInfo{color.CMYKModel, 4, 8}
		simg := img.(*image.CMYK)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.Gray16:
		info.Format = FmtInfo{color.Gray16Model, 4, 16}
		simg := img.(*image.Gray16)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.Gray:
		info.Format = FmtInfo{color.GrayModel, 4, 8}
		simg := img.(*image.Gray)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.NRGBA64:
		info.Format = FmtInfo{color.NRGBA64Model, 4, 16}
		simg := img.(*image.NRGBA64)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.NRGBA:
		info.Format = FmtInfo{color.NRGBAModel, 4, 8}
		simg := img.(*image.NRGBA)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.RGBA64:
		info.Format = FmtInfo{color.RGBA64Model, 4, 16}
		simg := img.(*image.RGBA64)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.RGBA:
		info.Format = FmtInfo{color.RGBAModel, 4, 8}
		simg := img.(*image.RGBA)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	default:
		return nil, info, unknownColourModelError{}
	}
	return
}

func imgPixToPixels(pix *[]uint8, info FmtInfo) *[]Pixel {
	bytesPerChannel := info.BitsPerChannel / bitsPerByte
	pixels := make([]Pixel, len(*pix) / int(info.ChannelsPerPix * bytesPerChannel))
	for i := range pixels {
		pixels[i] = Pixel{Channels:make([]uint16, info.ChannelsPerPix)}
		for j := uint8(0); j < info.ChannelsPerPix; j++ {
			//Image raw Pix arrays store multi-byte channel values in big-endian format
			//across separate indices (https://golang.org/src/image/image.go?s=8222:8528#L380)
			for k := uint8(0); k < info.BytesPerChannel(); k++ {
				pixels[i].Channels[j] <<= bitsPerByte
				pixels[i].Channels[j] += uint16((*pix)[i * int(info.ChannelsPerPix * bytesPerChannel) + int(j * bytesPerChannel)])
			}
		}
	}
	return &pixels
}

func updatePixWithPixels(pix *[]uint8, pixels *[]Pixel, info FmtInfo) {
	bytes := info.BytesPerChannel()
	for i := range *pixels {
		for j := range (*pixels)[i].Channels {
			for k := uint8(0); k < bytes; k++ {
				(*pix)[(i * int(info.ChannelsPerPix) + j) * int(bytes) + int(k)] =
					uint8(ReadFrom((*pixels)[i].Channels[j], (bytes - 1 - k) * bitsPerByte, bitsPerByte))
			}
		}
	}
}

func WriteImage(pixels *[]Pixel, info ImgInfo, outPath string) {
	var img image.Image
	switch info.Format.Model {
	case color.Alpha16Model:
		simg := image.NewAlpha16(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	case color.AlphaModel:
		simg := image.NewAlpha(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	case color.CMYKModel:
		simg := image.NewCMYK(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	case color.Gray16Model:
		simg := image.NewGray16(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	case color.GrayModel:
		simg := image.NewGray(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	case color.NRGBA64Model:
		simg := image.NewNRGBA64(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	case color.NRGBAModel:
		simg := image.NewNRGBA(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	case color.RGBA64Model:
		simg := image.NewRGBA64(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	case color.RGBAModel:
		simg := image.NewRGBA(image.Rectangle{Max: image.Point{int(info.W), int(info.H)}})
		updatePixWithPixels(&simg.Pix, pixels, info.Format)
		img = simg
	default:
		fmt.Println("Unknown image format.")
		return
	}

	f, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("There was an error creating the file '%v': %v\n", outPath, err.Error())
	}
	defer func() {
		if err = f.Close(); err != nil {
			fmt.Println("Error closing the file:", err.Error())
		}
	}()

	//TODO: Add support for additional format exports
	encoder := png.Encoder{CompressionLevel:png.BestCompression}
	err = encoder.Encode(f, img)
	if err != nil {
		fmt.Printf("There was an error encoding the image to the new file: %v\n", err.Error())
	}
}

/*func pixelToRgba(pix Pixel) color.Color {
	//fmt.Println(pix)
	return color.NRGBA{pix.R, pix.G, pix.B, pix.A}
}*/

func colorModelToStr(model color.Model) string {
	switch model {
	case color.Alpha16Model:
		return "Alpha16"
	case color.AlphaModel:
		return "Alpha"
	case color.CMYKModel:
		return "CMYK"
	case color.Gray16Model:
		return "Gray16"
	case color.GrayModel:
		return "Gray"
	case color.NRGBA64Model:
		return "NRGBA64"
	case color.NRGBAModel:
		return "NRGBA"
	case color.RGBA64Model:
		return "RGBA64"
	case color.RGBAModel:
		return "RGBA"
	case color.NYCbCrAModel:
		return "NYCbCrA"
	case color.YCbCrModel:
		return "YCbCr"
	default:
		return "<Unknown>"
	}
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

func GetMask(index, size uint8) uint16 {
	return ((1 << size) - 1) << index
}

func ReadFrom(data uint16, index, size uint8) uint16 {
	return (data & GetMask(index, size)) >> index
}

func WriteTo(data uint16, index, size uint8, value uint16) uint16 {
	return (data & (^GetMask(index, size))) | (value << index)
}
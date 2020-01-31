package steg

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"

	"github.com/zedseven/steg/pkg/binmani"
)

// Primary methods

func loadImage(imgPath string, outputLevel OutputLevel) (pixels *[]pixel, info imgInfo, err error) {
	imgFile, err := os.Open(imgPath)
	if err != nil {
		printlnLvl(outputLevel, OutputSteps, "Unable to open the image!", err.Error())
		return nil, imgInfo{}, err
	}

	defer func() {
		if err = imgFile.Close(); err != nil {
			printlnLvl(outputLevel, OutputSteps, fmt.Sprintf("Error closing the file '%v': %v", imgPath, err.Error()))
		}
	}()

	pixels, info, err = readPixels(imgFile)

	if err != nil {
		printlnLvl(outputLevel, OutputSteps, "The image couldn't be decoded:", err.Error())
		return nil, imgInfo{}, err
	}

	return
}

func writeImage(pixels *[]pixel, info imgInfo, outPath string, outputLevel OutputLevel) error {
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
		printlnLvl(outputLevel, OutputSteps, "Unknown image format.")
		return unknownColourModelError{}
	}

	f, err := os.Create(outPath)
	if err != nil {
		printlnLvl(outputLevel, OutputSteps, "There was an error creating the file '%v'.\n", outPath)
		return err
	}

	defer func() {
		if err = f.Close(); err != nil {
			printlnLvl(outputLevel, OutputSteps, "Error closing the file '%v': %v", outPath, err.Error())
		}
	}()

	// TODO: Add support for additional format exports
	encoder := png.Encoder{CompressionLevel:png.BestCompression}
	err = encoder.Encode(f, img)
	if err != nil {
		printlnLvl(outputLevel, OutputSteps, "There was an error encoding the image to the new file.")
		return err
	}

	return nil
}

// Helper functions

func readPixels(imgFile io.Reader) (pixels *[]pixel, info imgInfo, err error) {
	img, _, err := image.Decode(imgFile)

	if err != nil {
		return nil, imgInfo{}, err
	}

	dims := img.Bounds()
	w, h := dims.Max.X, dims.Max.Y

	info = imgInfo{W: uint(w), H:uint(h)}

	// Each colour model has to be handled individually
	switch img.(type) {
	case *image.Alpha16:
		info.Format = fmtInfo{color.Alpha16Model, 4, 16}
		simg := img.(*image.Alpha16)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.Alpha:
		info.Format = fmtInfo{color.AlphaModel, 4, 8}
		simg := img.(*image.Alpha)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.CMYK:
		info.Format = fmtInfo{color.CMYKModel, 4, 8}
		simg := img.(*image.CMYK)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.Gray16:
		info.Format = fmtInfo{color.Gray16Model, 4, 16}
		simg := img.(*image.Gray16)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.Gray:
		info.Format = fmtInfo{color.GrayModel, 4, 8}
		simg := img.(*image.Gray)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.NRGBA64:
		info.Format = fmtInfo{color.NRGBA64Model, 4, 16}
		simg := img.(*image.NRGBA64)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.NRGBA:
		info.Format = fmtInfo{color.NRGBAModel, 4, 8}
		simg := img.(*image.NRGBA)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.RGBA64:
		info.Format = fmtInfo{color.RGBA64Model, 4, 16}
		simg := img.(*image.RGBA64)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	case *image.RGBA:
		info.Format = fmtInfo{color.RGBAModel, 4, 8}
		simg := img.(*image.RGBA)
		pixels = imgPixToPixels(&simg.Pix, info.Format)
	default:
		return nil, info, unknownColourModelError{}
	}
	return
}

func imgPixToPixels(pix *[]uint8, info fmtInfo) *[]pixel {
	bytesPerChannel := info.BitsPerChannel / bitsPerByte
	pixels := make([]pixel, len(*pix) / int(info.ChannelsPerPix * bytesPerChannel))
	for i := range pixels {
		pixels[i] = make([]uint16, info.ChannelsPerPix)
		for j := uint8(0); j < info.ChannelsPerPix; j++ {
			// Image raw Pix arrays store multi-byte channel values in big-endian format
			// across separate indices (https://golang.org/src/image/image.go?s=8222:8528#L380)
			for k := uint8(0); k < info.bytesPerChannel(); k++ {
				pixels[i][j] <<= bitsPerByte
				pixels[i][j] += uint16((*pix)[i * int(info.ChannelsPerPix * bytesPerChannel) + int(j * bytesPerChannel)])
			}
		}
	}
	return &pixels
}

func updatePixWithPixels(pix *[]uint8, pixels *[]pixel, info fmtInfo) {
	bytes := info.bytesPerChannel()
	for i := range *pixels {
		for j := range (*pixels)[i] {
			for k := uint8(0); k < bytes; k++ {
				(*pix)[(i * int(info.ChannelsPerPix) + j) * int(bytes) + int(k)] =
					uint8(binmani.ReadFrom((*pixels)[i][j], (bytes - 1 - k) * bitsPerByte, bitsPerByte))
			}
		}
	}
}

func colourModelToStr(model color.Model) string {
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
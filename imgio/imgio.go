package imgio

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
)

type Pixel struct {
	R uint32
	G uint32
	B uint32
	A uint32
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

	for y := 0; y < h; y++ {
		row := make([]Pixel, 0, w)
		for x := 0; x < w; x++ {
			row = append(row, rgbaToPixel(img.At(x, y).RGBA()))
		}
		pixels = append(pixels, row)
	}

	return
}

func rgbaToPixel(r, g, b, a uint32) Pixel {
	return Pixel{r, g, b, a}
}
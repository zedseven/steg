package main

import (
	"flag"
	//"fmt"
	"github.com/zedseven/steg/op"
	//"os"
)

func main() {
	//hideCmd := flag.NewFlagSet("hide", flag.ExitOnError)
	//digCmd := flag.NewFlagset("dig", flag.ExitOnError)
	digToggle := flag.Bool("dig", false, "Whether to extract a file instead of hiding it")
	imgPath := flag.String("img", "", "The filepath to the image on disk")
	filePath := flag.String("file", "", "The filepath to the file on disk")
	outPath := flag.String("out", "", "The filepath to write the steg image to")
	patternPath := flag.String("pattern", "", "The filepath to the file used for the pattern hash")
	bits := flag.Uint("bits", 1, "The number of bits to modify per channel (1-8), at a maximum (working inwards as determined by -msb)")
	msb := flag.Bool("msb", false, "Whether to modify the most-significant bits instead - mostly for debugging")
	encodeAlpha := flag.Bool("alpha", false, "Whether to touch the alpha (transparency) channel")

	/*if len(os.Args) < 2 {
		fmt.Println("You have to specify what you want me to do!")
		return
	}*/
	flag.Parse()

	if len(*imgPath) <= 0 || len(*filePath) <= 0 || len(*outPath) <= 0 || *bits <= 0 || *bits > 8 {
		flag.PrintDefaults()
		return
	}
	if !*digToggle {
		op.Hide(*imgPath, *filePath, *outPath, *patternPath, uint8(*bits), *encodeAlpha, !*msb)
	} else {
		op.Dig(*imgPath, *outPath, *patternPath, uint8(*bits), *encodeAlpha, !*msb)
	}
}


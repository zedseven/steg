package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zedseven/steg"
	"github.com/zedseven/steg/internal/algos"
)

// Program entry point

func main() {
	if len(os.Args) < 2 {
		fmt.Println("You have to specify what you want me to do! The two subcommands are hide and dig.")
		return
	}

	var flagSet *flag.FlagSet

	// Flags unique to a command
	var filePath *string

	switch os.Args[1] {
	case "hide":
		flagSet = flag.NewFlagSet("hide", flag.ExitOnError)
		filePath = flagSet.String("file", "", "The filepath to the file on disk")
	case "dig":
		flagSet = flag.NewFlagSet("dig", flag.ExitOnError)
	default:
		fmt.Println("You have to specify what you want me to do! The two subcommands are hide and dig.")
		return
	}

	// Common flags
	imgPath := flagSet.String("img", "", "The filepath to the image on disk")
	outPath := flagSet.String("out", "", "The filepath to write the steg image to")
	algoType := flagSet.String("algo", "pattern", "The type of algorithm to use for hiding or digging")
	patternPath := flagSet.String("pattern", "", "The filepath to the file used for the pattern hash if an algorithm is chosen that requires one")
	bits := flagSet.Uint("bits", 1, "The number of bits to modify per channel (1-16), at a maximum (working inwards as determined by -msb)")
	msb := flagSet.Bool("msb", false, "Whether to modify the most-significant bits instead - mostly for debugging")
	encodeAlpha := flagSet.Bool("alpha", false, "Whether to touch the alpha (transparency) channel")
	outputLevel := flagSet.String("level", "info", "The output level or verbosity to use")

	if err := flagSet.Parse(os.Args[2:]); err != nil {
		fmt.Println("There was an issue parsing the flags!", err.Error())
		flagSet.PrintDefaults()
	}

	// Parse out which algorithm to use
	var algo algos.Algo
	algoTmp, err := strconv.ParseInt(*algoType, 10, 8)
	if err != nil || !algos.Algo(algoTmp).IsValid() {
		algo = algos.StringToAlgo(*algoType)
		if algo == algos.AlgoUnknown {
			flagSet.PrintDefaults()
			return
		}
	} else {
		algo = algos.Algo(algoTmp)
	}

	// Parse out which output level to use
	var level steg.OutputLevel
	levelTmp, err := strconv.ParseInt(*algoType, 10, 8)
	if err != nil {
		switch strings.ToLower(*outputLevel) {
		case "nothing":
			level = steg.OutputNothing
		case "steps":
			level = steg.OutputSteps
		case "info":
			level = steg.OutputInfo
		case "debug":
			level = steg.OutputDebug
		default:
			flagSet.PrintDefaults()
			return
		}
	} else {
		level = steg.OutputLevel(levelTmp)
	}

	// Run the appropriate command
	switch os.Args[1] {
	case "hide":
		config := steg.HideConfig{
			ImagePath:         *imgPath,
			FilePath:          *filePath,
			OutPath:           *outPath,
			PatternPath:       *patternPath,
			Algorithm:         algo,
			MaxBitsPerChannel: uint8(*bits),
			EncodeAlpha:       *encodeAlpha,
			EncodeMsb:         *msb,
			OutputLevel:       level,
		}
		if err := steg.Hide(config); err != nil {
			fmt.Println(err.Error())
			switch err.(type) {
			case *steg.InvalidFormatError:
				flagSet.PrintDefaults()
				return
			}
			return
		}
	case "dig":
		config := steg.DigConfig{
			ImagePath:         *imgPath,
			OutPath:           *outPath,
			PatternPath:       *patternPath,
			Algorithm:         algo,
			MaxBitsPerChannel: uint8(*bits),
			DecodeAlpha:       *encodeAlpha,
			DecodeMsb:         *msb,
			OutputLevel:       level,
		}
		if err := steg.Dig(config); err != nil {
			fmt.Println(err.Error())
			switch err.(type) {
			case *steg.InvalidFormatError:
				flagSet.PrintDefaults()
				return
			}
			return
		}
	default:
		fmt.Println("You have to specify what you want me to do! Either hide or dig.")
		return
	}
}


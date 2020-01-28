package main

import (
	"flag"
	"fmt"
	"github.com/zedseven/steg/internal/algos"
	"os"
	"strconv"

	"github.com/zedseven/steg"
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

	flagSet.Parse(os.Args[2:])

	// Parse out which algorithm to use
	var algo algos.Algo
	algoTmp, err := strconv.ParseInt(*algoType, 10, 8)
	if err != nil || algoTmp <= 0 || algoTmp > int64(algos.MaxAlgoVal) {
		algo = algos.StringToAlgo(*algoType)
		if algo == algos.AlgoUnknown {
			flagSet.PrintDefaults()
			return
		}
	} else {
		algo = algos.Algo(algoTmp)
	}

	// Run the appropriate command
	switch os.Args[1] {
	case "hide":
		//TODO: Make some of the below values constants
		if len(*imgPath) <= 0 || len(*filePath) <= 0 || len(*outPath) <= 0 || *bits <= 0 || *bits > 16 {
			flagSet.PrintDefaults()
			return
		}

		if err := steg.Hide(*imgPath, *filePath, *outPath, *patternPath, algo, uint8(*bits), *encodeAlpha, !*msb); err != nil {
			fmt.Println(err.Error())
			return
		}
	case "dig":
		if len(*imgPath) <= 0 || len(*outPath) <= 0 || *bits <= 0 || *bits > 16 {
			flagSet.PrintDefaults()
			return
		}

		if err := steg.Dig(*imgPath, *outPath, *patternPath, algo, uint8(*bits), *encodeAlpha, !*msb); err != nil {
			fmt.Println(err.Error())
			return
		}
	default:
		fmt.Println("You have to specify what you want me to do! Either hide or dig.")
		return
	}
}


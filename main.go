package main

import (
	"flag"
	"fmt"
	"github.com/zedseven/steg/hide"
	"os"
)

func main() {
	hideCmd := flag.NewFlagSet("hide", flag.ExitOnError)
	//digCmd := flag.NewFlagset("dig", flag.ExitOnError)
	imgPath := hideCmd.String("img", "", "The filepath to the image on disk")
	filePath := hideCmd.String("file", "", "The filepath to the file on disk")

	if len(os.Args) < 2 {
		fmt.Println("You have to specify what you want me to do!")
		return
	}

	switch os.Args[1] {
	case "hide":
		hideCmd.Parse(os.Args[2:])
		if len(*imgPath) <= 0 || len(*filePath) <= 0 {
			hideCmd.PrintDefaults()
			return
		}
		hide.Hide(*imgPath, *filePath)
	default:
		fmt.Println("Expected 'hide' or 'dig'")
		return
	}
	
	fmt.Println("Hello world! I'm steg. c:")
}


package steg

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/zedseven/steg/internal/algos"
	"github.com/zedseven/steg/internal/util"
	"github.com/zedseven/steg/pkg/binmani"
)

// Primary method

func Hide(imgPath, filePath, outPath, patternPath string, maxBitsPerChannel uint8, encodeAlpha, encodeLsb bool) error {
	if maxBitsPerChannel < 0 || maxBitsPerChannel > 16 {
		// TODO: Perhaps panic here instead
		return &InvalidFormatError{fmt.Sprintf("maxBitsPerChannel is outside the allowed range of 0-16: Provided %d", maxBitsPerChannel)}
	}

	if debugOutput {
		fmt.Println("This tool has been compiled and set to display debug output.")
	}

	fmt.Printf("Loading the image from '%v'...\n", imgPath)
	pixels, info, err := loadImage(imgPath)
	if err != nil {
		fmt.Printf("Unable to load the image at '%v'!\n", imgPath)
		return err
	}

	maxBitsPerChannel = uint8(util.Min(int(maxBitsPerChannel), int(info.Format.BitsPerChannel)))

	fmt.Printf("Image info:\n\tDimensions: %dx%d\n\tColour model: %v\n\tChannels per pixel: %d\n\tBits per channel: %d\n",
		info.W, info.H, colourModelToStr(info.Format.Model), info.Format.ChannelsPerPix, info.Format.BitsPerChannel)


	fmt.Printf("Opening the file at '%v'...\n", filePath)
	fileReader, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Unable to open the file at '%v'.\n", filePath)
		return err
	}

	defer func() {
		if err = fileReader.Close(); err != nil {
			fmt.Printf("Error closing the file '%v': %v", filePath, err.Error())
		}
	}()


	fmt.Println("Loading up the pattern key...")
	pHash, err := hashPatternFile(patternPath)
	if err != nil {
		fmt.Printf("Something went wrong while attempting to hash the pattern file '%v'.\n", patternPath)
		return err
	}
	fmt.Println("Pattern hash:", pHash)



	fmt.Println("Encoding the file into the image...")

	r := bufio.NewReader(fileReader)
	b := make([]byte, encodeChunkSize)

	channelsPerPix := info.Format.ChannelsPerPix
	if info.Format.SupportsAlpha() && !encodeAlpha {
		channelsPerPix--
	}
	if channelsPerPix <= 0 { // In the case of Alpha & Alpha16 models
		return &InsufficientHidingSpotsError{AdditionalInfo:fmt.Sprintf("The provided image is of the %v colour" +
			"model, but since alpha-channel encoding was not specified, there are no channels to hide data within.\n",
			colourModelToStr(info.Format.Model))}
	}

	channelCount := int64(len(*pixels)) * int64(channelsPerPix)
	fmt.Println("Maximum writable bits:", channelCount * int64(maxBitsPerChannel))
	// TODO: Add in proper CLI support for switching algos, and add more to choose from
	//f := algos.SequentialAddressor(channelCount, maxBitsPerChannel)
	f := algos.PatternAddressor(pHash, channelCount, maxBitsPerChannel)


	fmt.Println("Writing the steg header...")

	fileInfo, err := fileReader.Stat()
	if err != nil {
		fmt.Println("Unable to retrieve file info!")
		return err
	}

	b = []byte(fmt.Sprintf("steg%02d.%02d.%02d%v%019d", VersionMax, VersionMid, VersionMin, encodeHeaderSeparator, fileInfo.Size()))

	if debugOutput {
		fmt.Println("Encoding header:", string(b[0:]))
	}

	if err = encodeChunk(&f, info, pixels, channelsPerPix, maxBitsPerChannel, &b, int(encodeHeaderSize), encodeLsb); err != nil {
		switch err.(type) {
		case *algos.EmptyPoolError:
			return &InsufficientHidingSpotsError{InnerError:err}
		default:
			return err
		}
	}


	fmt.Println("Writing file data...")

	if debugOutput {
		for _, v := range b {
			fmt.Printf("%#08b\n", v)
		}
	}

	for {
		n, err := r.Read(b)
		if n > 0 {
			if err = encodeChunk(&f, info, pixels, channelsPerPix, maxBitsPerChannel, &b, n, encodeLsb); err != nil {
				switch err.(type) {
				case *algos.EmptyPoolError:
					return &InsufficientHidingSpotsError{InnerError:err}
				default:
					return err
				}
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Printf("An error occurred while reading the file '%v'", filePath)
				return err
			}
			break
		}
	}


	fmt.Printf("Writing the encoded image to '%v' now...\n", outPath)
	if err = writeImage(pixels, info, outPath); err != nil {
		fmt.Println("An error occurred while writing to the final image.")
		return err
	}


	fmt.Println("All done! c:")

	return nil
}

// Helper functions

func encodeChunk(pos *func() (int64, error), info imgInfo, pixels *[]pixel, channelCount, maxBitsPerChannel uint8, buf *[]byte, n int, lsb bool) error {
	for i := 0; i < n; i++ {
		for j := uint8(0); j < bitsPerByte; j++ {
			writeBit := binmani.ReadFrom(uint16((*buf)[i]), bitsPerByte - j - 1, 1)

			for {
				addr, err := (*pos)()
				if err != nil {
					return err
				}
				p, c, b := bitAddrToPCB(addr, channelCount, maxBitsPerChannel)

				if debugOutput {
					fmt.Printf("addr: %d, pixel: %d, channel: %d, bit: %d, RGBA: %v\n", addr, p, c, b, (*pixels)[p])
				}

				// TODO: Harden this for alpha models
				if (*pixels)[p][3] <= 0 {
					continue
				}

				if debugOutput {
					fmt.Printf("	Writing %d...\n", writeBit)
					fmt.Printf("	Channel before: %#016b - %v\n", (*pixels)[p][c], (*pixels)[p])
				}

				bitPos := b
				if !lsb {
					bitPos = info.Format.BitsPerChannel - b - 1
				}
				(*pixels)[p][c] = binmani.WriteTo((*pixels)[p][c], bitPos, 1, writeBit)

				if debugOutput {
					fmt.Printf("	Channel after:  %#016b - %v\n", (*pixels)[p][c], (*pixels)[p])
				}

				break
			}
		}
	}

	return nil
}
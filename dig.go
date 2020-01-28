package steg

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zedseven/steg/internal/algos"
	"github.com/zedseven/steg/internal/util"
	"github.com/zedseven/steg/pkg/binmani"
)

// Types

type BadHeaderError struct {}

func (e *BadHeaderError) Error() string {
	return "The read header is not valid!"
}

// Primary method

func Dig(imgPath, outPath, patternPath string, algo algos.Algo, maxBitsPerChannel uint8, encodeAlpha, encodeLsb bool) error {
	if maxBitsPerChannel < 0 || maxBitsPerChannel > 16 {
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


	fmt.Println("Loading up the pattern key...")
	pHash, err := hashPatternFile(patternPath)
	if err != nil {
		fmt.Printf("Something went wrong while attempting to hash the pattern file '%v'.\n", patternPath)
		return err
	}
	fmt.Println("Pattern hash:", pHash)



	fmt.Println("Reading the file from the image...")

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
	fmt.Println("Maximum readable bits:", channelCount * int64(maxBitsPerChannel))

	f, err := algos.AlgoAddressor(algo, pHash, channelCount, maxBitsPerChannel)
	if err != nil {
		return err
	}


	fmt.Println("Reading steg header...")

	b, header := make([]byte, encodeChunkSize), make([]byte, encodeHeaderSize)
	if err = decodeChunk(&f, info, pixels, channelsPerPix, maxBitsPerChannel, &header, int(encodeHeaderSize), encodeLsb); err != nil {
		switch err.(type) {
		case *algos.EmptyPoolError:
			return &InsufficientHidingSpotsError{InnerError:err}
		default:
			return err
		}
	}

	headerStr := string(header[0:])

	if debugOutput {
		fmt.Println("Encoding header:", headerStr)
		for _, v := range b {
			fmt.Printf("%#08b\n", v)
		}
	}

	headerParts := strings.Split(headerStr, encodeHeaderSeparator)
	if len(headerParts) < 2 {
		return &BadHeaderError{}
	}

	if debugOutput {
		fmt.Println("Header parts:", headerParts)
	}

	fileSize, err := strconv.ParseInt(headerParts[1], 10, 64)
	if err != nil {
		fmt.Println("The read file size is not valid!")
		return err
	}

	fmt.Printf("Output file size: %d B\n", fileSize)


	fmt.Printf("Creating the output file at '%v'...\n", outPath)
	outFile, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("There was an error creating the file '%v'.\n", outPath)
		return err
	}

	defer func() {
		if err = outFile.Close(); err != nil {
			fmt.Println("Error closing the file:", err.Error())
		}
	}()


	fmt.Printf("Writing to the output file at '%v'...\n", outPath)
	readBytes := int64(0)
	for readBytes < fileSize {
		n := util.Min(int(encodeChunkSize), int(fileSize - readBytes))
		if err = decodeChunk(&f, info, pixels, channelsPerPix, maxBitsPerChannel, &b, n, encodeLsb); err != nil {
			switch err.(type) {
			case *algos.EmptyPoolError:
				return &InsufficientHidingSpotsError{InnerError:err}
			default:
				return err
			}
		}
		if r, err := outFile.Write(b[:n]); err != nil {
			return err
		} else {
			readBytes += int64(r)
		}
	}


	fmt.Println("All done! c:")

	return nil
}

// Helper functions

func decodeChunk(pos *func() (int64, error), info imgInfo, pixels *[]pixel, channelCount, maxBitsPerChannel uint8, buf *[]byte, n int, lsb bool) error {
	for i := 0; i < n; i++ {
		for j := uint8(0); j < bitsPerByte; j++ {
			for {
				addr, err := (*pos)()
				if err != nil {
					return err
				}
				p, c, b := bitAddrToPCB(addr, channelCount, maxBitsPerChannel)

				if debugOutput {
					fmt.Printf("addr: %d, pixel: %d, channel: %d, bit: %d, RGBA: %v\n", addr, p, c, b, (*pixels)[p])
				}

				// TODO: Note that this has the potential to introduce nasty bugs if a (0,0,0,1) is turned into a (0,0,0,0)
				if (*pixels)[p][3] <= 0 {
					continue
				}

				bitPos := b
				if !lsb {
					bitPos = bitsPerByte - b - 1
				}

				readBit := binmani.ReadFrom((*pixels)[p][c], bitPos, 1)
				(*buf)[i] = byte(binmani.WriteTo(uint16((*buf)[i]), bitsPerByte - j - 1, 1, readBit))

				if debugOutput {
					fmt.Printf("	Read %d\n", readBit)
				}

				break
			}
		}
	}

	if debugOutput {
		fmt.Println("Read chunk:", string(*buf))
	}

	return nil
}
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

// DigConfig stores the configuration options for the Dig operation.
type DigConfig struct {
	ImagePath         string      // The path on disk to a supported image.
	OutPath           string      // The path on disk to write the output file.
	PatternPath       string      // The path on disk to the pattern file used in decoding.
	Algorithm         algos.Algo  // The algorithm to use in the operation.
	MaxBitsPerChannel uint8       // The maximum number of bits to write per pixel channel - the minimum of this and the supported max of the image format is used.
	EncodeAlpha       bool        // Whether or not to encode the alpha channel.
	DecodeMsb         bool        // Whether to encode the most-significant bits instead - mostly for debugging.
	OutputLevel       OutputLevel // The amount of output to provide.
}

// BadHeaderError is thrown when the read header is garbage. Likely caused by a bad configuration or source image.
type BadHeaderError struct {}

// Error returns a string that explains the BadHeaderError.
func (e *BadHeaderError) Error() string {
	return "The read header is not valid!"
}

// Primary method

// Dig extracts the binary data of a file from a provided image on disk, and saves the result to a new file.
// The configuration must perfectly match the one used in encoding in order to extract successfully.
func Dig(config DigConfig) error {
	// Input validation
	if len(config.ImagePath) <= 0 {
		return &InvalidFormatError{"ImagePath is empty."}
	}
	if len(config.OutPath) <= 0 {
		return &InvalidFormatError{"OutPath is empty."}
	}
	if len(config.PatternPath) <= 0 {
		return &InvalidFormatError{"PatternPath is empty."}
	}
	if !config.Algorithm.IsValid() {
		return &InvalidFormatError{"Algorithm is invalid."}
	}
	if config.MaxBitsPerChannel < 0 || config.MaxBitsPerChannel > 16 {
		return &InvalidFormatError{fmt.Sprintf("MaxBitsPerChannel is outside the allowed range of 0-16: Provided %d.", config.MaxBitsPerChannel)}
	}

	printlnLvl(config.OutputLevel, OutputDebug, "This tool has been set to display debug output.")

	printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("Loading the image from '%v'...", config.ImagePath))
	pixels, info, err := loadImage(config.ImagePath, config.OutputLevel)
	if err != nil {
		printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("Unable to load the image at '%v'!", config.ImagePath))
		return err
	}

	config.MaxBitsPerChannel = uint8(util.Min(int(config.MaxBitsPerChannel), int(info.Format.BitsPerChannel)))

	printlnLvl(config.OutputLevel, OutputInfo,
		fmt.Sprintf("Image info:\n\tDimensions: %dx%d px\n\tColour model: %v\n\tChannels per pixel: %d\n\tBits per channel: %d",
		info.W, info.H, colourModelToStr(info.Format.Model), info.Format.ChannelsPerPix, info.Format.BitsPerChannel))


	printlnLvl(config.OutputLevel, OutputSteps, "Loading up the pattern key...")
	pHash, err := hashPatternFile(config.PatternPath)
	if err != nil {
		printlnLvl(config.OutputLevel, OutputSteps,
			fmt.Sprintf("Something went wrong while attempting to hash the pattern file '%v'.", config.PatternPath))
		return err
	}
	printlnLvl(config.OutputLevel, OutputInfo, "Pattern hash:", pHash)


	printlnLvl(config.OutputLevel, OutputSteps, "Reading the file from the image...")

	channelsPerPix := info.Format.ChannelsPerPix
	if info.Format.supportsAlpha() && !config.EncodeAlpha {
		channelsPerPix--
	}
	if channelsPerPix <= 0 { // In the case of Alpha & Alpha16 models
		return &InsufficientHidingSpotsError{AdditionalInfo:fmt.Sprintf("The provided image is of the %v colour" +
			"model, but since alpha-channel encoding was not specified, there are no channels to hide data within.",
			colourModelToStr(info.Format.Model))}
	}

	channelCount := int64(len(*pixels)) * int64(channelsPerPix)
	printlnLvl(config.OutputLevel, OutputInfo, "Maximum readable bits:", channelCount * int64(config.MaxBitsPerChannel))

	f, err := algos.AlgoAddressor(config.Algorithm, pHash, channelCount, config.MaxBitsPerChannel)
	if err != nil {
		return err
	}


	printlnLvl(config.OutputLevel, OutputSteps, "Reading steg header...")

	b, header := make([]byte, encodeChunkSize), make([]byte, encodeHeaderSize)
	if err = decodeChunk(config, &f, pixels, channelsPerPix, &header, int(encodeHeaderSize)); err != nil {
		switch err.(type) {
		case *algos.EmptyPoolError:
			return &InsufficientHidingSpotsError{InnerError:err}
		default:
			return err
		}
	}

	headerStr := string(header[0:])

	if config.OutputLevel == OutputDebug {
		fmt.Println("Encoding header:", headerStr)
		for _, v := range b {
			fmt.Printf("%#08b\n", v)
		}
	}

	headerParts := strings.Split(headerStr, encodeHeaderSeparator)
	if len(headerParts) < 2 {
		return &BadHeaderError{}
	}

	printlnLvl(config.OutputLevel, OutputDebug, "Header parts:", headerParts)

	fileSize, err := strconv.ParseInt(headerParts[1], 10, 64)
	if err != nil {
		fmt.Println("The read file size is not valid!")
		return err
	}

	printlnLvl(config.OutputLevel, OutputInfo, fmt.Sprintf("Output file size: %d B", fileSize))


	printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("Creating the output file at '%v'...", config.OutPath))
	outFile, err := os.Create(config.OutPath)
	if err != nil {
		printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("There was an error creating the file '%v'.", config.OutPath))
		return err
	}

	defer func() {
		if err = outFile.Close(); err != nil {
			printlnLvl(config.OutputLevel, OutputSteps, "Error closing the file:", err.Error())
		}
	}()


	printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("Writing to the output file at '%v'...", config.OutPath))
	readBytes := int64(0)
	for readBytes < fileSize {
		n := util.Min(int(encodeChunkSize), int(fileSize - readBytes))
		if err = decodeChunk(config, &f, pixels, channelsPerPix, &b, n); err != nil {
			switch err.(type) {
			case *algos.EmptyPoolError:
				return &InsufficientHidingSpotsError{InnerError:err}
			default:
				return err
			}
		}
		r, err := outFile.Write(b[:n])
		if err != nil {
			return err
		}
		readBytes += int64(r)
	}


	printlnLvl(config.OutputLevel, OutputSteps, "All done! c:")

	return nil
}

// Helper functions

func decodeChunk(config DigConfig, pos *func() (int64, error), pixels *[]pixel, channelCount uint8, buf *[]byte, n int) error {
	for i := 0; i < n; i++ {
		for j := uint8(0); j < bitsPerByte; j++ {
			for {
				addr, err := (*pos)()
				if err != nil {
					return err
				}
				p, c, b := bitAddrToPCB(addr, channelCount, config.MaxBitsPerChannel)

				if config.OutputLevel == OutputDebug {
					fmt.Printf("addr: %d, pixel: %d, channel: %d, bit: %d, RGBA: %v\n", addr, p, c, b, (*pixels)[p])
				}

				// TODO: Note that this has the potential to introduce nasty bugs if a (0,0,0,1) is turned into a (0,0,0,0)
				if (*pixels)[p][3] <= 0 {
					continue
				}

				bitPos := b
				if config.DecodeMsb {
					bitPos = bitsPerByte - b - 1
				}

				readBit := binmani.ReadFrom((*pixels)[p][c], bitPos, 1)
				(*buf)[i] = byte(binmani.WriteTo(uint16((*buf)[i]), bitsPerByte - j - 1, 1, readBit))

				if config.OutputLevel == OutputDebug {
					fmt.Printf("	Read %d\n", readBit)
				}

				break
			}
		}
	}

	if config.OutputLevel == OutputDebug {
		fmt.Println("Read chunk:", string(*buf))
	}

	return nil
}
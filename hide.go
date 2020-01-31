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

// HideConfig stores the configuration options for the Hide operation.
type HideConfig struct {
	// ImagePath is the path on disk to a supported image.
	ImagePath         string
	// FilePath is the path on disk to the file to hide.
	FilePath          string
	// OutPath is the path on disk to write the output image.
	OutPath           string
	// PatternPath is the path on disk to the pattern file used in encoding.
	PatternPath       string
	// Algorithm is the algorithm to use in the operation.
	Algorithm         algos.Algo
	// MaxBitsPerChannel is the maximum number of bits to write per pixel channel.
	// The minimum of this and the supported max of the image format is used.
	MaxBitsPerChannel uint8
	// DecodeAlpha is whether or not to encode the alpha channel.
	EncodeAlpha       bool
	// EncodeMsb is whether to encode the most-significant bits instead - mostly for debugging.
	EncodeMsb         bool
	// OutputLevel is the amount of output to provide.
	OutputLevel       OutputLevel
}

// Hide hides the binary data of a file in a provided image on disk, and saves the result to a new image.
// It has the option of using one of several different encoding algorithms, depending on user needs.
func Hide(config HideConfig) error {
	// Input validation
	if len(config.ImagePath) <= 0 {
		return &InvalidFormatError{"ImagePath is empty."}
	}
	if len(config.FilePath) <= 0 {
		return &InvalidFormatError{"FilePath is empty."}
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

	printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("Opening the file at '%v'...", config.FilePath))
	fileReader, err := os.Open(config.FilePath)
	if err != nil {
		printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("Unable to open the file at '%v'.", config.FilePath))
		return err
	}

	defer func() {
		if err = fileReader.Close(); err != nil {
			printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("Error closing the file '%v': %v", config.FilePath, err.Error()))
		}
	}()


	printlnLvl(config.OutputLevel, OutputSteps, "Loading up the pattern key...")
	pHash, err := hashPatternFile(config.PatternPath)
	if err != nil {
		printlnLvl(config.OutputLevel, OutputSteps,
			fmt.Sprintf("Something went wrong while attempting to hash the pattern file '%v'.", config.PatternPath))
		return err
	}
	printlnLvl(config.OutputLevel, OutputInfo, "Pattern hash:", pHash)


	printlnLvl(config.OutputLevel, OutputSteps, "Encoding the file into the image...")

	r := bufio.NewReader(fileReader)
	b := make([]byte, encodeChunkSize)

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
	maxWritableBits := channelCount * int64(config.MaxBitsPerChannel)
	printlnLvl(config.OutputLevel, OutputInfo, "Maximum writable bits:", maxWritableBits)

	f, err := algos.AlgoAddressor(config.Algorithm, pHash, channelCount, config.MaxBitsPerChannel)
	if err != nil {
		return err
	}


	printlnLvl(config.OutputLevel, OutputSteps, "Writing steg header...")

	fileInfo, err := fileReader.Stat()
	if err != nil {
		printlnLvl(config.OutputLevel, OutputSteps, "Unable to retrieve file info!")
		return err
	}

	b = []byte(fmt.Sprintf("steg%02d.%02d.%02d%v%019d", VersionMax, VersionMid, VersionMin, encodeHeaderSeparator, fileInfo.Size()))
	bitsToWrite := fileInfo.Size() * int64(bitsPerByte)

	printlnLvl(config.OutputLevel, OutputInfo, fmt.Sprintf("Input file size: %d B", fileInfo.Size()))
	printlnLvl(config.OutputLevel, OutputInfo, "Bits to write:", bitsToWrite)

	if bitsToWrite > maxWritableBits {
		return &InsufficientHidingSpotsError{AdditionalInfo:fmt.Sprintf("Since the number of bits to write is %d " +
			"and the maximum possible with this configuration is %d, there is no way the input file will fit.", bitsToWrite, maxWritableBits)}
	}

	printlnLvl(config.OutputLevel, OutputDebug, "Encoding header:", string(b[0:]))

	if err = encodeChunk(config, info, &f, pixels, channelsPerPix, &b, int(encodeHeaderSize)); err != nil {
		switch err.(type) {
		case *algos.EmptyPoolError:
			return &InsufficientHidingSpotsError{InnerError:err}
		default:
			return err
		}
	}


	printlnLvl(config.OutputLevel, OutputSteps, "Writing file data...")

	if config.OutputLevel >= OutputDebug {
		for _, v := range b {
			fmt.Printf("%#08b\n", v)
		}
	}

	for {
		n, err := r.Read(b)
		if n > 0 {
			if err = encodeChunk(config, info, &f, pixels, channelsPerPix, &b, n); err != nil {
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
				printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("An error occurred while reading the file '%v'.", config.FilePath))
				return err
			}
			break
		}
	}


	printlnLvl(config.OutputLevel, OutputSteps, fmt.Sprintf("Writing the encoded image to '%v' now...", config.OutPath))
	if err = writeImage(pixels, info, config.OutPath, config.OutputLevel); err != nil {
		printlnLvl(config.OutputLevel, OutputSteps, "An error occurred while writing to the final image.")
		return err
	}


	printlnLvl(config.OutputLevel, OutputSteps, "All done! c:")

	return nil
}

// Helper functions

func encodeChunk(config HideConfig, info imgInfo, pos *func() (int64, error), pixels *[]pixel, channelCount uint8, buf *[]byte, n int) error {
	supportsAlpha := info.Format.supportsAlpha()
	alphaChannel := info.Format.alphaChannel()
	for i := 0; i < n; i++ {
		for j := uint8(0); j < bitsPerByte; j++ {
			writeBit := binmani.ReadFrom(uint16((*buf)[i]), bitsPerByte - j - 1, 1)

			for {
				addr, err := (*pos)()
				if err != nil {
					return err
				}
				p, c, b := bitAddrToPCB(addr, channelCount, config.MaxBitsPerChannel)

				if config.OutputLevel >= OutputDebug {
					fmt.Printf("addr: %d, pixel: %d, channel: %d, bit: %d, RGBA: %v\n", addr, p, c, b, (*pixels)[p])
				}

				if supportsAlpha && (*pixels)[p][alphaChannel] <= 0 {
					continue
				}

				if config.OutputLevel >= OutputDebug {
					fmt.Printf("	Writing %d...\n", writeBit)
					fmt.Printf("	Channel before: %#016b - %v\n", (*pixels)[p][c], (*pixels)[p])
				}

				bitPos := b
				if config.EncodeMsb {
					bitPos = info.Format.BitsPerChannel - b - 1
				}
				(*pixels)[p][c] = binmani.WriteTo((*pixels)[p][c], bitPos, 1, writeBit)

				if config.OutputLevel >= OutputDebug {
					fmt.Printf("	Channel after:  %#016b - %v\n", (*pixels)[p][c], (*pixels)[p])
				}

				break
			}
		}
	}

	return nil
}
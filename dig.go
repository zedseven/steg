package steg

import (
	"fmt"
	"os"

	"github.com/zedseven/bch"
	"github.com/zedseven/binmani"
	"github.com/zedseven/steg/internal/algos"
	"github.com/zedseven/steg/internal/util"
)

// Types

// DigConfig stores the configuration options for the Dig operation.
type DigConfig struct {
	// ImagePath is the path on disk to a supported image.
	ImagePath         string
	// OutPath is the path on disk to write the output image.
	OutPath           string
	// PatternPath is the path on disk to the pattern file used in decoding.
	PatternPath       string
	// Algorithm is the algorithm to use in the operation.
	Algorithm         algos.Algo
	// MaxCorrectableErrors is the number of bit errors to be able to correct for per file chunk. Setting it to 0 disables bit ECC.
	MaxCorrectableErrors uint8
	// MaxBitsPerChannel is the maximum number of bits to write per pixel channel.
	// The minimum of this and the supported max of the image format is used.
	MaxBitsPerChannel uint8
	// DecodeAlpha is whether or not to decode the alpha channel.
	DecodeAlpha       bool
	// DecodeMsb is whether to decode the most-significant bits instead - mostly for debugging.
	DecodeMsb         bool
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
func Dig(config *DigConfig, outputLevel OutputLevel) error {
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

	printlnLvl(outputLevel, OutputSteps, fmt.Sprintf("Steg v%d.%d.%d by Zacchary Dempsey-Plante.", VersionMax, VersionMid, VersionMin))
	printlnLvl(outputLevel, OutputDebug, "This tool has been set to display debug output.")

	printlnLvl(outputLevel, OutputSteps, fmt.Sprintf("Loading the image from '%v'...", config.ImagePath))
	pixels, info, err := loadImage(config.ImagePath, outputLevel)
	if err != nil {
		printlnLvl(outputLevel, OutputSteps, fmt.Sprintf("Unable to load the image at '%v'!", config.ImagePath))
		return err
	}

	config.MaxBitsPerChannel = uint8(util.Min(int(config.MaxBitsPerChannel), int(info.Format.BitsPerChannel)))

	printlnLvl(outputLevel, OutputInfo,
		fmt.Sprintf("Image info:\n\tDimensions: %dx%dpx\n\tColour model: %v\n\tChannels per pixel: %d\n\tBits per channel: %d",
		info.W, info.H, colourModelToStr(info.Format.Model), info.Format.ChannelsPerPix, info.Format.BitsPerChannel))


	printlnLvl(outputLevel, OutputSteps, "Loading up the pattern key...")
	pHash, err := hashPatternFile(config.PatternPath)
	if err != nil {
		printlnLvl(outputLevel, OutputSteps,
			fmt.Sprintf("Something went wrong while attempting to hash the pattern file '%v'.", config.PatternPath))
		return err
	}
	printlnLvl(outputLevel, OutputInfo, "Pattern hash:", pHash)


	printlnLvl(outputLevel, OutputSteps, "Reading the file from the image...")

	channelsPerPix := info.Format.ChannelsPerPix
	if info.Format.supportsAlpha() && !config.DecodeAlpha {
		channelsPerPix--
	}
	if channelsPerPix <= 0 { // In the case of Alpha & Alpha16 models
		return &InsufficientHidingSpotsError{AdditionalInfo:fmt.Sprintf("The provided image is of the %v colour" +
			"model, but since alpha-channel encoding was not specified, there are no channels to hide data within.",
			colourModelToStr(info.Format.Model))}
	}

	channelCount := int64(len(*pixels)) * int64(channelsPerPix)
	printlnLvl(outputLevel, OutputInfo, "Maximum readable bits:", channelCount * int64(config.MaxBitsPerChannel))

	f, err := algos.AlgoAddressor(config.Algorithm, pHash, channelCount, config.MaxBitsPerChannel)
	if err != nil {
		return err
	}


	var eccConfig *bch.EncodingConfig = nil
	if config.MaxCorrectableErrors > 0 {
		printlnLvl(outputLevel, OutputSteps, "Setting up data ECC...")
		chunkBitSize := util.Max(int(encodeChunkSize), int(encodeHeaderSize)) * int(bitsPerByte)
		codeLength, err := bch.TotalBitsForConfig(chunkBitSize, int(config.MaxCorrectableErrors))
		if err != nil {
			return err
		}

		eccConfig, err = bch.CreateConfig(codeLength, int(config.MaxCorrectableErrors))
		if err != nil {
			return err
		}
		printlnLvl(outputLevel, OutputInfo, fmt.Sprintf("Using a %v. This has a ratio (errors : bits) of %2.2f%%.",
			eccConfig, 100 * eccConfig.ECCRatio()))
	}


	eccErrors := 0

	printlnLvl(outputLevel, OutputSteps, "Reading steg header...")

	b, header := make([]byte, encodeChunkSize), make([]byte, encodeHeaderSize)
	if eccErrors, err = decodeChunk(config, eccConfig, info, &f, pixels, channelsPerPix, &header, int(encodeHeaderSize), outputLevel); err != nil {
		switch err.(type) {
		case *algos.EmptyPoolError:
			return &InsufficientHidingSpotsError{InnerError:err}
		default:
			return err
		}
	}

	headerStr := string(header[0:])

	if outputLevel == OutputDebug {
		fmt.Println("Encoding header:", headerStr)
		for _, v := range header {
			fmt.Printf("%#08b\n", v)
		}
	}

	encodeVersionMax := header[0]
	encodeVersionMid := header[1]
	encodeVersionMin := header[2]
	fileSize := int64(header[3])
	fileSize <<= 8
	fileSize += int64(header[4])
	fileSize <<= 8
	fileSize += int64(header[5])
	fileSize <<= 8
	fileSize += int64(header[6])

	/*if err != nil {
		fmt.Println("The read file size is not valid!")
		return err
	}*/

	printlnLvl(outputLevel, OutputInfo, fmt.Sprintf("This image was encoded with steg v%d.%d.%d.",
		encodeVersionMax, encodeVersionMid, encodeVersionMin))

	if encodeVersionMax != VersionMax || encodeVersionMid != VersionMid || encodeVersionMin != VersionMin {
		printlnLvl(outputLevel, OutputSteps,
			"This image was encoded with a different version of Steg. The program will continue, but in the case",
			"of strange errors or issues, try using the same version as the image was originally encoded with.")
	}

	printlnLvl(outputLevel, OutputInfo, fmt.Sprintf("Output file size: %d B", fileSize))


	printlnLvl(outputLevel, OutputSteps, fmt.Sprintf("Creating the output file at '%v'...", config.OutPath))
	outFile, err := os.Create(config.OutPath)
	if err != nil {
		printlnLvl(outputLevel, OutputSteps, fmt.Sprintf("There was an error creating the file '%v'.", config.OutPath))
		return err
	}

	defer func() {
		if err = outFile.Close(); err != nil {
			printlnLvl(outputLevel, OutputSteps, "Error closing the file:", err.Error())
		}
	}()


	printlnLvl(outputLevel, OutputSteps, fmt.Sprintf("Writing to the output file at '%v'...", config.OutPath))
	readBytes := int64(0)
	for readBytes < fileSize {
		n := util.Min(int(encodeChunkSize), int(fileSize - readBytes))
		if errors, err := decodeChunk(config, eccConfig, info, &f, pixels, channelsPerPix, &b, n, outputLevel); err != nil {
			switch err.(type) {
			case *algos.EmptyPoolError:
				return &InsufficientHidingSpotsError{InnerError:err}
			default:
				return err
			}
		} else {
			eccErrors += errors
		}
		r, err := outFile.Write(b[:n])
		if err != nil {
			return err
		}
		readBytes += int64(r)
	}

	if config.MaxCorrectableErrors > 0 {
		printlnLvl(outputLevel, OutputInfo, fmt.Sprintf("There were %d error(s) in the image.", eccErrors))
	}


	printlnLvl(outputLevel, OutputSteps, "All done! c:")

	return nil
}

// Helper functions

func decodeChunk(config *DigConfig, eccConfig *bch.EncodingConfig, info imgInfo, pos *func() (int64, error), pixels *[]pixel, channelCount uint8, buf *[]byte, n int, outputLevel OutputLevel) (int, error) {
	supportsAlpha := info.Format.supportsAlpha()
	alphaChannel := info.Format.alphaChannel()

	readLength := n * int(bitsPerByte)
	if eccConfig != nil {
		readLength += eccConfig.ChecksumBits()
	}
	codeBits := make([]uint8, readLength)
	for i := 0; i < readLength; i++ {
		for {
			addr, err := (*pos)()
			if err != nil {
				return -1, err
			}
			p, c, b := bitAddrToPCB(addr, channelCount, config.MaxBitsPerChannel)

			if outputLevel == OutputDebug {
				fmt.Printf("addr: %d, pixel: %d, channel: %d, bit: %d, RGBA: %v\n", addr, p, c, b, (*pixels)[p])
			}

			// TODO: Note that this has the potential to introduce nasty bugs if a (0,0,0,1) is turned into a (0,0,0,0)
			if supportsAlpha && (*pixels)[p][alphaChannel] <= 0 {
				continue
			}

			bitPos := b
			if config.DecodeMsb {
				bitPos = bitsPerByte - b - 1
			}

			readBit := binmani.ReadFrom((*pixels)[p][c], bitPos, 1)
			codeBits[i] = uint8(readBit)

			if outputLevel == OutputDebug {
				fmt.Printf("	Read %d\n", readBit)
			}

			break
		}
	}

	var eccErrors int
	if eccConfig != nil {
		printlnLvl(outputLevel, OutputDebug, codeBits)
		padBits := make([]uint8, eccConfig.CodeLength - readLength)
		codeBits = append(codeBits, padBits...)
		decodedBits, errors, err := bch.Decode(eccConfig, &codeBits)
		if err != nil {
			return -1, err
		}
		printlnLvl(outputLevel, OutputDebug, "Errors in this chunk:", errors)
		*buf = *binmani.BitsToBytes(decodedBits, false)
		eccErrors = errors
	} else {
		*buf = *binmani.BitsToBytes(codeBits, false)
	}

	if outputLevel == OutputDebug {
		fmt.Println("Read chunk:", string(*buf))
	}

	return eccErrors, nil
}
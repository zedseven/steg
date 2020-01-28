package steg

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/zedseven/steg/internal/algos"
	"github.com/zedseven/steg/pkg/binmani"
)

// Primary method

func Hide(imgPath, filePath, outPath, patternPath string, bpc uint8, alpha, lsb bool) {
	maxBitsPerChannel, encodeAlpha, encodeLsb = bpc, alpha, lsb

	fmt.Printf("Loading the image from '%v'...\n", imgPath)
	pixels, iinfo, err := loadImage(imgPath)

	if err != nil {
		fmt.Printf("Unable to load the image at '%v'! %v\n", imgPath, err.Error())
		return
	}

	fmt.Println("Image info:", iinfo)

	fmt.Printf("Opening the file at '%v'...\n", filePath)
	buf, err := os.Open(filePath)

	if err != nil {
		fmt.Printf("Unable to open the file at '%v'. %v\n", filePath, err.Error())
		return
	}

	defer func() {
		if err = buf.Close(); err != nil {
			fmt.Println("Error closing the file:", err.Error())
		}
	}()

	fmt.Println("Loading up the pattern key...")
	pHash := hashPatternFile(patternPath)
	fmt.Println("Pattern hash:", pHash)

	fmt.Println("Encoding the file into the image...")
	//Write the file data to the pixels data
	r := bufio.NewReader(buf)
	b := make([]byte, encodeChunkSize)
	channelsPerPix := uint8(3)
	if encodeAlpha {
		channelsPerPix = 4
	}
	channelCount := int64(len(*pixels)) * int64(channelsPerPix) //TODO: incorporate iinfo.Format.ChannelsPerPix
	//int64(len(pixels)) * int64(len(pixels[0])) * int64(channelsPerPix)
	fmt.Println("channelCount:", channelCount)
	//f := algos.SequentialAddressor(channelCount, maxBitsPerChannel)
	f := algos.PatternAddressor(pHash, channelCount, maxBitsPerChannel)

	//Write the encoding header
	info, err := buf.Stat()
	if err != nil {
		fmt.Println("Unable to retrieve file info:", err.Error())
	}

	b = []byte(fmt.Sprintf("steg%02d.%02d.%02d%v%019d", versionMax, versionMid, versionMin, encodeHeaderSeparator, info.Size()))
	fmt.Println("Encoding header:", string(b[0:]))
	encodeChunk(&f, iinfo, pixels, channelsPerPix, &b, int(encodeHeaderSize))

	for _, v := range b {
		fmt.Printf("%#08b\n", v)
	}

	for {
		n, err := r.Read(b)
		if n > 0 {
			//fmt.Println(string(b[0:n]))
			encodeChunk(&f, iinfo, pixels, channelsPerPix, &b, n)
		}
		if err != nil {
			if err != io.EOF {
				fmt.Println("An error occurred while reading the file:", err.Error())
			}
			break
		}
	}

	//Write the pixels data to file
	fmt.Printf("Writing the encoded image to '%v' now...\n", outPath)
	writeImage(pixels, iinfo, outPath)

	fmt.Println("All done! c:")
}

// Helper functions

func encodeChunk(pos *func() (int64, error), info imgInfo, pixels *[]pixel, channelCount uint8, buf *[]byte, n int) {
	//fmt.Println("Image dims:", info.W, info.H)

	for i := 0; i < n; i++ {
		//fmt.Printf("(%c) %#08b:\n", buf[i], buf[i])
		for j := uint8(0); j < bitsPerByte; j++ {
			//TODO: Look here first if errors with encoding file data correctly
			writeBit := binmani.ReadFrom(uint16((*buf)[i]), bitsPerByte - j - 1, 1)

			//fmt.Println(imgio.readFrom(buf[i], bitsPerByte - j - 1, 1))

			for {
				addr, err := (*pos)()
				if err != nil {
					fmt.Errorf("Something went seriously wrong when fetching the next bit address: %v\n", err.Error())
					panic("Something went seriously wrong when fetching the next bit address.")
				}
				p, c, b := bitAddrToPCB(addr, channelCount, maxBitsPerChannel)
				//x, y := imgio.posToXY(p, w)
				//fmt.Printf("addr: %d, pixel: (%d: %d, %d), channel: %d, bit: %d, RGBA: %v\n", addr, p, x, y, c, b, (*pixels)[y][x])
				fmt.Printf("addr: %d, pixel: %d, channel: %d, bit: %d, RGBA: %v\n", addr, p, c, b, (*pixels)[p])
				if (*pixels)[p][3] <= 0 {
					continue
				}

				fmt.Printf("	Writing %d...\n", writeBit)
				//fmt.Printf("writing (%d, %d: %d, %d)\n", x, y, c, b)

				var channelAddr *uint16
				switch c {
				case 0:
					channelAddr = &(*pixels)[p][0]
				case 1:
					channelAddr = &(*pixels)[p][1]
				case 2:
					channelAddr = &(*pixels)[p][2]
				case 3:
					channelAddr = &(*pixels)[p][3]
				}

				fmt.Printf("	Channel before: %#016b - %v\n", *channelAddr, (*pixels)[p])

				bitPos := b
				if !encodeLsb {
					bitPos = info.Format.BitsPerChannel - b - 1
				}
				*channelAddr = binmani.WriteTo(*channelAddr, bitPos, 1, writeBit)

				fmt.Printf("	Channel after:  %#016b - %v\n", *channelAddr, (*pixels)[p])

				break
			}
		}
	}
}
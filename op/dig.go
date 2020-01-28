package op

import (
	"fmt"
	"github.com/zedseven/steg/imgio"
	"strconv"
	"strings"
)

func Dig(imgPath, outPath, patternPath string, bpc uint8, alpha, lsb bool) {
	maxBitsPerChannel, encodeAlpha, encodeLsb = bpc, alpha, lsb

	fmt.Printf("Loading the image from '%v'...\n", imgPath)
	pixels, iinfo, err := imgio.LoadImage(imgPath)

	if err != nil {
		fmt.Printf("Unable to load the image at '%v'! %v\n", imgPath, err.Error())
		return
	}

	fmt.Println("Loading up the pattern key...")
	pHash := imgio.HashPatternFile(patternPath)
	fmt.Println("Pattern hash:", pHash)

	channelsPerPix := uint8(3)
	if encodeAlpha {
		channelsPerPix = 4
	}
	channelCount := int64(len(*pixels)) * int64(channelsPerPix) //TODO: incorporate iinfo.Format.ChannelsPerPix
	//int64(len(pixels)) * int64(len(pixels[0])) * int64(channelsPerPix)
	fmt.Println("channelCount:", channelCount)
	//f := sequentialAddressor(channelCount, maxBitsPerChannel)
	f := patternAddressor(pHash, channelCount, maxBitsPerChannel)

	_, header := make([]byte, encodeChunkSize), make([]byte, encodeHeaderSize)
	decodeChunk(&f, iinfo, pixels, channelsPerPix, &header, int(encodeHeaderSize))

	headerStr := string(header[0:])

	fmt.Println("Encoding header:", headerStr)

	for _, v := range header {
		fmt.Printf("%#08b\n", v)
	}

	headerParts := strings.Split(headerStr, encodeHeaderSeparator)
	if len(headerParts) < 2 {
		fmt.Println("The read header is not valid!")
	}
	fmt.Println("Header parts:", headerParts)

	fileSize, err := strconv.ParseInt(headerParts[1], 10, 64)
	if err != nil {
		fmt.Println("The read filesize is not valid!")
	}
	fmt.Println("File size:", fileSize)
}

func decodeChunk(pos *func() (int64, error), info imgio.ImgInfo, pixels *[]imgio.Pixel, channelCount uint8, buf *[]byte, n int) {
	for i := 0; i < n; i++ {
		for j := uint8(0); j < bitsPerByte; j++ {
			for {
				addr, err := (*pos)()
				if err != nil {
					fmt.Errorf("Something went seriously wrong when fetching the next bit address: %v\n", err.Error())
					panic("Something went seriously wrong when fetching the next bit address.")
				}
				p, c, b := imgio.BitAddrToPCB(addr, channelCount, maxBitsPerChannel)
				//x, y := imgio.PosToXY(p, int(info.W))
				//fmt.Printf("addr: %d, pixel: (%d: %d, %d), channel: %d, bit: %d, RGBA: %v\n", addr, p, x, y, c, b, (*pixels)[y][x])
				fmt.Printf("addr: %d, pixel: %d, channel: %d, bit: %d, RGBA: %v\n", addr, p, c, b, (*pixels)[p])
				if (*pixels)[p].Channels[3] <= 0 {
					continue
				}

				var channelAddr *uint16
				switch c {
				case 0:
					channelAddr = &(*pixels)[p].Channels[0]
				case 1:
					channelAddr = &(*pixels)[p].Channels[1]
				case 2:
					channelAddr = &(*pixels)[p].Channels[2]
				case 3:
					channelAddr = &(*pixels)[p].Channels[3]
				}

				bitPos := b
				if !encodeLsb {
					bitPos = bitsPerByte - b - 1
				}

				readBit := imgio.ReadFrom(*channelAddr, bitPos, 1)
				(*buf)[i] = byte(imgio.WriteTo(uint16((*buf)[i]), bitsPerByte - j - 1, 1, readBit))

				fmt.Printf("	Read %d\n", readBit)

				break
			}
		}
	}

	fmt.Println("Read chunk:", string(*buf))
}
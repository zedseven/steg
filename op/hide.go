package op

import (
	"bufio"
	"fmt"
	"github.com/zedseven/steg/imgio"
	"io"
	"os"
)

func Hide(imgPath, filePath, outPath, patternPath string, bpc uint8, alpha, lsb bool) {
	maxBitsPerChannel, encodeAlpha, encodeLsb = bpc, alpha, lsb

	fmt.Printf("Loading the image from '%v'...\n", imgPath)
	pixels, iinfo, err := imgio.LoadImage(imgPath)

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
	pHash := imgio.HashPatternFile(patternPath)
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
	//f := sequentialAddressor(channelCount, maxBitsPerChannel)
	f := patternAddressor(pHash, channelCount, maxBitsPerChannel)

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
	imgio.WriteImage(pixels, iinfo, outPath)

	fmt.Println("All done! c:")
}

func encodeChunk(pos *func() (int64, error), info imgio.ImgInfo, pixels *[]imgio.Pixel, channelCount uint8, buf *[]byte, n int) {
	//fmt.Println("Image dims:", info.W, info.H)

	for i := 0; i < n; i++ {
		//fmt.Printf("(%c) %#08b:\n", buf[i], buf[i])
		for j := uint8(0); j < bitsPerByte; j++ {
			//TODO: Look here first if errors with encoding file data correctly
			writeBit := imgio.ReadFrom(uint16((*buf)[i]), bitsPerByte - j - 1, 1)

			//fmt.Println(imgio.ReadFrom(buf[i], bitsPerByte - j - 1, 1))

			for {
				addr, err := (*pos)()
				if err != nil {
					fmt.Errorf("Something went seriously wrong when fetching the next bit address: %v\n", err.Error())
					panic("Something went seriously wrong when fetching the next bit address.")
				}
				p, c, b := imgio.BitAddrToPCB(addr, channelCount, maxBitsPerChannel)
				//x, y := imgio.PosToXY(p, w)
				//fmt.Printf("addr: %d, pixel: (%d: %d, %d), channel: %d, bit: %d, RGBA: %v\n", addr, p, x, y, c, b, (*pixels)[y][x])
				fmt.Printf("addr: %d, pixel: %d, channel: %d, bit: %d, RGBA: %v\n", addr, p, c, b, (*pixels)[p])
				if (*pixels)[p].Channels[3] <= 0 {
					continue
				}

				fmt.Printf("	Writing %d...\n", writeBit)
				//fmt.Printf("writing (%d, %d: %d, %d)\n", x, y, c, b)

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

				fmt.Printf("	Channel before: %#016b - %v\n", *channelAddr, (*pixels)[p])

				bitPos := b
				if !encodeLsb {
					bitPos = info.Format.BitsPerChannel - b - 1
				}
				*channelAddr = imgio.WriteTo(*channelAddr, bitPos, 1, writeBit)

				fmt.Printf("	Channel after:  %#016b - %v\n", *channelAddr, (*pixels)[p])

				break
			}
		}
	}
}
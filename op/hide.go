package op

import (
	"bufio"
	"fmt"
	"github.com/zedseven/steg/imgio"
	"io"
	"math/rand"
	"os"
)

const bitsPerByte uint8 = 8
const encodeChunkSize uint8 = 32
var maxBitsPerChannel uint8 = 1
var encodeAlpha = true
var encodeLsb = true //For debugging

func Hide(imgPath, filePath, outPath, patternPath string, bpc uint8, alpha, lsb bool) {
	maxBitsPerChannel, encodeAlpha, encodeLsb = bpc, alpha, lsb

	fmt.Printf("Loading the image from '%v'...\n", imgPath)
	pixels, err := imgio.LoadImage(imgPath)

	if err != nil {
		fmt.Printf("Unable to load the image at '%v'! %v\n", imgPath, err.Error())
		return
	}

	fmt.Printf("Opening the file at '%v'...\n", filePath)
	buf, err := os.Open(filePath)

	if err != nil {
		fmt.Printf("Unable to open the file at '%v'. %v\n", filePath, err.Error())
		return
	}

	defer func() {
		if err = buf.Close(); err != nil {
			fmt.Println("Error closing the file", err.Error())
		}
	}()

	fmt.Println("Loading up the pattern key...")
	pHash := imgio.HashPatternFile(patternPath)
	fmt.Println("Pattern hash:", pHash)

	fmt.Println("Encoding the file into the image...")
	//Write the file data to the pixels data
	r := bufio.NewReader(buf)
	b := make([]byte, encodeChunkSize)
	channelsPerPix := uint8(4)
	if !encodeAlpha {
		channelsPerPix = 3
	}
	channelCount := int64(len(pixels)) * int64(len(pixels[0])) * int64(channelsPerPix)
	fmt.Println("channelCount:", channelCount)
	f, _ := bitAddresser(pHash, channelCount, maxBitsPerChannel)
	for {
		n, err := r.Read(b)
		if n > 0 {
			//fmt.Println(string(b[0:n]))
			encodeChunk(&f, &pixels, channelsPerPix, b, n)
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
	imgio.WriteImage(&pixels, outPath)

	fmt.Println("All done! c:")
}

func makeRange(max int64) []int64 {
	r := make([]int64, max)
	for i := range r {
		r[i] = int64(i)
	}
	return r
}

type emptyPoolError struct {}

func (e emptyPoolError) Error() string {
	return "The pool of bit addresses is empty."
}

func bitAddresser(seed, channels int64, bitsPerChannel uint8) (func() (int64, error), *[]int64) {
	poolSize := channels * int64(bitsPerChannel)
	pool := makeRange(poolSize)
	rand.Seed(seed)
	fmt.Println("poolSize:", poolSize)
	//An implementation of the Fisher-Yates shuffling algorithm, slightly re-purposed
	return func() (int64, error) {
		if poolSize <= 0 {
			return -1, &emptyPoolError{}
		}

		j := rand.Int63n(poolSize) //I'm aware this isn't crypto/rand, but I needed to be able to seed it

		poolSize--

		p := pool[j]

		pool[j] = pool[poolSize]
		pool = pool[:poolSize]

		return p, nil
	}, &pool
}

func encodeChunk(pos *func() (int64, error), pixels *[][]imgio.Pixel, channelCount uint8, buf []byte, n int) {
	encodePattern(pos, pixels, channelCount, buf, n)
}

func encodePattern(pos *func() (int64, error), pixels *[][]imgio.Pixel, channelCount uint8, buf []byte, n int) {
	w/*, h*/ := len((*pixels)[0])//, len(*pixels)

	//fmt.Println("Image dims:", w, h)

	for i := 0; i < n; i++ {
		//fmt.Printf("(%c) %#08b:\n", buf[i], buf[i])
		for j := uint8(0); j < bitsPerByte; j++ {
			writeBit := imgio.ReadFrom(buf[i], bitsPerByte - j - 1, 1)

			//fmt.Println(imgio.ReadFrom(buf[i], bitsPerByte - j - 1, 1))

			for {
				addr, err := (*pos)()
				if err != nil {
					fmt.Errorf("Something went seriously wrong when fetching the next bit address: %v\n", err.Error())
					panic("Something went seriously wrong when fetching the next bit address.")
				}
				p, c, b := imgio.BitAddrToPCB(addr, channelCount, maxBitsPerChannel)
				x, y := imgio.PosToXY(p, w)
				//fmt.Printf("addr: %d, pixel: (%d: %d, %d), channel: %d, bit: %d, RGBA: %v\n", addr, p, x, y, c, b, (*pixels)[y][x])
				if (*pixels)[y][x].A <= 0 {
					continue
				}

				//fmt.Printf("	Writing %d...\n", writeBit)
				//fmt.Printf("writing (%d, %d: %d, %d)\n", x, y, c, b)

				var channelAddr *uint8
				switch c {
				case 0:
					channelAddr = &(*pixels)[y][x].R
				case 1:
					channelAddr = &(*pixels)[y][x].G
				case 2:
					channelAddr = &(*pixels)[y][x].B
				case 3:
					channelAddr = &(*pixels)[y][x].A
				}

				//fmt.Printf("	Pix before: %#08b - %v\n", *channelAddr, (*pixels)[y][x])

				bitPos := b
				if !encodeLsb {
					bitPos = bitsPerByte - b - 1
				}
				*channelAddr = imgio.WriteTo(*channelAddr, bitPos, 1, writeBit)

				//fmt.Printf("	Pix after:  %#08b - %v\n", *channelAddr, (*pixels)[y][x])

				break
			}
		}
	}
}
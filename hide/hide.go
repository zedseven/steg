package hide

import (
	"bufio"
	"fmt"
	"github.com/zedseven/steg/imgio"
	"os"
)

func Hide(imgPath, filePath string) {
	pixels, err := imgio.LoadImage(imgPath)

	if err != nil {
		fmt.Printf("Unable to load the image at '%v'! %v\n", imgPath, err.Error())
		return
	}

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

	fmt.Println(pixels)

	r := bufio.NewReader(buf)
	b := make([]byte, 32)
	for {
		n, err := r.Read(b)
		if err != nil {
			fmt.Println("Error: ", err.Error())
			break
		}
		fmt.Println(string(b[0:n]))
	}
}

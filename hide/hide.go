package hide

import (
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

	r, err := os.Open(filePath)

	if err != nil {
		fmt.Printf("Unable to open the file at '%v'. %v\n", filePath, err.Error())
		return
	}

	b := make([]byte, 0, 32)
	for {
		n, err := r.Read(b)
		fmt.Println(n, err)
	}
	fmt.Println(pixels)
}


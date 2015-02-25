package main

import (
	"flag"
	"fmt"
)

var IMAGE_PATH = flag.String("image", "", "The path to the image that should be processed.")
var POOL_PATH = flag.String("pool", "", "The path to the directory of images that can be used for tiling.")
var NUM_SPLITS = flag.Int("splits", 10, "The number of images that each dimension should be converted into.")

func main() {
	flag.Parse()
	// Load the image
	img := LoadImage(*IMAGE_PATH)

	// Split the image into `NUM_SPLITS` pieces.
	SplitImage(img, *NUM_SPLITS)
	// Gaze on each one of them.
	//	Gaze(img)

	for i := 0; i < len(img.Gazelings); i++ {
		SaveImage(img.Gazelings[i], fmt.Sprintf("image-%d", i))
	}

	rebuilt := Assemble(img)
	SaveImage(&GazeImage{
		Image: rebuilt,
	}, "full")

	//	SaveImage(img, "test")
}

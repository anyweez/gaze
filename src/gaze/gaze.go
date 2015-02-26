package main

import (
	"flag"
	"fmt"
	"log"
)

var IMAGE_PATH = flag.String("image", "", "The path to the image that should be processed.")
var POOL_PATH = flag.String("pool", "", "The path to the directory of images that can be used for tiling.")
var NUM_SPLITS = flag.Int("splits", 10, "The number of images that each dimension should be converted into.")

func main() {
	flag.Parse()
	// Load the image
	img := LoadImage(*IMAGE_PATH)
	firstX := img.Image.Bounds().Size().X
	firstY := img.Image.Bounds().Size().Y

	// Split the image into `NUM_SPLITS` pieces.
	SplitImage(img, *NUM_SPLITS)
	log.Println(fmt.Sprintf("Working image is `%s`.\n  - Starting dimensions: %dx%d\n  - Trimmed dimensions: %dx%d",
		*IMAGE_PATH,
		firstX, firstY,
		img.Image.Bounds().Size().X, img.Image.Bounds().Size().Y,
	))

	pool := LoadPool(*POOL_PATH, img)
	log.Println(fmt.Sprintf("Loaded %d files into the image pool.", len(pool)))
	if len(pool) == 0 {
		log.Fatal("No images available in the image pool. Cannot continue.")
	}

	// Gaze on each one of them.
	Gaze(img, pool)

	rebuilt := Assemble(img)
	SaveImage(&GazeImage{
		Image: rebuilt,
	}, "full")

	//	SaveImage(img, "test")
}

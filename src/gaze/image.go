package main

import (
	"fmt"
	image "image"
	_ "image/gif"
	_ "image/jpeg"
	png "image/png"
	"log"
	"os"
)

/**
 * A GazeImage is a representation of an image as well as it's constituent
 * 'gazelings', or the subimages that compose the image. The data structure
 * is recursive and can theoretically contain many depths of GazeImages
 * within GazeImages.
 */
type GazeImage struct {
	// The root image for this GazeImage (raw image data)
	Image *image.Image
	Gazed *image.Image
	// A pointer to the parent GazeImage, if any. If no parent then this is null.
	Parent *GazeImage
	// Children GazeImage's that occupy a subset of this image space.
	Gazelings []*GazeImage

	// These coordinates specify the location in the parent image that this
	// GazeImage occupies. `Slices` is the maximum number of slices.
	X      int32
	Y      int32
	Slices int32
}

/**
 * Read in an image from disk. This function returns a GazeImage,
 * which is a standard golang Image wrapped with some additional
 * metadata.
 */
func LoadImage(filename string) *GazeImage {
	reader, err := os.Open(filename)
	defer reader.Close()

	if err != nil {
		log.Fatal(fmt.Sprintf("Can't open image file `%s`; are you sure the file exist?", filename))
	}

	img, _, derr := image.Decode(reader)
	if derr != nil {
		log.Fatal(fmt.Sprintf("Couldn't decode image file `%s`; are you sure it's a valid image file?", filename))
	}

	return &GazeImage{
		Image: &img,
	}
}

/**
 * Split the image into `n` different parts, all as close to
 * equally sized as possible. Each part hsould be
 */
func SplitImage(img *GazeImage, n int) {
	bounds := img.Image.Bounds()
	xDiff := bounds.Max.X - bounds.Min.X
	yDiff := bounds.Max.Y - bounds.Min.Y

	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			// Get the subset of the image.
			img.Gazelings = append(img.Gazelings, &GazeImage{
				Image:  subset,
				Parent: img,
			})
		}
	}

	img.Slices = n
}

/**
 * Convert all of the gazelings for `img` into their nearest images.
 * This function is recursive and will run on all images gazelings that
 * have not yet been gazed upon. When this function completes all gazelings
 * as well as the provided GazeImage should have their .Gazed field set.
 *
 * This function also generates the gazed image of itself, which is stored
 * in the .Gazed field.
 */
func Gaze(img *GazeImage) {
	if len(img.Gazelings) > 0 {
		// Check to make sure that all sub-images have also been gazed. Once
		// all have been gazed then gazing yourself is just a matter of
		// assembling all of the gazed sub-images.
		for i := 0; i < len(img.Gazelings); i++ {
			if img.Gazelings[i].Gazed == nil {
				Gaze(img.Gazelings[i])
			}
		}
	} else {
		// For a leaf image, run the actual gaze algorithm.
		// TODO: write the gaze algorithm :)
	}

	// Combine all gazelings into a single gazed image.
	img.Gazed = Assemble(img)
}

/**
 * Create a new image of the gazed version of the image and
 * return just the image (NOT the GazeImage). In most cases
 * this should simply populate the .Gazed field of the provided
 * GazedImage.
 */
func Assemble(img *GazeImage) *image.Image {
	// Create an image of the same size.
	newImg := image.NewAlpha(img.Image.Bounds())

	xDiff := (newImg.Bounds().Max.X - newImg.Bounds().Min.X) / img.Slices
	yDiff := (newImg.Bounds().Max.Y - newImg.Bounds().Min.Y) / img.Slices

	// Run through each of the subimages and copy over pixels.
	for i := 0; i < img.Slices; i++ {
		sub := img.Gazelings[i]

		for x := 0; x < xDiff; x++ {
			for y := 0; y < yDiff; y++ {
				newImg.SetAlpha(x*sub.X, y*sub.Y, sub.At(x, y))
			}
		}
	}

	return newImg
}

/**
 * Saves the original and gazed image to disk. One file is
 * called orig_{fnSuffix} while the other is called gazed_{fnSuffix}.
 */
func SaveImage(img *GazeImage, fnSuffix string) {
	fp, err := os.Create(fmt.Sprintf("orig_%s.png", fnSuffix))
	png.Encode(fp, img.Image)

	fp, err = os.Create(fmt.Sprintf("gazed_%s.png", fnSuffix))
	png.Encode(fp, img.Image)
}

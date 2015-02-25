package main

import (
	"fmt"
	"github.com/daddye/vips"
	image "image"
	"image/draw"
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
	Image *image.RGBA
	Gazed *image.RGBA
	// A pointer to the parent GazeImage, if any. If no parent then this is null.
	Parent *GazeImage
	// Children GazeImage's that occupy a subset of this image space.
	Gazelings []*GazeImage

	// These coordinates specify the location in the parent image that this
	// GazeImage occupies. `Slices` is the maximum number of slices.
	X      int
	Y      int
	Slices int
}

/**
 * Return a configuration for the given image that should be applied to each gazeling
 * during the splitting process.
 */
func getVipsConfig(img *GazeImage) vips.Options {
	return vips.Options{
		Width:        (img.Image.Bounds().Max.X - img.Image.Bounds().Min.X) / img.Slices,
		Height:       (img.Image.Bounds().Max.Y - img.Image.Bounds().Min.Y) / img.Slices,
		Crop:         false,
		Extend:       vips.EXTEND_WHITE,
		Interpolator: vips.BILINEAR,
		Gravity:      vips.CENTRE,
		Quality:      95,
	}
}

/**
 * Copy an image into a new RGBA object.
 */
func imgToRGBA(img *image.Image) *image.RGBA {
	b := (*img).(interface {
		Bounds() image.Rectangle
	}).Bounds()
	dst := image.NewRGBA(b)

	draw.Draw(dst, b, *img, b.Min, draw.Src)
	return dst
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
		Image: imgToRGBA(&img),
	}
}

/**
 * Split the image into `n` different parts, all as close to
 * equally sized as possible. This adds the data to the object
 * passed in as the first parameter and does not create a copy.
 */
func SplitImage(img *GazeImage, n int) {
	xDiff := (img.Image.Bounds().Max.X - img.Image.Bounds().Min.X) / n
	yDiff := (img.Image.Bounds().Max.Y - img.Image.Bounds().Min.Y) / n

	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			// Create the rectangle that should be extracted.
			region := image.Rect(x*xDiff, y*yDiff, (x+1)*xDiff, (y+1)*yDiff)
			// Get the subimage defined by the region above.
			subset := img.Image.SubImage(region)

			// Build a GazeImage from the subimage and associated metadata.
			img.Gazelings = append(img.Gazelings, &GazeImage{
				Image:  imgToRGBA(&subset),
				Parent: img,
				X:      x,
				Y:      y,
				Slices: n,
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
func Assemble(img *GazeImage) *image.RGBA {
	// Create an image of the same size.
	newImg := image.NewRGBA(img.Image.Bounds())

	xDiff := newImg.Bounds().Size().X / img.Slices
	yDiff := newImg.Bounds().Size().Y / img.Slices

	// Run through each of the subimages and copy over pixels.
	for i := 0; i < (img.Slices * img.Slices); i++ {
		sub := img.Gazelings[i]

		fmt.Println(fmt.Sprintf("Assembling gazeling (%d, %d) at (%d, %d)", sub.X, sub.Y, sub.X*xDiff, sub.Y*yDiff))

		area := image.Rect(sub.X*xDiff, sub.Y*yDiff, (sub.X+1)*xDiff, (sub.Y+1)*yDiff)
		draw.Draw(newImg, area, sub.Image, sub.Image.Bounds().Min, draw.Src)
	}

	return newImg
}

/**
 * Saves the original and gazed image to disk. One file is
 * called orig_{fnSuffix} while the other is called gazed_{fnSuffix}.
 */
func SaveImage(img *GazeImage, fnSuffix string) {
	fp, err := os.Create(fmt.Sprintf("output/orig_%s.png", fnSuffix))
	png.Encode(fp, img.Image)

	if err != nil {
		log.Fatal("Couldn't save image to " + fmt.Sprintf("output/orig_%s.png", fnSuffix))
	}

	if img.Gazed != nil {
		fp, err = os.Create(fmt.Sprintf("output/gazed_%s.png", fnSuffix))
		png.Encode(fp, img.Image)

		if err != nil {
			log.Fatal("Couldn't save image to " + fmt.Sprintf("output/gazed_%s.png", fnSuffix))
		}
	}
}

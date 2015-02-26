package main

import (
	"fmt"
	"math"
	//	"github.com/daddye/vips"
	"github.com/nfnt/resize"
	image "image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	png "image/png"
	"io/ioutil"
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

func Abs(val int) int {
	if val > 0 {
		return val
	} else {
		return val * -1
	}
}

func Round(val float64) float64 {
	return math.Floor(val + .5)
}

/**
 * This function measures the "difference" between two images. It's currently the sum of
 * the pixelwise euclidean distance which is going to be slow and potentially produce
 * mediocre results but it's an easy enough place to start from an implementation perspective.
 */
func ImageDiff(target *GazeImage, comp *GazeImage) int {
	// Ensure that the two images are the same size.
	resizedComp := resize.Thumbnail(
		uint(target.Image.Bounds().Size().X),
		uint(target.Image.Bounds().Size().Y),
		comp.Image,
		resize.NearestNeighbor,
	)

	resizedTarget := resize.Thumbnail(
		uint(resizedComp.Bounds().Size().X),
		uint(resizedComp.Bounds().Size().Y),
		target.Image,
		resize.NearestNeighbor,
	)

	newComp := imgToRGBA(&resizedComp)
	newTarget := imgToRGBA(&resizedTarget)

	distance := 0

	for x := 0; x < newTarget.Bounds().Size().X; x++ {
		for y := 0; y < newTarget.Bounds().Size().Y; y++ {
			r1, b1, g1, _ := newTarget.At(x, y).RGBA()
			r2, b2, g2, _ := newComp.At(x, y).RGBA()

			distance += Abs(int(r1)-int(r2)) + Abs(int(b1)-int(b2)) + Abs(int(g1)-int(g2))
		}
	}

	// The distance is the average Euclient distance per pixel.
	return distance / (newTarget.Bounds().Size().X * newTarget.Bounds().Size().Y)
}

/**
 * Loads all images from the provided directory that match the dimension ratio of the target image.
 */
func LoadPool(directory string, target *GazeImage) []*GazeImage {
	if target.Slices == 0 {
		log.Println("Run SplitImage() before loading the pool to reduce pool storage requirements.")
	}

	files, err := ioutil.ReadDir(directory)
	var pool []*GazeImage

	targetImageRatio := float64(target.Image.Bounds().Size().X) / float64(target.Image.Bounds().Size().Y)
	targetImageRatio = Round(targetImageRatio*10) / 10
	log.Println(fmt.Sprintf("targetImageRatio = %f", targetImageRatio))

	if err != nil {
		log.Fatal("Couldn't read provided pool directory.")
	}

	// Read in each file in the directory that can be parsed as an image file.
	for i, file := range files {
		if !file.IsDir() {
			gaze := LoadImage(fmt.Sprintf("%s/%s", directory, file.Name()))

			// Trim the image before checking the image ratio.
			TrimImage(gaze, target.Slices)

			newImageRatio := float64(gaze.Image.Bounds().Size().X) / float64(gaze.Image.Bounds().Size().Y)
			newImageRatio = Round(newImageRatio*10) / 10

			// If the ratios match, resize the image to be the max size of a gazeling and store it in the pool.
			if newImageRatio == targetImageRatio {
				if target.Slices > 0 {
					thumb := resize.Thumbnail(
						uint(target.Image.Bounds().Size().X/target.Slices),
						uint(target.Image.Bounds().Size().Y/target.Slices),
						gaze.Image,
						resize.NearestNeighbor,
					)
					gaze.Image = imgToRGBA(&thumb)
				}

				pool = append(pool, gaze)
				// TODO: temporary; remove this
				SaveImage(gaze, fmt.Sprintf("pool-%d", i))
			} else {
				log.Println(fmt.Sprintf("Skipping image `%s` with image ratio of %f", file.Name(), newImageRatio))
			}
		}
	}

	return pool
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
 * Gets rid of extra pixels that will lead to rounding error if gone unchecked.
 */
func TrimImage(img *GazeImage, n int) {
	// Resize the image to get rid of rounding error.
	sub := img.Image.SubImage(image.Rect(
		0,
		0,
		img.Image.Bounds().Size().X-(img.Image.Bounds().Size().X%n),
		img.Image.Bounds().Size().Y-(img.Image.Bounds().Size().Y%n),
	))
	img.Image = imgToRGBA(&sub)
}

/**
 * Split the image into `n` different parts, all as close to
 * equally sized as possible. This adds the data to the object
 * passed in as the first parameter and does not create a copy.
 */
func SplitImage(img *GazeImage, n int) {
	TrimImage(img, n)

	xDiff := img.Image.Bounds().Size().X / n
	yDiff := img.Image.Bounds().Size().Y / n

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
func Gaze(img *GazeImage, pool []*GazeImage) {
	// Recursive case checks that all gazelings have been gazed.
	if len(img.Gazelings) > 0 {
		// Check to make sure that all sub-images have also been gazed. Once
		// all have been gazed then gazing yourself is just a matter of
		// assembling all of the gazed sub-images.
		for i := 0; i < len(img.Gazelings); i++ {
			if img.Gazelings[i].Gazed == nil {
				Gaze(img.Gazelings[i], pool)
			}
		}
	} else {
		// For a leaf image, run the actual gaze algorithm.
		bestScore := 0
		bestIndex := -1

		// For each image in the pool, check to see how different it is
		// from the source image and find the one that is the most similar.
		for i := 0; i < len(pool); i++ {
			score := ImageDiff(img, pool[i])

			if score < bestScore || bestIndex < 0 {
				bestScore = score
				bestIndex = i
			}
		}

		// Resize the image in the pool to the appropriate size for this GazeImage.
		resized := resize.Thumbnail(
			uint(img.Image.Bounds().Size().X),
			uint(img.Image.Bounds().Size().Y),
			pool[bestIndex].Image,
			resize.NearestNeighbor,
		)

		img.Gazed = imgToRGBA(&resized)
	}
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
	for x := 0; x < img.Slices; x++ {
		for y := 0; y < img.Slices; y++ {
			sub := img.Gazelings[y*img.Slices+x]

			// fmt.Println(fmt.Sprintf("(%d, %d) to (%d, %d)", x*xDiff, y*yDiff, (x+1)*xDiff, (y+1)*yDiff))
			// fmt.Println(fmt.Sprintf("Dimensions of gazeling: %d,%d", sub.Image.Bounds().Size().X, sub.Image.Bounds().Size().Y))
			area := image.Rect(x*xDiff, y*yDiff, (x+1)*xDiff, (y+1)*yDiff)
			if sub.Gazed != nil {
				draw.Draw(newImg, area, sub.Gazed, sub.Gazed.Bounds().Min, draw.Src)
			} else {
				draw.Draw(newImg, area, sub.Image, sub.Image.Bounds().Min, draw.Src)
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
	fp, err := os.Create(fmt.Sprintf("output/%s_orig.png", fnSuffix))
	png.Encode(fp, img.Image)

	if err != nil {
		log.Fatal("Couldn't save image to " + fmt.Sprintf("output/%s_orig.png", fnSuffix))
	}

	if img.Gazed != nil {
		fp, err = os.Create(fmt.Sprintf("output/%s_gazed.png", fnSuffix))
		png.Encode(fp, img.Gazed)

		if err != nil {
			log.Fatal("Couldn't save image to " + fmt.Sprintf("output/%s_gazed.png", fnSuffix))
		}
	}
}

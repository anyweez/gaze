package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const API_URL = "https://api.flickr.com/services/rest/?method=flickr.photosets.getPhotos&api_key=%s&photoset_id=%d&extras=url_l,license&media=photos&format=json&nojsoncallback=1"
const API_KEY = ""

type FlickrAPIResponse struct {
	Photoset FlickrAPIPhotoset
}

type FlickrAPIPhotoset struct {
	Photo []FlickrAPIPhoto
}

type FlickrAPIPhoto struct {
	URL    string `json:"url_l"`
	Id     string
	Secret string
	Server string
}

func (p *FlickrAPIPhoto) Filename() string {
	parts := strings.Split(p.URL, ".")
	ext := parts[len(parts)-1]

	return fmt.Sprintf("%s-%s-%s.%s", p.Id, p.Secret, p.Server, ext)
}

func loadPhotosets(filename string) []uint64 {
	var ids []uint64

	fp, err := os.Open(filename)

	if err != nil {
		log.Fatal("Photoset file doesn't exist or can't be opened: " + err.Error())
	}

	data, err := ioutil.ReadAll(fp)
	for _, val := range strings.Split(string(data), "\n") {
		psid, perr := strconv.ParseInt(val, 10, 64)

		if perr == nil {
			ids = append(ids, uint64(psid))
		} else {
			log.Println("Warning: invalid data in photoset file: " + perr.Error())

		}

	}

	return ids
}

func main() {
	ps := loadPhotosets("photosets")

	// Run through each photoset and grab the URL for each image in the
	// photoset.
	for _, id := range ps {
		log.Println(fmt.Sprintf("Fetching photoset %d", id))
		resp, err := http.Get(fmt.Sprintf(API_URL, API_KEY, id))

		fmt.Println(fmt.Sprintf(API_URL, API_KEY, id))

		if err != nil {
			log.Fatal("Couldn't fetch photoset: " + err.Error())
		}

		body, berr := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if berr != nil {
			log.Fatal("Couldn't read response:" + err.Error())
		}

		flresp := FlickrAPIResponse{}
		json.Unmarshal(body, &flresp)

		log.Println(len(flresp.Photoset.Photo))

		// Download and store each image.
		for _, photo := range flresp.Photoset.Photo {
			// In certain cases the URL field is empty. If that's the case, skip it.
			if len(photo.URL) > 0 {
				fmt.Println(fmt.Sprintf("  Downloading %s...", photo.URL))
				pResp, perr := http.Get(photo.URL)

				if perr != nil {
					log.Fatal("Couldn't read image from URL: " + photo.URL)
				}

				outfile, oerr := os.Create("pool/" + photo.Filename())
				log.Println("Saving file to pool/" + photo.Filename())

				if oerr != nil {
					log.Fatal("Couldn't open file for writing:" + oerr.Error())
				}

				// Copy the actual image data.
				io.Copy(outfile, pResp.Body)

				// Cleanup.
				outfile.Close()
				pResp.Body.Close()
			}
		}
	}
}

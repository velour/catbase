package cowboy

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path"
	"sync"

	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/plugins/emojy"
)

var (
	cowboyCache = map[string]image.Image{}
	cowboyMutex = sync.Mutex{}
)

func getEmojy(emojyPath, baseEmojyURL, name string) (image.Image, error) {
	files, _, err := emojy.AllFiles(emojyPath, baseEmojyURL)
	if err != nil {
		return nil, err
	}
	fname, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("could not find emojy: %s", name)
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func getCowboyHat(emojyPath, overlay string) (image.Image, error) {
	emojies, _, err := emojy.AllFiles(emojyPath, "")
	if err != nil {
		return nil, err
	}
	overlay, ok := emojies[overlay]
	if !ok {
		return nil, fmt.Errorf("could not find overlay %s", overlay)
	}
	overlay = path.Clean(overlay)
	f, err := os.Open(overlay)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func cowboyifyImage(emojyPath, overlay string, input image.Image) (image.Image, error) {
	hat, err := getCowboyHat(emojyPath, overlay)
	if err != nil {
		return nil, err
	}
	inputW, inputH := float64(input.Bounds().Max.X), float64(input.Bounds().Max.Y)
	hatW, hatH := float64(hat.Bounds().Max.X), float64(hat.Bounds().Max.Y)
	targetSZ := math.Max(inputW, inputH)
	dst := image.NewRGBA(image.Rect(0, 0, int(targetSZ), int(targetSZ)))
	newH := uint(targetSZ / hatW * hatH)
	hat = resize.Resize(uint(targetSZ), newH, hat, resize.Lanczos3)
	draw.Draw(dst, input.Bounds(), input, image.Point{}, draw.Src)
	draw.Draw(dst, hat.Bounds(), hat, image.Point{}, draw.Over)
	return dst, nil
}

func cowboy(emojyPath, baseEmojyURL, overlay, name string) (image.Image, error) {
	cowboyMutex.Lock()
	defer cowboyMutex.Unlock()
	if img, ok := cowboyCache[name]; ok {
		log.Debug().Msgf(":cowboy_using_cached_image: %s", name)
		return img, nil
	}
	log.Debug().Msgf(":cowboy_generating_image: %s with overlay %s", name, overlay)
	emjy, err := getEmojy(emojyPath, baseEmojyURL, name)
	if err != nil {
		return nil, err
	}
	img, err := cowboyifyImage(emojyPath, overlay, emjy)
	if err != nil {
		return nil, err
	}
	cowboyCache[name] = img
	return img, nil
}

func encode(img image.Image, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	w := bytes.NewBuffer([]byte{})
	err = png.Encode(w, img)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func cowboyClearCache() {
	cowboyMutex.Lock()
	defer cowboyMutex.Unlock()
	cowboyCache = map[string]image.Image{}
}

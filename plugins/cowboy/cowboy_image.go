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
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/plugins/emojy"
)

var (
	cowboyCache = map[string][]byte{}
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

func getCowboyHat(c *config.Config, emojyPath string) (image.Image, error) {
	p := path.Join(emojyPath, c.Get("cowboy.hatname", "hat.png"))
	p = path.Clean(p)
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func cowboyifyImage(c *config.Config, emojyPath string, input image.Image) (image.Image, error) {
	hat, err := getCowboyHat(c, emojyPath)
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

func cowboy(c *config.Config, emojyPath, baseEmojyURL, name string) ([]byte, error) {
	cowboyMutex.Lock()
	defer cowboyMutex.Unlock()
	if img, ok := cowboyCache[name]; ok {
		log.Debug().Msgf(":cowboy_using_cached_image: %s", name)
		return img, nil
	}
	log.Debug().Msgf(":cowboy_generating_image: %s", name)
	emjy, err := getEmojy(emojyPath, baseEmojyURL, name)
	if err != nil {
		return nil, err
	}
	img, err := cowboyifyImage(c, emojyPath, emjy)
	if err != nil {
		return nil, err
	}
	w := bytes.NewBuffer([]byte{})
	err = png.Encode(w, img)
	if err != nil {
		return nil, err
	}
	cowboyCache[name] = w.Bytes()
	return w.Bytes(), nil
}

func cowboyClearCache() {
	cowboyMutex.Lock()
	defer cowboyMutex.Unlock()
	cowboyCache = map[string][]byte{}
}

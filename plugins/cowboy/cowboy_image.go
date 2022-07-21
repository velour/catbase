package cowboy

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path"

	"github.com/nfnt/resize"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/plugins/emojy"
)

func getEmojy(c *config.Config, name string) (image.Image, error) {
	files, _, err := emojy.AllFiles(c)
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

func getCowboyHat(c *config.Config) (image.Image, error) {
	emojyPath := c.Get("emojy.path", "emojy")
	p := path.Join(emojyPath, c.Get("cowboy.hatname", "hat.png"))
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

func cowboyifyImage(c *config.Config, input image.Image) (image.Image, error) {
	hat, err := getCowboyHat(c)
	if err != nil {
		return nil, err
	}
	targetW := uint(c.GetInt("cowboy.targetw", 64))
	hatW, hatH := float64(hat.Bounds().Max.X), float64(hat.Bounds().Max.Y)
	newH := uint(float64(targetW) / hatW * hatH)
	hat = resize.Resize(targetW, newH, hat, resize.MitchellNetravali)
	input = resize.Resize(targetW, targetW, input, resize.MitchellNetravali)
	dst := image.NewRGBA(image.Rect(0, 0, 64, 64))
	draw.Draw(dst, input.Bounds(), input, image.Point{}, draw.Src)
	draw.Draw(dst, hat.Bounds(), hat, image.Point{}, draw.Over)
	return dst, nil
}

func cowboy(c *config.Config, name string) ([]byte, error) {
	emojy, err := getEmojy(c, name)
	if err != nil {
		return nil, err
	}
	img, err := cowboyifyImage(c, emojy)
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

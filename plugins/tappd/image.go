package tappd

import (
	"bytes"
	"github.com/disintegration/imageorient"
	"github.com/fogleman/gg"
	"github.com/gabriel-vasile/mimetype"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/plugins/meme"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"path"
)

func (p *Tappd) getImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	img, _, err := imageorient.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	r := img.Bounds()
	w := r.Dx()
	h := r.Dy()

	maxSz := p.c.GetFloat64("maxImgSz", 750.0)

	if w > h {
		scale := maxSz / float64(w)
		w = int(float64(w) * scale)
		h = int(float64(h) * scale)
	} else {
		scale := maxSz / float64(h)
		w = int(float64(w) * scale)
		h = int(float64(h) * scale)
	}

	log.Debug().Msgf("trynig to resize to %v, %v", w, h)
	img = resize.Resize(uint(w), uint(h), img, resize.Lanczos3)
	r = img.Bounds()
	w = r.Dx()
	h = r.Dy()
	log.Debug().Msgf("resized to %v, %v", w, h)

	return img, nil
}

type textSpec struct {
	text string
	// percentage location of text center
	x float64
	y float64
}

func defaultSpec() textSpec {
	return textSpec{
		x: 0.5,
		y: 0.9,
	}
}

func (p *Tappd) overlay(img image.Image, texts []textSpec) ([]byte, error) {
	font := p.c.Get("meme.font", "impact.ttf")
	fontSizes := []float64{48, 36, 24, 16, 12}
	r := img.Bounds()
	w := r.Dx()
	h := r.Dy()

	txts := []string{}
	for _, t := range texts {
		txts = append(txts, t.text)
	}

	fontSize := meme.FindFontSize(p.c, txts, font, w, h, fontSizes)

	m := gg.NewContext(w, h)
	m.DrawImage(img, 0, 0)
	for _, spec := range texts {
		// write some stuff on the image here
		if err := m.LoadFontFace(font, fontSize); err != nil {
			return nil, err
		}

		// Apply black stroke
		m.SetHexColor("#000")
		strokeSize := 6
		for dy := -strokeSize; dy <= strokeSize; dy++ {
			for dx := -strokeSize; dx <= strokeSize; dx++ {
				// give it rounded corners
				if dx*dx+dy*dy >= strokeSize*strokeSize {
					continue
				}
				x := float64(w)*spec.x + float64(dx)
				y := float64(h)*spec.y + float64(dy)
				m.DrawStringAnchored(spec.text, x, y, 0.5, 0.5)
			}
		}

		m.SetHexColor("#FFF")
		x := float64(w) * spec.x
		y := float64(h) * spec.y
		m.DrawStringAnchored(spec.text, x, y, 0.5, 0.5)
	}
	i := bytes.Buffer{}
	if err := png.Encode(&i, m.Image()); err != nil {
		return nil, err
	}
	return i.Bytes(), nil
}

func (p *Tappd) getAndOverlay(id, srcURL string, texts []textSpec) (imageInfo, error) {
	baseURL := p.c.Get("BaseURL", ``)
	u, _ := url.Parse(baseURL)
	u.Path = path.Join(u.Path, "tappd", id)
	img, err := p.getImage(srcURL)
	if err != nil {
		return imageInfo{}, err
	}
	data, err := p.overlay(img, texts)
	if err != nil {
		return imageInfo{}, err
	}
	bounds := img.Bounds()
	mime := mimetype.Detect(data)
	info := imageInfo{
		ID:       id,
		SrcURL:   srcURL,
		BotURL:   u.String(),
		Img:      img,
		Repr:     data,
		W:        bounds.Dx(),
		H:        bounds.Dy(),
		MimeType: mime.String(),
		FileName: id + mime.Extension(),
	}
	p.imageMap[id] = info
	log.Debug().
		Interface("BotURL", info.BotURL).
		Str("ID", id).
		Msgf("here's some info")
	return info, nil
}

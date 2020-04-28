package meme

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/fogleman/gg"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type MemePlugin struct {
	bot bot.Bot
	c   *config.Config

	images map[string][]byte
}

func New(b bot.Bot) *MemePlugin {
	mp := &MemePlugin{
		bot:    b,
		c:      b.Config(),
		images: make(map[string][]byte),
	}

	b.Register(mp, bot.Message, mp.message)
	b.Register(mp, bot.Help, mp.help)
	mp.registerWeb(b.DefaultConnector())

	return mp
}

func (p *MemePlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	return false
}

func (p *MemePlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	formats := p.c.GetMap("meme.memes", defaultFormats)
	msg := "Use `/meme [format] [text]` to create a meme.\nI know the following formats:"
	for k := range formats {
		msg += "\n" + k
	}
	p.bot.Send(c, bot.Message, message.Channel, msg)
	return true
}

func (p *MemePlugin) registerWeb(c bot.Connector) {
	http.HandleFunc("/slash/meme", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		log.Debug().Msgf("Meme:\n%+v", r.PostForm.Get("text"))
		channel := r.PostForm.Get("channel_id")
		user := r.PostForm.Get("user_name")
		log.Debug().Msgf("channel: %s", channel)

		parts := strings.SplitN(r.PostForm.Get("text"), " ", 2)
		log.Debug().Strs("parts", parts).Msgf("Meme:\n%+v", r.PostForm.Get("text"))
		w.WriteHeader(200)

		id := p.genMeme(parts[0], parts[1])
		baseURL := p.c.Get("BaseURL", `https://catbase.velour.ninja`)
		u, _ := url.Parse(baseURL)
		u.Path = path.Join(u.Path, "meme", "img", id)

		log.Debug().Msgf("image is at %s", u.String())
		p.bot.Send(c, bot.Message, channel, "", bot.ImageAttachment{
			URL:    u.String(),
			AltTxt: user,
		})
		w.Write(nil)
	})

	http.HandleFunc("/meme/img/", func(w http.ResponseWriter, r *http.Request) {
		_, file := path.Split(r.URL.Path)
		id := file
		img := p.images[id]
		w.Write(img)
	})
}

func DownloadTemplate(u *url.URL) image.Image {
	res, err := http.Get(u.String())
	if err != nil {
		log.Error().Msgf("template from %s failed because of %v", u.String(), err)
	}
	defer res.Body.Close()
	image, _, err := image.Decode(res.Body)
	if err != nil {
		log.Error().Msgf("Could not decode %v because of %v", u, err)
	}
	return image
}

var defaultFormats = map[string]string{
	"fry":    "Futurama-Fry.jpg",
	"aliens": "Ancient-Aliens.jpg",
	"doge":   "Doge.jpg",
	"simply": "One-Does-Not-Simply.jpg",
	"wonka":  "Creepy-Condescending-Wonka.jpg",
	"grumpy": "Grumpy-Cat.jpg",
	"raptor": "Philosoraptor.jpg",
}

func (p *MemePlugin) genMeme(meme, text string) string {
	const fontSize = 36

	formats := p.c.GetMap("meme.memes", defaultFormats)

	path := uuid.New().String()

	imgName, ok := formats[meme]
	if !ok {
		imgName = meme
	}

	u, err := url.Parse(imgName)
	if err != nil {
		u, _ = url.Parse("https://imgflip.com/s/meme/" + imgName)
	}

	img := DownloadTemplate(u)
	r := img.Bounds()
	w := r.Dx()
	h := r.Dy()

	m := gg.NewContext(w, h)
	m.DrawImage(img, 0, 0)
	fontLocation := p.c.Get("meme.font", "impact.ttf")
	err = m.LoadFontFace(fontLocation, fontSize) // problem
	if err != nil {
		log.Error().Err(err).Msg("could not load font")
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
			x := float64(w/2 + dx)
			y := float64(h - fontSize + dy)
			m.DrawStringAnchored(text, x, y, 0.5, 0.5)
		}
	}

	// Apply white fill
	m.SetHexColor("#FFF")
	m.DrawStringAnchored(text, float64(w)/2, float64(h)-fontSize, 0.5, 0.5)

	i := bytes.Buffer{}
	png.Encode(&i, m.Image())
	p.images[path] = i.Bytes()

	log.Debug().Msgf("Saved to %s\n", path)

	return path
}

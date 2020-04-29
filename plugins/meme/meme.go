package meme

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

type MemePlugin struct {
	bot bot.Bot
	c   *config.Config

	images cachedImages
}

type cachedImage struct {
	created time.Time
	repr    []byte
}

var horizon = 24 * 7

type cachedImages map[string]*cachedImage

func (ci cachedImages) cleanup() {
	for key, img := range ci {
		if time.Now().After(img.created.Add(time.Hour * time.Duration(horizon))) {
			delete(ci, key)
		}
	}
}

func New(b bot.Bot) *MemePlugin {
	mp := &MemePlugin{
		bot:    b,
		c:      b.Config(),
		images: make(cachedImages),
	}

	horizon = mp.c.GetInt("meme.horizon", horizon)

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
		channelName := r.PostForm.Get("channel_name")
		from := r.PostForm.Get("user_name")
		log.Debug().Msgf("channel: %s", channel)

		parts := strings.SplitN(r.PostForm.Get("text"), " ", 2)
		isCmd, message := bot.IsCmd(p.c, parts[1])
		format := parts[0]

		log.Debug().Strs("parts", parts).Msgf("Meme:\n%+v", r.PostForm.Get("text"))
		w.WriteHeader(200)

		top, bottom := "", message
		parts = strings.Split(message, "\n")
		if len(parts) > 1 {
			top, bottom = parts[0], parts[1]
		}

		id := p.genMeme(format, top, bottom)
		baseURL := p.c.Get("BaseURL", `https://catbase.velour.ninja`)
		u, _ := url.Parse(baseURL)
		u.Path = path.Join(u.Path, "meme", "img", id)

		log.Debug().Msgf("image is at %s", u.String())
		p.bot.Send(c, bot.Message, channel, "", bot.ImageAttachment{
			URL:    u.String(),
			AltTxt: fmt.Sprintf("%s: %s", from, message),
		})
		w.Write(nil)
		m := msg.Message{
			User: &user.User{
				ID:    from,
				Name:  from,
				Admin: false,
			},
			Channel:     channel,
			ChannelName: channelName,
			Body:        message,
			Command:     isCmd,
			Time:        time.Now(),
		}

		p.bot.Receive(c, bot.Message, m)
	})

	http.HandleFunc("/meme/img/", func(w http.ResponseWriter, r *http.Request) {
		_, file := path.Split(r.URL.Path)
		id := file
		if img, ok := p.images[id]; ok {
			w.Write(img.repr)
		} else {
			w.WriteHeader(404)
			w.Write([]byte("not found"))
		}
		p.images.cleanup()
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

func (p *MemePlugin) genMeme(meme, top, bottom string) string {
	fontSizes := []float64{96, 48, 24, 12}
	fontSize := fontSizes[0]

	formats := p.c.GetMap("meme.memes", defaultFormats)

	path := uuid.New().String()

	imgName, ok := formats[meme]
	if !ok {
		imgName = meme
	}

	u, err := url.Parse(imgName)
	if err != nil || u.Scheme == "" {
		log.Debug().Err(err).Str("imgName", imgName).Msgf("url not detected")
		u, _ = url.Parse("https://imgflip.com/s/meme/" + imgName)
	}

	log.Debug().Msgf("Attempting to download url: %s", u.String())

	img := DownloadTemplate(u)
	r := img.Bounds()
	w := r.Dx()
	h := r.Dy()

	m := gg.NewContext(w, h)
	m.DrawImage(img, 0, 0)
	fontLocation := p.c.Get("meme.font", "impact.ttf")
	for _, sz := range fontSizes {
		err = m.LoadFontFace(fontLocation, sz) // problem
		if err != nil {
			log.Error().Err(err).Msg("could not load font")
		}
		topW, _ := m.MeasureString(top)
		botW, _ := m.MeasureString(bottom)
		if topW < float64(w) && botW < float64(w) {
			fontSize = sz
			break
		}
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
			y := float64(h) - fontSize + float64(dy)
			m.DrawStringAnchored(top, x, fontSize, 0.5, 0.5)
			m.DrawStringAnchored(bottom, x, y, 0.5, 0.5)
		}
	}

	// Apply white fill
	m.SetHexColor("#FFF")
	m.DrawStringAnchored(top, float64(w)/2, fontSize, 0.5, 0.5)
	m.DrawStringAnchored(bottom, float64(w)/2, float64(h)-fontSize, 0.5, 0.5)

	i := bytes.Buffer{}
	png.Encode(&i, m.Image())
	p.images[path] = &cachedImage{time.Now(), i.Bytes()}

	log.Debug().Msgf("Saved to %s\n", path)

	return path
}

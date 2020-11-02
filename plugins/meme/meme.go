package meme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
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

type memeText struct {
	XPerc float64 `json:"x"`
	YPerc float64 `json:"y"`
	Text  string  `json:"t"`
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

var cmdMatch = regexp.MustCompile(`(?i)meme (.+)`)

func (p *MemePlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if message.Command && cmdMatch.MatchString(message.Body) {
		subs := cmdMatch.FindStringSubmatch(message.Body)
		if len(subs) != 2 {
			p.bot.Send(c, bot.Message, message.Channel, "Invalid meme request.")
			return true
		}
		minusMeme := subs[1]
		p.sendMeme(c, message.Channel, message.ChannelName, message.ID, message.User, minusMeme)
		return true
	}
	return false
}

func (p *MemePlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	webRoot := p.c.Get("BaseURL", "https://catbase.velour.ninja")
	formats := p.c.GetMap("meme.memes", defaultFormats)
	msg := "Use `/meme [format] [text]` to create a meme.\nI know the following formats:"
	msg += "\n`[format]` can be a URL"
	msg += fmt.Sprintf("\nor a format from the list of %d pre-made memes listed on the website", len(formats))
	msg += fmt.Sprintf("\nHead over to %s/meme to view and add new meme formats", webRoot)
	msg += "\nYou can use `_` as a placeholder for empty text and a newline (|| on Discord) to separate top vs bottom."
	msg += "\nOR you can configure the text yourself with JSON. Send a code quoted message like:"
	msg += "```/meme doge `[{\"x\": 0.2, \"y\": 0.1, \"t\": \"such image\"},{\"x\": 0.7, \"y\": 0.5, \"t\": \"much meme\"},{\"x\": 0.4, \"y\": 0.8, \"t\": \"wow\"}]` ```"
	p.bot.Send(c, bot.Message, message.Channel, msg)
	return true
}

func (p *MemePlugin) bully(c bot.Connector, format, id string) (image.Image, string) {
	bullyIcon := ""

	for _, bully := range p.c.GetArray("meme.bully", []string{}) {
		if format == bully {
			if u, err := c.Profile(bully); err == nil {
				bullyIcon = u.Icon
			} else {
				log.Debug().Err(err).Msgf("could not get profile for %s", format)
			}
			formats := p.c.GetMap("meme.memes", defaultFormats)
			format = randEntry(formats)
			log.Debug().Msgf("After bullying:\nFormat: %s", format)
			break
		}
	}

	if u, err := c.Profile(id); bullyIcon == "" && err == nil {
		if u.IconImg != nil {
			return u.IconImg, format
		}
		bullyIcon = u.Icon
	}

	u, err := url.Parse(bullyIcon)
	if err != nil {
		log.Error().Err(err).Msg("error with bully URL")
	}
	bullyImg, err := DownloadTemplate(u)
	if err != nil {
		log.Error().Err(err).Msg("error downloading bully icon")
	}
	return bullyImg, format
}

func (p *MemePlugin) sendMeme(c bot.Connector, channel, channelName, msgID string, from *user.User, text string) {
	parts := strings.SplitN(text, " ", 2)
	if len(parts) != 2 {
		log.Debug().Msgf("Bad meme request: %v, %v", from, text)
		p.bot.Send(c, bot.Message, channel, fmt.Sprintf("%v tried to send me a bad meme request.", from.Name))
		return
	}
	isCmd, message := bot.IsCmd(p.c, parts[1])
	format := parts[0]

	log.Debug().Strs("parts", parts).Msgf("Meme:\n%+v", text)

	go func() {
		var config []memeText

		message = strings.TrimPrefix(message, "`")
		message = strings.TrimSuffix(message, "`")

		err := json.Unmarshal([]byte(message), &config)
		if err == nil {
			message = ""
			for _, c := range config {
				message += c.Text + "\n"
			}
		} else {
			if strings.Contains(message, "||") {
				parts = strings.Split(message, "||")
			} else {
				parts = strings.Split(message, "\n")
			}

			allConfigs := p.c.GetMap("meme.memeconfigs", map[string]string{})
			configtxt, ok := allConfigs[format]
			if !ok {
				config = defaultFormatConfig()
				log.Debug().Msgf("Did not find %s in %+v", format, allConfigs)
			} else {
				err = json.Unmarshal([]byte(configtxt), &config)
				if err != nil {
					log.Error().Err(err).Msgf("Could not parse config for %s:\n%s", format, configtxt)
					config = defaultFormatConfig()
				}
			}

			j := 0
			for i := range config {
				if len(parts) > i {
					if parts[j] != "_" {
						config[i].Text = parts[j]
					}
					j++
				}
			}
		}

		bullyImg, format := p.bully(c, format, from.ID)

		id, w, h, err := p.genMeme(format, bullyImg, config)
		if err != nil {
			msg := fmt.Sprintf("Hey %v, I couldn't download that image you asked for.", from.Name)
			p.bot.Send(c, bot.Ephemeral, channel, from.ID, msg)
			return
		}
		baseURL := p.c.Get("BaseURL", ``)
		u, _ := url.Parse(baseURL)
		u.Path = path.Join(u.Path, "meme", "img", id)

		log.Debug().Msgf("image is at %s", u.String())
		_, err = p.bot.Send(c, bot.Message, channel, "", bot.ImageAttachment{
			URL:    u.String(),
			AltTxt: fmt.Sprintf("%s: %s", from.Name, message),
			Width:  w,
			Height: h,
		})

		if err == nil && msgID != "" {
			p.bot.Send(c, bot.Delete, channel, msgID)
		}

		m := msg.Message{
			User: &user.User{
				ID:    from.ID,
				Name:  from.Name,
				Admin: false,
			},
			Channel:     channel,
			ChannelName: channelName,
			Body:        message,
			Command:     isCmd,
			Time:        time.Now(),
		}

		p.bot.Receive(c, bot.Message, m)
	}()

}

func (p *MemePlugin) slashMeme(c bot.Connector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		log.Debug().Msgf("Meme:\n%+v", r.PostForm.Get("text"))
		channel := r.PostForm.Get("channel_id")
		channelName := r.PostForm.Get("channel_name")
		userID := r.PostForm.Get("user_id")
		from := r.PostForm.Get("user_name")
		text := r.PostForm.Get("text")
		log.Debug().Msgf("channel: %s", channel)

		user := &user.User{
			ID:   userID, // HACK but should work fine
			Name: from,
		}

		p.sendMeme(c, channel, channelName, "", user, text)

		w.WriteHeader(200)
		w.Write(nil)
	}
}

func randEntry(m map[string]string) string {
	for _, e := range m {
		return e
	}
	return ""
}

func DownloadTemplate(u *url.URL) (image.Image, error) {
	res, err := http.Get(u.String())
	if err != nil {
		log.Error().Msgf("template from %s failed because of %v", u.String(), err)
		return nil, err
	}
	defer res.Body.Close()
	image, _, err := image.Decode(res.Body)
	if err != nil {
		log.Error().Msgf("Could not decode %v because of %v", u, err)
		return nil, err
	}
	return image, nil
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

func (p *MemePlugin) findFontSize(config []memeText, w, h int, sizes []float64) float64 {
	fontSize := 12.0

	m := gg.NewContext(w, h)
	fontLocation := p.c.Get("meme.font", "impact.ttf")

	longestStr, longestW := "", 0.0

	for _, s := range config {
		err := m.LoadFontFace(fontLocation, 12) // problem
		if err != nil {
			log.Error().Err(err).Msg("could not load font")
			return fontSize
		}

		w, _ := m.MeasureString(s.Text)
		if w > longestW {
			longestStr = s.Text
			longestW = w
		}
	}

	for _, sz := range sizes {
		err := m.LoadFontFace(fontLocation, sz) // problem
		if err != nil {
			log.Error().Err(err).Msg("could not load font")
			return fontSize
		}

		topW, _ := m.MeasureString(longestStr)
		if topW < float64(w) {
			fontSize = sz
			break
		}
	}
	return fontSize
}

func defaultFormatConfig() []memeText {
	return []memeText{
		{XPerc: 0.5, YPerc: 0.05},
		{XPerc: 0.5, YPerc: 0.95},
	}
}

func (p *MemePlugin) genMeme(meme string, bully image.Image, config []memeText) (string, int, int, error) {
	fontSizes := []float64{48, 36, 24, 16, 12}
	formats := p.c.GetMap("meme.memes", defaultFormats)

	path := uuid.New().String()

	imgName, ok := formats[meme]
	if !ok {
		imgName = meme
	}

	u, err := url.Parse(imgName)
	if err != nil || u.Scheme == "" {
		log.Debug().Err(err).Str("imgName", imgName).Msgf("url not detected")
		if u, err = url.Parse("https://imgflip.com/s/meme/" + imgName); err != nil {
			return "", 0, 0, err
		}
	}

	log.Debug().Msgf("Attempting to download url: %s", u.String())

	img, err := DownloadTemplate(u)
	if err != nil {
		log.Debug().Msgf("failed to download image: %s", err)
		return "", 0, 0, err
	}

	r := img.Bounds()
	w := r.Dx()
	h := r.Dy()

	// /meme2 5guys [{"x": 0.1, "y": 0.1, "t": "test"}]

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

	if bully != nil {
		img = p.applyBully(img, bully)
	}

	m := gg.NewContext(w, h)
	m.DrawImage(img, 0, 0)
	fontLocation := p.c.Get("meme.font", "impact.ttf")
	m.LoadFontFace(fontLocation, p.findFontSize(config, w, h, fontSizes))

	// Apply black stroke
	m.SetHexColor("#000")
	strokeSize := 6
	for dy := -strokeSize; dy <= strokeSize; dy++ {
		for dx := -strokeSize; dx <= strokeSize; dx++ {
			// give it rounded corners
			if dx*dx+dy*dy >= strokeSize*strokeSize {
				continue
			}
			for _, c := range config {
				x := float64(w)*c.XPerc + float64(dx)
				y := float64(h)*c.YPerc + float64(dy)
				m.DrawStringAnchored(c.Text, x, y, 0.5, 0.5)
			}
		}
	}

	// Apply white fill
	m.SetHexColor("#FFF")
	for _, c := range config {
		x := float64(w) * c.XPerc
		y := float64(h) * c.YPerc
		m.DrawStringAnchored(c.Text, x, y, 0.5, 0.5)
	}

	i := bytes.Buffer{}
	png.Encode(&i, m.Image())
	p.images[path] = &cachedImage{time.Now(), i.Bytes()}

	log.Debug().Msgf("Saved to %s\n", path)

	return path, w, h, nil
}

func (p *MemePlugin) applyBully(img, bullyImg image.Image) image.Image {
	dst := image.NewRGBA(img.Bounds())

	scaleFactor := p.c.GetFloat64("meme.bullyScale", 0.1)
	position := p.c.GetString("meme.bullyPosition", "botright")

	scaleFactor = float64(img.Bounds().Max.X) * scaleFactor / float64(bullyImg.Bounds().Max.X)

	newSzX := uint(float64(bullyImg.Bounds().Max.X) * scaleFactor)
	newSzY := uint(float64(bullyImg.Bounds().Max.Y) * scaleFactor)

	bullyImg = resize.Resize(newSzX, newSzY, bullyImg, resize.Lanczos3)

	draw.Draw(dst, img.Bounds(), img, image.Point{}, draw.Src)
	srcSz := img.Bounds().Size()

	w, h := bullyImg.Bounds().Max.X, bullyImg.Bounds().Max.Y

	pt := image.Point{}
	switch position {
	case "botright":
		pt = image.Point{srcSz.X - w, srcSz.Y - h}
	case "botleft":
		pt = image.Point{0, srcSz.Y - h}
	case "topright":
		pt = image.Point{srcSz.X - w, 0}
	case "topleft":
		pt = image.Point{0, 0}
	}
	rect := image.Rect(pt.X, pt.Y, srcSz.X, srcSz.Y)

	draw.DrawMask(dst, rect, bullyImg, image.Point{}, &circle{image.Point{w / 2, h / 2}, w / 2}, image.Point{}, draw.Over)
	return dst
}

// the following is ripped off of https://blog.golang.org/image-draw
type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

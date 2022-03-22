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
	Text  string  `json:"t",omitempty`
	Caps  bool    `json:"c"`
	Font  string  `json:"f",omitempty`
}

type specification struct {
	ImageURL string
	StampURL string
	Configs  []memeText
}

func (s specification) toJSON() string {
	out, _ := json.Marshal(s)
	return string(out)
}

func SpecFromJSON(input []byte) (specification, error) {
	out := specification{}
	err := json.Unmarshal(input, &out)
	return out, err
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

	b.RegisterRegexCmd(mp, bot.Message, cmdMatch, mp.message)
	b.Register(mp, bot.Help, mp.help)
	mp.registerWeb(b.DefaultConnector())

	return mp
}

var cmdMatch = regexp.MustCompile(`(?i)^meme (?P<content>.+)$`)

func (p *MemePlugin) message(r bot.Request) bool {
	minusMeme := r.Values["content"]
	log.Debug().Msgf("Calling sendMeme with text: %s", minusMeme)
	p.sendMeme(r.Conn, r.Msg.Channel, r.Msg.ChannelName, r.Msg.ID, r.Msg.User, minusMeme)
	return true
}

func (p *MemePlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
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

func (p *MemePlugin) stamp(c bot.Connector, format, id string) string {
	iconURL := ""

	for _, bully := range p.c.GetArray("meme.bully", []string{}) {
		if format == bully {
			if u, err := c.Profile(bully); err == nil {
				iconURL = u.Icon
			} else {
				log.Debug().Err(err).Msgf("could not get profile for %s", format)
			}
			formats := p.c.GetMap("meme.memes", defaultFormats)
			format = randEntry(formats)
			log.Debug().Msgf("After bullying:\nFormat: %s", format)
			break
		}
	}

	if u, err := c.Profile(id); iconURL == "" && err == nil {
		iconURL = u.Icon
	}

	return iconURL
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

	formats := p.c.GetMap("meme.memes", defaultFormats)
	imgURL, ok := formats[format]
	if !ok {
		imgURL = format
	}

	stampURL := p.stamp(c, format, from.ID)

	spec := specification{
		ImageURL: imgURL,
		StampURL: stampURL,
		Configs:  config,
	}

	encodedSpec, _ := json.Marshal(spec)

	w, h, err := p.checkMeme(imgURL)
	if err != nil {
		msg := fmt.Sprintf("Hey %v, I couldn't download that image you asked for.", from.Name)
		p.bot.Send(c, bot.Ephemeral, channel, from.ID, msg)
		return
	}
	baseURL := p.c.Get("BaseURL", ``)
	u, _ := url.Parse(baseURL)
	u.Path = path.Join(u.Path, "meme", "img")
	q := u.Query()
	q.Add("spec", string(encodedSpec))
	u.RawQuery = q.Encode()

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
	"fry":    "https://imgflip.com/s/meme/Futurama-Fry.jpg",
	"aliens": "https://imgflip.com/s/meme/Ancient-Aliens.jpg",
	"doge":   "https://imgflip.com/s/meme/Doge.jpg",
	"simply": "https://imgflip.com/s/meme/One-Does-Not-Simply.jpg",
	"wonka":  "https://imgflip.com/s/meme/Creepy-Condescending-Wonka.jpg",
	"grumpy": "https://imgflip.com/s/meme/Grumpy-Cat.jpg",
	"raptor": "https://imgflip.com/s/meme/Philosoraptor.jpg",
}

func (p *MemePlugin) findFontSize(config []memeText, fontLocation string, w, h int, sizes []float64) float64 {
	fontSize := 12.0

	m := gg.NewContext(w, h)

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
		{XPerc: 0.5, YPerc: 0.05, Caps: true},
		{XPerc: 0.5, YPerc: 0.95, Caps: true},
	}
}

func defaultFormatConfigJSON() string {
	c, _ := json.Marshal(defaultFormatConfig())
	return string(c)
}

func (p *MemePlugin) checkMeme(imgURL string) (int, int, error) {
	u, err := url.Parse(imgURL)
	if err != nil || u.Scheme == "" {
		log.Debug().Err(err).Str("imgName", imgURL).Msgf("url not detected")
		return 0, 0, fmt.Errorf("URL not valid (%s): %w", imgURL, err)
	}

	img, err := DownloadTemplate(u)
	if err != nil {
		return 0, 0, err
	}

	return img.Bounds().Dx(), img.Bounds().Dy(), err
}

func (p *MemePlugin) genMeme(spec specification) ([]byte, error) {
	fontSizes := []float64{48, 36, 24, 16, 12}

	jsonSpec := spec.toJSON()
	if cached, ok := p.images[jsonSpec]; ok {
		log.Debug().Msgf("Returning cached image for %s", jsonSpec)
		return cached.repr, nil
	}

	u, err := url.Parse(spec.ImageURL)
	if err != nil || u.Scheme == "" {
		log.Debug().Err(err).Str("imgName", spec.ImageURL).Msgf("url not detected")
		return nil, fmt.Errorf("Error with meme URL: %w", err)
	}

	log.Debug().Msgf("Attempting to download url: %s", u.String())

	img, err := DownloadTemplate(u)
	if err != nil {
		log.Debug().Msgf("failed to download image: %s", err)
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

	if spec.StampURL != "" {
		img, err = p.applyStamp(img, spec.StampURL)
		if err != nil {
			log.Error().
				Err(err).
				Interface("spec", spec).
				Msg("could not apply stamp")
		}
	}

	m := gg.NewContext(w, h)
	m.DrawImage(img, 0, 0)
	defaultFont := p.c.Get("meme.font", "impact.ttf")

	for i, c := range spec.Configs {
		if c.Caps {
			spec.Configs[i].Text = strings.ToUpper(c.Text)
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
			for _, c := range spec.Configs {
				fontLocation := c.Font
				if fontLocation == "" {
					fontLocation = defaultFont
				}
				fontSize := p.findFontSize(spec.Configs, fontLocation, w, h, fontSizes)
				m.LoadFontFace(fontLocation, fontSize)
				x := float64(w)*c.XPerc + float64(dx)
				y := float64(h)*c.YPerc + float64(dy)
				m.DrawStringAnchored(c.Text, x, y, 0.5, 0.5)
			}
		}
	}

	// Apply white fill
	m.SetHexColor("#FFF")
	for _, c := range spec.Configs {
		fontLocation := c.Font
		if fontLocation == "" {
			fontLocation = defaultFont
		}
		fontSize := p.findFontSize(spec.Configs, fontLocation, w, h, fontSizes)
		m.LoadFontFace(fontLocation, fontSize)
		x := float64(w) * c.XPerc
		y := float64(h) * c.YPerc
		m.DrawStringAnchored(c.Text, x, y, 0.5, 0.5)
	}

	i := bytes.Buffer{}
	png.Encode(&i, m.Image())
	p.images[jsonSpec] = &cachedImage{time.Now(), i.Bytes()}

	log.Debug().Msgf("Saved to %s\n", jsonSpec)

	return p.images[jsonSpec].repr, nil
}

func (p *MemePlugin) applyStamp(img image.Image, bullyURL string) (image.Image, error) {
	u, _ := url.Parse(bullyURL)
	bullyImg, err := DownloadTemplate(u)
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(img.Bounds())

	scaleFactor := p.c.GetFloat64("meme.stampScale", 0.1)
	position := p.c.GetString("meme.stampPosition", "botright")

	scaleFactor = float64(img.Bounds().Max.X) * scaleFactor / float64(bullyImg.Bounds().Max.X)

	newSzX := uint(float64(bullyImg.Bounds().Max.X) * scaleFactor)
	newSzY := uint(float64(bullyImg.Bounds().Max.Y) * scaleFactor)

	bullyImg = resize.Resize(newSzX, newSzY, bullyImg, resize.Lanczos3)

	draw.Draw(dst, img.Bounds(), img, image.Point{}, draw.Src)
	srcSz := img.Bounds().Size()

	w, h := bullyImg.Bounds().Max.X, bullyImg.Bounds().Max.Y

	pt := image.Point{srcSz.X - w, srcSz.Y - h}
	switch position {
	case "botleft":
		pt = image.Point{0, srcSz.Y - h}
	case "topright":
		pt = image.Point{srcSz.X - w, 0}
	case "topleft":
		pt = image.Point{0, 0}
	}
	rect := image.Rect(pt.X, pt.Y, srcSz.X, srcSz.Y)

	draw.DrawMask(dst, rect, bullyImg, image.Point{}, &circle{image.Point{w / 2, h / 2}, w / 2}, image.Point{}, draw.Over)
	return dst, nil
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
